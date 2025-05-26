package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSslMode  string

	AWSRegion        string
	SQSEventQueueURL string
	IoTMQTTEndpoint  string

	JWTSecret          string        // Secret key cho JWT
	JWTExpirationHours time.Duration // Thời gian hết hạn của JWT

	// NEW: Gate Event Settings
	GateEventTimeoutMinutes  int           // Thời gian timeout cho gate events (default: 5 phút)
	GateEventCleanupInterval time.Duration // Interval cho cleanup job (default: 1 phút)
	LPRConfidenceThreshold   float32       // Ngưỡng confidence để auto-create session (default: 0.8)

	// WebSocket Settings
	WebSocketReadBufferSize  int // Default: 1024
	WebSocketWriteBufferSize int // Default: 1024

	// Logging Settings
	EnableStructuredLogging bool   // Enable structured JSON logging
	LogLevel                string // DEBUG, INFO, WARN, ERROR
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Cảnh báo: Không thể tải file .env: %v", err)
	}

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))

	jwtExpHours, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24")) // Mặc định 24 giờ

	// NEW: Gate Event Config
	gateEventTimeout, _ := strconv.Atoi(getEnv("GATE_EVENT_TIMEOUT_MINUTES", "5"))
	cleanupIntervalMin, _ := strconv.Atoi(getEnv("GATE_EVENT_CLEANUP_INTERVAL_MINUTES", "1"))
	lprThreshold, _ := strconv.ParseFloat(getEnv("LPR_CONFIDENCE_THRESHOLD", "0.8"), 32)

	// WebSocket Config
	wsReadBuffer, _ := strconv.Atoi(getEnv("WEBSOCKET_READ_BUFFER_SIZE", "1024"))
	wsWriteBuffer, _ := strconv.Atoi(getEnv("WEBSOCKET_WRITE_BUFFER_SIZE", "1024"))

	// Logging Config
	enableStructuredLogging, _ := strconv.ParseBool(getEnv("ENABLE_STRUCTURED_LOGGING", "false"))

	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     dbPort,
		DBUser:     getEnv("DB_USER", "youruser"),         // << THAY THẾ
		DBPassword: getEnv("DB_PASSWORD", "yourpassword"), // << THAY THẾ
		DBName:     getEnv("DB_NAME", "parking_db"),       // << THAY THẾ
		DBSslMode:  getEnv("DB_SSLMODE", "disable"),

		AWSRegion:        getEnv("AWS_REGION", "ap-southeast-1"), // << THAY BẰNG REGION CỦA BẠN
		SQSEventQueueURL: getEnv("SQS_EVENT_QUEUE_URL", ""),      // << ĐIỀN URL SQS QUEUE
		IoTMQTTEndpoint:  getEnv("IOT_MQTT_ENDPOINT", ""),        // << ĐIỀN AWS IOT ENDPOINT

		JWTSecret:          getEnv("JWT_SECRET", "your-very-secret-key-for-jwt-!@#$"), // << THAY BẰNG SECRET KEY MẠNH HƠN
		JWTExpirationHours: time.Duration(jwtExpHours) * time.Hour,

		// NEW: Gate Event Settings
		GateEventTimeoutMinutes:  gateEventTimeout,
		GateEventCleanupInterval: time.Duration(cleanupIntervalMin) * time.Minute,
		LPRConfidenceThreshold:   float32(lprThreshold),

		// WebSocket Settings
		WebSocketReadBufferSize:  wsReadBuffer,
		WebSocketWriteBufferSize: wsWriteBuffer,

		// Logging Settings
		EnableStructuredLogging: enableStructuredLogging,
		LogLevel:                getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Biến môi trường '%s' không được đặt, sử dụng giá trị mặc định: '%s'", key, fallback)
	return fallback
}
