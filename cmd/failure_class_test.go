package cmd

import (
	"errors"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
)

func TestFailureClassLineUsesStableClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "recoverable remote",
			err:  errors.New("API httpResponse is 500 Internal Server Error"),
			want: "GoSungrow-Failure-Class: recoverable_remote",
		},
		{
			name: "Docker DNS",
			err:  errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: no such host"),
			want: "GoSungrow-Failure-Class: docker_dns",
		},
		{
			name: "non-recoverable",
			err:  errors.New("invalid local configuration"),
			want: "GoSungrow-Failure-Class: non_recoverable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FailureClassLine(tc.err); got != tc.want {
				t.Fatalf("FailureClassLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFailureClassLineDoesNotReclassifyAggregatedDockerDNSText(t *testing.T) {
	err := iSolarCloud.SummarizeLoginAttemptFailures([]iSolarCloud.LoginAttemptFailure{
		{
			Attempt: iSolarCloud.LoginAttempt{Host: "https://gateway.isolarcloud.eu"},
			Err:     errors.New("login rejected by gateway"),
		},
		{
			Attempt: iSolarCloud.LoginAttempt{Host: "https://gateway.isolarcloud.com.cn"},
			Err:     errors.New("dial tcp: lookup gateway.isolarcloud.com.cn on 127.0.0.11:53: no such host"),
		},
	})
	if err == nil {
		t.Fatal("expected summarized error")
	}

	want := "GoSungrow-Failure-Class: recoverable_remote"
	if got := FailureClassLine(err); got != want {
		t.Fatalf("FailureClassLine() = %q, want %q", got, want)
	}
}
