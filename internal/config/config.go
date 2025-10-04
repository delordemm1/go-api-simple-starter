package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all the configuration for the application.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Google    GoogleConfig
	Apple     AppleConfig
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
	Port string `mapstructure:"PORT"`
	Env  string `mapstructure:"ENV"`
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	URL string `mapstructure:"URL"`
}

// RedisConfig holds the Redis configuration.
type RedisConfig struct {
	URL string `mapstructure:"URL"`
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

	// --- Read Configuration ---
	if err := viper.ReadInConfig(); err != nil {
		// Only log a warning if the .env file is not found.
		// We can still proceed if all config is set via environment variables.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("❌ Error reading config file: %s", err)
		}
	}

	// --- Unmarshal configuration into our struct ---
	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("❌ Unable to decode config into struct: %v", err)
	}

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
