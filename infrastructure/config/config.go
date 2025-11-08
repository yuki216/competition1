package config

import (
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"
    "strings"

    "github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	JWTPrivateKey      string
	JWTPublicKey       string
	JWTAlgorithm       string
	RefreshTokenSalt   string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	ServerPort         string
	ServerHost         string
	Environment        string
	RecaptchaSecret    string
	RecaptchaEnabled   bool
	RecaptchaSkip      bool
	RecaptchaTimeout   time.Duration
	RecaptchaSiteKey   string
	RedisURL           string
	RateLimitEnabled   bool
	RateLimitIPAttempts int
	RateLimitIPWindow  time.Duration
	RateLimitUserAttempts int
	RateLimitUserWindow time.Duration
	RateLimitBlockDuration time.Duration
	LogLevel           string
	LogFormat          string
	LogCorrelationIDHeader string
	LogEnableRequestLog  bool
	LogEnableResponseLog bool

	// AI configuration (partial, aligned with internal)
	AIProvider        string
	AIEmbeddingModel  string
	AISuggestionModel string
	AIEmbeddingDim    int
	AITopK            int
	AIMinConfidence   float64
	AITimeoutMs       int
	AIEnableCache     bool
	AICacheTTLMin     int
	AIChunkSize       int
	AIChunkOverlap    int
	AIBatchSize       int
	AIMaxConcurrency  int
	AIMockMode        bool
	OpenAIAPIKey      string
	ZAIAPIKey         string

	// CORS configuration
	CORSEnabled          bool
	CORSAllowedOrigins   []string
	CORSAllowCredentials bool

	// SSE configuration
	SSEEnabled           bool
	SSEFlushInterval     time.Duration
	SSEHeartbeatInterval time.Duration
	SSEMaxConnections    int
	SSEMessageBufferSize int
	SSEClientTimeout     time.Duration
}

