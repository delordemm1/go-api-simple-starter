package config

import (
	"log"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all the configuration for the application.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Google    GoogleConfig
	Apple     AppleConfig
	SMTP      SMTPConfig
	JWTSecret string `env:"JWT_SECRET,required"`
}

type GoogleConfig struct {
	ClientID     string `env:"GOOGLE_CLIENT_ID"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	RedirectURL  string `env:"GOOGLE_REDIRECT_URL"`
}

type AppleConfig struct {
	ClientID    string `env:"APPLE_CLIENT_ID"`
	TeamID      string `env:"APPLE_TEAM_ID"`
	KeyID       string `env:"APPLE_KEY_ID"`
	PrivateKey  string `env:"APPLE_PRIVATE_KEY"`
	RedirectURL string `env:"APPLE_REDIRECT_URL"`
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
	From     string `env:"SMTP_FROM"`
	Password string `env:"SMTP_PASSWORD"`
	Username string `env:"SMTP_USERNAME"`
	Port     int    `env:"SMTP_PORT"`
	Host     string `env:"SMTP_HOST"`
}

// Load creates a new Config object from environment variables.
func Load() *Config {
	// --- Set up Viper ---
	// Set the file name of the configurations file
	viper.SetConfigName(".env")
	// Set the type of the configuration file
	viper.SetConfigType("env")
	// Add the path to look for the configurations file
	viper.AddConfigPath(".")
	// Automatically read environment variables
	viper.AutomaticEnv()
	// Use a replacer to map env vars like SERVER_PORT to Server.Port
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Load .env into process environment for BindEnv to work with file-based envs
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ godotenv could not load .env: %v", err)
	} else {
		log.Printf("ℹ️ .env loaded into process environment via godotenv")
	}

	// Bind structured keys to environment variables
	_ = viper.BindEnv("server.port", "SERVER_PORT")
	_ = viper.BindEnv("server.env", "SERVER_ENV")
	_ = viper.BindEnv("database.url", "DATABASE_URL")
	_ = viper.BindEnv("redis.url", "REDIS_URL")
	_ = viper.BindEnv("jwtsecret", "JWT_SECRET")
	_ = viper.BindEnv("google.clientid", "GOOGLE_CLIENT_ID")
	_ = viper.BindEnv("google.clientsecret", "GOOGLE_CLIENT_SECRET")
	_ = viper.BindEnv("google.redirecturl", "GOOGLE_REDIRECT_URL")
	_ = viper.BindEnv("apple.clientid", "APPLE_CLIENT_ID")
	_ = viper.BindEnv("apple.teamid", "APPLE_TEAM_ID")
	_ = viper.BindEnv("apple.keyid", "APPLE_KEY_ID")
	_ = viper.BindEnv("apple.privatekey", "APPLE_PRIVATE_KEY")
	_ = viper.BindEnv("apple.redirecturl", "APPLE_REDIRECT_URL")

	// --- Read Configuration ---
	if err := viper.ReadInConfig(); err != nil {
		// Only log a warning if the .env file is not found.
		// We can still proceed if all config is set via environment variables.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("❌ Error reading config file: %s", err)
		} else {
			log.Printf("⚠️ .env file not found, relying on environment variables")
		}
	} else {
		log.Printf("ℹ️ Using config file: %s", viper.ConfigFileUsed())
	}

	// Debug what Viper currently sees for relevant keys
	log.Printf("🔎 Viper debug (raw keys): SERVER_PORT=%q SERVER_ENV=%q DATABASE_URL=%q REDIS_URL=%q JWT_SECRET=%q GOOGLE_CLIENT_ID=%q",
		viper.GetString("SERVER_PORT"),
		viper.GetString("SERVER_ENV"),
		viper.GetString("DATABASE_URL"),
		viper.GetString("REDIS_URL"),
		viper.GetString("JWT_SECRET"),
		viper.GetString("GOOGLE_CLIENT_ID"),
	)
	log.Printf("🔎 Viper debug (structured keys): server.port=%q server.env=%q database.url=%q redis.url=%q google.clientid=%q jwtsecret=%q",
		viper.GetString("server.port"),
		viper.GetString("server.env"),
		viper.GetString("database.url"),
		viper.GetString("redis.url"),
		viper.GetString("google.clientid"),
		viper.GetString("jwtsecret"),
	)

	// --- Unmarshal configuration into our struct ---
	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("❌ Unable to decode config into struct: %v", err)
	}

	// Log resulting struct values for verification
	log.Printf("🔎 Config after Unmarshal: Server.Port=%q Server.Env=%q Database.URL=%q Redis.URL=%q JWTSecretEmpty=%t",
		cfg.Server.Port,
		cfg.Server.Env,
		cfg.Database.URL,
		cfg.Redis.URL,
		cfg.JWTSecret == "",
	)

	// --- Set default values ---
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Env == "" {
		cfg.Server.Env = "development"
	}

	log.Println("✅ Configuration loaded successfully")
	return &cfg
}
