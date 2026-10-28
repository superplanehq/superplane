package polar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHasPaymentMethod(t *testing.T) {
	now := time.Now()
	for _, test := range []struct {
		name      string
		status    int
		items     []map[string]any
		want      bool
		wantError bool
	}{
		{name: "customer without a card", status: 200},
		{name: "missing customer", status: 404},
		{name: "confirmed saved card", status: 200, items: []map[string]any{{"id": "pm_card", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() + 1, "exp_month": 1}}}, want: true},
		{name: "expired card", status: 200, items: []map[string]any{{"id": "pm_old", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() - 1, "exp_month": 1}}}},
		{name: "card without an id", status: 200, items: []map[string]any{{"type": "kr_card"}}},
		{name: "card without expiry", status: 200, items: []map[string]any{{"id": "pm_card", "type": "card"}}},
		{name: "invalid expiry month", status: 200, items: []map[string]any{{"id": "pm_card", "type": "card", "method_metadata": map[string]int{"exp_year": now.Year() + 1, "exp_month": 13}}}},
		{name: "non-card method", status: 200, items: []map[string]any{{"id": "pm_bank", "type": "bank"}}},
		{name: "unavailable provider", status: 503, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/customers/external/org_test/payment-methods", r.URL.Path)
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				w.Header().Set(apiVersionHeader, APIVersion())
				w.WriteHeader(test.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"items": test.items, "pagination": map[string]int{"max_page": 1}})
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "token", server.Client()).HasPaymentMethod(context.Background(), "org_test")
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got, test.name)
		})
	}
}

type paymentMethodTransport func(*http.Request) (*http.Response, error)

func (f paymentMethodTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func paymentMethodResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{apiVersionHeader: []string{APIVersion()}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHasPaymentMethodRecoversConnectionReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		attempts := 0
		client := NewClient("https://polar.test", "token", &http.Client{
			Transport: paymentMethodTransport(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "/customers/external/org_test/payment-methods", req.URL.Path)
				attempts++
				if attempts == 1 {
					return nil, fmt.Errorf("interrupted read: %w", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET})
				}
				return paymentMethodResponse(`{"items":[{"id":"pm_card","type":"kr_card"}],"pagination":{"max_page":1}}`), nil
			}),
		})
		hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
		require.NoError(t, err)
		require.True(t, hasCard)
		require.Equal(t, 2, attempts)
	})
}

type interruptedPaymentMethodBody struct {
	io.Reader
	err    error
	closed bool
}

func (b *interruptedPaymentMethodBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if errors.Is(err, io.EOF) {
		return n, b.err
	}
	return n, err
}

func (b *interruptedPaymentMethodBody) Close() error {
	b.closed = true
	return nil
}

func TestHasPaymentMethodDroppedConnections(t *testing.T) {
	for _, cause := range []error{syscall.ECONNRESET, syscall.EPIPE, io.EOF, io.ErrUnexpectedEOF} {
		for _, duringBodyRead := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/body=%t", cause, duringBodyRead), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					attempts := 0
					var interruptedBody *interruptedPaymentMethodBody
					client := NewClient("https://polar.test", "token", &http.Client{
						Transport: paymentMethodTransport(func(_ *http.Request) (*http.Response, error) {
							attempts++
							if attempts == 1 {
								wrapped := fmt.Errorf("interrupted read: %w", cause)
								if !duringBodyRead {
									return nil, wrapped
								}
								interruptedBody = &interruptedPaymentMethodBody{Reader: strings.NewReader(`{"items":[{"id":"pm_partial","type":"kr_card"}`), err: wrapped}
								response := paymentMethodResponse("")
								response.Body = interruptedBody
								return response, nil
							}
							if interruptedBody != nil {
								require.True(t, interruptedBody.closed, "close the failed body before retrying")
							}
							return paymentMethodResponse(`{"items":[],"pagination":{"max_page":1}}`), nil
						}),
					})
					started := time.Now()
					hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
					require.NoError(t, err)
					require.False(t, hasCard, "an interrupted card response does not confirm a saved card")
					require.Equal(t, 2, attempts)
					require.Equal(t, 100*time.Millisecond, time.Since(started))
				})
			})
		}
	}
}

