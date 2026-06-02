package netguard

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidatePublicHTTPURL rejects local and private network targets before
// platform clients fetch user-configured media URLs.
func ValidatePublicHTTPURL(ctx context.Context, rawURL string, purpose string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s URL must be a valid HTTP or HTTPS URL", purpose)
	}

	host := parsed.Hostname()
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("%s URL must include a host", purpose)
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return fmt.Errorf("%s URL must not target local or private network addresses", purpose)
	}

	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("%s URL must not target local or private network addresses", purpose)
		}
		return nil
	}

	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve %s URL host: %w", purpose, err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("%s URL host did not resolve", purpose)
	}
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			return fmt.Errorf("%s URL must not target local or private network addresses", purpose)
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return !(ip.IsUnspecified() ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast())
}
