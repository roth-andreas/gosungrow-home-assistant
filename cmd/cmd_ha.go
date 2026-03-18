package cmd

import (
	"fmt"

	"github.com/MickMake/GoUnify/Only"
	"github.com/MickMake/GoUnify/cmdHelp"
	"github.com/spf13/cobra"
)

//goland:noinspection GoNameStartsWithPackageName
type CmdHa CmdDefault

func NewCmdHa() *CmdHa {
	var ret *CmdHa

	for range Only.Once {
		ret = &CmdHa{
			Error:   nil,
			cmd:     nil,
			SelfCmd: nil,
		}
	}

	return ret
}

func (c *CmdHa) AttachCommand(cmd *cobra.Command) *cobra.Command {
	for range Only.Once {
		if cmd == nil {
			break
		}
		c.cmd = cmd

		c.SelfCmd = &cobra.Command{
			Use:                   "ha",
			Aliases:               []string{},
			Annotations:           map[string]string{"group": "Ha"},
			Short:                 fmt.Sprintf("Home Assistant commands."),
			Long:                  fmt.Sprintf("Home Assistant commands."),
			DisableFlagParsing:    false,
			DisableFlagsInUseLine: false,
			PreRunE:               cmds.SunGrowArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			},
			Args: cobra.MinimumNArgs(1),
		}
		cmd.AddCommand(c.SelfCmd)
		c.SelfCmd.Example = cmdHelp.PrintExamples(c.SelfCmd, "install-dashboard")

		c.SelfCmd.AddCommand(c.newInstallDashboardCommand())
	}
	return c.SelfCmd
}
