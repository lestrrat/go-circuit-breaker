package breaker

type breakerOpenErr struct {}

func (e breakerOpenErr) Error() string {
	return "breaker open"
}

func (e breakerOpenErr) State() State {
	return Open
}

type breakerTimeoutErr struct {}

func (e breakerTimeoutErr) Error() string {
	return "breaker timeout"
}

func (e breakerTimeoutErr) IsTimeout() bool {
	return true
}

type causer interface {
	Cause() error
}

type stater interface {
	State() State
}

type isTimeouter interface {
	IsTimeout() bool
}

// IsOpen returns true if the error is caused by a "breaker open" error.
func IsOpen(err error) bool {
	for err != nil {
		if bserr, ok := err.(stater); ok {
			return bserr.State() == Open
		}

		if cerr, ok := err.(causer); ok {
			err = cerr.Cause()
		}
	}
	return false
}

// IsTimeout returns true if the error is caused by a "breaker timeout" error.
func IsTimeout(err error) bool {
	for err != nil {
		if bterr, ok := err.(isTimeouter); ok {
			return bterr.IsTimeout()
		}

		if cerr, ok := err.(causer); ok {
			err = cerr.Cause()
		}
	}
	return false
}