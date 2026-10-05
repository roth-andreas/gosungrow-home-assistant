package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MickMake/GoUnify/Only"
	"github.com/MickMake/GoUnify/cmdHelp"
	"github.com/MickMake/GoUnify/cmdLog"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron"
	"github.com/roth-andreas/gosungrow-home-assistant/cmdHassio"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/queryDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DefaultServiceName = "GoSungrow"
	DefaultServiceArea = "Roof"
	DefaultVendor      = "Andreas Roth"
	flagMqttUsername   = "mqtt-user"
	flagMqttPassword   = "mqtt-password"
	flagMqttHost       = "mqtt-host"
	flagMqttPort       = "mqtt-port"
	startupRetryMax    = 3
)

const (
	realtimeEndpointName = "queryDeviceRealTimeDataByPsKeys"
)

var dockerDNSRetryDelays = []time.Duration{
	15 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	300 * time.Second,
}

type realtimePsKeyTarget struct {
	PsID            string
	PsKey           string
	DeviceType      int64
	SelectionSource string
}

type mqttEndpointBatch struct {
	Endpoints []string
	Args      []string
}

var mqttApiLogin = func(force bool) error {
	return cmds.Api.ApiLogin(force)
}

var mqttLoadDevices = func() (getDeviceList.Devices, error) { return cmds.Api.SunGrow.GetDeviceList() }

var mqttLoadPlantTrees = func() (iSolarCloud.PsTrees, error) {
	return cmds.Api.SunGrow.PsTreeMenu()
}

var mqttLoadDevicePoints = func() (map[string]getDevicePointAttrs.Points, error) {
	return cmds.Api.SunGrow.DevicePointAttrsByDevice()
}

//goland:noinspection GoNameStartsWithPackageName
type CmdMqtt struct {
	CmdDefault

	// HASSIO MQTT
	Username string
	Password string
	Host     string
	Port     string

	Client    *cmdHassio.Mqtt
	endpoints MqttEndPoints
	points    getDevicePointAttrs.PointsMap
	previous  map[string]*api.DataEntries

	log                 cmdLog.Log
	optionSleepDelay    time.Duration
	optionFetchSchedule time.Duration
	dockerDNSHintLogged bool
	dockerDNSErrorCount int
	dockerDNSOutageAt   time.Time
	now                 func() time.Time
	syncCycle           uint64
	currentSyncEndpoint string
	plantTopologies     map[string]iSolarCloud.PlantTopology
	plantInventories    map[string]iSolarCloud.PlantInventory
	rediscoveryPending  bool
}

func NewCmdMqtt(logLevel string) *CmdMqtt {
	var ret *CmdMqtt

	for range Only.Once {
		if logLevel == "" {
			logLevel = cmdLog.LogLevelInfoStr
		}

		ret = &CmdMqtt{
			CmdDefault: CmdDefault{
				Error:   nil,
				cmd:     nil,
				SelfCmd: nil,
			},

			log:                 cmdLog.New(logLevel),
			optionSleepDelay:    time.Second * 40, // Takes up to 40 seconds for data to come in.
			optionFetchSchedule: time.Minute * 5,
			previous:            make(map[string]*api.DataEntries, 0),
			plantTopologies:     make(map[string]iSolarCloud.PlantTopology),
			plantInventories:    make(map[string]iSolarCloud.PlantInventory),
			now:                 time.Now,
		}
	}

	return ret
}

func (c *CmdMqtt) AttachCommand(cmd *cobra.Command) *cobra.Command {
	for range Only.Once {
		if cmd == nil {
			break
		}
		c.cmd = cmd

		// ******************************************************************************** //
		var cmdRoot = &cobra.Command{
			Use:                   "mqtt",
			Aliases:               []string{""},
			Annotations:           map[string]string{"group": "MQTT"},
			Short:                 "Connect to a HASSIO broker.",
			Long:                  "Connect to a HASSIO broker.",
			DisableFlagParsing:    false,
			DisableFlagsInUseLine: false,
			PreRunE:               nil,
			RunE:                  c.CmdMqtt,
			Args:                  cobra.MinimumNArgs(1),
		}
		cmd.AddCommand(cmdRoot)
		cmdRoot.Example = cmdHelp.PrintExamples(cmdRoot, "run", "sync")

		// ******************************************************************************** //
		var cmdMqttRun = &cobra.Command{
			Use:                   "run",
			Aliases:               []string{""},
			Annotations:           map[string]string{"group": "MQTT"},
			Short:                 "One-off sync to a HASSIO broker.",
			Long:                  "One-off sync to a HASSIO broker.",
			DisableFlagParsing:    false,
			DisableFlagsInUseLine: false,
			PreRunE: func(cmd *cobra.Command, args []string) error {
				cmds.Error = cmds.SunGrowArgs(cmd, args)
				if cmds.Error != nil {
					return cmds.Error
				}
				cmds.Error = cmds.Mqtt.MqttArgs(cmd, args)
				if cmds.Error != nil {
					return cmds.Error
				}
				return nil
			},
			RunE: cmds.Mqtt.CmdMqttRun,
			Args: cobra.RangeArgs(0, 1),
		}
		cmdRoot.AddCommand(cmdMqttRun)
		cmdMqttRun.Example = cmdHelp.PrintExamples(cmdMqttRun, "")

		// ******************************************************************************** //
		var cmdMqttSync = &cobra.Command{
			Use:                   "sync",
			Aliases:               []string{""},
			Annotations:           map[string]string{"group": "MQTT"},
			Short:                 "Sync to a HASSIO MQTT broker.",
			Long:                  "Sync to a HASSIO MQTT broker.",
			DisableFlagParsing:    false,
			DisableFlagsInUseLine: false,
			PreRunE: func(cmd *cobra.Command, args []string) error {
				cmds.Error = cmds.SunGrowArgs(cmd, args)
				if cmds.Error != nil {
					return cmds.Error
				}
				cmds.Error = cmds.Mqtt.MqttArgs(cmd, args)
				if cmds.Error != nil {
					return cmds.Error
				}
				return nil
			},
			RunE: cmds.Mqtt.CmdMqttSync,
			Args: cobra.RangeArgs(0, 1),
		}
		cmdRoot.AddCommand(cmdMqttSync)
		cmdMqttSync.Example = cmdHelp.PrintExamples(cmdMqttSync, "", "all")
	}
	return c.SelfCmd
}

