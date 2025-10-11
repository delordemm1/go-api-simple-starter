package config

import (
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all the configuration for the application.
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Google       GoogleConfig       `mapstructure:"google"`
	Apple        AppleConfig        `mapstructure:"apple"`
	SMTP         SMTPConfig         `mapstructure:"smtp"`
	Templates    TemplatesConfig    `mapstructure:"templates"`
	Verification VerificationConfig `mapstructure:"verification"`
	ResetToken   ResetTokenConfig   `mapstructure:"reset_token"`
	JWTSecret    string             `mapstructure:"jwt_secret" env:"JWT_SECRET"`
}

type GoogleConfig struct {
	ClientID     string `mapstructure:"client_id" env:"GOOGLE_CLIENT_ID"`
	ClientSecret string `mapstructure:"client_secret" env:"GOOGLE_CLIENT_SECRET"`
	RedirectURL  string `mapstructure:"redirect_url" env:"GOOGLE_REDIRECT_URL"`
}

type AppleConfig struct {
	ClientID    string `mapstructure:"client_id" env:"APPLE_CLIENT_ID"`
	TeamID      string `mapstructure:"team_id" env:"APPLE_TEAM_ID"`
	KeyID       string `mapstructure:"key_id" env:"APPLE_KEY_ID"`
	PrivateKey  string `mapstructure:"private_key" env:"APPLE_PRIVATE_KEY"`
	RedirectURL string `mapstructure:"redirect_url" env:"APPLE_REDIRECT_URL"`
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

// RedisConfig holds the Redis configuration.
type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type SMTPConfig struct {
	From     string `mapstructure:"from" env:"SMTP_FROM"`
	Password string `mapstructure:"password" env:"SMTP_PASSWORD"`
	Username string `mapstructure:"username" env:"SMTP_USERNAME"`
	Port     int    `mapstructure:"port" env:"SMTP_PORT"`
	Host     string `mapstructure:"host" env:"SMTP_HOST"`
}

type TemplatesConfig struct {
	Dir    string `mapstructure:"dir" env:"EMAIL_TEMPLATES_DIR"`
	Reload bool   `mapstructure:"reload" env:"TEMPLATES_RELOAD"`
}

type VerificationConfig struct {
	TTLMinutes            int `mapstructure:"ttl_minutes" env:"VERIFICATION_TTL_MINUTES"`
	ResendCooldownSeconds int `mapstructure:"resend_cooldown_seconds" env:"VERIFICATION_RESEND_COOLDOWN_SECONDS"`
	MaxAttempts           int `mapstructure:"max_attempts" env:"VERIFICATION_MAX_ATTEMPTS"`
}

type ResetTokenConfig struct {
	TTLMinutes int `mapstructure:"ttl_minutes" env:"RESET_TOKEN_TTL_MINUTES"`
}

// --- Helpers for auto-binding env vars ---

var (
	reFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	reAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(s string) string {
	snake := reFirstCap.ReplaceAllString(s, "${1}_${2}")
	snake = reAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func bindEnvsFromStruct(prefix string, t reflect.Type) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}

		keyPart := f.Tag.Get("mapstructure")
		if keyPart == "" {
			keyPart = toSnakeCase(f.Name)
		}

		key := keyPart
		if prefix != "" {
			key = prefix + "." + keyPart
		}

		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			bindEnvsFromStruct(key, ft)
			continue
		}

		envName := f.Tag.Get("env")
		if envName == "" {
			envName = strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		}
		_ = viper.BindEnv(key, envName)
	}
}

// Load creates a new Config object from environment variables.
func Load() *Config {
	// Load .env into process environment (no Viper file read)
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ godotenv could not load .env: %v", err)
	} else {
		log.Printf("ℹ️ .env loaded into process environment via godotenv")
	}

	// Read environment variables only
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Sane defaults (ensure keys exist for Unmarshal)
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.env", "development")
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("templates.reload", false)

	// Verification & Reset token defaults
	viper.SetDefault("verification.ttl_minutes", 10)
	viper.SetDefault("verification.resend_cooldown_seconds", 60)
	viper.SetDefault("verification.max_attempts", 5)
	viper.SetDefault("reset_token.ttl_minutes", 15)

	// Auto-bind env vars for all config leaves
	bindEnvsFromStruct("", reflect.TypeOf(Config{}))

	// Unmarshal configuration into our struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("❌ Unable to decode config into struct: %v", err)
	}

	log.Println("✅ Configuration loaded successfully")
	return &cfg
}
