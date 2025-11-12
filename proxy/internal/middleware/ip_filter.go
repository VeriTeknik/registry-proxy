package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/veriteknik/registry-proxy/internal/utils"
	"go.uber.org/zap"
)

var (
	allowedNetworks []*net.IPNet
	allowedIPs      []net.IP
)

// InitMetricsIPFilter parses the METRICS_ALLOWED_IPS environment variable
func InitMetricsIPFilter() error {
	allowedIPsStr := os.Getenv("METRICS_ALLOWED_IPS")
	if allowedIPsStr == "" {
		// Default to localhost only if not specified
		allowedIPsStr = "127.0.0.1,::1"
		utils.Logger.Warn("METRICS_ALLOWED_IPS not set, defaulting to localhost only")
	}

	allowedNetworks = []*net.IPNet{}
	allowedIPs = []net.IP{}

	entries := strings.Split(allowedIPsStr, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Check if it's CIDR notation
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return fmt.Errorf("invalid CIDR '%s': %w", entry, err)
			}
			allowedNetworks = append(allowedNetworks, network)
			utils.Logger.Info("Allowed metrics access from CIDR", zap.String("cidr", entry))
		} else {
			// Single IP
			ip := net.ParseIP(entry)
			if ip == nil {
				return fmt.Errorf("invalid IP address '%s'", entry)
			}
			allowedIPs = append(allowedIPs, ip)
			utils.Logger.Info("Allowed metrics access from IP", zap.String("ip", entry))
		}
	}

	return nil
}

// MetricsIPFilter is middleware that restricts access to metrics endpoint by IP
func MetricsIPFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		// Check if IP is allowed
		if !isIPAllowed(clientIP) {
			utils.Logger.Warn("Metrics access denied",
				zap.String("client_ip", clientIP),
				zap.String("path", r.URL.Path),
				zap.String("user_agent", r.UserAgent()),
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// IP is allowed, continue
		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first (for requests through reverse proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can be a comma-separated list, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isIPAllowed checks if the given IP is in the allowed list
func isIPAllowed(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check individual IPs
	for _, allowedIP := range allowedIPs {
		if ip.Equal(allowedIP) {
			return true
		}
	}

	// Check CIDR ranges
	for _, network := range allowedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
