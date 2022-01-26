package chaos

// Switchable is the base interface for chaos adapters created for testing.
type Switchable interface {
	// Breaking returns true if the failure behaviour is activated.
	Breaking() bool
	// Break starts the failure behaviour.
	Break()
	// Fix stops the failure behaviour.
	Fix()
}
