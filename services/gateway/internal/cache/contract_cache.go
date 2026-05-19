package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Contract is the runtime representation of a data contract.
// This is what lives in the cache and what the validator works against.
// It is intentionally separate from the contracts service API response type
// so changes to the API shape do not ripple into the validation hot path.
type Contract struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Topic   string          `json:"topic"`
	Version string          `json:"version"`
	Schema  json.RawMessage `json:"schema"`
	Fields  []FieldPolicy   `json:"fields"`

	// FailOpen controls gateway behavior when this contract's cache entry
	// is stale and a re-fetch has failed. true = allow the message through
	// with a warning metric. false = reject. Set per-contract, not globally,
	// because criticality varies across topics.
	FailOpen bool `json:"fail_open"`
}

type FieldPolicy struct {
	Path       string `json:"path"`
	Required   bool   `json:"required"`
	PIIClass   string `json:"pii_class,omitempty"` // PERSONAL | FINANCIAL | HEALTH | empty
	MaskingOp  string `json:"masking_op,omitempty"` // REDACT | TOKENIZE | MASK
}

type entry struct {
	contract  *Contract
	fetchedAt time.Time
}

// ContractCache holds active contracts in memory and refreshes them in the background.
// The cache is the reason the validation hot path has no network dependency.
//
// Concurrency model: sync.RWMutex gives us concurrent reads (the common case)
// with exclusive writes only during cache fills. Under high message throughput
// with infrequent contract changes this is the correct tradeoff.
type ContractCache struct {
	mu         sync.RWMutex
	entries    map[string]*entry // keyed by topic name
	ttl        time.Duration
	httpClient *http.Client
	serviceURL string
	log        zerolog.Logger
	metrics    *cacheMetrics
}

func New(serviceURL string, ttl time.Duration, fetchTimeout time.Duration, log zerolog.Logger) *ContractCache {
	return &ContractCache{
		entries:    make(map[string]*entry),
		ttl:        ttl,
		serviceURL: serviceURL,
		httpClient: &http.Client{Timeout: fetchTimeout},
		log:        log.With().Str("component", "contract_cache").Logger(),
		metrics:    newCacheMetrics(),
	}
}

// Warm fetches all active contracts at startup.
// The gateway refuses to serve traffic until this completes successfully,
// because serving without any contracts would silently allow all messages through.
func (c *ContractCache) Warm(ctx context.Context) error {
	contracts, err := c.fetchAll(ctx)
	if err != nil {
		return fmt.Errorf("warming contract cache: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for _, contract := range contracts {
		c.entries[contract.Topic] = &entry{
			contract:  contract,
			fetchedAt: now,
		}
	}

	c.log.Info().Int("count", len(contracts)).Msg("contract cache warmed")
	return nil
}

// Get returns the contract for a topic. It refreshes the entry if it has exceeded
// the TTL. The refresh happens synchronously on the first stale read; subsequent
// reads during an in-flight refresh return the stale entry rather than blocking.
//
// This is an intentional tradeoff: a short staleness window is acceptable,
// but blocking the hot path on a network call is not.
func (c *ContractCache) Get(ctx context.Context, topic string) (*Contract, error) {
	c.mu.RLock()
	e, ok := c.entries[topic]
	c.mu.RUnlock()

	if ok && time.Since(e.fetchedAt) < c.ttl {
		c.metrics.hits.Inc()
		return e.contract, nil
	}

	c.metrics.misses.Inc()
	return c.refresh(ctx, topic, e)
}

// refresh fetches a single contract and updates the cache.
// If the fetch fails and a stale entry exists, behavior depends on FailOpen.
func (c *ContractCache) refresh(ctx context.Context, topic string, stale *entry) (*Contract, error) {
	contract, err := c.fetchOne(ctx, topic)
	if err != nil {
		c.metrics.fetchErrors.Inc()
		c.log.Warn().Err(err).Str("topic", topic).Msg("contract fetch failed")

		if stale != nil && stale.contract.FailOpen {
			c.metrics.failOpenAllows.Inc()
			c.log.Warn().Str("topic", topic).Msg("serving stale contract: fail_open=true")
			return stale.contract, nil
		}

		if stale != nil {
			return nil, fmt.Errorf("contract fetch failed and fail_open=false for topic %q: %w", topic, err)
		}

		return nil, fmt.Errorf("no contract found for topic %q: %w", topic, err)
	}

	c.mu.Lock()
	c.entries[topic] = &entry{contract: contract, fetchedAt: time.Now()}
	c.mu.Unlock()

	return contract, nil
}

func (c *ContractCache) fetchAll(ctx context.Context) ([]*Contract, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serviceURL+"/internal/v1/contracts/active", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("contracts service returned %d", resp.StatusCode)
	}

	var contracts []*Contract
	if err := json.NewDecoder(resp.Body).Decode(&contracts); err != nil {
		return nil, fmt.Errorf("decoding contracts response: %w", err)
	}

	return contracts, nil
}

func (c *ContractCache) fetchOne(ctx context.Context, topic string) (*Contract, error) {
	url := fmt.Sprintf("%s/internal/v1/contracts/by-topic/%s", c.serviceURL, topic)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no active contract for topic %q", topic)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("contracts service returned %d", resp.StatusCode)
	}

	var contract Contract
	if err := json.NewDecoder(resp.Body).Decode(&contract); err != nil {
		return nil, fmt.Errorf("decoding contract: %w", err)
	}

	return &contract, nil
}
