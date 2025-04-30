package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	defer func() { _ = resp.Body.Close() }()

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
