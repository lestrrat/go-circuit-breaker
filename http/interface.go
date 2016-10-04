package http

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/lestrrat/go-circuit-breaker/breaker"
)

var ErrBadStatus = errors.New("bad HTTP status")

type Option interface {
	Name() string
	Get() interface{}
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
	Get(string) (*http.Response, error)
	Head(string) (*http.Response, error)
	Post(string, string, io.Reader) (*http.Response, error)
	PostForm(string, url.Values) (*http.Response, error)
}

// Client is a wrapper around http.Client that provides circuit breaker capabilities.
type Client struct {
	client         HTTPClient
	errOnBadStatus bool
	// BreakerTripped func()
	// BreakerReset   func()
	// BreakerLookup  func(*HTTPClient, interface{}) *breaker.Breaker
	// Panel          *Panel
	lookup  BreakerLookupper
	timeout time.Duration
}

type doCtx struct {
	Client           HTTPClient
	Error            error
	ErrorOnBadStatus bool
	Request          *http.Request
	Response         *http.Response
}

type getCtx struct {
	Client           HTTPClient
	Error            error
	ErrorOnBadStatus bool
	URL              string
	Response         *http.Response
}

type headCtx getCtx

type postCtx struct {
	Body             io.Reader
	BodyType         string
	Client           HTTPClient
	Error            error
	ErrorOnBadStatus bool
	URL              string
	Response         *http.Response
}

type postFormCtx struct {
	Client           HTTPClient
	Data             url.Values
	Error            error
	ErrorOnBadStatus bool
	URL              string
	Response         *http.Response
}

type BreakerLookupper interface {
	BreakerLookup(interface{}) breaker.Breaker
}

type PerHostLookup struct {
	hosts breaker.Map
}
