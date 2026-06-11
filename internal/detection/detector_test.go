package detection

import (
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		input string
		want  QueryType
	}{
		{"8.8.8.8", TypeIPv4},
		{"192.168.1.1", TypeIPv4},
		{"10.0.0.1", TypeIPv4},
		{"256.256.256.256", TypeUnknown},
		{"10.0.0.0/24", TypeCIDR},
		{"192.168.0.0/16", TypeCIDR},
		{"::1", TypeIPv6},
		{"2001:db8::1", TypeIPv6},
		{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", TypeIPv6},
		{"example.com", TypeDomain},
		{"google.com", TypeDomain},
		{"sub.domain.example.com", TypeSubdomain},
		{"api.v3.service.example.org", TypeSubdomain},
		{"3g2upl4pq6kufc4m.onion", TypeOnion},
		{"facebookwkhpilnemxj7asaniu7vnjjbiltxjqhye3mhbshg7kx5tfyd.onion", TypeOnion},
		{"user@example.com", TypeEmail},
		{"john.doe@company.co.uk", TypeEmail},
		{"username123", TypeUsername},
		{"john_doe_42", TypeUsername},
		{"John Doe", TypePersonName},
		{"Alice Bob", TypePersonName},
		{"Acme Corp", TypeCompanyName},
		{"Tech Inc", TypeCompanyName},
		{"Global Solutions LLC", TypeCompanyName},
		{"+1234567890", TypePhone},
		{"+1-555-123-4567", TypePhone},
		{"0x742d35Cc6634C0532925a3b844Bc454e4438f44e", TypeEthereumAddress},
		{"0x0000000000000000000000000000000000000000", TypeEthereumAddress},
		{"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", TypeBitcoinAddress},
		{"bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq", TypeBitcoinAddress},
		{"3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy", TypeBitcoinAddress},
		{"https://example.com", TypeURL},
		{"http://google.com/path?q=1", TypeURL},
		{"d41d8cd98f00b204e9800998ecf8427e", TypeHashMD5},
		{"5d41402abc4b2a76b9719d911017c592", TypeHashMD5},
		{"da39a3ee5e6b4b0d3255bfef95601890afd80709", TypeHashSHA1},
		{"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", TypeHashSHA256},
		{"", TypeUnknown},
		{"   ", TypeUnknown},
		{"ab", TypeUnknown},
		{"a", TypeUnknown},
		{"plaintext with spaces and numbers 123", TypeUnknown},
	}

	for _, tt := range tests {
		got := Detect(tt.input)
		if got != tt.want {
			t.Errorf("Detect(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectPriority(t *testing.T) {
	// IPv4 should be detected before domain
	if Detect("8.8.8.8") != TypeIPv4 {
		t.Error("IPv4 not detected correctly")
	}
	// Email should be detected before domain
	if Detect("user@example.com") != TypeEmail {
		t.Error("Email not detected correctly")
	}
	// URL should be detected before domain
	if Detect("https://example.com") != TypeURL {
		t.Error("URL not detected correctly")
	}
	// Onion should be detected
	if Detect("3g2upl4pq6kufc4m.onion") != TypeOnion {
		t.Error("Onion not detected correctly")
	}

	// Try a realistic onion
	if Detect("facebookwkhpilnemxj7asaniu7vnjjbiltxjqhye3mhbshg7kx5tfyd.onion") != TypeOnion {
		t.Error("Long onion not detected correctly")
	}
}

func TestDetectEdgeCases(t *testing.T) {
	// Long query should return unknown
	long := make([]byte, 600)
	for i := range long {
		long[i] = 'a'
	}
	if Detect(string(long)) != TypeUnknown {
		t.Error("Long query should return unknown")
	}

	// Just within limit should still be detected as username
	short := make([]byte, 10)
	for i := range short {
		short[i] = 'a'
	}
	if Detect(string(short)) != TypeUsername {
		t.Errorf("Short alpha query should be username, got %q", Detect(string(short)))
	}

	// 2 chars should still be unknown
	if Detect("ab") != TypeUnknown {
		t.Errorf("2-char query should be unknown, got %q", Detect("ab"))
	}

	// 3 chars should be username
	if Detect("abc") != TypeUsername {
		t.Errorf("3-char query should be username, got %q", Detect("abc"))
	}
}

func TestIsAlphaNumericOnly(t *testing.T) {
	if !isAlphaNumericOnly("hello123") {
		t.Error("expected true for alphanumeric")
	}
	if !isAlphaNumericOnly("hello_123") {
		t.Error("expected true for alphanumeric with underscore")
	}
	if isAlphaNumericOnly("hello world") {
		t.Error("expected false for string with space")
	}
	if isAlphaNumericOnly("hello@world") {
		t.Error("expected false for string with @")
	}
	if isAlphaNumericOnly("") {
		t.Error("expected false for empty string")
	}
}

func TestIsTitleCase(t *testing.T) {
	if !isTitleCase([]string{"John", "Doe"}) {
		t.Error("expected true for title case")
	}
	if isTitleCase([]string{"john", "doe"}) {
		t.Error("expected false for lowercase")
	}
	if isTitleCase([]string{"John", "doe"}) {
		t.Error("expected false for mixed case")
	}
}
