package cmd

import "github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"

const failureClassPrefix = "GoSungrow-Failure-Class: "

// FailureClassLine renders the machine-readable app-wrapper classification line.
func FailureClassLine(err error) string {
	return failureClassPrefix + string(iSolarCloud.ClassifyFailure(err))
}
