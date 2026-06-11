package normalization

import (
	"net"
	"net/mail"
	"strings"
	"unicode"
)

func NormalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	query = strings.ToLower(query)
	return query
}

func NormalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.Split(domain, "/")[0]
	domain = strings.Split(domain, ":")[0]
	return domain
}

func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}
