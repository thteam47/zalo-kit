package health

import (
	"errors"
	"net"
	"strings"
)

type FailureKind string

const (
	FailureUnknown FailureKind = "unknown"
	FailureNetwork FailureKind = "network"
	FailureAuth    FailureKind = "auth"
)

// Classify is deliberately conservative: an unknown error must never force a
// user to scan a new QR code. Only narrow, known auth signals return Auth.
func Classify(err error) FailureKind {
	if err == nil {
		return FailureUnknown
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return FailureNetwork
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"invalid cookie", "session expired", "unauthorized", "login required"} {
		if strings.Contains(message, marker) {
			return FailureAuth
		}
	}
	for _, marker := range []string{"timeout", "connection reset", "proxy", "temporary", "eof"} {
		if strings.Contains(message, marker) {
			return FailureNetwork
		}
	}
	return FailureUnknown
}
