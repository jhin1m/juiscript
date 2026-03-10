package cache

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/system"
)

// phpVersionRe validates PHP version format (e.g. "8.3", "7.4").
var phpVersionRe = regexp.MustCompile(`^\d+\.\d+$`)

// CacheStatus holds Redis and Opcache service status.
type CacheStatus struct {
	RedisRunning bool
	RedisVersion string
	RedisMemory  string // human-readable used memory, e.g. "1.2M"
}

// Manager wraps Redis and Opcache operations via system commands.
type Manager struct {
	executor system.Executor
	cfg      *config.Config
}

// NewManager creates a cache manager.
func NewManager(exec system.Executor, cfg *config.Config) *Manager {
	return &Manager{executor: exec, cfg: cfg}
}

// Status checks Redis connectivity and returns service info.
func (m *Manager) Status(ctx context.Context) (*CacheStatus, error) {
	status := &CacheStatus{}

	// Check Redis via PING
	out, err := m.executor.Run(ctx, "redis-cli", "PING")
	if err != nil {
		return status, nil // Redis not running — not an error
	}
	status.RedisRunning = strings.TrimSpace(out) == "PONG"

	// Fetch version and memory if running
	if status.RedisRunning {
		info, _ := m.executor.Run(ctx, "redis-cli", "INFO", "server", "memory")
		status.RedisVersion = parseRedisField(info, "redis_version")
		status.RedisMemory = parseRedisField(info, "used_memory_human")
	}

	return status, nil
}

// EnableRedis ensures Redis service is running for a site.
// It validates the DB number and starts Redis if not active.
func (m *Manager) EnableRedis(ctx context.Context, domain string, db int) error {
	if err := m.validateDB(db); err != nil {
		return err
	}

	// Check if Redis is already active
	_, err := m.executor.Run(ctx, "systemctl", "is-active", "redis-server")
	if err != nil {
		// Try to start Redis
		if _, startErr := m.executor.Run(ctx, "systemctl", "start", "redis-server"); startErr != nil {
			return fmt.Errorf("start redis: %w", startErr)
		}
	}

	// Verify connectivity after start
	out, err := m.executor.Run(ctx, "redis-cli", "PING")
	if err != nil || strings.TrimSpace(out) != "PONG" {
		return fmt.Errorf("redis not responding after start")
	}

	return nil
}

// DisableRedis is a placeholder for MVP. App-level config changes (Laravel .env,
// WP wp-config.php) are left to the user. This method exists for interface completeness.
func (m *Manager) DisableRedis(ctx context.Context, domain string) error {
	return nil
}

// FlushDB flushes a specific Redis database by number.
func (m *Manager) FlushDB(ctx context.Context, db int) error {
	if err := m.validateDB(db); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "redis-cli", "-n", strconv.Itoa(db), "FLUSHDB")
	if err != nil {
		return fmt.Errorf("flush redis db %d: %w", db, err)
	}
	return nil
}

// FlushAll flushes all Redis databases. Destructive — caller should confirm first.
func (m *Manager) FlushAll(ctx context.Context) error {
	_, err := m.executor.Run(ctx, "redis-cli", "FLUSHALL")
	if err != nil {
		return fmt.Errorf("flush all redis: %w", err)
	}
	return nil
}

// ResetOpcache resets PHP Opcache by restarting the PHP-FPM service.
func (m *Manager) ResetOpcache(ctx context.Context, phpVersion string) error {
	if phpVersion == "" {
		return fmt.Errorf("php version required")
	}
	if !phpVersionRe.MatchString(phpVersion) {
		return fmt.Errorf("invalid php version format: %q (expected X.Y)", phpVersion)
	}
	svc := fmt.Sprintf("php%s-fpm", phpVersion)
	_, err := m.executor.Run(ctx, "systemctl", "restart", svc)
	if err != nil {
		return fmt.Errorf("restart %s: %w", svc, err)
	}
	return nil
}

// validateDB checks that db is within the configured range.
func (m *Manager) validateDB(db int) error {
	max := m.cfg.Redis.MaxDatabases
	if max <= 0 {
		max = 16
	}
	if db < 0 || db >= max {
		return fmt.Errorf("redis database must be 0-%d, got %d", max-1, db)
	}
	return nil
}

// parseRedisField extracts a value from redis INFO output (key:value format).
func parseRedisField(info, field string) string {
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
		}
	}
	return ""
}
