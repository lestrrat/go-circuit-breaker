package http

import (
	"net/url"

	"github.com/lestrrat/go-circuit-breaker/breaker"
)

func NewPerHostLookup(hosts breaker.Map) *PerHostLookup {
	return &PerHostLookup{
		hosts: hosts,
	}
}

const defaultBreakerName = "_default"
func (l *PerHostLookup) BreakerLookup(v interface{}) *breaker.Breaker {
	rawURL := v.(string)
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		breaker, _ := l.hosts.Get(defaultBreakerName)
		return breaker
	}

	host := parsedURL.Host
	cb, ok := l.hosts.Get(host)
	if !ok {
		return nil
/*
		cb = breaker.New(breaker.WithTripper(breaker.ThresholdTripper(l.threshold)))
		l.hosts.Set(host, cb)
*/
	}
	return cb
}
