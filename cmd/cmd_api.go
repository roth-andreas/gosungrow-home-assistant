package cmd

import (
	"errors"
	"strings"
	"time"

	"github.com/MickMake/GoSungrow/iSolarCloud"
	"github.com/MickMake/GoSungrow/iSolarCloud/AppService/login"
	"github.com/MickMake/GoUnify/Only"
	"github.com/MickMake/GoUnify/cmdHelp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagApiUrl       = "host"
	flagApiTimeout   = "timeout"
	flagApiUsername  = "user"
	flagApiPassword  = "password"
	flagApiAppKey    = "appkey"
	flagApiLastLogin = "token-expiry"

	oldLoginAppKey    = "A5C22A880B97303FCB902069C6B042AB"
	legacyLoginAppKey = "93D72E60331ABDCDC7B39ADC2D1F32B3"
)

type loginAttempt struct {
	host   string
	appKey string
}

//goland:noinspection GoNameStartsWithPackageName
type CmdApi struct {
	CmdDefault

	ApiTimeout   time.Duration
	Url          string
	Username     string
	Password     string
	AppKey       string
	LastLogin    string
	ApiToken     string
	ApiTokenFile string

	SunGrow *iSolarCloud.SunGrow
}

func NewCmdApi() *CmdApi {
	var ret *CmdApi

	for range Only.Once {
		ret = &CmdApi{
			CmdDefault: CmdDefault{
				Error:   nil,
				cmd:     nil,
				SelfCmd: nil,
			},
			ApiTimeout:   iSolarCloud.DefaultTimeout,
			Url:          iSolarCloud.DefaultHost,
			Username:     "",
			Password:     "",
			AppKey:       iSolarCloud.DefaultApiAppKey,
			LastLogin:    "",
			ApiToken:     "",
			ApiTokenFile: "",
			SunGrow:      nil,
		}
	}

	return ret
}

func (c *CmdApi) AttachCommand(cmd *cobra.Command) *cobra.Command {
	for range Only.Once {
		if cmd == nil {
			break
		}
		c.cmd = cmd

		cmdApi := &cobra.Command{
			Use:                   "api",
			Annotations:           map[string]string{"group": "Api"},
			Short:                 "Low-level authentication helpers for iSolarCloud.",
			Long:                  "Low-level authentication helpers for iSolarCloud.",
			DisableFlagParsing:    false,
			DisableFlagsInUseLine: false,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return cmd.Help()
			},
			Args: cobra.ArbitraryArgs,
		}
		cmd.AddCommand(cmdApi)
		cmdApi.Example = cmdHelp.PrintExamples(cmdApi, "login")

		cmdApiLogin := &cobra.Command{
			Use:                   "login",
			Annotations:           map[string]string{"group": "Api"},
			Short:                 "Log in to iSolarCloud and refresh the cached token.",
			Long:                  "Log in to iSolarCloud and refresh the cached token.",
			DisableFlagParsing:    false,
			DisableFlagsInUseLine: false,
			PreRunE:               cmds.SunGrowArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				c.Error = c.ApiLogin(true)
				if c.Error != nil {
					return c.Error
				}

				c.SunGrow.Auth.Print()
				return nil
			},
			Args: cobra.NoArgs,
		}
		cmdApi.AddCommand(cmdApiLogin)
		cmdApiLogin.Example = cmdHelp.PrintExamples(cmdApiLogin, "")
	}

	return c.SelfCmd
}

func (c *CmdApi) AttachFlags(cmd *cobra.Command, viper *viper.Viper) {
	for range Only.Once {
		cmd.PersistentFlags().StringVarP(&c.Username, flagApiUsername, "u", "", "SunGrow: api username.")
		viper.SetDefault(flagApiUsername, "")
		cmd.PersistentFlags().StringVarP(&c.Password, flagApiPassword, "p", "", "SunGrow: api password.")
		viper.SetDefault(flagApiPassword, "")
		cmd.PersistentFlags().StringVarP(&c.AppKey, flagApiAppKey, "", iSolarCloud.DefaultApiAppKey, "SunGrow: api application key.")
		viper.SetDefault(flagApiAppKey, iSolarCloud.DefaultApiAppKey)
		cmd.PersistentFlags().StringVarP(&c.Url, flagApiUrl, "", iSolarCloud.DefaultHost, "SunGrow: Provider API URL.")
		viper.SetDefault(flagApiUrl, iSolarCloud.DefaultHost)
		c.ApiTimeout = iSolarCloud.DefaultTimeout
		cmd.PersistentFlags().StringVar(&c.LastLogin, flagApiLastLogin, "", "SunGrow: last login.")
		viper.SetDefault(flagApiLastLogin, "")
	}
}

