package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger interface untuk structured logging
type Logger interface {
	Info(ctx context.Context, message string, fields map[string]interface{})
	Error(ctx context.Context, message string, err error, fields map[string]interface{})
	Warn(ctx context.Context, message string, fields map[string]interface{})
	Debug(ctx context.Context, message string, fields map[string]interface{})
	WithFields(fields map[string]interface{}) Logger
}

// structuredLogger implementasi Logger dengan logrus
type structuredLogger struct {
	logger *logrus.Logger
	fields map[string]interface{}
}

// LogEntry representasi structured log entry
type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	CorrelationID string               `json:"correlation_id,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Service     string                 `json:"service"`
	Method      string                 `json:"method,omitempty"`
	Duration    string                 `json:"duration,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	IP          string                 `json:"ip,omitempty"`
}

// LoggerConfig configuration untuk logger
type LoggerConfig struct {
	Level                  string
	Format                 string
	CorrelationIDHeader    string
	EnableRequestLog       bool
	EnableResponseLog      bool
	ServiceName           string
}

// NewStructuredLogger membuat instance baru dari structured logger
func NewStructuredLogger(config LoggerConfig) Logger {
	logrusLogger := logrus.New()
	
	// Set level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrusLogger.SetLevel(level)
	
	// Set format
	if config.Format == "json" {
		logrusLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	} else {
		logrusLogger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339Nano,
			FullTimestamp:   true,
		})
	}
	
	// Set output
	logrusLogger.SetOutput(os.Stdout)
	
	return &structuredLogger{
		logger: logrusLogger,
		fields: map[string]interface{}{
			"service": config.ServiceName,
		},
	}
}

// Info logging untuk informational messages
func (l *structuredLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	entry := l.createEntry(ctx, "INFO", message, nil, fields)
	l.log(entry)
}

// Error logging untuk error messages
func (l *structuredLogger) Error(ctx context.Context, message string, err error, fields map[string]interface{}) {
	entry := l.createEntry(ctx, "ERROR", message, err, fields)
	l.log(entry)
}

// Warn logging untuk warning messages
func (l *structuredLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	entry := l.createEntry(ctx, "WARN", message, nil, fields)
	l.log(entry)
}

// Debug logging untuk debug messages
func (l *structuredLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	entry := l.createEntry(ctx, "DEBUG", message, nil, fields)
	l.log(entry)
}

// WithFields membuat logger baru dengan additional fields
func (l *structuredLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &structuredLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// createEntry membuat log entry dengan context information
func (l *structuredLogger) createEntry(ctx context.Context, level, message string, err error, fields map[string]interface{}) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}
	
	// Add correlation ID from context
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		entry.CorrelationID = correlationID.(string)
	}
	
	// Add error if present
	if err != nil {
		entry.Error = err.Error()
	}
	
	// Add base fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	
	// Add additional fields
	for k, v := range fields {
		entry.Fields[k] = v
	}
	
	// Add caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		funcName := runtime.FuncForPC(pc).Name()
		entry.Fields["caller"] = fmt.Sprintf("%s:%d %s", file, line, funcName)
	}
	
	return entry
}

// log menulis log entry
func (l *structuredLogger) log(entry LogEntry) {
	fields := logrus.Fields{}
	
	// Add correlation ID
	if entry.CorrelationID != "" {
		fields["correlation_id"] = entry.CorrelationID
	}
	
	// Add error
	if entry.Error != "" {
		fields["error"] = entry.Error
	}
	
	// Add other fields
	for k, v := range entry.Fields {
		fields[k] = v
	}
	
	// Marshal to JSON for structured logging
	if jsonData, err := json.Marshal(entry); err == nil {
		fields["structured_data"] = string(jsonData)
	}
	
	// Log based on level
	switch entry.Level {
	case "INFO":
		l.logger.WithFields(fields).Info(entry.Message)
	case "ERROR":
		l.logger.WithFields(fields).Error(entry.Message)
	case "WARN":
		l.logger.WithFields(fields).Warn(entry.Message)
	case "DEBUG":
		l.logger.WithFields(fields).Debug(entry.Message)
	default:
		l.logger.WithFields(fields).Info(entry.Message)
	}
}

// Helper functions untuk common logging scenarios

// LogAuthEvent untuk authentication events
func LogAuthEvent(ctx context.Context, logger Logger, event string, userID, ip string, success bool, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["event_type"] = "auth"
	fields["auth_event"] = event
	fields["user_id"] = userID
	fields["ip"] = ip
	fields["success"] = success
	
	level := "INFO"
	message := fmt.Sprintf("Auth event: %s", event)
	if !success {
		level = "WARN"
		message = fmt.Sprintf("Auth event failed: %s", event)
	}
	
	switch level {
	case "INFO":
		logger.Info(ctx, message, fields)
	case "WARN":
		logger.Warn(ctx, message, fields)
	}
}

// LogSecurityEvent untuk security events
func LogSecurityEvent(ctx context.Context, logger Logger, event string, severity string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["event_type"] = "security"
	fields["security_event"] = event
	fields["severity"] = severity
	
	message := fmt.Sprintf("Security event: %s", event)
	
	switch severity {
	case "HIGH":
		logger.Error(ctx, message, nil, fields)
	case "MEDIUM":
		logger.Warn(ctx, message, fields)
	case "LOW":
		logger.Info(ctx, message, fields)
	default:
		logger.Info(ctx, message, fields)
	}
}

// LogPerformance untuk performance metrics
func LogPerformance(ctx context.Context, logger Logger, operation string, duration time.Duration, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["event_type"] = "performance"
	fields["operation"] = operation
	fields["duration_ms"] = duration.Milliseconds()
	fields["duration_human"] = duration.String()
	
	message := fmt.Sprintf("Performance: %s took %s", operation, duration)
	logger.Info(ctx, message, fields)
}