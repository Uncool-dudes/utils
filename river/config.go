package river

import (
	"time"

	riv "github.com/riverqueue/river"
)

// Config holds River async job queue settings.
type Config struct {
	Enabled                     bool                   `json:"enabled,omitempty"`
	Queues                      map[string]QueueConfig `json:"queues,omitempty"`
	MaxAttempts                 int                    `json:"maxattempts,omitempty"`
	JobTimeoutSeconds           int                    `json:"jobtimeoutseconds,omitempty"`
	FetchCooldownMs             int                    `json:"fetchcooldownms,omitempty"`
	FetchPollIntervalMs         int                    `json:"fetchpollintervalms,omitempty"`
	RescueStuckJobsAfterMinutes int                    `json:"rescuestuckjobsafterminutes,omitempty"`
	CompletedRetentionHours     int                    `json:"completedretentionhours,omitempty"`
	DiscardedRetentionHours     int                    `json:"discardedretentionhours,omitempty"`
	CancelledRetentionHours     int                    `json:"cancelledretentionhours,omitempty"`
}

// QueueConfig controls per-queue worker concurrency.
type QueueConfig struct {
	MaxWorkers int `json:"maxworkers" validate:"required,min=1"`
}

const (
	// DefaultMaxWorkers is the default per-queue worker concurrency.
	DefaultMaxWorkers = 10
	// DefaultMaxAttempts is the default number of job attempts before discarding.
	DefaultMaxAttempts = 5
	// DefaultCompletedRetentionPeriod is how long completed jobs are kept.
	// Completed jobs may carry PII in error/result fields — keep short.
	DefaultCompletedRetentionPeriod = 24 * time.Hour
	// DefaultDiscardedRetentionPeriod is how long discarded jobs are kept.
	// Discarded = exceeded max attempts. Keep longer for saga investigation.
	DefaultDiscardedRetentionPeriod = 72 * time.Hour
	// DefaultCancelledRetentionPeriod is how long cancelled jobs are kept.
	DefaultCancelledRetentionPeriod = 24 * time.Hour
)

// Defaults provides a single default queue with DefaultMaxWorkers.
var Defaults = Config{
	MaxAttempts: DefaultMaxAttempts,
	Queues: map[string]QueueConfig{
		riv.QueueDefault: {MaxWorkers: DefaultMaxWorkers},
	},
}

// EnsureQueue guarantees a named queue exists in *cfgp. Existing entries are not overwritten.
func EnsureQueue(cfgp **Config, name string, maxWorkers int) {
	if *cfgp == nil {
		*cfgp = &Config{}
	}
	if (*cfgp).Queues == nil {
		(*cfgp).Queues = map[string]QueueConfig{}
	}
	if _, ok := (*cfgp).Queues[name]; !ok {
		(*cfgp).Queues[name] = QueueConfig{MaxWorkers: maxWorkers}
	}
}

// BuildRiverConfig returns a river.Config with cfg applied over defaults.
// Pass nil to get pure defaults.
func BuildRiverConfig(cfg *Config, workers *riv.Workers) *riv.Config {
	rc := &riv.Config{
		CompletedJobRetentionPeriod: DefaultCompletedRetentionPeriod,
		DiscardedJobRetentionPeriod: DefaultDiscardedRetentionPeriod,
		CancelledJobRetentionPeriod: DefaultCancelledRetentionPeriod,
		Queues: map[string]riv.QueueConfig{
			riv.QueueDefault: {MaxWorkers: DefaultMaxWorkers},
		},
		Workers:     workers,
		MaxAttempts: DefaultMaxAttempts,
	}
	if cfg == nil {
		return rc
	}
	for name, v := range cfg.Queues {
		if v.MaxWorkers == 0 {
			delete(rc.Queues, name)
			continue
		}
		rc.Queues[name] = riv.QueueConfig{MaxWorkers: v.MaxWorkers}
	}
	if len(rc.Queues) == 0 {
		rc.Queues = nil
		rc.Workers = nil
	}
	if cfg.MaxAttempts > 0 {
		rc.MaxAttempts = cfg.MaxAttempts
	}
	if cfg.JobTimeoutSeconds > 0 {
		rc.JobTimeout = time.Duration(cfg.JobTimeoutSeconds) * time.Second
	}
	if cfg.FetchCooldownMs > 0 {
		rc.FetchCooldown = time.Duration(cfg.FetchCooldownMs) * time.Millisecond
	}
	if cfg.FetchPollIntervalMs > 0 {
		rc.FetchPollInterval = time.Duration(cfg.FetchPollIntervalMs) * time.Millisecond
	}
	if cfg.RescueStuckJobsAfterMinutes > 0 {
		rc.RescueStuckJobsAfter = time.Duration(cfg.RescueStuckJobsAfterMinutes) * time.Minute
	}
	if cfg.CompletedRetentionHours > 0 {
		rc.CompletedJobRetentionPeriod = time.Duration(cfg.CompletedRetentionHours) * time.Hour
	}
	if cfg.DiscardedRetentionHours > 0 {
		rc.DiscardedJobRetentionPeriod = time.Duration(cfg.DiscardedRetentionHours) * time.Hour
	}
	if cfg.CancelledRetentionHours > 0 {
		rc.CancelledJobRetentionPeriod = time.Duration(cfg.CancelledRetentionHours) * time.Hour
	}
	return rc
}