func (c *CmdMqtt) AttachFlags(cmd *cobra.Command, viper *viper.Viper) {
	for range Only.Once {
		cmd.PersistentFlags().StringVarP(&c.Username, flagMqttUsername, "", "", "HASSIO: mqtt username.")
		viper.SetDefault(flagMqttUsername, "")
		cmd.PersistentFlags().StringVarP(&c.Password, flagMqttPassword, "", "", "HASSIO: mqtt password.")
		viper.SetDefault(flagMqttPassword, "")
		cmd.PersistentFlags().StringVarP(&c.Host, flagMqttHost, "", "", "HASSIO: mqtt host.")
		viper.SetDefault(flagMqttHost, "")
		cmd.PersistentFlags().StringVarP(&c.Port, flagMqttPort, "", "", "HASSIO: mqtt port.")
		viper.SetDefault(flagMqttPort, "")
	}
}

func (c *CmdMqtt) MqttArgs(_ *cobra.Command, _ []string) error {
	for range Only.Once {
		c.log.Info("Connecting to MQTT HASSIO Service...\n")
		c.Client = cmdHassio.New(cmdHassio.Mqtt{
			ClientId:     DefaultServiceName,
			EntityPrefix: DefaultServiceName,
			Username:     c.Username,
			Password:     c.Password,
			Host:         c.Host,
			Port:         c.Port,
		})
		c.Error = c.Client.GetError()
		if c.Error != nil {
			break
		}

		c.log.Info("Connecting to SunGrow...\n")
		c.Error = c.retryStartupRecoverable("device discovery", func() error {
			var err error
			c.Client.SungrowDevices, err = cmds.Api.SunGrow.GetDeviceList()
			return err
		})
		if c.Error != nil {
			break
		}

		c.log.Info("Found SunGrow %d devices\n", len(c.Client.SungrowDevices))
		c.log.Info("SunGrow device types: %s\n", formatSungrowDeviceTypeSummary(c.Client.SungrowDevices))
		c.logRealtimePsKeySelections(c.Client.SungrowDevices)
		c.refreshPlantTopologies()
		c.Client.DeviceName = DefaultServiceName
		_, c.Error = c.Client.SetDeviceConfig(
			c.Client.DeviceName, c.Client.DeviceName,
			"virtual", "virtual", "", DefaultServiceName,
			DefaultServiceArea,
		)
		if c.Error != nil {
			break
		}

		_, c.Error = c.Client.SetDeviceConfig(
			c.Client.DeviceName, c.Client.DeviceName,
			"system", "system", "", DefaultServiceName,
			DefaultServiceArea,
		)
		if c.Error != nil {
			break
		}

		for _, psId := range c.Client.SungrowDevices {
			// ca.Error = ca.Mqtt.Mqtt.SetDeviceConfig(DefaultServiceName, strconv.FormatInt(id, 10), DefaultServiceName, model[0], "Sungrow", DefaultServiceArea)
			parent := psId.PsId.String()
			if parent == psId.PsKey.Value() {
				parent = c.Client.DeviceName
			}

			_, c.Error = c.Client.SetDeviceConfig(
				DefaultServiceName, DefaultServiceName,
				psId.PsId.String(), psId.FactoryName.Value(), psId.FactoryName.Value(), psId.FactoryName.Value(),
				DefaultServiceArea,
			)
			if c.Error != nil {
				break
			}

			_, c.Error = c.Client.SetDeviceConfig(
				DefaultServiceName, parent,
				psId.PsKey.Value(), psId.DeviceName.Value(), psId.DeviceModel.Value(), psId.FactoryName.Value(),
				DefaultServiceArea,
			)
			if c.Error != nil {
				break
			}

			c.Client.SungrowPsIds[psId.PsId] = true
		}

		c.Error = c.Client.Connect()
		if c.Error != nil {
			break
		}

		c.Error = c.Options()
		if c.Error != nil {
			break
		}

		c.log.Info("Caching Sungrow metadata...\n")
		c.Error = c.retryStartupRecoverable("metadata discovery", func() error {
			return c.GetEndPoints()
		})
		if c.Error != nil {
			break
		}

		c.Error = c.retryStartupRecoverable("device point discovery", func() error {
			return c.refreshPlantInventories()
		})
		if c.Error != nil {
			break
		}
		c.log.Info("Cached %d Sungrow data points...\n", len(c.points))
	}

	return c.Error
}

