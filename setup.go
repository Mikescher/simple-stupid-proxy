package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

var expectedAuthKey string

func setupLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func loadConfig() {
	expectedAuthKey = os.Getenv("PROXY_AUTH_KEY")
	if expectedAuthKey == "" {
		log.Fatal().Msg("PROXY_AUTH_KEY environment variable not set")
	}
	log.Info().Msg("Configuration loaded")
}
