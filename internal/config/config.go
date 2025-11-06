package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/fixora/fixora/internal/ports"
)

// Config represents application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	AI       AIConfig       `json:"ai"`
	Redis    RedisConfig    `json:"redis"`
	Logging  LoggingConfig  `json:"logging"`
	Security SecurityConfig `json:"security"`
	SSE      SSEConfig      `json:"sse"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port         string        `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	Environment  string        `json:"environment"`
	Debug        bool          `json:"debug"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	DBName          string        `json:"dbname"`
	SSLMode         string        `json:"sslmode"`
	MaxConnections  int           `json:"max_connections"`
	MaxIdleTime     time.Duration `json:"max_idle_time"`
	ConnectTimeout  time.Duration `json:"connect_timeout"`
	QueryTimeout    time.Duration `json:"query_timeout"`
	MigrationsPath  string        `json:"migrations_path"`
}

// AIConfig represents AI service configuration
type AIConfig struct {
	Provider          string            `json:"provider"`
	APIKey           string            `json:"api_key"`
	EmbeddingModel   string            `json:"embedding_model"`
	SuggestionModel  string            `json:"suggestion_model"`
	EmbeddingDim     int               `json:"embedding_dim"`
	TopK             int               `json:"top_k"`
	MinConfidence    float64           `json:"min_confidence"`
	TimeoutMs        int               `json:"timeout_ms"`
	EnableCache      bool              `json:"enable_cache"`
	CacheTTLMin      int               `json:"cache_ttl_min"`
	ChunkSize        int               `json:"chunk_size"`
	ChunkOverlap     int               `json:"chunk_overlap"`
	BatchSize        int               `json:"batch_size"`
	MaxConcurrency   int               `json:"max_concurrency"`
	MockMode         bool              `json:"mock_mode"`
	Providers        map[string]string `json:"providers"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Password string        `json:"password"`
	DB       int           `json:"db"`
	PoolSize int           `json:"pool_size"`
	Timeout  time.Duration `json:"timeout"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"` // json, text
	Output     string `json:"output"` // stdout, file
	File       string `json:"file"`
	MaxSize    int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age_days"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	JWTSecret           string        `json:"jwt_secret"`
	JWTExpiration       time.Duration `json:"jwt_expiration"`
	CORSOrigins         []string      `json:"cors_origins"`
	RateLimitEnabled    bool          `json:"rate_limit_enabled"`
	RateLimitRequests   int           `json:"rate_limit_requests"`
	RateLimitWindow     time.Duration `json:"rate_limit_window"`
	EncryptionKey       string        `json:"encryption_key"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	PasswordMinLength   int           `json:"password_min_length"`
	RequireHTTPS        bool          `json:"require_https"`
}

// SSEConfig represents SSE streaming configuration
type SSEConfig struct {
	Enabled           bool          `json:"enabled"`
	FlushInterval     time.Duration `json:"flush_interval"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	MaxConnections    int           `json:"max_connections"`
	MessageBufferSize int          `json:"message_buffer_size"`
	ClientTimeout     time.Duration `json:"client_timeout"`
}

