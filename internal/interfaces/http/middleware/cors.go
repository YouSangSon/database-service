package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS는 Cross-Origin Resource Sharing 미들웨어입니다
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID, X-API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// CORSWithConfig는 설정 가능한 CORS 미들웨어입니다
func CORSWithConfig(allowedOrigins []string, allowedMethods []string, allowedHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

			// Set allowed headers
			if len(allowedHeaders) > 0 {
				headers := ""
				for i, header := range allowedHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
			}

			// Set allowed methods
			if len(allowedMethods) > 0 {
				methods := ""
				for i, method := range allowedMethods {
					if i > 0 {
						methods += ", "
					}
					methods += method
				}
				c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
			}

			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