func TestHasPaymentMethodExhaustsDroppedConnectionRetries(t *testing.T) {
	for _, duringBodyRead := range []bool{false, true} {
		t.Run(fmt.Sprintf("body=%t", duringBodyRead), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				attempts := 0
				cause := fmt.Errorf("provider connection: %w", syscall.ECONNRESET)
				var bodies []*interruptedPaymentMethodBody
				client := NewClient("https://polar.test", "token", &http.Client{
					Transport: paymentMethodTransport(func(_ *http.Request) (*http.Response, error) {
						for _, body := range bodies {
							require.True(t, body.closed)
						}
						attempts++
						if duringBodyRead {
							body := &interruptedPaymentMethodBody{Reader: strings.NewReader(""), err: cause}
							bodies = append(bodies, body)
							response := paymentMethodResponse("")
							response.Body = body
							return response, nil
						}
						return nil, cause
					}),
				})
				started := time.Now()
				hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
				require.False(t, hasCard)
				require.ErrorIs(t, err, cause)
				require.ErrorIs(t, err, syscall.ECONNRESET)
				require.ErrorIs(t, err, ErrOutcomeUnknown)
				require.Equal(t, 3, attempts)
				require.Equal(t, 300*time.Millisecond, time.Since(started))
				for _, body := range bodies {
					require.True(t, body.closed)
				}
			})
		})
	}
}

func TestHasPaymentMethodDoesNotRetryProviderRejections(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		version string
		body    string
		class   error
	}{
		{name: "unauthorized", status: 401, class: ErrUnauthorized},
		{name: "rate limited", status: 429, class: ErrRateLimited},
		{name: "unavailable", status: 503, class: ErrOutcomeUnknown},
		{name: "invalid request", status: 422, class: ErrInvalidRequest},
		{name: "unsupported version", status: 404, class: ErrUnsupportedAPIVersion},
		{name: "malformed JSON", status: 200, version: APIVersion(), body: `{"items":`, class: ErrOutcomeUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			client := NewClient("https://polar.test", "token", &http.Client{
				Transport: paymentMethodTransport(func(_ *http.Request) (*http.Response, error) {
					attempts++
					response := paymentMethodResponse(test.body)
					response.StatusCode = test.status
					response.Header.Set(apiVersionHeader, test.version)
					return response, nil
				}),
			})
			hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
			require.False(t, hasCard)
			require.ErrorIs(t, err, test.class)
			require.Equal(t, 1, attempts)
		})
	}
}

func TestHasPaymentMethodCancellation(t *testing.T) {
	for _, beforeRequest := range []bool{true, false} {
		t.Run(fmt.Sprintf("before request=%t", beforeRequest), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				attempts := 0
				client := NewClient("https://polar.test", "token", &http.Client{
					Transport: paymentMethodTransport(func(_ *http.Request) (*http.Response, error) {
						attempts++
						return nil, syscall.ECONNRESET
					}),
				})
				if beforeRequest {
					cancel()
				}
				done := make(chan error, 1)
				go func() {
					hasCard, err := client.HasPaymentMethod(ctx, "org_test")
					if hasCard {
						t.Error("canceled lookup must not confirm a card")
					}
					done <- err
				}()
				synctest.Wait()
				cancel()
				err := <-done
				require.ErrorIs(t, err, context.Canceled)
				require.ErrorIs(t, err, ErrOutcomeUnknown)
				if beforeRequest {
					require.Zero(t, attempts)
					return
				}
				require.Equal(t, 1, attempts)
				require.ErrorIs(t, err, syscall.ECONNRESET)
			})
		})
	}
}

