package ssp

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

// AuthMiddleware checks for the presence and correctness of the 'auth' query parameter.
func AuthMiddleware() gin.HandlerFunc {
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

func LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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

	}
}
