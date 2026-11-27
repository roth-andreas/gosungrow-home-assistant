package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/defaults"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
)

const (
	cloudLogin              = "login"
	cloudStartupDevices     = "startup_device_discovery"
	cloudTopology           = "topology"
	cloudPointMetadata      = "point_metadata"
	cloudLivePlantLookup    = "live_plant_lookup"
	cloudLiveCollection     = "live_collection"
	cloudRediscovery        = "runtime_rediscovery"
	cloudDashboardDiscovery = "dashboard_discovery"
)

// cloudDenialLog belongs to the CLI command that runs the synchronous cloud requests.
type cloudDenialLog struct {
	command   string
	operation string
	cycle     uint64
	output    io.Writer
}

func (c *CmdApi) setCloudOperation(operation string, cycle uint64) {
	if c != nil && c.cloudDenials != nil {
		c.cloudDenials.operation = operation
		c.cloudDenials.cycle = cycle
	}
}

func (l *cloudDenialLog) report(denial api.ForbiddenResponse) {
	operation := l.operation
	if denial.Authentication {
		operation = cloudLogin
	}
	cycle := ""
	if l.cycle != 0 {
		cycle = fmt.Sprintf(" sync_cycle=%d", l.cycle)
	}
	output := l.output
	if output == nil {
		output = os.Stderr
	}
	// Log delivery does not affect request classification or recovery.
	fmt.Fprintf(output, "iSolarCloud HTTP denial: at=%s version=%s command=%q operation=%s%s endpoint=%s path=%s status=%d gateway=%s session_origin=%s\n",
		denial.At.Format(time.RFC3339Nano), defaults.BinaryVersion, l.command, operation, cycle,
		denial.Endpoint, denial.Path, denial.Status, denial.Gateway, denial.SessionOrigin)
}