// Load loads configuration from environment variables and defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			Environment:  getEnv("ENVIRONMENT", "development"),
			Debug:        getEnvBool("DEBUG", true),
		},
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           getEnvInt("DB_PORT", 5432),
			User:           getEnv("DB_USER", "postgres"),
			Password:       getEnv("DB_PASSWORD", ""),
			DBName:         getEnv("DB_NAME", "fixora"),
			SSLMode:        getEnv("DB_SSLMODE", "disable"),
			MaxConnections: getEnvInt("DB_MAX_CONNECTIONS", 20),
			MaxIdleTime:    getEnvDuration("DB_MAX_IDLE_TIME", 30*time.Minute),
			ConnectTimeout: getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:   getEnvDuration("DB_QUERY_TIMEOUT", 30*time.Second),
			MigrationsPath: getEnv("DB_MIGRATIONS_PATH", "./migrations"),
		},
		AI: AIConfig{
			Provider:          getEnv("AI_PROVIDER", "mock"),
			APIKey:           getEnv("AI_API_KEY", ""),
			EmbeddingModel:   getEnv("AI_EMBEDDING_MODEL", "text-embedding-ada-002"),
			SuggestionModel:  getEnv("AI_SUGGESTION_MODEL", "gpt-3.5-turbo"),
			EmbeddingDim:     getEnvInt("AI_EMBEDDING_DIM", 1536),
			TopK:             getEnvInt("AI_TOP_K", 10),
			MinConfidence:    getEnvFloat("AI_MIN_CONFIDENCE", 0.4),
			TimeoutMs:        getEnvInt("AI_TIMEOUT_MS", 5000),
			EnableCache:      getEnvBool("AI_ENABLE_CACHE", true),
			CacheTTLMin:      getEnvInt("AI_CACHE_TTL_MIN", 60),
			ChunkSize:        getEnvInt("AI_CHUNK_SIZE", 800),
			ChunkOverlap:     getEnvInt("AI_CHUNK_OVERLAP", 150),
			BatchSize:        getEnvInt("AI_BATCH_SIZE", 32),
			MaxConcurrency:   getEnvInt("AI_MAX_CONCURRENCY", 5),
			MockMode:         getEnvBool("AI_MOCK_MODE", true),
			Providers: map[string]string{
				"openai": getEnv("OPENAI_API_KEY", ""),
				"zai":    getEnv("ZAI_API_KEY", ""),
			},
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
			Timeout:  getEnvDuration("REDIS_TIMEOUT", 5*time.Second),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			File:       getEnv("LOG_FILE", ""),
			MaxSize:    getEnvInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
			MaxAge:     getEnvInt("LOG_MAX_AGE_DAYS", 28),
		},
		Security: SecurityConfig{
			JWTSecret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			JWTExpiration:     getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			CORSOrigins:       getEnvSlice("CORS_ORIGINS", []string{"*"}),
			RateLimitEnabled:  getEnvBool("RATE_LIMIT_ENABLED", false),
			RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:   getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
			EncryptionKey:     getEnv("ENCRYPTION_KEY", ""),
			SessionTimeout:    getEnvDuration("SESSION_TIMEOUT", time.Hour),
			PasswordMinLength: getEnvInt("PASSWORD_MIN_LENGTH", 8),
			RequireHTTPS:      getEnvBool("REQUIRE_HTTPS", false),
		},
		SSE: SSEConfig{
			Enabled:           getEnvBool("SSE_ENABLED", true),
			FlushInterval:     getEnvDuration("SSE_FLUSH_INTERVAL", 250*time.Millisecond),
			HeartbeatInterval: getEnvDuration("SSE_HEARTBEAT_INTERVAL", 15*time.Second),
			MaxConnections:    getEnvInt("SSE_MAX_CONNECTIONS", 1000),
			MessageBufferSize: getEnvInt("SSE_MESSAGE_BUFFER_SIZE", 256),
			ClientTimeout:     getEnvDuration("SSE_CLIENT_TIMEOUT", 30*time.Second),
		},
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if c.AI.Provider == "" {
		return fmt.Errorf("AI provider is required")
	}

	if c.AI.Provider != "mock" && c.AI.APIKey == "" {
		return fmt.Errorf("AI API key is required for provider: %s", c.AI.Provider)
	}

	if c.Security.JWTSecret == "" || c.Security.JWTSecret == "your-secret-key-change-in-production" {
		if c.Server.Environment == "production" {
			return fmt.Errorf("JWT secret must be set in production")
		}
	}

	return nil
}

// IsProduction checks if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment checks if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// GetDatabaseURL returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// GetRedisURL returns the Redis connection URL
func (c *Config) GetRedisURL() string {
	if c.Redis.Password != "" {
		return fmt.Sprintf(":%s@%s:%d/%d", c.Redis.Password, c.Redis.Host, c.Redis.Port, c.Redis.DB)
	}
	return fmt.Sprintf("%s:%d/%d", c.Redis.Host, c.Redis.Port, c.Redis.DB)
}

// ToAIConfig converts to ports.AIConfig
func (c *Config) ToAIConfig() ports.AIConfig {
	return ports.AIConfig{
		Provider:         c.AI.Provider,
		APIKey:          c.AI.APIKey,
		EmbeddingModel:  c.AI.EmbeddingModel,
		SuggestionModel: c.AI.SuggestionModel,
		EmbeddingDim:    c.AI.EmbeddingDim,
		TopK:            c.AI.TopK,
		MinConfidence:   c.AI.MinConfidence,
		TimeoutMs:       c.AI.TimeoutMs,
		EnableCache:     c.AI.EnableCache,
		CacheTTLMin:     c.AI.CacheTTLMin,
	}
}

// Helper functions for environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		return []string{value}
	}
	return defaultValue
}