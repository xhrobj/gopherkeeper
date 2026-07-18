package tui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type serverStatusResultMsg struct {
	requestID uint64
	status    string
	err       error
}

type serverStatusFailure struct {
	status string
	reason string
}

func serverStatusCmd(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	minimumDuration time.Duration,
) tea.Cmd {
	return func() tea.Msg {
		startedAt := time.Now()
		status, err := backend.Health(ctx)
		if remaining := minimumDuration - time.Since(startedAt); remaining > 0 {
			timer := time.NewTimer(remaining)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
			}
		}
		return serverStatusResultMsg{
			requestID: requestID,
			status:    status,
			err:       err,
		}
	}
}

func describeServerStatusError(err error) serverStatusFailure {
	if err == nil {
		return serverStatusFailure{}
	}

	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)

	switch {
	case strings.Contains(lower, "connection refused"):
		return serverStatusFailure{status: "Unreachable", reason: "Connection refused"}
	case strings.Contains(lower, "no such host") || strings.Contains(lower, "name or service not known"):
		return serverStatusFailure{status: "Unreachable", reason: "Host not found"}
	case strings.Contains(lower, "network is unreachable"):
		return serverStatusFailure{status: "Unreachable", reason: "Network unreachable"}
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return serverStatusFailure{status: "Unreachable", reason: "Connection timed out"}
	case strings.Contains(lower, "x509") || strings.Contains(lower, "certificate"):
		return serverStatusFailure{status: "TLS error", reason: "Certificate verification failed"}
	case strings.Contains(lower, "tls"):
		return serverStatusFailure{status: "TLS error", reason: "TLS handshake failed"}
	case strings.Contains(lower, "server gave http response to https client"):
		return serverStatusFailure{status: "TLS error", reason: "Server does not support HTTPS"}
	default:
		return serverStatusFailure{status: "Connection error", reason: cleanServerStatusReason(message)}
	}
}

func cleanServerStatusReason(message string) string {
	for _, prefix := range []string{
		"send health request: ",
		"create health request: ",
		"read health response: ",
		"decode health response: ",
	} {
		message = strings.TrimPrefix(message, prefix)
	}

	if index := strings.LastIndex(message, `": `); index >= 0 {
		message = message[index+3:]
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return "Unknown connection error"
	}

	return capitalizeFirst(message)
}
