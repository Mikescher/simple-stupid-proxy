package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	ssp "go-proxy"
	"os"
)

func main() {
	ssp.SetupLogger()
	ssp.LoadConfig()

	gin.SetMode(gin.ReleaseMode) // Use ReleaseMode for production
	router := gin.New()          // Use gin.New() instead of gin.Default() for custom middleware/logging

	// Add Gin's recovery middleware
	router.Use(gin.Recovery())

	// Custom logger middleware using Zerolog
	router.Use(ssp.LogMiddleware())

	// Define the proxy route with authentication middleware
	// It captures everything after the initial '/'
	proxyGroup := router.Group("/")
	proxyGroup.Use(ssp.AuthMiddleware())
	{
		// Handle ANY HTTP method for the proxy path
		proxyGroup.Any("/*proxyPath", ssp.ProxyHandler)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "80" // Default port
	}

	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("address", addr).Msg("Starting server")

	// Start the server
	if err := router.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
