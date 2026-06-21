package middleware

import (
	"net/http"

	"github.com/uncool-dudes/utils/resilience"
)

type cbTransport struct {
	cb   *resilience.CircuitBreaker
	next http.RoundTripper
}

// CircuitBreakerTransport wraps an http.RoundTripper with a circuit breaker.
//
//	client := &http.Client{
//	    Transport: middleware.CircuitBreakerTransport(cb, http.DefaultTransport),
//	}
func CircuitBreakerTransport(cb *resilience.CircuitBreaker, next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &cbTransport{cb: cb, next: next}
}

func (t *cbTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	result, err := t.cb.Execute(func() (any, error) {
		return t.next.RoundTrip(req)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*http.Response), nil
}
