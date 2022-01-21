package chaos

type Switchable interface {
	Breaking() bool
	Break()
	Fix()
}
