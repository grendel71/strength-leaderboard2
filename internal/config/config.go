package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3PublicURL string
}

func Load() Config {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
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
	return c
}
