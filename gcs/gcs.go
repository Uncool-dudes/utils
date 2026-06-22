package gcs

import (
	"context"
	"io"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Bucket is the common interface for uploading, downloading, and signing URLs.
// Both *Service and *ScopedService implement it.
type Bucket interface {
	Upload(ctx context.Context, key string, r io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	SignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	SignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Service manages a GCS client and bucket handle.
type Service struct {
	config Config
	client *storage.Client
	bucket *storage.BucketHandle
}

// New returns an uninitiated Service. Connection is deferred to OnStart via fx.Module.
func New(cfg Config) *Service {
	return &Service{config: cfg}
}

func (s *Service) connect(ctx context.Context) error {
	var opts []option.ClientOption
	if s.config.CredentialsFile != "" {
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, s.config.CredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return Domain.Mark(err, ErrConnFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}

	if _, err := client.Bucket(s.config.Bucket).Attrs(ctx); err != nil {
		_ = client.Close()
		return Domain.Mark(err, ErrConnFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}

	s.client = client
	s.bucket = client.Bucket(s.config.Bucket)
	return nil
}

// Scope returns a ScopedService that prefixes all keys with prefix.
// Use it to partition a single bucket into logical namespaces.
func (s *Service) Scope(prefix string) *ScopedService {
	return &ScopedService{svc: s, prefix: prefix}
}

// Upload writes r to key in the configured bucket.
func (s *Service) Upload(ctx context.Context, key string, r io.Reader) error {
	return upload(ctx, s.bucket, s.fullKey(key), r)
}

// Download returns a reader for the object at key. Caller must close it.
func (s *Service) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return download(ctx, s.bucket, s.fullKey(key))
}

// Delete removes the object at key.
func (s *Service) Delete(ctx context.Context, key string) error {
	return del(ctx, s.bucket, s.fullKey(key))
}

// SignedDownloadURL returns a v4-signed URL granting GET access to key for expiry duration.
func (s *Service) SignedDownloadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return signedURL(s.bucket, s.fullKey(key), "GET", expiry, s.config.ServiceAccount)
}

// SignedUploadURL returns a v4-signed URL granting PUT access to key for expiry duration.
// Use this to let clients upload directly to GCS without proxying through your server.
func (s *Service) SignedUploadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return signedURL(s.bucket, s.fullKey(key), "PUT", expiry, s.config.ServiceAccount)
}

// Close releases the GCS client.
func (s *Service) Close() error {
	if s.client != nil {
		return Domain.Wrap(s.client.Close(), "close")
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

func (s *Service) fullKey(key string) string {
	return joinPrefix(s.config.Prefix, key)
}

// ScopedService is a prefix-scoped view of a Service. No lifecycle of its own.
type ScopedService struct {
	svc    *Service
	prefix string
}

// Upload writes r to key under this scope's prefix.
func (ss *ScopedService) Upload(ctx context.Context, key string, r io.Reader) error {
	return upload(ctx, ss.svc.bucket, ss.fullKey(key), r)
}

// Download returns a reader for the object at key under this scope's prefix. Caller must close it.
func (ss *ScopedService) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return download(ctx, ss.svc.bucket, ss.fullKey(key))
}

// Delete removes the object at key under this scope's prefix.
func (ss *ScopedService) Delete(ctx context.Context, key string) error {
	return del(ctx, ss.svc.bucket, ss.fullKey(key))
}

// SignedDownloadURL returns a v4-signed URL granting GET access to key under this scope's prefix.
func (ss *ScopedService) SignedDownloadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return signedURL(ss.svc.bucket, ss.fullKey(key), "GET", expiry, ss.svc.config.ServiceAccount)
}

// SignedUploadURL returns a v4-signed URL granting PUT access to key under this scope's prefix.
func (ss *ScopedService) SignedUploadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return signedURL(ss.svc.bucket, ss.fullKey(key), "PUT", expiry, ss.svc.config.ServiceAccount)
}

func (ss *ScopedService) fullKey(key string) string {
	return ss.svc.fullKey(joinPrefix(ss.prefix, key))
}

// --- shared helpers ---

func upload(ctx context.Context, bkt *storage.BucketHandle, key string, r io.Reader) error {
	w := bkt.Object(key).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return Domain.Mark(err, ErrUploadFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	if err := w.Close(); err != nil {
		return Domain.Mark(err, ErrUploadFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	return nil
}

func download(ctx context.Context, bkt *storage.BucketHandle, key string) (io.ReadCloser, error) {
	r, err := bkt.Object(key).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, Domain.Mark(err, ErrNotFound) //nolint:wrapcheck // Domain.Mark is the wrapping layer
		}
		return nil, Domain.Wrap(err, "open object")
	}
	return r, nil
}

func del(ctx context.Context, bkt *storage.BucketHandle, key string) error {
	err := bkt.Object(key).Delete(ctx)
	if err == storage.ErrObjectNotExist {
		return Domain.Mark(err, ErrNotFound) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	return Domain.Wrap(err, "delete object")
}

func signedURL(bkt *storage.BucketHandle, key, method string, expiry time.Duration, serviceAccount string) (string, error) {
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  method,
		Expires: time.Now().Add(expiry),
	}
	if serviceAccount != "" {
		opts.GoogleAccessID = serviceAccount
	}
	url, err := bkt.SignedURL(key, opts)
	if err != nil {
		return "", Domain.Wrap(err, "sign url")
	}
	return url, nil
}

func joinPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(key, "/")
}
