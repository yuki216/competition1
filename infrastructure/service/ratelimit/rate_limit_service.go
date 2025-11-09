package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RateLimitService mendefinisikan interface untuk rate limiting
type RateLimitService interface {
	CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	Increment(ctx context.Context, key string, window time.Duration) error
	Block(ctx context.Context, key string, duration time.Duration, reason string) error
	IsBlocked(ctx context.Context, key string) (bool, error)
	GetAttempts(ctx context.Context, key string) (int, error)
}

// rateLimitService implementasi RateLimitService dengan Redis
type rateLimitService struct {
	redisClient *redis.Client
	logger      *logrus.Logger
	enabled     bool
}

// RateLimitConfig configuration untuk rate limiting
type RateLimitConfig struct {
	Enabled         bool
	RedisURL        string
	IPAttempts      int
	IPWindow        time.Duration
	UserAttempts    int
	UserWindow      time.Duration
	BlockDuration   time.Duration
}

// NewRateLimitService membuat instance baru dari RateLimitService
func NewRateLimitService(config RateLimitConfig, logger *logrus.Logger) (RateLimitService, error) {
	if !config.Enabled {
		logger.Info("Rate limiting disabled")
		return &noopRateLimitService{}, nil
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	redisClient := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"ip_attempts":   config.IPAttempts,
		"ip_window":     config.IPWindow,
		"user_attempts": config.UserAttempts,
		"user_window":   config.UserWindow,
		"block_duration": config.BlockDuration,
	}).Info("Rate limiting service initialized")

	return &rateLimitService{
		redisClient: redisClient,
		logger:      logger,
		enabled:     true,
	}, nil
}

// CheckLimit mengecek apakah limit telah tercapai
func (s *rateLimitService) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	currentCount, err := s.GetAttempts(ctx, key)
	if err != nil {
		return false, err
	}

	isUnderLimit := currentCount < limit
	
	s.logger.WithContext(ctx).WithFields(logrus.Fields{
		"key":        key,
		"current":    currentCount,
		"limit":      limit,
		"under_limit": isUnderLimit,
	}).Debug("Rate limit check")

	return isUnderLimit, nil
}

// Increment menambah counter untuk key tertentu
func (s *rateLimitService) Increment(ctx context.Context, key string, window time.Duration) error {
	pipeline := s.redisClient.Pipeline()
	
	// Increment counter
	incrCmd := pipeline.Incr(ctx, key)
	
	// Set expiration jika key baru
	pipeline.Expire(ctx, key, window)
	
	_, err := pipeline.Exec(ctx)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to increment rate limit counter")
		return fmt.Errorf("failed to increment rate limit: %w", err)
	}

	count := incrCmd.Val()
	
	s.logger.WithContext(ctx).WithFields(logrus.Fields{
		"key":   key,
		"count": count,
		"window": window,
	}).Info("Rate limit incremented")

	return nil
}

// Block memblokir key untuk durasi tertentu
func (s *rateLimitService) Block(ctx context.Context, key string, duration time.Duration, reason string) error {
	blockKey := fmt.Sprintf("blocked:%s", key)
	
	// Simpan informasi block
	blockData := map[string]interface{}{
		"reason":     reason,
		"blocked_at": time.Now().Unix(),
		"duration":   duration.Seconds(),
		"correlation_id": ctx.Value("correlation_id"),
	}

	pipeline := s.redisClient.Pipeline()
	pipeline.HSet(ctx, blockKey, blockData)
	pipeline.Expire(ctx, blockKey, duration)
	
	_, err := pipeline.Exec(ctx)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to block key")
		return fmt.Errorf("failed to block key: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logrus.Fields{
		"key":      key,
		"duration": duration,
		"reason":   reason,
	}).Warn("Key blocked due to rate limit exceeded")

	return nil
}

// IsBlocked mengecek apakah key sedang diblokir
func (s *rateLimitService) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockKey := fmt.Sprintf("blocked:%s", key)
	
	exists, err := s.redisClient.Exists(ctx, blockKey).Result()
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to check block status")
		return false, fmt.Errorf("failed to check block status: %w", err)
	}

	isBlocked := exists > 0
	
	if isBlocked {
		s.logger.WithContext(ctx).WithFields(logrus.Fields{
			"key": key,
		}).Warn("Key is blocked")
	}

	return isBlocked, nil
}

// GetAttempts mendapatkan jumlah attempts untuk key
func (s *rateLimitService) GetAttempts(ctx context.Context, key string) (int, error) {
	count, err := s.redisClient.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // Key doesn't exist, return 0
		}
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get attempts count")
		return 0, fmt.Errorf("failed to get attempts: %w", err)
	}

	return count, nil
}

// noopRateLimitService implementasi no-op untuk ketika rate limiting disabled
type noopRateLimitService struct{}

func (n *noopRateLimitService) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return true, nil // Always allow
}

func (n *noopRateLimitService) Increment(ctx context.Context, key string, window time.Duration) error {
	return nil // No-op
}

func (n *noopRateLimitService) Block(ctx context.Context, key string, duration time.Duration, reason string) error {
	return nil // No-op
}

func (n *noopRateLimitService) IsBlocked(ctx context.Context, key string) (bool, error) {
	return false, nil // Never blocked
}

func (n *noopRateLimitService) GetAttempts(ctx context.Context, key string) (int, error) {
	return 0, nil // Always 0
}