package errors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	stderrors "errors"
	"net"
	"os"
	"strings"
	"syscall"
)

const (
	Timeout           = "timeout"
	DNS               = "DNS"
	ConnectionRefused = "connection_refused"
	TLS               = "TLS"
	HTTP4xx           = "HTTP_4xx"
	HTTP5xx           = "HTTP_5xx"
	BodyRead          = "body/read_error"
	Unknown           = "unknown"
)

func Classify(err error) string {
	if err == nil {
		return ""
	}
	if stderrors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return Timeout
	}
	var dnsErr *net.DNSError
	if stderrors.As(err, &dnsErr) {
		return DNS
	}
	if stderrors.Is(err, syscall.ECONNREFUSED) || strings.Contains(strings.ToLower(err.Error()), "connection refused") {
		return ConnectionRefused
	}
	var certErr x509.UnknownAuthorityError
	if stderrors.As(err, &certErr) {
		return TLS
	}
	var hostnameErr x509.HostnameError
	if stderrors.As(err, &hostnameErr) {
		return TLS
	}
	var recordErr tls.RecordHeaderError
	if stderrors.As(err, &recordErr) {
		return TLS
	}
	if strings.Contains(strings.ToLower(err.Error()), "tls") {
		// Some TLS failures arrive wrapped by transports that do not expose a
		// concrete TLS error type. The string fallback is intentionally narrow.
		return TLS
	}
	return Unknown
}

func ClassifyStatus(status int) string {
	switch {
	case status >= 400 && status < 500:
		return HTTP4xx
	case status >= 500 && status < 600:
		return HTTP5xx
	default:
		return Unknown
	}
}
