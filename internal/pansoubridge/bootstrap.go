package pansoubridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pansou/config"
	"pansou/plugin"
	"pansou/service"
	"pansou/util"
	"pansou/util/cache"
)

const (
	maxSearches      = 2
	maxAsyncSearches = 8
	asyncLifetime    = 30 * time.Second
)

type Bootstrap struct {
	adapter     searchAdapter
	cache       *cache.DelayedBatchWriteManager
	mainCache   *cache.EnhancedTwoLevelCache
	gate        chan struct{}
	mu          sync.Mutex
	closed      bool
	async       map[uint64]context.CancelFunc
	doneRefresh map[string]time.Time
	nextID      uint64
	active      sync.WaitGroup
	shutdown    sync.Once
	shutdownErr error
}

// New mirrors PanSou main.go initialization without starting its HTTP server.
func New(dataDir string) (*Bootstrap, error) {
	cacheDir := filepath.Join(dataDir, "pansou-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create PanSou cache directory: %w", err)
	}
	setDefault("CACHE_ENABLED", "true")
	_ = os.Setenv("CACHE_PATH", cacheDir)
	setDefault("ASYNC_PLUGIN_ENABLED", "true")
	setDefault("ASYNC_MAX_BACKGROUND_WORKERS", "4")
	setDefault("ASYNC_MAX_BACKGROUND_TASKS", "8")
	setDefault("ASYNC_RESPONSE_TIMEOUT", "4")
	setDefault("PLUGIN_TIMEOUT", "12")
	setDefault("PLUGIN_BATCH_TIMEOUT_SECONDS", "12")
	setDefault("TG_CHANNEL_REQUEST_TIMEOUT_SECONDS", "8")
	setDefault("TG_CHANNEL_TIMEOUT_SECONDS", "8")
	setDefault("OUTBOUND_MAX_CONCURRENCY", "8")
	setDefault("CHANNELS", "tgsearchers7")
	plugins := strings.TrimSpace(os.Getenv("MEDIA_CENTER_PANSOU_PLUGINS"))
	if plugins == "" {
		plugins = "5266ys,quark4k"
	}
	_ = os.Setenv("ENABLED_PLUGINS", plugins)

	config.Init()
	// Keep resource limits fixed even if upstream environment variables are present.
	config.AppConfig.CacheEnabled = true
	config.AppConfig.CachePath = cacheDir
	config.AppConfig.CacheMaxSizeMB = min(256, max(32, config.AppConfig.CacheMaxSizeMB))
	config.AppConfig.CacheTTLMinutes = min(24*60, max(1, config.AppConfig.CacheTTLMinutes))
	config.AppConfig.AsyncCacheTTLHours = min(24, max(1, config.AppConfig.AsyncCacheTTLHours))
	config.AppConfig.AsyncPluginEnabled = true
	config.AppConfig.AsyncMaxBackgroundWorkers = 4
	config.AppConfig.AsyncMaxBackgroundTasks = 8
	config.AppConfig.AsyncResponseTimeout = 4
	config.AppConfig.AsyncResponseTimeoutDur = 4 * time.Second
	config.AppConfig.PluginTimeout = 12 * time.Second
	config.AppConfig.PluginBatchTimeout = 12 * time.Second
	config.AppConfig.TGChannelRequestTimeout = 8 * time.Second
	config.AppConfig.TGChannelTimeout = 8 * time.Second
	config.AppConfig.OutboundMaxConcurrency = 8
	if len(config.AppConfig.DefaultChannels) > 8 {
		config.AppConfig.DefaultChannels = config.AppConfig.DefaultChannels[:8]
	}
	if config.AppConfig.DefaultConcurrency > 4 {
		config.AppConfig.DefaultConcurrency = 4
	}

	util.InitHTTPClient()
	writer, err := cache.NewDelayedBatchWriteManager()
	if err != nil {
		return nil, fmt.Errorf("create PanSou cache writer: %w", err)
	}
	if err := writer.Initialize(); err != nil {
		return nil, fmt.Errorf("initialize PanSou cache writer: %w", err)
	}
	service.SetGlobalCacheWriteManager(writer)
	plugin.InitAsyncPluginSystem()
	manager := plugin.NewPluginManager()
	manager.RegisterGlobalPluginsWithFilter(config.AppConfig.EnabledPlugins)
	config.UpdateDefaultConcurrency(min(4, max(1, len(manager.GetPlugins()))))
	search := service.NewSearchService(manager) // cache initialization and updater installation are synchronous here
	return &Bootstrap{adapter: searchAdapter{service: search}, cache: writer, mainCache: service.GetEnhancedTwoLevelCache(), gate: make(chan struct{}, maxSearches), async: make(map[uint64]context.CancelFunc), doneRefresh: make(map[string]time.Time)}, nil
}

func setDefault(key, value string) {
	if strings.TrimSpace(os.Getenv(key)) == "" {
		_ = os.Setenv(key, value)
	}
}

func (b *Bootstrap) Search(parent context.Context, keyword, source string, cloudTypes []string, refresh bool) (interface{}, error) {
	if parent == nil {
		parent = context.Background()
	}
	select {
	case b.gate <- struct{}{}:
	case <-parent.Done():
		return nil, parent.Err()
	}
	defer func() { <-b.gate }()
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, errors.New("resource search is shutting down")
	}
	if source != "all" && source != "tg" && source != "plugin" {
		b.mu.Unlock()
		return nil, errors.New("unsupported source")
	}
	if refresh {
		key := strings.ToLower(strings.TrimSpace(keyword))
		if at, ok := b.doneRefresh[key]; ok && time.Since(at) < 10*time.Second {
			b.mu.Unlock()
			return nil, errors.New("refresh rate limited; retry later")
		}
		if len(b.doneRefresh) >= 256 {
			var oldestKey string
			var oldest time.Time
			for k, at := range b.doneRefresh {
				if oldest.IsZero() || at.Before(oldest) {
					oldestKey, oldest = k, at
				}
			}
			delete(b.doneRefresh, oldestKey)
		}
		b.doneRefresh[key] = time.Now()
	}
	if len(b.async) >= maxAsyncSearches {
		b.mu.Unlock()
		return nil, errors.New("too many pending resource searches")
	}
	b.nextID++
	id := b.nextID
	ctx, cancel := context.WithTimeout(context.Background(), asyncLifetime)
	b.async[id] = cancel
	b.active.Add(1)
	b.mu.Unlock()
	defer b.active.Done()
	go func() { <-ctx.Done(); b.mu.Lock(); delete(b.async, id); b.mu.Unlock() }()

	result, err := b.adapter.Search(ctx, keyword, source, cloudTypes, refresh)
	if err != nil {
		cancel()
		return nil, err
	}
	return result, nil
}

func (b *Bootstrap) Shutdown(ctx context.Context) error {
	b.shutdown.Do(func() {
		b.mu.Lock()
		b.closed = true
		for _, cancel := range b.async {
			cancel()
		}
		b.mu.Unlock()
		done := make(chan struct{})
		go func() { b.active.Wait(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			b.shutdownErr = ctx.Err()
			return
		}
		if err := plugin.WaitForAsyncTasks(ctx); err != nil && b.shutdownErr == nil {
			b.shutdownErr = err
		}
		if err := b.cache.Shutdown(5 * time.Second); err != nil {
			b.shutdownErr = err
		}
		cache.StopGlobalCleanupTask()
		if b.mainCache != nil {
			if err := b.mainCache.FlushMemoryToDisk(); err != nil && b.shutdownErr == nil {
				b.shutdownErr = err
			}
		}
	})
	return b.shutdownErr
}
