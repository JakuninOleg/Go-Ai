package providers

import (
	"context"
	"errors"
	"net"
	"net/url"
)

type UpstreamError struct {
	Provider string
	Category string
	err      error
}

func (e UpstreamError) Error() string {
	return "upstream " + e.Category + " error for provider: " + e.Provider
}

func (e UpstreamError) Unwrap() error {
	return e.err
}

func ClassifyUpstreamError(provider string, err error) error {
	if err == nil {
		return nil
	}

	return UpstreamError{
		Provider: provider,
		Category: upstreamErrorCategory(err),
		err:      err,
	}
}

func upstreamErrorCategory(err error) string {
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return "timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	return "transport"
}
