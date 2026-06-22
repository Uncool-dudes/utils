package gcs

import "time"

// Config holds connection and bucket settings for the GCS client.
// When CredentialsFile is empty, credentials are resolved via ADC:
// Workload Identity on GCP, or GOOGLE_APPLICATION_CREDENTIALS env var locally.
type Config struct {
	Bucket          string `koanf:"bucket"           validate:"required"`
	Prefix          string `koanf:"prefix"`
	CredentialsFile string `koanf:"credentials_file"` // path to service account JSON; omit for ADC

	// ServiceAccount is the service account email used for signing URLs.
	// Optional when running on GCP with Workload Identity — the client
	// uses IAM sign-blob via the attached service account automatically.
	ServiceAccount string `koanf:"service_account"`

	SignedURLExpiry time.Duration `koanf:"signed_url_expiry"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	SignedURLExpiry: 15 * time.Minute,
}
