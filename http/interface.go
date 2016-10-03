package http

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/lestrrat/go-circuit-breaker/breaker"
)

type Option interface {
	Name() string
	Get() interface{}
}

// Client is a wrapper around http.Client that provides circuit breaker capabilities.
//
// By default, the client will use its defaultBreaker. A BreakerLookup function may be
// provided to allow different breakers to be used based on the circumstance. See the
// implementation of NewHostBasedHTTPClient for an example of this.
type Client struct {
	client *http.Client
	// BreakerTripped func()
	// BreakerReset   func()
	// BreakerLookup  func(*HTTPClient, interface{}) *breaker.Breaker
	// Panel          *Panel
	lookup  BreakerLookupper
	timeout time.Duration
}

type doCtx struct {
	Client   *http.Client
	Error    error
	Request  *http.Request
	Response *http.Response
}

type getCtx struct {
	Client   *http.Client
	Error    error
	URL      string
	Response *http.Response
}

type headCtx getCtx

type postCtx struct {
	Body     io.Reader
	BodyType string
	Client   *http.Client
	Error    error
	URL      string
	Response *http.Response
}

type postFormCtx struct {
	Client   *http.Client
	Data     url.Values
	Error    error
	URL      string
	Response *http.Response
}

type BreakerLookupper interface {
	BreakerLookup(interface{}) *breaker.Breaker
}

type PerHostLookup struct {
	hosts breaker.Map
}
