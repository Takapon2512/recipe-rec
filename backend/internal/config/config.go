package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DBDSN             string
	CognitoRegion     string
	CognitoUserPoolID string
	CognitoClientID   string
	BedrockRegion     string
	BedrockModelID    string
	AllowedOrigins    []string
	Env               string
}

func Load() (*Config, error) {
	// .env が存在する場合のみ読み込む（本番環境では環境変数を直接設定するため無視）
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DBDSN:             os.Getenv("DB_DSN"),
		CognitoRegion:     os.Getenv("COGNITO_REGION"),
		CognitoUserPoolID: os.Getenv("COGNITO_USER_POOL_ID"),
		CognitoClientID:   os.Getenv("COGNITO_CLIENT_ID"),
		BedrockRegion:     os.Getenv("BEDROCK_REGION"),
		BedrockModelID:    os.Getenv("BEDROCK_MODEL_ID"),
		AllowedOrigins:    strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		Env:               getEnv("ENV", "development"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) validate() error {
	required := map[string]string{
		"DB_DSN":               c.DBDSN,
		"COGNITO_REGION":       c.CognitoRegion,
		"COGNITO_USER_POOL_ID": c.CognitoUserPoolID,
		"COGNITO_CLIENT_ID":    c.CognitoClientID,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("環境変数 %s が設定されていません", key)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
