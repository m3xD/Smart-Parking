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
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Cảnh báo: Không thể tải file .env: %v", err)
	}

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))

	jwtExpHours, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24")) // Mặc định 24 giờ

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
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Biến môi trường '%s' không được đặt, sử dụng giá trị mặc định: '%s'", key, fallback)
	return fallback
}
