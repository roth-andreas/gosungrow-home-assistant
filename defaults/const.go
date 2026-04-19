package defaults

import _ "embed"

//go:embed README.md
var Readme string

//go:embed EXAMPLES.md
var Examples string

const (
	Description   = "GoSungrow for Home Assistant and Sungrow iSolarCloud MQTT publishing"
	BinaryName    = "GoSungrow"
	BinaryVersion = "3.0.16"
	SourceRepo    = "github.com/roth-andreas/gosungrow-home-assistant"
	BinaryRepo    = "github.com/roth-andreas/gosungrow-home-assistant"

	EnvPrefix = "GOSUNGROW"

	HelpSummary = `
# GoSungrow for Home Assistant.

This repository is maintained primarily as a Home Assistant app for Sungrow iSolarCloud systems.
It logs in to iSolarCloud, publishes MQTT discovery/state data, and can install a managed Home Assistant dashboard.

`
)
