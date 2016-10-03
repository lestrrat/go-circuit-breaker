package http

import (
	"io"
	"net/http"
	"net/url"

	"github.com/lestrrat/go-circuit-breaker/breaker"
)

func New(l BreakerLookupper, options ...Option) *Client {
	var cl *http.Client
	for _, option := range options {
		switch option.Name() {
		case "Client":
			cl = option.Get().(*http.Client)
		}
	}
	if cl == nil {
		cl = &http.Client{}
	}

	return &Client{
		client: cl,
		lookup: l,
	}
}

var defaultBreakerName = "_default"

// Do wraps http.Client Do()
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	breaker := c.breakerLookup(req.URL.String())
	if breaker == nil {
		return c.client.Do(req)
	}

	ctx := getDoCtx()
	defer releaseDoCtx(ctx)

	ctx.Client = c.client
	ctx.Request = req
	breaker.Call(ctx, c.timeout)
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
	ctx.URL = url
	breaker.Call(ctx, c.timeout)
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
	ctx.URL = url
	breaker.Call(ctx, c.timeout)
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
	ctx.URL = url
	ctx.Body = body
	ctx.BodyType = bodyType
	breaker.Call(ctx, c.timeout)
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
	ctx.URL = url
	ctx.Data = data
	breaker.Call(ctx, c.timeout)
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
