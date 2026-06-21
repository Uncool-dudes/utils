package consul

import (
	"fmt"
	"net"

	"github.com/hashicorp/consul/api"
)

// Client wraps the Consul API client with registration helpers.
type Client struct {
	raw   *api.Client
	cfg   Config
	svcID string
}

// New creates a Client from cfg. Returns error if Consul is unreachable at construction time only
// when the addr is unparseable — liveness is checked at registration, not here.
func New(cfg Config) (*Client, error) {
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, Domain.Mark(fmt.Errorf("invalid addr %q: %w", cfg.Addr, err), ErrInvalidAddr)
	}
	_ = port

	c := api.DefaultConfig()
	c.Address = net.JoinHostPort(host, port)

	raw, err := api.NewClient(c)
	if err != nil {
		return nil, Domain.Mark(err, ErrConnect)
	}
	return &Client{raw: raw, cfg: cfg}, nil
}

// Register registers svcName on httpPort with standard /healthz and /readyz checks.
// The service ID is "<svcName>-<httpPort>" so multiple instances don't collide.
// The "prometheus" tag and metrics_path meta are included by default so Prometheus
// Consul SD picks the service up automatically.
func (c *Client) Register(svcName string, httpPort int) error {
	baseURL := fmt.Sprintf("http://host.docker.internal:%d", httpPort)

	id := fmt.Sprintf("%s-%d", svcName, httpPort)
	c.svcID = id

	reg := &api.AgentServiceRegistration{
		ID:   id,
		Name: svcName,
		Port: httpPort,
		Tags: ensureTag(c.cfg.Tags, "prometheus"),
		Meta: c.cfg.Meta,
		Checks: api.AgentServiceChecks{
			{
				CheckID:                        id + ":liveness",
				Name:                           "liveness",
				HTTP:                           baseURL + "/healthz",
				Interval:                       "10s",
				Timeout:                        "3s",
				DeregisterCriticalServiceAfter: "1m",
			},
			{
				CheckID:  id + ":readiness",
				Name:     "readiness",
				HTTP:     baseURL + "/readyz",
				Interval: "10s",
				Timeout:  "3s",
			},
		},
	}
	if err := c.raw.Agent().ServiceRegister(reg); err != nil {
		return Domain.Wrapf(err, "register %s", svcName)
	}
	return nil
}

// ensureTag returns tags with tag appended if not already present.
func ensureTag(tags []string, tag string) []string {
	for _, t := range tags {
		if t == tag {
			return tags
		}
	}
	out := make([]string, len(tags)+1)
	copy(out, tags)
	out[len(tags)] = tag
	return out
}

// Lookup returns host:port of a healthy instance of svcName.
// Uses the first passing instance; add round-robin on top if needed.
func (c *Client) Lookup(svcName string) (string, error) {
	entries, _, err := c.raw.Health().Service(svcName, "", true, nil)
	if err != nil {
		return "", Domain.Mark(err, ErrLookup)
	}
	if len(entries) == 0 {
		return "", Domain.Wrapf(ErrNoInstances, "service %s", svcName)
	}
	e := entries[0]
	addr := e.Service.Address
	if addr == "" {
		addr = e.Node.Address
	}
	return net.JoinHostPort(addr, fmt.Sprint(e.Service.Port)), nil
}

// Deregister removes the service registered by Register.
func (c *Client) Deregister() error {
	if c.svcID == "" {
		return nil
	}
	if err := c.raw.Agent().ServiceDeregister(c.svcID); err != nil {
		return Domain.Mark(err, ErrDeregister)
	}
	return nil
}
