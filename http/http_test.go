package http_test

import (
	"github.com/lestrrat/go-circuit-breaker/breaker"
	"github.com/lestrrat/go-circuit-breaker/http"
)

func ExampleHTTPWithBreaker() {
	m := breaker.NewMap()
	m.Set("_default", breaker.New())
	m.Set("example.com", breaker.New())

	l := http.NewPerHostLookup(m)
	cl := http.New(l)

	cl.Get("http://example.com")
}
