package errors

import (
	"context"
	stderrors "errors"
	"net"
	"syscall"
	"testing"
)

func TestClassifyTimeout(t *testing.T) {
	if got := Classify(context.DeadlineExceeded); got != Timeout {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyDNS(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "example.invalid"}
	if got := Classify(err); got != DNS {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyConnectionRefused(t *testing.T) {
	if got := Classify(stderrors.Join(syscall.ECONNREFUSED)); got != ConnectionRefused {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyStatus(t *testing.T) {
	if got := ClassifyStatus(404); got != HTTP4xx {
		t.Fatalf("got %q", got)
	}
	if got := ClassifyStatus(503); got != HTTP5xx {
		t.Fatalf("got %q", got)
	}
}
