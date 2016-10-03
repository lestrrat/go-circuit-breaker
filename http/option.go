package http

import (
	"net/http"

	"github.com/lestrrat/go-circuit-breaker/internal/option"
)

func WithClient(c *http.Client) Option {
	return option.NewValue("Client", c)
}
