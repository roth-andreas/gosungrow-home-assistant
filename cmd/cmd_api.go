package cmd

import (
	"errors"
	"time"

	"github.com/MickMake/GoUnify/Only"
	"github.com/MickMake/GoUnify/cmdHelp"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/login"
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
)

type loginAttempt = iSolarCloud.LoginAttempt

var apiWriteConfig = func() error { return cmds.Unify.WriteConfig() }

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
	if c.SunGrow == nil {
		c.Error = errors.New("sungrow instance not configured")
		return c.Error
	}
	auth := login.SunGrowAuth{AppKey: iSolarCloud.NormalizeLoginAppKey(c.AppKey), UserAccount: c.Username, UserPassword: c.Password, TokenFile: c.ApiTokenFile, Force: force}
	if c.SunGrow.AuthDetails != nil {
		auth.TokenPath = c.SunGrow.AuthDetails.TokenPath
	}
	c.Error = c.SunGrow.AuthenticateSession(auth, force, c.persistSession)
	return c.Error
}

// persistSession stages CLI values for WriteConfig and restores them on failure.
// It runs before the candidate replaces the active iSolarCloud client.
func (c *CmdApi) persistSession(candidate *iSolarCloud.SunGrow) error {
	oldURL, oldKey, oldLogin, oldToken := c.Url, c.AppKey, c.LastLogin, c.ApiToken
	c.Url, c.AppKey = candidate.ApiRoot.ServerUrl.String(), candidate.GetAppKey()
	c.LastLogin, c.ApiToken = candidate.GetLastLogin(), candidate.GetToken()
	if c.ApiToken != oldToken {
		if err := apiWriteConfig(); err != nil {
			c.Url, c.AppKey, c.LastLogin, c.ApiToken = oldURL, oldKey, oldLogin, oldToken
			return err
		}
	}
	return nil
}

func normalizeLoginAppKey(appKey string) string {
	return iSolarCloud.NormalizeLoginAppKey(appKey)
}

func buildLoginAttempts(host string, appKey string) []loginAttempt {
	return iSolarCloud.BuildLoginAttempts(host, appKey)
}

func shouldTryNextLoginAttempt(err error) bool {
	return iSolarCloud.ShouldTryNextLoginAttempt(err)
}
