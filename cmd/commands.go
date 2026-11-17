package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/MickMake/GoUnify/Only"
	"github.com/MickMake/GoUnify/Unify"
	"github.com/roth-andreas/gosungrow-home-assistant/defaults"
	"github.com/spf13/cobra"
)

type Cmds struct {
	Unify *Unify.Unify
	Api   *CmdApi
	Mqtt  *CmdMqtt
	Ha    *CmdHa

	ConfigDir   string
	CacheDir    string
	ConfigFile  string
	WriteConfig bool
	Quiet       bool
	Debug       bool

	Args []string

	Error error
}

//goland:noinspection GoNameStartsWithPackageName
type CmdDefault struct {
	Error   error
	cmd     *cobra.Command
	SelfCmd *cobra.Command
}

var cmds Cmds

func init() {
	for range Only.Once {
		cmds.Unify = Unify.New(
			Unify.Options{
				Description:   defaults.Description,
				BinaryName:    defaults.BinaryName,
				BinaryVersion: defaults.BinaryVersion,
				SourceRepo:    defaults.SourceRepo,
				BinaryRepo:    defaults.BinaryRepo,
				EnvPrefix:     defaults.EnvPrefix,
				HelpSummary:   defaults.HelpSummary,
				ReadMe:        defaults.Readme,
				Examples:      defaults.Examples,
			},
			Unify.Flags{
				MergeRun: true,
			},
		)

		cmdRoot := cmds.Unify.GetCmd()

		cmds.Api = NewCmdApi()
		cmds.Api.AttachCommand(cmdRoot)
		cmds.Api.AttachFlags(cmdRoot, cmds.Unify.GetViper())

		cmds.Mqtt = NewCmdMqtt("")
		cmds.Mqtt.AttachCommand(cmdRoot)
		cmds.Mqtt.AttachFlags(cmdRoot, cmds.Unify.GetViper())

		cmds.Ha = NewCmdHa()
		cmds.Ha.AttachCommand(cmdRoot)
		// GoUnify applies config/environment values during initialization. Keep
		// strict Boolean validation after that step: its generic flag visitor
		// can overwrite a conversion error while visiting subsequent flags.
		cobra.OnInitialize(validateRuntimeDashboardConfiguration)
	}
}

func validateRuntimeDashboardConfiguration() {
	root := cmds.Unify.GetCmd()
	initialize := root.PersistentPreRunE
	explicit := make(map[string]string)
	for _, name := range []string{"dashboard-enabled", "dashboard-force-update"} {
		if flag := root.PersistentFlags().Lookup(name); flag != nil && flag.Changed {
			explicit[name] = flag.Value.String()
		}
	}
	root.PersistentPreRunE = func(command *cobra.Command, args []string) error {
		if initialize != nil {
			if err := initialize(command, args); err != nil {
				return err
			}
		}
		for _, setting := range []struct {
			name  string
			value *bool
		}{{"dashboard-enabled", &cmds.Mqtt.DashboardEnabled}, {"dashboard-force-update", &cmds.Mqtt.DashboardForceUpdate}} {
			raw, set := explicit[setting.name]
			if !set {
				raw = fmt.Sprint(cmds.Unify.GetViper().Get(setting.name))
			}
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("invalid Boolean configuration for --%s", setting.name)
			}
			*setting.value = value
		}
		return nil
	}
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmds.Unify.GetCmd().SetContext(ctx)
	var err error

	for range Only.Once {
		// Execute adds all child commands to the root command and sets flags appropriately.
		// This is called by main.main(). It only needs to happen once to the rootCmd.
		err = cmds.Unify.Execute()
		if err != nil {
			break
		}
	}

	return err
}

func (ca *Cmds) ProcessArgs(_ *cobra.Command, args []string) error {
	for range Only.Once {
		ca.Args = args

		ca.ConfigDir = cmds.Unify.GetConfigDir()
		ca.ConfigFile = cmds.Unify.GetConfigFile()
		ca.CacheDir = cmds.Unify.GetCacheDir()
		ca.Debug = cmds.Unify.Flags.Debug
		ca.Quiet = cmds.Unify.Flags.Quiet
	}

	return ca.Error
}
