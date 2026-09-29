package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// parseOrigins splits a comma-separated CORS_ORIGIN value
// (e.g. "http://localhost:7540,http://192.168.1.10:7540").
func parseOrigins(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(strings.TrimSuffix(p, "/")); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// originAllowed reports whether origin may call the API. Exact allowlist
// matches always pass; loopback origins (localhost / 127.0.0.1 / ::1, any
// port, http or https) pass for local dev over LAN-agnostic hostnames.
func originAllowed(origin string, allowed []string) bool {
	norm := strings.TrimSuffix(strings.TrimSpace(origin), "/")
	for _, a := range allowed {
		if norm == a {
			return true
		}
	}
	lower := strings.ToLower(norm)
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		if strings.HasPrefix(lower, "http://"+host) || strings.HasPrefix(lower, "https://"+host) {
			return true
		}
	}
	return false
}

func CORS(allowedOrigins string) gin.HandlerFunc {
	allowed := parseOrigins(allowedOrigins)
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// Echo the request origin when it is allowlisted (or when it is
			// a loopback dev origin). Echoing keeps `Allow-Credentials: true`
			// valid for browsers; unknown origins get no ACAO header.
			if originAllowed(origin, allowed) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		} else if len(allowed) > 0 {
			c.Header("Access-Control-Allow-Origin", allowed[0])
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Agent-ID, X-Browse-Token")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