func TestHasPaymentMethodDeadlineDuringRetryWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		attempts := 0
		client := NewClient("https://polar.test", "token", &http.Client{
			Transport: paymentMethodTransport(func(_ *http.Request) (*http.Response, error) {
				attempts++
				return nil, io.ErrUnexpectedEOF
			}),
		})
		started := time.Now()
		hasCard, err := client.HasPaymentMethod(ctx, "org_test")
		require.False(t, hasCard)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, ErrOutcomeUnknown)
		require.Equal(t, 1, attempts)
		require.Equal(t, 50*time.Millisecond, time.Since(started))
	})
}

func TestHasPaymentMethodPaginationRecoversInterruptedPage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var pages []string
		var interruptedBody *interruptedPaymentMethodBody
		client := NewClient("https://polar.test", "token", &http.Client{
			Transport: paymentMethodTransport(func(req *http.Request) (*http.Response, error) {
				pages = append(pages, req.URL.Query().Get("page"))
				switch len(pages) {
				case 1:
					return nil, syscall.EPIPE
				case 2:
					return paymentMethodResponse(`{"items":[{"id":"pm_bank","type":"bank"}],"pagination":{"max_page":3}}`), nil
				case 3, 4:
					if interruptedBody != nil {
						require.True(t, interruptedBody.closed)
					}
					interruptedBody = &interruptedPaymentMethodBody{Reader: strings.NewReader(`{"items":[{"id":"pm_partial","type":"kr_card"}`), err: io.ErrUnexpectedEOF}
					response := paymentMethodResponse("")
					response.Body = interruptedBody
					return response, nil
				case 5:
					require.True(t, interruptedBody.closed)
					return paymentMethodResponse(`{"items":[{"id":"pm_bank","type":"bank"}],"pagination":{"max_page":3}}`), nil
				default:
					return paymentMethodResponse(`{"items":[{"id":"pm_card","type":"kr_card"}],"pagination":{"max_page":3}}`), nil
				}
			}),
		})
		hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
		require.NoError(t, err)
		require.True(t, hasCard)
		require.Equal(t, []string{"1", "1", "2", "2", "2", "3"}, pages)
	})
}

func TestHasPaymentMethodSharesTimeoutAcrossPagesAndRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := time.Now()
		var pages []string
		client := NewClient("https://polar.test", "token", &http.Client{
			Transport: paymentMethodTransport(func(req *http.Request) (*http.Response, error) {
				deadline, ok := req.Context().Deadline()
				require.True(t, ok)
				require.Equal(t, started.Add(15*time.Second), deadline)
				pages = append(pages, req.URL.Query().Get("page"))
				switch len(pages) {
				case 1:
					time.Sleep(6 * time.Second)
					return paymentMethodResponse(`{"items":[{"id":"pm_bank","type":"bank"}],"pagination":{"max_page":2}}`), nil
				case 2:
					time.Sleep(6 * time.Second)
					return nil, syscall.ECONNRESET
				default:
					<-req.Context().Done()
					return nil, req.Context().Err()
				}
			}),
		})
		hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
		require.False(t, hasCard)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, ErrOutcomeUnknown)
		require.Equal(t, []string{"1", "2", "2"}, pages)
		require.Equal(t, 15*time.Second, time.Since(started))
	})
}

func TestHasPaymentMethodPreservesShorterHTTPTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		attempts := 0
		client := NewClient("https://polar.test", "token", &http.Client{
			Timeout: 20 * time.Millisecond,
			Transport: paymentMethodTransport(func(req *http.Request) (*http.Response, error) {
				attempts++
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		})
		started := time.Now()
		hasCard, err := client.HasPaymentMethod(context.Background(), "org_test")
		require.False(t, hasCard)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, ErrOutcomeUnknown)
		require.Equal(t, 1, attempts)
		require.Equal(t, 20*time.Millisecond, time.Since(started))
	})
}