var (
	ErrMissingDatabaseURL  = errors.New("DATABASE_URL is required")
	ErrMissingJWTSecret    = errors.New("JWT_SECRET is required")
	ErrMissingRefreshSalt  = errors.New("REFRESH_TOKEN_SALT is required")
	ErrInvalidTokenTTL     = errors.New("invalid token TTL format")
	ErrInvalidJWTAlgorithm = errors.New("invalid JWT algorithm")
	ErrMissingRecaptchaSecret = errors.New("RECAPTCHA_SECRET is required when reCAPTCHA is enabled")
)

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()
	
	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTPrivateKey:      os.Getenv("JWT_PRIVATE_KEY"),
		JWTPublicKey:       os.Getenv("JWT_PUBLIC_KEY"),
		JWTAlgorithm:       getEnvOrDefault("JWT_ALG", "HS256"),
		RefreshTokenSalt:   os.Getenv("REFRESH_TOKEN_SALT"),
		ServerPort:         getEnvOrDefault("SERVER_PORT", "8080"),
		ServerHost:         getEnvOrDefault("SERVER_HOST", "localhost"),
		Environment:        getEnvOrDefault("ENV", "development"),
		RecaptchaSecret:    os.Getenv("RECAPTCHA_SECRET"),
		RecaptchaEnabled:   getEnvOrDefaultBool("RECAPTCHA_ENABLED", false),
		RecaptchaSkip:      getEnvOrDefaultBool("RECAPTCHA_SKIP", false),
		RecaptchaSiteKey:   os.Getenv("RECAPTCHA_SITE_KEY"),
		RedisURL:           getEnvOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		RateLimitEnabled:   getEnvOrDefaultBool("RATE_LIMIT_ENABLED", true),
		LogLevel:           getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:          getEnvOrDefault("LOG_FORMAT", "json"),
		LogCorrelationIDHeader: getEnvOrDefault("LOG_CORRELATION_ID_HEADER", "X-Correlation-ID"),
		LogEnableRequestLog:  getEnvOrDefaultBool("LOG_ENABLE_REQUEST_LOG", true),
		LogEnableResponseLog: getEnvOrDefaultBool("LOG_ENABLE_RESPONSE_LOG", false),

		// AI configuration
		AIProvider:        getEnvOrDefault("AI_PROVIDER", "mock"),
		OpenAIAPIKey:      getEnvOrDefault("OPENAI_API_KEY", ""),
		ZAIAPIKey:         getEnvOrDefault("ZAI_API_KEY", ""),
		AIEmbeddingModel:  getEnvOrDefault("AI_EMBEDDING_MODEL", "text-embedding-ada-002"),
		AISuggestionModel: getEnvOrDefault("AI_SUGGESTION_MODEL", "gpt-3.5-turbo"),
		AIEmbeddingDim:    getEnvOrDefaultInt("AI_EMBEDDING_DIM", 1536),
		AITopK:            getEnvOrDefaultInt("AI_TOP_K", 10),
		AIMinConfidence:   getEnvOrDefaultFloat("AI_MIN_CONFIDENCE", 0.4),
		AITimeoutMs:       getEnvOrDefaultInt("AI_TIMEOUT_MS", 5000),
		AIEnableCache:     getEnvOrDefaultBool("AI_ENABLE_CACHE", true),
		AICacheTTLMin:     getEnvOrDefaultInt("AI_CACHE_TTL_MIN", 60),
		AIChunkSize:       getEnvOrDefaultInt("AI_CHUNK_SIZE", 800),
		AIChunkOverlap:    getEnvOrDefaultInt("AI_CHUNK_OVERLAP", 150),
		AIBatchSize:       getEnvOrDefaultInt("AI_BATCH_SIZE", 32),
		AIMaxConcurrency:  getEnvOrDefaultInt("AI_MAX_CONCURRENCY", 5),
		AIMockMode:        getEnvOrDefaultBool("AI_MOCK_MODE", true),

		CORSEnabled:          getEnvOrDefaultBool("CORS_ENABLED", true),
		CORSAllowCredentials: getEnvOrDefaultBool("CORS_ALLOW_CREDENTIALS", true),
		CORSAllowedOrigins:   parseAllowedOrigins(getEnvOrDefault("CORS_ALLOWED_ORIGINS", "")),

		// SSE configuration
		SSEEnabled:           getEnvOrDefaultBool("SSE_ENABLED", true),
		SSEFlushInterval:     getEnvOrDefaultDuration("SSE_FLUSH_INTERVAL", 250*time.Millisecond),
		SSEHeartbeatInterval: getEnvOrDefaultDuration("SSE_HEARTBEAT_INTERVAL", 15*time.Second),
		SSEMaxConnections:    getEnvOrDefaultInt("SSE_MAX_CONNECTIONS", 1000),
		SSEMessageBufferSize: getEnvOrDefaultInt("SSE_MESSAGE_BUFFER_SIZE", 256),
		SSEClientTimeout:     getEnvOrDefaultDuration("SSE_CLIENT_TIMEOUT", 30*time.Second),
	}
	
	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, ErrMissingDatabaseURL
	}
	
	// Validate JWT configuration
	if cfg.JWTAlgorithm != "HS256" && cfg.JWTAlgorithm != "RS256" && cfg.JWTAlgorithm != "ES256" {
		return nil, ErrInvalidJWTAlgorithm
	}
	
	if cfg.JWTAlgorithm == "HS256" && cfg.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	
	if cfg.RefreshTokenSalt == "" {
		return nil, ErrMissingRefreshSalt
	}
	
	// Parse token TTLs
	accessTokenTTL, err := parseTokenTTL(getEnvOrDefault("JWT_ACCESS_TOKEN_TTL", "900"))
	if err != nil {
		return nil, ErrInvalidTokenTTL
	}
	cfg.AccessTokenTTL = accessTokenTTL
	
	refreshTokenTTL, err := parseTokenTTL(getEnvOrDefault("JWT_REFRESH_TOKEN_TTL", "2592000"))
	if err != nil {
		return nil, ErrInvalidTokenTTL
	}
	cfg.RefreshTokenTTL = refreshTokenTTL
	
	// Parse reCAPTCHA timeout
	recaptchaTimeout, err := parseTokenTTL(getEnvOrDefault("RECAPTCHA_TIMEOUT", "5"))
	if err != nil {
		return nil, ErrInvalidTokenTTL
	}
	cfg.RecaptchaTimeout = recaptchaTimeout
	
	// Validate reCAPTCHA secret when enabled (and not skipped)
	if cfg.RecaptchaEnabled && !cfg.RecaptchaSkip && cfg.RecaptchaSecret == "" {
		return nil, ErrMissingRecaptchaSecret
	}

	// Validate AI provider keys if provider selected
	if cfg.AIProvider == "openai" && cfg.OpenAIAPIKey == "" {
		logMissing("OPENAI_API_KEY for AI_PROVIDER=openai")
	}
	if cfg.AIProvider == "zai" && cfg.ZAIAPIKey == "" {
		logMissing("ZAI_API_KEY for AI_PROVIDER=zai")
	}
	
	// Parse rate limiting config
	cfg.RateLimitIPAttempts = getEnvOrDefaultInt("RATE_LIMIT_IP_ATTEMPTS", 5)
	cfg.RateLimitUserAttempts = getEnvOrDefaultInt("RATE_LIMIT_USER_ATTEMPTS", 10)
	
	ipWindow, err := parseTokenTTL(getEnvOrDefault("RATE_LIMIT_IP_WINDOW", "900"))
	if err != nil {
		return nil, ErrInvalidTokenTTL
	}
	cfg.RateLimitIPWindow = ipWindow
	
	userWindow, err := parseTokenTTL(getEnvOrDefault("RATE_LIMIT_USER_WINDOW", "3600"))
	if err != nil {
		return nil, ErrInvalidTokenTTL
	}
	cfg.RateLimitUserWindow = userWindow
	
	blockDuration, err := parseTokenTTL(getEnvOrDefault("RATE_LIMIT_BLOCK_DURATION", "1800"))
	if err != nil {
		return nil, ErrInvalidTokenTTL
	}
	cfg.RateLimitBlockDuration = blockDuration
	
	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}

func parseTokenTTL(value string) (time.Duration, error) {
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

func parseAllowedOrigins(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

// getEnvOrDefaultFloat returns float64 env or default

func getEnvOrDefaultFloat(key string, defaultValue float64) float64 {
    if value := os.Getenv(key); value != "" {
        parsed, err := strconv.ParseFloat(value, 64)
        if err != nil {
            return defaultValue
        }
        return parsed
    }
    return defaultValue
}

func getEnvOrDefaultDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        // interpret as seconds if numeric, else parse like Go duration
        if n, err := strconv.Atoi(value); err == nil {
            return time.Duration(n) * time.Second
        }
        d, err := time.ParseDuration(value)
        if err != nil {
            return defaultValue
        }
        return d
    }
    return defaultValue
}

func logMissing(msg string) {
    // lightweight warning for missing optional AI keys
    // avoid introducing a logger dependency here
    fmt.Fprintf(os.Stderr, "[config] missing %s\n", msg)
}