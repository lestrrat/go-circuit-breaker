package breaker

// Execute executes the given function
func (c CircuitFunc) Execute() error {
	return c()
}
