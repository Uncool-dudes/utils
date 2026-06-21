package river

var (
	ErrInvalidConfig = Domain.New("invalid river config")
	ErrConnect       = Domain.New("failed to create river client")
	ErrMigrate       = Domain.New("river migration failed")
)
