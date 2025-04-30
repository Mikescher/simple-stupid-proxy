package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func main() {
	setupLogger()
	loadConfig()

	gin.SetMode(gin.ReleaseMode) // Use ReleaseMode for production
	router := gin.New()          // Use gin.New() instead of gin.Default() for custom middleware/logging

	// Add Gin's recovery middleware
	router.Use(gin.Recovery())

	// Custom logger middleware using Zerolog
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next() // Process request

		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		logEvent := log.Info()
		if statusCode >= 500 {
			logEvent = log.Error().Str("error", errorMessage)
		} else if statusCode >= 400 {
			logEvent = log.Warn()
		}

		if raw != "" {
			path = path + "?" + raw
		}

		logEvent.
			Int("status", statusCode).
			Str("method", method).
			Str("path", path).
			Str("ip", clientIP).
			Dur("latency", latency).
			Str("user_agent", c.Request.UserAgent()).
			Msg("Request completed")

	})

	// Define the proxy route with authentication middleware
	// It captures everything after the initial '/'
	proxyGroup := router.Group("/")
	proxyGroup.Use(authMiddleware())
	{
		// Handle ANY HTTP method for the proxy path
		proxyGroup.Any("/*proxyPath", proxyHandler)
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