func (c *CmdMqtt) CmdMqtt(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

func (c *CmdMqtt) CmdMqttRun(_ *cobra.Command, _ []string) error {
	for range Only.Once {
		c.Error = c.Cron()
		if c.Error != nil {
			break
		}

		c.log.Info("Starting ticker...\n")
		c.log.Info("Fetch Schedule: %s\n", c.GetFetchSchedule())
		c.log.Info("Sleep Delay:    %s\n", c.GetSleepDelay())
		for {
			delay := c.nextSyncDelay()
			c.log.Debug("Next sync in %s\n", delay)
			timer := time.NewTimer(delay)
			<-timer.C
			c.log.Debug("Update: %s\n", c.now().String())
			c.Error = c.Cron()
			if c.Error != nil {
				break
			}
		}
	}

	return c.Error
}

func (c *CmdMqtt) CmdMqttSync(_ *cobra.Command, args []string) error {
	cronString := "*/5 * * * *"
	if len(args) > 0 {
		if len(args) < 5 {
			return errors.New("cron expression requires five fields")
		}
		cronString = strings.ReplaceAll(strings.Join(args[:5], " "), ".", "*")
	}
	scheduler := gocron.NewScheduler(time.UTC)
	ticks := make(chan struct{}, 1)
	_, err := scheduler.Cron(cronString).SingletonMode().Do(func() {
		select {
		case ticks <- struct{}{}:
		default:
		}
	})
	if err != nil {
		return err
	}
	if err = c.Cron(); err != nil {
		return err
	}
	scheduler.StartAsync()
	defer scheduler.Stop()
	c.log.Info("Created job schedule using '%s'\n", cronString)
	return c.runScheduledSync(ticks, c.Cron, time.After)
}

// runScheduledSync serializes cron and DNS attempts. Cron ticks cannot shorten
// an active DNS deadline, and fatal cycle errors terminate the scheduler owner.
func (c *CmdMqtt) runScheduledSync(ticks <-chan struct{}, attempt func() error, after func(time.Duration) <-chan time.Time) error {
	for {
		var retry <-chan time.Time
		if c.dockerDNSErrorCount > 0 {
			retry = after(c.nextSyncDelay())
		}
	wait:
		for {
			select {
			case <-ticks:
				if retry == nil {
					break wait
				}
			case <-retry:
				break wait
			}
		}
		if err := attempt(); err != nil {
			return err
		}
		if retry != nil {
		drain:
			for {
				select {
				case <-ticks:
				default:
					break drain
				}
			}
		}
	}
}

// -------------------------------------------------------------------------------- //

func (c *CmdMqtt) Cron() error {
	if c == nil {
		return errors.New("mqtt not available")
	}
	if cmds.Api.SunGrow == nil {
		return errors.New("sungrow not available")
	}
	if c.now == nil {
		c.now = time.Now
	}
	sg := cmds.Api.SunGrow
	sg.BeginRetry()
	if c.isRecoverableGatewayError(c.Error) {
		c.Error = nil
	}
	if sg.Error != nil {
		return sg.Error
	}
	if c.Error != nil {
		return c.Error
	}
	if c.Client.IsFirstRun() {
		c.Client.UnsetFirstRun()
	} else if c.dockerDNSErrorCount == 0 {
		time.Sleep(c.optionSleepDelay)
	}
	started := c.now()
	c.syncCycle++
	c.log.Info("Starting iSolarCloud sync cycle %d.\n", c.syncCycle)
	c.currentSyncEndpoint = "authentication"
	c.Error = c.syncAttempt(c.Client.IsNewDay())
	if c.Error != nil {
		if c.isTokenInvalidError(c.Error) {
			cmds.Api.SunGrow.RequireAuthentication()
		}
		c.log.Info("iSolarCloud sync cycle %d failed at %s after %s: %s\n", c.syncCycle, c.currentSyncEndpoint, c.now().Sub(started).Round(time.Millisecond), c.Error)
		if c.isRecoverableGatewayError(c.Error) {
			if c.isDockerDNSError(c.Error) {
				c.recordDockerDNSError(c.Error)
			} else {
				c.log.Info("Recoverable API/gateway error during sync. Keeping service alive and retrying on next cycle: %s\n", c.Error)
			}
			c.Error = nil
			return nil
		}
		c.log.Error("%s\n", c.Error)
		return c.Error
	}
	c.clearDockerDNSOutage()
	c.Client.LastRefresh = c.now()
	c.log.Info("Completed iSolarCloud sync cycle %d in %s.\n", c.syncCycle, c.now().Sub(started).Round(time.Millisecond))
	return nil
}

func (c *CmdMqtt) syncAttempt(newDay bool) error {
	loginUsed := false
	recover := func() error {
		loginUsed = true
		c.currentSyncEndpoint = "authentication"
		if err := mqttApiLogin(true); err != nil {
			return err
		}
		c.rediscoveryPending = true
		return nil
	}
	if cmds.Api.SunGrow.NeedLogin {
		if err := recover(); err != nil {
			return err
		}
	}
	if c.rediscoveryPending {
		if err := c.rediscover(); err != nil {
			return err
		}
	}
	err := c.collectAndPublish(newDay)
	if err == nil || !c.isTokenInvalidError(err) {
		return err
	}
	cmds.Api.SunGrow.RequireAuthentication()
	if loginUsed {
		return err
	}
	if err = recover(); err != nil {
		return err
	}
	if err = c.rediscover(); err != nil {
		return err
	}
	// One complete collection replay; any failure is handled by Cron.
	return c.collectAndPublish(newDay)
}

func (c *CmdMqtt) rediscover() error {
	c.currentSyncEndpoint = "device rediscovery"
	devices, err := mqttLoadDevices()
	if err != nil {
		return err
	}
	staged := *c
	client := *c.Client
	client.SungrowDevices = devices
	staged.Client = &client
	if err = staged.refreshPlantInventories(); err != nil {
		return err
	}
	c.currentSyncEndpoint = "plant topology refresh"
	staged.refreshPlantTopologies()
	if cmds.Api.SunGrow.NeedLogin {
		if cmds.Api.SunGrow.Error != nil {
			return cmds.Api.SunGrow.Error
		}
		return errors.New("need to login again")
	}
	c.Client.SungrowDevices = devices
	c.plantInventories, c.points, c.plantTopologies = staged.plantInventories, staged.points, staged.plantTopologies
	c.rediscoveryPending = false
	return nil
}

func (c *CmdMqtt) collectAndPublish(newDay bool) error {
	batches := buildMqttEndpointBatches(c.endpoints.Names(), c.getRealtimePsKeyTargets())
	for _, batch := range batches {
		if err := c.collectAndPublishBatch(batch, newDay); err != nil {
			return err
		}
	}

	return nil
}

func (c *CmdMqtt) collectAndPublishBatch(batch mqttEndpointBatch, newDay bool) error {
	if len(batch.Endpoints) == 0 {
		return nil
	}

	c.currentSyncEndpoint = strings.Join(batch.Endpoints, ", ")
	c.log.Info("Sync cycle %d: requesting %s.\n", c.syncCycle, c.currentSyncEndpoint)

	data := cmds.Api.SunGrow.NewSunGrowData()
	data.SetCacheTimeout(c.optionFetchSchedule)

	data.SetPsIds()
	if data.Error != nil {
		return data.Error
	}

	data.SetArgs(batch.Args...)
	data.SetEndpoints(batch.Endpoints...)
	if err := data.GetData(); err != nil {
		return err
	}

	resultKeys := make([]string, 0, len(data.Results))
	for key := range data.Results {
		resultKeys = append(resultKeys, key)
	}
	sort.Strings(resultKeys)
	for _, key := range resultKeys {
		result := data.Results[key]
		if result.EndPointName.String() == "queryDeviceList" {
			if err := c.publishPlantPVPower(&result.Response.Data, newDay); err != nil {
				return err
			}
		}
		if err := c.Update(result.EndPointName.String(), result.Response.Data, newDay); err != nil {
			return err
		}
	}
	c.log.Info("Sync cycle %d: completed %s.\n", c.syncCycle, c.currentSyncEndpoint)

	return nil
}

func (c *CmdMqtt) refreshPlantTopologies() {
	trees, err := mqttLoadPlantTrees()
	if err != nil && cmds.Api.SunGrow != nil && !cmds.Api.SunGrow.NeedLogin {
		// Topology failure is explicitly nonfatal; retire its operation error.
		cmds.Api.SunGrow.Error = nil
	}
	topologies := iSolarCloud.BuildPlantTopologies(trees)
	plantIDs := mqttPlantIDs(c.Client.SungrowDevices)
	for _, psID := range plantIDs {
		topology, ok := topologies[psID]
		if !ok {
			topology = iSolarCloud.PlantTopology{PsID: psID, Complete: false, Reason: "plant topology unavailable"}
			topologies[psID] = topology
		}
		if err != nil || !topology.Complete {
			reason := topology.Reason
			if reason == "" && err != nil {
				reason = err.Error()
			}
			c.log.Info("Plant PV aggregation warning: ps_id=%s topology unavailable: %s\n", psID, reason)
		}
	}
	c.plantTopologies = topologies
}

func (c *CmdMqtt) refreshPlantInventories() error {
	byDevice, err := mqttLoadDevicePoints()
	if err != nil {
		return err
	}
	inventories := iSolarCloud.BuildPlantInventories(c.Client.SungrowDevices, byDevice)
	c.plantInventories = inventories
	c.points = make(getDevicePointAttrs.PointsMap)
	keys := make([]string, 0, len(byDevice))
	for key := range byDevice {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for i := range byDevice[key] {
			point := byDevice[key][i]
			c.points[point.Id.String()] = &point
		}
	}
	return nil
}

func mqttPlantIDs(devices getDeviceList.Devices) []string {
	seen := make(map[string]bool)
	for _, device := range devices {
		psID := strings.TrimSpace(device.PsId.String())
		if psID != "" {
			seen[psID] = true
		}
	}
	plantIDs := make([]string, 0, len(seen))
	for psID := range seen {
		plantIDs = append(plantIDs, psID)
	}
	sort.Strings(plantIDs)
	return plantIDs
}

// publishPlantPVPower evaluates exactly one requested plant snapshot and routes its
// canonical entity through the ordinary retained discovery/state pipeline.
func (c *CmdMqtt) publishPlantPVPower(data *api.DataMap, newDay bool) error {
	if data.EndPoint == nil {
		return nil
	}
	endpoint := queryDeviceList.Assert(data.EndPoint)
	psID := endpoint.Request.PsId.String()
	topology := c.plantTopologies[psID]
	topology.PsID = psID
	if inventory, ok := c.plantInventories[psID]; ok {
		topology.Inventory = inventory
	}
	result := iSolarCloud.AddCanonicalPlantPVPower(data, topology)
	outcome, reason := "suppressed", result.Reason
	var err error
	if result.Added {
		key := "virtual." + psID + ".pv_power"
		entry := data.Map[key].GetEntry(api.LastEntry)
		switch {
		case !c.endpoints.IsOK(entry):
			outcome, reason = "filtered", "endpoint_filter"
		default:
			if _, known := c.Client.MqttDevices[psID]; !known {
				outcome, reason = "filtered", "unknown_parent"
			} else {
				canonical := api.NewDataMap()
				canonical.Map[key] = data.Map[key]
				err = c.Update(queryDeviceList.EndPointName, canonical, newDay)
				if err != nil {
					outcome, reason = "publication_failed", "publication_failed"
				} else {
					outcome, reason = "published", "none"
				}
			}
		}
		// Do not publish the same canonical point again with the remaining snapshot.
	}
	delete(data.Map, "virtual."+psID+".pv_power")
	expected := "unknown"
	if result.ExpectedKnown {
		expected = fmt.Sprint(result.Expected)
	}
	c.log.Info("Plant PV aggregation: ps_id=%s cycle=%d outcome=%s source=%s expected=%s received=%d valid_ac=%d valid_dc=%d reason=%s\n",
		psID, c.syncCycle, outcome, result.Source, expected, result.Received, result.ValidAC, result.ValidDC, reason)
	for _, record := range result.Records {
		c.log.Debug("Plant PV aggregation detail: device_key=%s device_type=%d point_id=%s unit=%s valid=%t rejection=%s\n",
			record.DeviceKey, record.DeviceType, record.PointID, record.Unit, record.Valid, record.Reason)
	}
	c.log.Debug("Plant PV aggregation details: omitted=%d\n", result.OmittedRecords)
	return err
}

func (c *CmdMqtt) getRealtimePsKeyTargets() []realtimePsKeyTarget {
	return selectRealtimePsKeyTargets(c.Client.SungrowDevices)
}

func (c *CmdMqtt) isTokenInvalidError(err error) bool {
	if err == nil {
		return false
	}
	// A classified sequence describes its current outcome; earlier token text
	// in its causal summary cannot authorize another login or override fatality.
	var classified interface {
		FailureClass() iSolarCloud.FailureClass
	}
	if errors.As(err, &classified) {
		return false
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "er_token_login_invalid") {
		return true
	}
	if strings.Contains(msg, "need to login again") {
		return true
	}

	return false
}

func (c *CmdMqtt) isRecoverableGatewayError(err error) bool {
	if err == nil {
		return false
	}
	return iSolarCloud.ShouldRecoverGatewayError(err)
}

func (c *CmdMqtt) retryStartupRecoverable(step string, fn func() error) error {
	var err error
	for attempt := 1; attempt <= startupRetryMax; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !c.isRecoverableGatewayError(err) {
			return err
		}
		if c.isDockerDNSError(err) {
			c.logDockerDNSHint(err)
			return err
		}

		c.log.Info("Recoverable API/gateway error during %s (attempt %d/%d). Re-authenticating: %s\n", step, attempt, startupRetryMax, err)
		c.logDockerDNSHint(err)
		if loginErr := mqttApiLogin(true); loginErr != nil {
			return loginErr
		}

		if attempt < startupRetryMax {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return err
}

func (c *CmdMqtt) isDockerDNSError(err error) bool {
	return iSolarCloud.IsDockerDNSError(err)
}

func (c *CmdMqtt) logDockerDNSHint(err error) {
	if c.dockerDNSHintLogged || !c.isDockerDNSError(err) {
		return
	}

	c.dockerDNSHintLogged = true
	c.log.Info("Docker DNS resolver 127.0.0.11 could not resolve iSolarCloud. This is usually a Home Assistant/Docker DNS issue; check Settings > System > Network DNS and restart Home Assistant/Docker if other apps also cannot resolve hostnames.\n")
}

func (c *CmdMqtt) nextSyncDelay() time.Duration {
	if c.dockerDNSErrorCount == 0 {
		return c.optionFetchSchedule
	}
	idx := c.dockerDNSErrorCount - 1
	if idx >= len(dockerDNSRetryDelays) {
		idx = len(dockerDNSRetryDelays) - 1
	}
	return dockerDNSRetryDelays[idx]
}

func (c *CmdMqtt) recordDockerDNSError(err error) {
	if c.dockerDNSErrorCount == 0 {
		c.dockerDNSOutageAt = c.now()
		c.logDockerDNSHint(err)
	}
	c.dockerDNSErrorCount++
	c.log.Info("iSolarCloud DNS unavailable (attempt %d). MQTT remains connected; next retry in %s.\n", c.dockerDNSErrorCount, c.nextSyncDelay())
}

func (c *CmdMqtt) clearDockerDNSOutage() {
	if c.dockerDNSErrorCount == 0 {
		return
	}
	duration := c.now().Sub(c.dockerDNSOutageAt).Round(time.Second)
	c.log.Info("iSolarCloud DNS recovered after %s. Resuming the normal %s sync schedule.\n", duration, c.GetFetchSchedule())
	c.dockerDNSErrorCount = 0
	c.dockerDNSOutageAt = time.Time{}
	c.dockerDNSHintLogged = false
}

func (c *CmdMqtt) Update(endpoint string, data api.DataMap, newDay bool) error {
	c.Error = nil
	for range Only.Once {
		// Also getPowerStatistics, getHouseholdStoragePsReport, getPsList, getUpTimePoint,
		c.log.Info("Syncing %d entries with HASSIO from %s.\n", len(data.Map), endpoint)

		for _, o := range data.Sort() {
			_, previouslySeen := c.previous[o]
			refreshConfig := newDay || !previouslySeen

			entries := data.Map[o]
			r := entries.GetEntry(api.LastEntry) // Gets the last entry

			// if strings.Contains(r.EndPoint, "active") {
			// 	fmt.Printf("EMPTY[%s] -> %s\n", r.EndPoint, r.Value.String())
			// }

			if _, ok := c.previous[o]; ok {
				previous := c.previous[o].GetEntry(api.LastEntry)
				if r.Value.String() != previous.Value.String() {
					refreshConfig = true
				}
			}

			if !r.Point.Valid {
				// Any point that shouldn't be passed through to MQTT is ignored
				// - includes child points of an aggregate of several points.
				c.log.PlainInfo("-")
				c.log.Debug("Ignored: [%s] = '%s'\n", r.EndPoint, r.Value.String())
				continue
			}

			if !c.endpoints.IsOK(r) {
				continue
			}

			if !r.Value.Valid {
				// Point doesn't have a valid value.
				// Usually a float or int that cannot be converted or is empty.
				c.log.PlainInfo("?")
				c.log.Debug("Invalid: [%s] = '%s'\n", r.EndPoint, r.Value.String())
				continue
			}

			_ = c.UpdatePoint(r)
			entityUnit := normalizeEntityMeasurement(r)

			id := r.EndPoint
			name := c.friendlyEntityName(r)

			// if r.Point.Unit == "" {
			// 	r.Point.Unit = r.Point.ValueType
			// }
			// if r.Point.Unit == "Bool" {
			// 	r.Point.Unit = mmHa.LabelBinarySensor
			// }
			// if r.Point.ValueType == "Bool" {
			// 	r.Point.Unit = mmHa.LabelBinarySensor
			// }

			re := cmdHassio.EntityConfig{
				Name:       name, // mmHa.JoinStringsForName(" - ", id), // r.Point.Name, // PointName,
				SubName:    "",
				ParentId:   r.EndPoint,
				ParentName: r.Parent.Key,
				UniqueId:   r.Point.Id,
				// UniqueId:    r.Id,
				FullId: id, // string(r.FullId),	// WAS r.Point.FullId
				// FullName:    r.Point.Name,
				Units: entityUnit,
				// ValueName:   r.Point.Description,
				// ValueName:   r.Point.Id,
				DeviceClass: "",
				StateClass:  r.Point.UpdateFreq,
				Value:       &r.Value,
				Point:       r.Point,
				Icon:        r.Current.PointIcon(),
				UpdateFreq:  r.Current.DataStructure.PointUpdateFreq,

				// LastReset:   "",
				// LastResetValueTemplate: "",
			}

			re.FixConfig()
			if re.LastResetValueTemplate != "" {
				re.LastReset = r.Point.WhenReset(r.Date)
			}

			// if strings.Contains(r.EndPoint, "active") {
			// 	fmt.Printf("EMPTY[%s] -> %s\n", r.EndPoint, r.Value.String())
			// }

			if refreshConfig {
				c.log.PlainInfo("C")
				c.log.Debug("Config: [%s]\n", r.EndPoint)
				c.Error = c.Client.BinarySensorPublishConfig(re)
				if c.Error != nil {
					break
				}

				c.Error = c.Client.SensorPublishConfig(re)
				if c.Error != nil {
					break
				}
			}

			c.log.PlainInfo("U")
			c.log.Debug("Update: [%s] = '%s' %s\n", r.EndPoint, r.Value.String(), r.Value.Unit())
			c.Error = c.Client.BinarySensorPublishValue(re)
			if c.Error != nil {
				break
			}

			c.Error = c.Client.SensorPublishValue(re)
			if c.Error != nil {
				break
			}
			c.previous[o] = entries

		}
		c.log.PlainInfo("\n")
	}
	return c.Error
}

// normalizeEntityMeasurement keeps the published state and discovery metadata
// on one unit. The point synchronization is intentionally limited to the
// explicit reactive-power allowlist so all unrelated sensor behavior remains
// unchanged.
func normalizeEntityMeasurement(entry *api.DataEntry) string {
	if entry == nil || entry.Point == nil {
		return ""
	}

	reactivePower := valueTypes.IsReactivePowerUnit(entry.Value.Unit())
	entry.Value.UnitValueFix()
	if reactivePower {
		entry.Point.Unit = entry.Value.Unit()
		entry.Point.ValueType = entry.Value.Type()
	}
	return entry.Point.Unit
}

func (c *CmdMqtt) friendlyEntityName(r *api.DataEntry) string {
	group := strings.TrimSpace(r.Point.GroupName)
	desc := strings.TrimSpace(r.Point.Description)

	switch {
	case group != "" && desc != "":
		if strings.EqualFold(group, desc) {
			return cmdHassio.JoinStringsForName(" - ", group)
		}
		return cmdHassio.JoinStringsForName(" - ", group, desc)
	case desc != "":
		return cmdHassio.JoinStringsForName(" - ", desc)
	case group != "":
		return cmdHassio.JoinStringsForName(" - ", group)
	case r.Point.Id != "":
		return r.Point.Id
	default:
		return r.EndPoint
	}
}

func (c *CmdMqtt) GetEndPoints() error {
	for range Only.Once {
		fn := filepath.Join(cmds.ConfigDir, "mqtt_endpoints.json")
		if _, err := os.Stat(fn); errors.Is(err, os.ErrNotExist) {
			c.Error = os.WriteFile(fn, []byte(DefaultMqttFile), 0644)
			if c.Error != nil {
				break
			}
		} else if err != nil {
			c.Error = err
			break
		}

		raw, err := os.ReadFile(fn)
		if err != nil {
			c.Error = err
			break
		}
		c.Error = json.Unmarshal(raw, &c.endpoints)
		if c.Error != nil {
			break
		}
		endpointsChanged, err := mergeDefaultMqttEndpoints(&c.endpoints)
		if err != nil {
			c.Error = err
			break
		}
		if endpointsChanged {
			updated, err := json.MarshalIndent(c.endpoints, "", "\t")
			if err != nil {
				c.Error = err
				break
			}
			updated = append(updated, '\n')
			c.Error = os.WriteFile(fn, updated, 0644)
			if c.Error != nil {
				break
			}
			c.log.Info("Updated MQTT endpoint config with required GoSungrow defaults.\n")
		}
		c.log.Info("Loaded MQTT endpoints: %s\n", strings.Join(sortedStrings(c.endpoints.Names()), ", "))

		// All := []string{ "queryDeviceList", "getPsList", "getPsDetailWithPsType", "getPsDetail", "getKpiInfo"}
		// All := []string{ "queryDeviceList", "getPsList", "getPsDetailWithPsType", "getPsDetail", "getKpiInfo"}	//, queryMutiPointDataList, getDevicePointMinuteDataList }
		// All := []string{ "WebIscmAppService.getDeviceModel" }
		for name := range c.endpoints {
			_, c.Error = c.Client.SetDeviceConfig(
				DefaultServiceName, DefaultServiceName,
				name, DefaultServiceName+"."+name, DefaultServiceName, DefaultVendor,
				DefaultServiceArea,
			)
			if c.Error != nil {
				break
			}
		}
	}
	return c.Error
}

// UpdatePoint - Set Point values to something resembling sanity based off the points metadata.
func (c *CmdMqtt) UpdatePoint(entry *api.DataEntry) error {
	for range Only.Once {
		// if !c.points.Exists(entry.Point.Id) {
		// 	c.LogDebug("Point Meta: %s - Not found.\n", entry.Point.Id)
		// 	break
		// }
		p := c.points.Get(entry.Point.Id)
		if p == nil {
			c.log.Debug("Point Meta: %s - Not found.\n", entry.Point.Id)
			break
		}

		// {
		// 	fmt.Printf("entry.Point: %s - FOUND - %v\n", entry.Point.Id, p)
		// 	// fmt.Printf("\tValue   - Description:'-'\t\tUnit:'%s'\tGroupName:'-'\tValueType:'%s'\n",
		// 	// 	r.Value.UnitValue, r.Point.ValueType)
		// 	fmt.Printf("\tDescription:'%s'\tPointName:'%s' - SAME:%t\n",
		// 		entry.Point.Description, entry.Current.DataStructure.PointName, entry.Current.DataStructure.PointName == entry.Point.Description)
		// 	fmt.Printf("\tUnit:'%s'\tPointUnit:'%s' - SAME:%t\n",
		// 		entry.Point.Unit, entry.Current.DataStructure.PointUnit, entry.Current.DataStructure.PointUnit == entry.Point.Unit)
		// 	fmt.Printf("\tGroupName:'%s'\tPointGroupName:'%s' - SAME:%t\n",
		// 		entry.Point.GroupName, entry.Current.DataStructure.PointGroupName, entry.Current.DataStructure.PointGroupName == entry.Point.GroupName)
		// 	fmt.Printf("\tValueType:'%s'\tValueType:'%s' - SAME:%t\n",
		// 		entry.Point.ValueType, entry.Current.DataStructure.ValueType, entry.Current.DataStructure.ValueType == entry.Point.ValueType)
		// }

		// // If Point description matches ...
		// if p.Name.String() == entry.Point.Description {
		// 	break
		// }
		//
		// // If Point unit matches ...
		// if p.Unit.String() == entry.Point.Unit {
		// 	break
		// }

		// Unit
		if entry.Value.Unit() == "" {
			entry.Value.SetUnit(p.Unit.String())
			// fmt.Printf("[%s] -> %s\n", entry.EndPoint, entry.Value.String())
		}
		if entry.Point.Unit == "" {
			entry.Point.Unit = p.Unit.String()
			// fmt.Printf("[%s] -> %s\n", entry.EndPoint, entry.Value.String())
		}
		if entry.Point.Unit == "" {
			entry.Point.Unit = entry.Point.ValueType
		}
		// if entry.Point.Unit == "Bool" {
		// 	entry.Point.Unit = mmHa.LabelBinarySensor
		// }
		// if entry.Point.ValueType == "Bool" {
		// 	entry.Point.Unit = mmHa.LabelBinarySensor
		// }
		// if entry.Value.TypeValue == "Bool" {
		// 	entry.Value.UnitValue = mmHa.LabelBinarySensor
		// }

		// Parent
		if len(entry.Point.Parents.Map) == 0 {
		}
		if entry.Parent.Key == "" {
		}

		// GroupName
		if entry.Point.GroupName == "" {
			entry.Point.Description = p.Name.String()
			entry.Point.GroupName = p.PointGroupName
		}

		// ValueType
		if entry.Point.ValueType == "" {
			entry.Point.ValueType = p.UnitType.String()
		}
	}

	return c.Error
}

const (
	OptionLogLevel      = "loglevel"
	OptionFetchSchedule = "fetchschedule"
	OptionSleepDelay    = "sleepdelay"
	OptionServiceState  = "servicestate"
)

func (c *CmdMqtt) Options() error {
	for range Only.Once {
		c.Error = c.Client.CreateOption(OptionLogLevel, "Log Level",
			c.optionFuncLogLevel,
			cmdLog.LogLevelErrorStr, cmdLog.LogLevelWarningStr, cmdLog.LogLevelInfoStr, cmdLog.LogLevelDebugStr)
		if c.Error != nil {
			break
		}
		c.Error = c.Client.SetOption(OptionLogLevel, c.log.GetLogLevel())
		if c.Error != nil {
			break
		}

		c.Error = c.Client.CreateOption(OptionFetchSchedule, "Fetch Schedule",
			c.optionFuncFetchSchedule,
			"2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "10m")
		if c.Error != nil {
			break
		}
		c.Error = c.Client.SetOption(OptionFetchSchedule, c.GetFetchSchedule())
		if c.Error != nil {
			break
		}

		c.Error = c.Client.CreateOption(OptionSleepDelay, "Sleep Delay After Schedule",
			c.optionFuncSleepDelay,
			"0s", "10s", "20s", "30s", "40s", "50s", "60s")
		if c.Error != nil {
			break
		}
		c.Error = c.Client.SetOption(OptionSleepDelay, c.GetSleepDelay())
		if c.Error != nil {
			break
		}

		c.Error = c.Client.CreateOption(OptionServiceState, "Service State",
			c.optionFuncServiceState,
			"Run", "Restart", "Stop")
		if c.Error != nil {
			break
		}
		c.Error = c.Client.SetOption(OptionServiceState, "Run")
		if c.Error != nil {
			break
		}
	}
	return c.Error
}

func (c *CmdMqtt) optionFuncLogLevel(_ mqtt.Client, msg mqtt.Message) {
	for range Only.Once {
		request := strings.ToLower(string(msg.Payload()))
		c.log.Info("Option[%s] set to '%s'\n", OptionLogLevel, request)
		c.Error = c.Client.SetOption(OptionLogLevel, request)
		if c.Error != nil {
			c.log.Error("%s\n", c.Error)
			break
		}
		c.log.SetLogLevel(request)
	}
}

func (c *CmdMqtt) optionFuncFetchSchedule(_ mqtt.Client, msg mqtt.Message) {
	for range Only.Once {
		request := strings.ToLower(string(msg.Payload()))
		c.log.Info("Option[%s] set to '%s'\n", OptionFetchSchedule, request)
		c.optionFetchSchedule, c.Error = time.ParseDuration(request)
		if c.Error != nil {
			c.log.Error("%s\n", c.Error)
			break
		}

		c.Error = c.Client.SetOption(OptionFetchSchedule, c.GetFetchSchedule())
		if c.Error != nil {
			c.log.Error("%s\n", c.Error)
			break
		}

		// c.optionCacheTimeout = c.optionFetchSchedule
	}
}

func (c *CmdMqtt) GetFetchSchedule() string {
	return fmt.Sprintf("%.0fm", c.optionFetchSchedule.Minutes())
}

func (c *CmdMqtt) optionFuncSleepDelay(_ mqtt.Client, msg mqtt.Message) {
	for range Only.Once {
		request := strings.ToLower(string(msg.Payload()))
		c.log.Info("Option[%s] set to '%s'\n", OptionSleepDelay, request)
		c.optionSleepDelay, c.Error = time.ParseDuration(request)
		if c.Error != nil {
			c.log.Error("%s\n", c.Error)
			break
		}

		c.Error = c.Client.SetOption(OptionSleepDelay, request)
		if c.Error != nil {
			c.log.Error("%s\n", c.Error)
			break
		}
	}
}

func (c *CmdMqtt) GetSleepDelay() string {
	return fmt.Sprintf("%.0fs", c.optionSleepDelay.Seconds())
}

func (c *CmdMqtt) optionFuncServiceState(_ mqtt.Client, msg mqtt.Message) {
	for range Only.Once {
		request := strings.ToLower(string(msg.Payload()))
		c.log.Info("Option[%s] set to '%s'\n", OptionServiceState, request)
		switch request {
		case "Run":
		case "Restart":
		case "Stop":
		}

		c.Error = c.Client.SetOption(OptionServiceState, request)
		if c.Error != nil {
			c.log.Error("%s\n", c.Error)
			break
		}
	}
}

// -------------------------------------------------------------------------------- //

type MqttEndPoints map[string]MqttEndPoint
type MqttEndPoint struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

func (c *MqttEndPoints) Names() []string {
	var ret []string
	for name := range *c {
		ret = append(ret, name)
	}
	return ret
}

func (c *MqttEndPoints) IsOK(check *api.DataEntry) bool {
	var yes bool
	for range Only.Once {
		field := check.Current.GetFieldPath()
		name := field.First()

		var ep MqttEndPoint
		if ep, yes = (*c)[name]; !yes {
			yes = false
			break
		}

		if len(ep.Include) == 0 {
			yes = false
			break
		}

		for _, reStr := range ep.Exclude {
			reStr = strings.ReplaceAll(reStr, `.`, `\.`)
			reStr = strings.ReplaceAll(reStr, `*`, `.*?`)
			reStr = "^" + strings.TrimPrefix(reStr, "^")
			re := regexp.MustCompile(reStr)
			if re.MatchString(check.EndPoint) {
				return false
			}
			if re.MatchString(check.Current.FieldPath.String()) {
				return false
			}
			if re.MatchString(check.Current.DataStructure.Endpoint.String()) {
				return false
			}
		}

		for _, reStr := range ep.Include {
			reStr = strings.ReplaceAll(reStr, `.`, `\.`)
			reStr = strings.ReplaceAll(reStr, `*`, `.*`)
			reStr = "^" + strings.TrimPrefix(reStr, "^")
			re := regexp.MustCompile(reStr)
			if re.MatchString(check.EndPoint) {
				yes = true
				break
			}
			if re.MatchString(check.Current.FieldPath.String()) {
				yes = true
				break
			}
			if re.MatchString(check.Current.DataStructure.Endpoint.String()) {
				yes = true
				break
			}
		}
	}
	return yes
}

func sortedStrings(values []string) []string {
	ret := append([]string(nil), values...)
	sort.Strings(ret)
	return ret
}

func formatSungrowDeviceTypeSummary(devices getDeviceList.Devices) string {
	if len(devices) == 0 {
		return "none"
	}

	counts := make(map[int64]int)
	for _, device := range devices {
		counts[device.DeviceType.Value()]++
	}

	types := make([]int64, 0, len(counts))
	for deviceType := range counts {
		types = append(types, deviceType)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})

	parts := make([]string, 0, len(types))
	for _, deviceType := range types {
		parts = append(parts, fmt.Sprintf("%d=%d", deviceType, counts[deviceType]))
	}
	return strings.Join(parts, ", ")
}

func (c *CmdMqtt) logRealtimePsKeySelections(devices getDeviceList.Devices) {
	targets := selectRealtimePsKeyTargets(devices)
	warnings := realtimePsKeySelectionWarnings(devices, targets)
	c.log.Info("Realtime source selections: %s\n", describeRealtimePsKeySelection(devices))
	for _, target := range targets {
		c.log.Info("Realtime source target: ps_id=%s ps_key=%s device_type=%d source=%s\n", target.PsID, target.PsKey, target.DeviceType, target.SelectionSource)
	}
	for _, warning := range warnings {
		c.log.Info("Realtime source warning: %s\n", warning)
	}
}

func describeRealtimePsKeySelection(devices getDeviceList.Devices) string {
	targets := selectRealtimePsKeyTargets(devices)
	switch len(targets) {
	case 0:
		return "no usable ps_key available"
	case 1:
		target := targets[0]
		return fmt.Sprintf("1 plant: ps_id=%s ps_key=%s device_type=%d source=%s", target.PsID, target.PsKey, target.DeviceType, target.SelectionSource)
	default:
		return fmt.Sprintf("%d plants: %s", len(targets), strings.Join(formatRealtimePsKeyTargets(targets), "; "))
	}
}

func formatRealtimePsKeyTargets(targets []realtimePsKeyTarget) []string {
	parts := make([]string, 0, len(targets))
	for _, target := range targets {
		parts = append(parts, fmt.Sprintf("ps_id=%s ps_key=%s device_type=%d source=%s", target.PsID, target.PsKey, target.DeviceType, target.SelectionSource))
	}
	return parts
}

func selectRealtimePsKeyTargets(devices getDeviceList.Devices) []realtimePsKeyTarget {
	bestByPlant := make(map[string]realtimePsKeyTarget)
	bestRankByPlant := make(map[string]int)
	for _, device := range devices {
		psKey := strings.TrimSpace(device.PsKey.String())
		if psKey == "" {
			continue
		}

		psID := realtimeDevicePsID(device)
		if psID == "" {
			continue
		}

		deviceType := device.DeviceType.Value()
		rank := preferredSungrowDeviceTypeRank(deviceType)
		currentRank, exists := bestRankByPlant[psID]
		if exists && (rank > currentRank || (rank == currentRank && psKey >= bestByPlant[psID].PsKey)) {
			continue
		}

		bestRankByPlant[psID] = rank
		bestByPlant[psID] = realtimePsKeyTarget{
			PsID:            psID,
			PsKey:           psKey,
			DeviceType:      deviceType,
			SelectionSource: realtimeSelectionSourceForDeviceType(deviceType),
		}
	}

	psIDs := make([]string, 0, len(bestByPlant))
	for psID := range bestByPlant {
		psIDs = append(psIDs, psID)
	}
	sort.Strings(psIDs)

	targets := make([]realtimePsKeyTarget, 0, len(psIDs))
	for _, psID := range psIDs {
		targets = append(targets, bestByPlant[psID])
	}
	return targets
}

func realtimeDevicePsID(device getDeviceList.Device) string {
	psID := strings.TrimSpace(device.PsId.String())
	if psID != "" {
		return psID
	}

	psKey := strings.TrimSpace(device.PsKey.String())
	if psKey == "" {
		return ""
	}
	parts := strings.Split(psKey, "_")
	return strings.TrimSpace(parts[0])
}

func realtimePsKeySelectionWarnings(devices getDeviceList.Devices, targets []realtimePsKeyTarget) []string {
	selected := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		selected[target.PsID] = struct{}{}
	}

	seenPlants := make(map[string]struct{})
	for _, device := range devices {
		psID := realtimeDevicePsID(device)
		if psID == "" {
			continue
		}
		seenPlants[psID] = struct{}{}
	}

	psIDs := make([]string, 0, len(seenPlants))
	for psID := range seenPlants {
		if _, ok := selected[psID]; ok {
			continue
		}
		psIDs = append(psIDs, psID)
	}
	sort.Strings(psIDs)

	warnings := make([]string, 0, len(psIDs))
	for _, psID := range psIDs {
		warnings = append(warnings, fmt.Sprintf("ps_id=%s has no usable realtime ps_key", psID))
	}
	return warnings
}

func buildMqttEndpointBatches(endpoints []string, targets []realtimePsKeyTarget) []mqttEndpointBatch {
	nonRealtime := make([]string, 0, len(endpoints))
	hasRealtime := false
	for _, endpoint := range endpoints {
		if endpoint == realtimeEndpointName {
			hasRealtime = true
			continue
		}
		nonRealtime = append(nonRealtime, endpoint)
	}

	batches := make([]mqttEndpointBatch, 0, 1+len(targets))
	if len(nonRealtime) > 0 {
		batches = append(batches, mqttEndpointBatch{Endpoints: nonRealtime})
	}
	if hasRealtime {
		for _, target := range targets {
			if strings.TrimSpace(target.PsKey) == "" {
				continue
			}
			batches = append(batches, mqttEndpointBatch{
				Endpoints: []string{realtimeEndpointName},
				Args:      []string{"PsKeyList:" + target.PsKey},
			})
		}
	}
	return batches
}

func defaultMqttEndpoints() (MqttEndPoints, error) {
	endpoints := MqttEndPoints{}
	if err := json.Unmarshal([]byte(DefaultMqttFile), &endpoints); err != nil {
		return nil, err
	}
	return endpoints, nil
}

func mergeDefaultMqttEndpoints(endpoints *MqttEndPoints) (bool, error) {
	defaults, err := defaultMqttEndpoints()
	if err != nil {
		return false, err
	}
	if *endpoints == nil {
		*endpoints = MqttEndPoints{}
	}

	changed := false
	for name, defaultEndpoint := range defaults {
		current, exists := (*endpoints)[name]
		if !exists {
			(*endpoints)[name] = MqttEndPoint{
				Include: copyStringSlice(defaultEndpoint.Include),
				Exclude: copyStringSlice(defaultEndpoint.Exclude),
			}
			changed = true
			continue
		}

		for _, include := range defaultEndpoint.Include {
			if stringSliceContains(current.Include, include) {
				continue
			}
			current.Include = append(current.Include, include)
			changed = true
		}
		(*endpoints)[name] = current
	}
	return changed, nil
}

func stringSliceContains(values []string, value string) bool {
	for _, entry := range values {
		if entry == value {
			return true
		}
	}
	return false
}

func copyStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	ret := make([]string, len(values))
	copy(ret, values)
	return ret
}

const DefaultMqttFile = `{
	"queryDeviceList": {
		"include": [
			"virtual.*"
		],
		"exclude": [
			"queryDeviceList.*.devices.*",
			"queryDeviceList.*.device_types.*"
		]
	},
	"queryDeviceRealTimeDataByPsKeys": {
		"include": [
			"*"
		],
		"exclude": [
		]
	},
	"getPsList": {
		"include": [
			"virtual.*"
		],
		"exclude": [
		]
	},
	"getPsDetail": {
		"include": [
			"virtual.*"
		],
		"exclude": [
		]
	}
}
`
