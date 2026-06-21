package consul

var (
	ErrInvalidAddr = Domain.New("invalid consul address")
	ErrConnect     = Domain.New("failed to connect to consul")
	ErrRegister    = Domain.New("service registration failed")
	ErrDeregister  = Domain.New("service deregistration failed")
	ErrLookup      = Domain.New("service lookup failed")
	ErrNoInstances = Domain.New("no healthy instances found")
)
