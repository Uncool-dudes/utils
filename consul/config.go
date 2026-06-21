package consul

type Config struct {
	Addr string   `koanf:"addr" validate:"required"`
	Tags []string `koanf:"tags"`
	// Meta is included in the Consul service registration and exposed to Prometheus
	// via Consul SD. metrics_path tells Prometheus which path to scrape.
	Meta map[string]string `koanf:"meta"`
}

//nolint:gochecknoglobals
var Defaults = Config{
	Addr: "localhost:8500",
	Tags: []string{"prometheus"},
	Meta: map[string]string{"metrics_path": "/metrics"},
}
