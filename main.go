package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	expectedAuthKey string
)

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

// authMiddleware checks for the presence and correctness of the 'auth' query parameter.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authKey := c.Query("auth")
		if authKey == "" {
			log.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Missing 'auth' query parameter")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing 'auth' query parameter"})
			return
		}
		if authKey != expectedAuthKey {
			log.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Invalid 'auth' query parameter")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid 'auth' query parameter"})
			return
		}
		log.Debug().Msg("Authentication successful")
		c.Next()
	}
}

// proxyHandler forwards the request to the target URL.
func proxyHandler(c *gin.Context) {
	targetPath := c.Param("proxyPath")
	if targetPath == "" {
		log.Error().Msg("proxyPath parameter is empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target path in URL"})
		return
	}

	// Remove leading slash if present
	targetPath = strings.TrimPrefix(targetPath, "/")

	// Construct the target URL (always use HTTPS)
	targetURL := url.URL{
		Scheme: "https",
		Host:   targetPath, // The path parameter contains host/path initially
	}

	// Separate host and actual path
	parts := strings.SplitN(targetPath, "/", 2)
	targetURL.Host = parts[0]
	if len(parts) > 1 {
		targetURL.Path = "/" + parts[1]
	} else {
		targetURL.Path = "/"
	}

	// Prepare query parameters, excluding 'auth'
	originalQuery := c.Request.URL.Query()
	targetQuery := url.Values{}
	for key, values := range originalQuery {
		if key != "auth" {
			targetQuery[key] = values
		}
	}
	targetURL.RawQuery = targetQuery.Encode()

	targetURLString := targetURL.String()
	log.Info().
		Str("method", c.Request.Method).
		Str("target_url", targetURLString).
		Msg("Proxying request")

	// Create the request to the target server
	proxyReq, err := http.NewRequest(c.Request.Method, targetURLString, c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create proxy request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	// Copy headers from the original request to the proxy request
	// Note: http.Client handles some headers like Host, Content-Length automatically.
	// Avoid copying connection-specific headers.
	for key, values := range c.Request.Header {
		// Filter out hop-by-hop headers and auth header if needed (though auth is query param here)
		lowerKey := strings.ToLower(key)
		if lowerKey == "connection" || lowerKey == "proxy-authenticate" || lowerKey == "proxy-authorization" || lowerKey == "te" || lowerKey == "trailers" || lowerKey == "transfer-encoding" || lowerKey == "upgrade" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}
	// Set Host header explicitly based on the target URL
	proxyReq.Host = targetURL.Host

	// Execute the request using the default HTTP client
	client := &http.Client{
		// Prevent following redirects automatically, return the redirect response directly
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second, // Add a reasonable timeout
	}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Error().Err(err).Str("target_url", targetURLString).Msg("Failed to execute proxy request")
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to reach target server: %v", err)})
		return
	}
	defer resp.Body.Close()

	log.Info().
		Str("target_url", targetURLString).
		Int("status_code", resp.StatusCode).
		Msg("Received response from target")

	// Copy headers from the target response to the client response
	for key, values := range resp.Header {
		for _, value := range values {
			// Use Add to handle multiple headers with the same name correctly
			c.Writer.Header().Add(key, value)
		}
	}

	// Set the status code
	c.Writer.WriteHeader(resp.StatusCode)

	// Copy the response body
	writtenBytes, err := io.Copy(c.Writer, resp.Body)
	if err != nil {
		// Log error, but response headers and status might already be sent
		log.Error().Err(err).Msg("Failed to copy response body")
	}
	log.Debug().Int64("bytes_written", writtenBytes).Msg("Response body copied")
}

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
		port = "8080" // Default port
	}

	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("address", addr).Msg("Starting server")

	// Start the server
	if err := router.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