func (ca *Cmds) SunGrowArgs(cmd *cobra.Command, args []string) error {
	for range Only.Once {
		ca.Error = cmds.ProcessArgs(cmd, args)
		if ca.Error != nil {
			break
		}

		ca.Api.SunGrow = iSolarCloud.NewSunGro(ca.Api.Url, ca.CacheDir)
		if ca.Api.SunGrow.Error != nil {
			ca.Error = ca.Api.SunGrow.Error
			break
		}

		ca.Error = ca.Api.SunGrow.Init()
		if ca.Error != nil {
			break
		}

		if ca.Api.AppKey == "" {
			ca.Api.AppKey = iSolarCloud.DefaultApiAppKey
		}

		ca.Error = ca.Api.ApiLogin(false)
		if ca.Error != nil {
			break
		}

		if ca.Debug {
			ca.Api.SunGrow.Auth.Print()
		}
	}

	return ca.Error
}

func (c *CmdApi) ApiLogin(force bool) error {
	for range Only.Once {
		if c.SunGrow == nil {
			c.Error = errors.New("sungrow instance not configured")
			break
		}

		c.AppKey = normalizeLoginAppKey(c.AppKey)
		candidates := buildLoginAttempts(c.Url, c.AppKey)

		cacheDir := c.SunGrow.ApiRoot.GetCacheDir()
		var firstRetriableErr error
		var lastErr error
		exhaustedRetriable := true
		for idx, attempt := range candidates {
			if idx > 0 && c.SunGrow != nil {
				c.SunGrow.Logout()
			}
			c.SunGrow = iSolarCloud.NewSunGro(attempt.host, cacheDir)
			if c.SunGrow.Error != nil {
				c.Error = c.SunGrow.Error
				break
			}
			c.Error = c.SunGrow.Init()
			if c.Error != nil {
				break
			}

			auth := login.SunGrowAuth{
				AppKey:       attempt.appKey,
				UserAccount:  c.Username,
				UserPassword: c.Password,
				TokenFile:    c.ApiTokenFile,
				Force:        force,
			}
			c.Error = c.SunGrow.Login(auth)
			if c.Error == nil {
				c.Url = attempt.host
				c.AppKey = attempt.appKey
				break
			}
			lastErr = c.Error
			if !shouldTryNextLoginAttempt(c.Error) {
				exhaustedRetriable = false
				break
			}
			if firstRetriableErr == nil {
				firstRetriableErr = c.Error
			}
		}
		if c.Error != nil {
			if exhaustedRetriable && firstRetriableErr != nil {
				c.Error = firstRetriableErr
			} else if lastErr != nil {
				c.Error = lastErr
			}
		}
		if c.Error != nil {
			break
		}

		if c.SunGrow.HasTokenChanged() {
			c.LastLogin = c.SunGrow.GetLastLogin()
			c.ApiToken = c.SunGrow.GetToken()
			c.Error = cmds.Unify.WriteConfig()
		}
	}
	return c.Error
}

func normalizeLoginAppKey(appKey string) string {
	appKey = strings.TrimSpace(appKey)
	if appKey == "" || appKey == legacyLoginAppKey {
		return iSolarCloud.DefaultApiAppKey
	}
	return appKey
}

func appendUniqueLoginAttempt(list []loginAttempt, item loginAttempt) []loginAttempt {
	host := strings.TrimSpace(item.host)
	key := strings.TrimSpace(item.appKey)
	if host == "" || key == "" {
		return list
	}
	for _, existing := range list {
		if existing.host == host && existing.appKey == key {
			return list
		}
	}
	return append(list, loginAttempt{host: host, appKey: key})
}

func buildLoginAttempts(host string, appKey string) []loginAttempt {
	candidates := make([]loginAttempt, 0)
	hosts := []string{
		host,
		iSolarCloud.DefaultHost,
		"https://gateway.isolarcloud.com",
		"https://gateway.isolarcloud.eu",
		"https://gateway.isolarcloud.com.cn",
	}
	appKeys := []string{
		normalizeLoginAppKey(appKey),
		iSolarCloud.DefaultApiAppKey,
		oldLoginAppKey,
		legacyLoginAppKey,
	}
	for _, host := range hosts {
		for _, appKey := range appKeys {
			candidates = appendUniqueLoginAttempt(candidates, loginAttempt{host: host, appKey: appKey})
		}
	}
	return candidates
}

func shouldTryNextLoginAttempt(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "login_state=-1") ||
		strings.Contains(msg, "login rejected by gateway") ||
		strings.Contains(msg, "appkey is incorrect") ||
		strings.Contains(msg, "need to login again") ||
		strings.Contains(msg, "er_token_login_invalid") ||
		strings.Contains(msg, "cannot login")
}
