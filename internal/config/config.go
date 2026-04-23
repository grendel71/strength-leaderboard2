package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	Port       string
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3PublicURL string
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.PathEscape(c.DBUser),
		url.PathEscape(c.DBPassword),
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func Load() Config {
	c := Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),
		Port:       os.Getenv("PORT"),
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3Region:    os.Getenv("S3_REGION"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3PublicURL: os.Getenv("S3_PUBLIC_URL"),
	}
	if c.Port == "" {
		c.Port = "3000"
	}
	if c.DBPort == "" {
		c.DBPort = "5432"
	}
	if c.DBSSLMode == "" {
		c.DBSSLMode = "disable"
	}
	return c
}
