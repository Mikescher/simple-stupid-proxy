package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

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
