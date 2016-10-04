package http

import (
	"io"
	"net/http"
	"net/url"

	"github.com/lestrrat/go-circuit-breaker/breaker"
)

// NewClient creates a new HTTP Client where requests are controlled via
// the provided circuit breaker(s). The mandatory argument `l` is an object
// that provides the breaker to be used for the given request.
//
// Possible optional parameters:
// * WithClient: specify the HTTP Client instance
// * WithErrorOnBadStatus: specify if you want the breaker to consider 5XX status codes as errors
func NewClient(l BreakerLookupper, options ...Option) *Client {
	var cl HTTPClient
	errOnBadStatus := true
	for _, option := range options {
		switch option.Name() {
		case "Client":
			cl = option.Get().(HTTPClient)
		case "ErrorOnBadStatus":
			errOnBadStatus = option.Get().(bool)
		}
	}
	if cl == nil {
		cl = &http.Client{}
	}

	return &Client{
		client:         cl,
		errOnBadStatus: errOnBadStatus,
		lookup:         l,
	}
}

// Do wraps http.Client Do()
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	breaker := c.breakerLookup(req.URL.String())
	if breaker == nil {
		return c.client.Do(req)
	}

	ctx := getDoCtx()
	defer releaseDoCtx(ctx)

	ctx.Client = c.client
	ctx.ErrorOnBadStatus = c.errOnBadStatus
	ctx.Request = req
	if err := breaker.Call(ctx, c.timeout); err != nil {
		return nil, err
	}
	return ctx.Response, ctx.Error
}

// Get wraps http.Client Get()
func (c *Client) Get(url string) (*http.Response, error) {
	breaker := c.breakerLookup(url)
	if breaker == nil {
		return c.client.Get(url)
	}

	ctx := getGetCtx()
	defer releaseGetCtx(ctx)

	ctx.Client = c.client
	ctx.ErrorOnBadStatus = c.errOnBadStatus
	ctx.URL = url
	if err := breaker.Call(ctx, c.timeout); err != nil {
		return nil, err
	}
	return ctx.Response, ctx.Error
}

// Head wraps http.Client Head()
func (c *Client) Head(url string) (*http.Response, error) {
	breaker := c.breakerLookup(url)
	if breaker == nil {
		return c.client.Head(url)
	}

	ctx := getHeadCtx()
	defer releaseHeadCtx(ctx)

	ctx.Client = c.client
	ctx.ErrorOnBadStatus = c.errOnBadStatus
	ctx.URL = url
	if err := breaker.Call(ctx, c.timeout); err != nil {
		return nil, err
	}
	return ctx.Response, ctx.Error
}

// Post wraps http.Client Post()
func (c *Client) Post(url string, bodyType string, body io.Reader) (*http.Response, error) {
	breaker := c.breakerLookup(url)
	if breaker == nil {
		return c.client.Head(url)
	}

	ctx := getPostCtx()
	defer releasePostCtx(ctx)

	ctx.Client = c.client
	ctx.ErrorOnBadStatus = c.errOnBadStatus
	ctx.URL = url
	ctx.Body = body
	ctx.BodyType = bodyType
	if err := breaker.Call(ctx, c.timeout); err != nil {
		return nil, err
	}
	return ctx.Response, ctx.Error
}

// PostForm wraps http.Client PostForm()
func (c *Client) PostForm(url string, data url.Values) (*http.Response, error) {
	breaker := c.breakerLookup(url)
	if breaker == nil {
		return c.client.PostForm(url, data)
	}

	ctx := getPostFormCtx()
	defer releasePostFormCtx(ctx)

	ctx.Client = c.client
	ctx.ErrorOnBadStatus = c.errOnBadStatus
	ctx.URL = url
	ctx.Data = data
	if err := breaker.Call(ctx, c.timeout); err != nil {
		return nil, err
	}
	return ctx.Response, ctx.Error
}

func (c *Client) breakerLookup(val interface{}) *breaker.Breaker {
	return c.lookup.BreakerLookup(val)
}

/*

func (c *Client) runBreakerTripped() {
	if c.BreakerTripped != nil {
		c.BreakerTripped()
	}
}

func (c *Client) runBreakerReset() {
	if c.BreakerReset != nil {
		c.BreakerReset()
	}
}
*/
