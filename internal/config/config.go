// Package config handles application configuration using environment variables.
// Configuration follows the 12-factor app methodology with sensible defaults.
package config

import (
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	RabbitMQ  RabbitMQConfig
	Redis     RedisConfig
	WhatsApp  WhatsAppConfig
	Auth      AuthConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Host           string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port           int           `envconfig:"SERVER_PORT" default:"9090"`
	ReadTimeout    time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"30s"`
	WriteTimeout   time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout    time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"120s"`
	MaxHeaderBytes int           `envconfig:"SERVER_MAX_HEADER_BYTES" default:"1048576"`
}

type DatabaseConfig struct {
	Host         string        `envconfig:"DB_HOST" default:"localhost"`
	Port         int           `envconfig:"DB_PORT" default:"5432"`
	User         string        `envconfig:"DB_USER" default:"wachat"`
	Password     string        `envconfig:"DB_PASSWORD" default:"wachat_secret"`
	Database     string        `envconfig:"DB_NAME" default:"wa_gateway"`
	MaxOpenConns int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	MaxLifetime  time.Duration `envconfig:"DB_MAX_LIFETIME" default:"5m"`
}

func (d DatabaseConfig) DSN() string {
	return "host=" + d.Host + " port=" + itoa(d.Port) + " user=" + d.User + " password=" + d.Password + " dbname=" + d.Database + " sslmode=disable"
}

type RabbitMQConfig struct {
	Host     string `envconfig:"RABBITMQ_HOST" default:"localhost"`
	Port     int    `envconfig:"RABBITMQ_PORT" default:"5672"`
	User     string `envconfig:"RABBITMQ_USER" default:"wachat"`
	Password string `envconfig:"RABBITMQ_PASSWORD" default:"wachat_secret"`
	VHost    string `envconfig:"RABBITMQ_VHOST" default:"/"`
}

func (r RabbitMQConfig) URL() string {
	return "amqp://" + r.User + ":" + r.Password + "@" + r.Host + ":" + itoa(r.Port) + r.VHost
}

type RedisConfig struct {
	Host     string `envconfig:"REDIS_HOST" default:"localhost"`
	Port     int    `envconfig:"REDIS_PORT" default:"6379"`
	Password string `envconfig:"REDIS_PASSWORD" default:"wachat_secret"`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func (r RedisConfig) Addr() string {
	return r.Host + ":" + itoa(r.Port)
}

type WhatsAppConfig struct {
	PhoneNumberID string `envconfig:"WHATSAPP_PHONE_NUMBER_ID" default:"test_phone_id"`
	WABAID        string `envconfig:"WHATSAPP_WABA_ID" default:""`
	AccessToken   string `envconfig:"WHATSAPP_ACCESS_TOKEN" default:"test_token"`
	APIVersion    string `envconfig:"WHATSAPP_API_VERSION" default:"v20.0"`
	VerifyToken   string `envconfig:"WHATSAPP_VERIFY_TOKEN" default:"verify_token"`
	WebhookSecret string `envconfig:"WHATSAPP_WEBHOOK_SECRET" default:""`
}

type AuthConfig struct {
	JWT secretConfig
	API APIKeyConfig
}

type secretConfig struct {
	Secret          string        `envconfig:"JWT_SECRET" default:"dev-secret-key"`
	ExpiryDuration  time.Duration `envconfig:"JWT_EXPIRY_DURATION" default:"24h"`
	RefreshDuration time.Duration `envconfig:"JWT_REFRESH_DURATION" default:"168h"`
}

type APIKeyConfig struct {
	HeaderName string `envconfig:"API_KEY_HEADER" default:"X-API-Key"`
}

type RateLimitConfig struct {
	RequestsPerSecond int           `envconfig:"RATE_LIMIT_RPS" default:"20"`
	Burst             int           `envconfig:"RATE_LIMIT_BURST" default:"50"`
	RedisKeyPrefix    string        `envconfig:"RATE_LIMIT_KEY_PREFIX" default:"ratelimit:"`
	WindowSize        time.Duration `envconfig:"RATE_LIMIT_WINDOW" default:"1s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadFromFile(path string) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := parseYAML(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseYAML(data []byte, cfg *Config) error {
	return yaml.Unmarshal(data, cfg)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var result []byte
	negative := false
	if i < 0 {
		negative = true
		i = -i
	}

	for i > 0 {
		result = append([]byte{'0' + byte(i%10)}, result...)
		i /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}
