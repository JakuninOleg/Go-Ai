package config

import (
	"os"	

	"github.com/joho/godotenv"
)


type Config struct {
	Port string
}


func Load() Config {

	godotenv.Load()

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	return Config{
		Port: port,
	}
}