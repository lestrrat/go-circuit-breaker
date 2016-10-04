package http_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cenk/backoff"
	"github.com/facebookgo/clock"
	"github.com/lestrrat/go-circuit-breaker/breaker"
	httpb "github.com/lestrrat/go-circuit-breaker/http"
	"github.com/stretchr/testify/assert"
)

func ExampleHTTPWithBreaker() {
	m := breaker.NewMap()
	m.Set("_default", breaker.New())
	m.Set("example.com", breaker.New())

	l := httpb.NewPerHostLookup(m)
	cl := httpb.NewClient(l)

	cl.Get("http://example.com")
}

func TestTreshold(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("fail") == "" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)

	c := clock.NewMock()
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.Clock = c

	m := breaker.NewMap()
	m.Set(u.Host, breaker.New(
		breaker.WithClock(c),
		breaker.WithBackOff(bo),
		breaker.WithTripper(breaker.ThresholdTripper(1)),
	))
	l := httpb.NewPerHostLookup(m)
	cl := httpb.NewClient(l)
	res, err := cl.Get(s.URL)
	if !assert.NoError(t, err, "Get should succeed") {
		return
	}
	t.Logf("%#v", res)

	res, err = cl.Get(s.URL + "?fail=true")
	if !assert.Error(t, err, "Get should fail") {
		return
	}
	for i := 0; i < 10; i++ {
		res, err = cl.Get(s.URL)
		if !assert.Error(t, err, "Get should fail") {
			return
		}
	}
}
