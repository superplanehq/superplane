package core

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// ProviderAPIError classifies a failure from a BYOK provider's live API
// (Claude, OpenAI, OpenRouter). It preserves the original, human-readable
// message text (Error() is unchanged from the wrapped error), while letting
// callers use errors.As to recover the HTTP status code and classify the
// failure without string matching.
type ProviderAPIError struct {
	// StatusCode is the HTTP status code returned by the provider. It is zero
	// for transport-level failures (network error, timeout) that never
	// received a response.
	StatusCode int
	message    string
	err        error
}

// NewProviderAPIError wraps err as a provider API error carrying the given
// HTTP status code. message is used as the Error() text, so existing
// string-matching tests and log lines are unaffected.
func NewProviderAPIError(statusCode int, message string, err error) *ProviderAPIError {
	return &ProviderAPIError{StatusCode: statusCode, message: message, err: err}
}

// NewProviderTransportError wraps a transport-level failure (network error,
// timeout) as a provider API error with no HTTP status code.
func NewProviderTransportError(message string, err error) *ProviderAPIError {
	return &ProviderAPIError{StatusCode: 0, message: message, err: err}
}

func (e *ProviderAPIError) Error() string {
	return e.message
}

func (e *ProviderAPIError) Unwrap() error {
	return e.err
}

// IsAuth reports whether the provider rejected the request as unauthenticated
// or unauthorized (invalid or revoked API key).
func (e *ProviderAPIError) IsAuth() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// IsRateLimited reports whether the provider throttled the request.
func (e *ProviderAPIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsUnavailable reports whether the provider returned a server-side error.
func (e *ProviderAPIError) IsUnavailable() bool {
	return e.StatusCode >= http.StatusInternalServerError
}

// IsTransport reports whether the request never reached the provider (network
// error, connection reset, timeout) as opposed to receiving an HTTP error
// response.
func (e *ProviderAPIError) IsTransport() bool {
	return e.StatusCode == 0
}

// IsProviderAuthOrNetworkError reports whether err represents a class of
// provider failure that a user can plausibly fix (invalid or revoked API
// key), or a transient condition (network error, timeout, provider outage)
// rather than a bug in SuperPlane.
func IsProviderAuthOrNetworkError(err error) bool {
	var providerErr *ProviderAPIError
	if errors.As(err, &providerErr) {
		return providerErr.IsAuth() || providerErr.IsTransport() || providerErr.IsUnavailable()
	}
	return isContextTimeout(err)
}

// IsProviderRateLimited reports whether err represents a rate-limit response
// from a BYOK provider.
func IsProviderRateLimited(err error) bool {
	var providerErr *ProviderAPIError
	if errors.As(err, &providerErr) {
		return providerErr.IsRateLimited()
	}
	return false
}

func isContextTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
