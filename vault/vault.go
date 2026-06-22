package vault

import (
	"context"
	"fmt"
	"net/http"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// DynamicSecret holds credentials and lease metadata from a dynamic secret engine.
type DynamicSecret struct {
	Data      map[string]any
	LeaseID   string
	TTL       time.Duration
	Renewable bool
}

// Service manages a Vault client and its lifecycle.
type Service struct {
	config     Config
	client     *vaultapi.Client
	kv         *vaultapi.KVv2
	tokenWatch *vaultapi.LifetimeWatcher
}

// New returns an uninitiated Service. Connection is deferred to OnStart via fx.Module.
func New(cfg Config) *Service {
	return &Service{config: cfg}
}

func (s *Service) connect(ctx context.Context) error {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = s.config.Addr
	if s.config.Timeout > 0 {
		cfg.HttpClient.Timeout = s.config.Timeout
	}

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return Domain.Mark(err, ErrConnFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	if s.config.Namespace != "" {
		client.SetNamespace(s.config.Namespace)
	}

	switch s.config.Auth.Method {
	case "token":
		client.SetToken(s.config.Auth.Token)
	case "approle":
		secret, err := client.Logical().WriteWithContext(ctx, "auth/approle/login", map[string]any{
			"role_id":   s.config.Auth.RoleID,
			"secret_id": s.config.Auth.SecretID,
		})
		if err != nil {
			return Domain.Mark(err, ErrConnFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
		}
		client.SetToken(secret.Auth.ClientToken)
		if err := s.startTokenRenewal(client, secret); err != nil {
			return Domain.Wrap(err, "start token renewal")
		}
	}

	s.client = client
	s.kv = client.KVv2(s.config.KV.MountPath)
	return nil
}

func (s *Service) startTokenRenewal(client *vaultapi.Client, secret *vaultapi.Secret) error {
	if !secret.Auth.Renewable {
		return nil
	}
	watcher, err := client.NewLifetimeWatcher(&vaultapi.LifetimeWatcherInput{Secret: secret})
	if err != nil {
		return err //nolint:wrapcheck // wrapped by caller
	}
	s.tokenWatch = watcher
	go watcher.Start()
	go func() {
		for {
			select {
			case <-watcher.DoneCh():
				return
			case <-watcher.RenewCh():
			}
		}
	}()
	return nil
}

// Get returns all fields at path from KV v2.
func (s *Service) Get(ctx context.Context, path string) (map[string]any, error) {
	secret, err := s.kv.Get(ctx, path)
	if err != nil {
		return nil, s.classifyErr(err, "get")
	}
	return secret.Data, nil
}

// GetString returns a single string field from a KV v2 secret.
func (s *Service) GetString(ctx context.Context, path, field string) (string, error) {
	data, err := s.Get(ctx, path)
	if err != nil {
		return "", err
	}
	v, ok := data[field]
	if !ok {
		return "", Domain.Newf("field %q not found in secret %q", field, path) //nolint:wrapcheck // Domain.Newf is the error origin
	}
	str, ok := v.(string)
	if !ok {
		return "", Domain.Newf("field %q in secret %q is not a string", field, path) //nolint:wrapcheck // Domain.Newf is the error origin
	}
	return str, nil
}

// GetVersion returns a specific version of a KV v2 secret.
func (s *Service) GetVersion(ctx context.Context, path string, version int) (map[string]any, error) {
	secret, err := s.kv.GetVersion(ctx, path, version)
	if err != nil {
		return nil, s.classifyErr(err, "get version")
	}
	return secret.Data, nil
}

// List returns all keys under prefix in KV v2.
func (s *Service) List(ctx context.Context, prefix string) ([]string, error) {
	// KV v2 list uses the metadata path
	listPath := s.config.KV.MountPath + "/metadata/" + prefix
	secret, err := s.client.Logical().ListWithContext(ctx, listPath)
	if err != nil {
		return nil, s.classifyErr(err, "list")
	}
	if secret == nil {
		return nil, nil
	}
	raw, ok := secret.Data["keys"].([]any)
	if !ok {
		return nil, nil
	}
	keys := make([]string, 0, len(raw))
	for _, k := range raw {
		if str, ok := k.(string); ok {
			keys = append(keys, str)
		}
	}
	return keys, nil
}

// Put writes data to path in KV v2.
func (s *Service) Put(ctx context.Context, path string, data map[string]any) error {
	_, err := s.kv.Put(ctx, path, data)
	return s.classifyErr(err, "put")
}

// Delete soft-deletes the latest version at path in KV v2.
func (s *Service) Delete(ctx context.Context, path string) error {
	return s.classifyErr(s.kv.Delete(ctx, path), "delete")
}

// GetDynamic fetches credentials from a dynamic secret engine path.
// e.g. "database/creds/my-role", "aws/creds/my-role"
func (s *Service) GetDynamic(ctx context.Context, path string) (*DynamicSecret, error) {
	secret, err := s.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, s.classifyErr(err, "get dynamic")
	}
	if secret == nil {
		return nil, Domain.Mark(fmt.Errorf("no secret at %q", path), ErrSecretNotFound) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	return &DynamicSecret{
		Data:      secret.Data,
		LeaseID:   secret.LeaseID,
		TTL:       time.Duration(secret.LeaseDuration) * time.Second,
		Renewable: secret.Renewable,
	}, nil
}

// WatchLease renews leaseID in the background until ctx is cancelled or renewal fails.
// onRenew fires on each successful renewal with the new TTL.
// onExpiry fires when renewal fails or the lease expires — caller should re-fetch credentials.
// Returns a stop function that cancels the watcher.
func (s *Service) WatchLease(
	ctx context.Context,
	leaseID string,
	ttl time.Duration,
	onRenew func(newTTL time.Duration),
	onExpiry func(err error),
) (stop func(), err error) {
	synthetic := &vaultapi.Secret{
		LeaseID:       leaseID,
		LeaseDuration: int(ttl.Seconds()),
		Renewable:     true,
	}
	watcher, err := s.client.NewLifetimeWatcher(&vaultapi.LifetimeWatcherInput{Secret: synthetic})
	if err != nil {
		return nil, Domain.Wrap(err, "create lease watcher")
	}
	go watcher.Start()
	go func() {
		for {
			select {
			case err := <-watcher.DoneCh():
				if onExpiry != nil {
					onExpiry(err)
				}
				return
			case renewal := <-watcher.RenewCh():
				if onRenew != nil {
					onRenew(time.Duration(renewal.Secret.LeaseDuration) * time.Second)
				}
			case <-ctx.Done():
				watcher.Stop()
				return
			}
		}
	}()
	return watcher.Stop, nil
}

// Close stops token renewal and releases resources.
func (s *Service) Close() error {
	if s.tokenWatch != nil {
		s.tokenWatch.Stop()
	}
	return nil
}

// NewConnected creates and immediately connects a Service. Use in tests and CLIs.
func NewConnected(ctx context.Context, cfg Config) (*Service, error) {
	svc := New(cfg)
	if err := svc.connect(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Service) classifyErr(err error, op string) error {
	if err == nil {
		return nil
	}
	if re, ok := err.(*vaultapi.ResponseError); ok { //nolint:errorlint // vault SDK returns concrete type
		switch re.StatusCode {
		case http.StatusNotFound:
			return Domain.Mark(err, ErrSecretNotFound) //nolint:wrapcheck // Domain.Mark is the wrapping layer
		case http.StatusForbidden:
			return Domain.Mark(err, ErrAccessDenied) //nolint:wrapcheck // Domain.Mark is the wrapping layer
		}
	}
	return Domain.Wrapf(err, "%s", op)
}
