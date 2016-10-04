# go-circuit-breaker

Circuit Breaker Pattern for Go

[![Build Status](https://travis-ci.org/lestrrat/go-circuit-breaker.svg?branch=master)](https://travis-ci.org/lestrrat/go-circuit-breaker)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-circuit-breaker?status.svg)](https://godoc.org/github.com/lestrrat/go-circuit-breaker)

# SYNOPSIS

```go
import "github.com/lestrrat/go-circuit-breaker/breaker"

func main() {
  b := breaker.New(
    breaker.WithTripper(breaker.ThresholdTripper(10)),
  )

  err := b.Call(breaker.CircuitFucn(func() error {
    ...
  })
  if err != nil {
    // either the `circuit` passed to Call() failed
    // or the call was made while the breaker was
    // open/tripped.
    ...
  }
}
```

# DESCRIPTION

This is a fork of https://github.com/rubyist/circuitbreaker.
There were few issues (timing sensitive tests) and tweaks that I wanted to see
(e.g. separating the event subscription from the core breaker), but they mostly
required API changes, which is a very hard to thing to press for (and understandably so)

# CONTRIBUTING

PRs are welcome. If you have a new patch, please attach a test.
If you are suggesting new API, or change, please attach a failing test
to demonstrate what you are looking for. In short: please attach a test.
