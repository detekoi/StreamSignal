package netguard

import (
	"context"
	"testing"
)

func TestValidatePublicHTTPURLRejectsLocalAndPrivateTargets(t *testing.T) {
	tests := []string{
		"http://localhost/image.png",
		"http://127.0.0.1/image.png",
		"http://10.0.0.10/image.png",
		"http://172.16.0.10/image.png",
		"http://192.168.1.10/image.png",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/image.png",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			err := ValidatePublicHTTPURL(context.Background(), value, "Test image")
			if err == nil {
				t.Fatal("expected URL to be rejected")
			}
		})
	}
}

func TestValidatePublicHTTPURLAcceptsPublicIP(t *testing.T) {
	if err := ValidatePublicHTTPURL(context.Background(), "https://93.184.216.34/image.png", "Test image"); err != nil {
		t.Fatalf("expected public URL to be accepted: %v", err)
	}
}

func TestValidatePublicHTTPURLRejectsNonHTTP(t *testing.T) {
	err := ValidatePublicHTTPURL(context.Background(), "file:///tmp/image.png", "Test image")
	if err == nil || err.Error() != "Test image URL must be a valid HTTP or HTTPS URL" {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
}
