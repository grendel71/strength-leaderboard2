package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() Config {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}
	if c.Port == "" {
		c.Port = "3000"
	}
	return c
}
