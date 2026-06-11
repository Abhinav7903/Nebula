package detection

import (
	"net"
	"regexp"
	"strings"
	"unicode"
)

type QueryType string

const (
	TypeUnknown          QueryType = "unknown"
	TypeIPv4             QueryType = "ipv4"
	TypeCIDR             QueryType = "cidr"
	TypeIPv6             QueryType = "ipv6"
	TypeDomain           QueryType = "domain"
	TypeSubdomain        QueryType = "subdomain"
	TypeOnion            QueryType = "onion"
	TypeEmail            QueryType = "email"
	TypeUsername         QueryType = "username"
	TypePersonName       QueryType = "person_name"
	TypeCompanyName      QueryType = "company_name"
	TypePhone            QueryType = "phone"
	TypeEthereumAddress  QueryType = "ethereum_address"
	TypeBitcoinAddress   QueryType = "bitcoin_address"
	TypeSolanaAddress    QueryType = "solana_address"
	TypeURL              QueryType = "url"
	TypeHashMD5          QueryType = "hash_md5"
	TypeHashSHA1         QueryType = "hash_sha1"
	TypeHashSHA256       QueryType = "hash_sha256"
)

var (
	reEmail   = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	rePhone   = regexp.MustCompile(`^\+?[\d\-\(\)]{7,15}$`)
	reEthAddr = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	reBtcAddr = regexp.MustCompile(`^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^(bc1)[a-zA-HJ-NP-Z0-9]{25,59}$`)
	reSolAddr = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)
	reURL     = regexp.MustCompile(`^https?://\S+$`)
	reMD5     = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
	reSHA1    = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	reSHA256  = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	reAlpha   = regexp.MustCompile(`^[a-zA-Z]+$`)
	reCompany = regexp.MustCompile(`(?i)\b(Inc|LLC|Ltd|Corp|GmbH|SA|BV)\b$`)
)

func Detect(query string) QueryType {
	query = strings.TrimSpace(query)
	if query == "" {
		return TypeUnknown
	}

	if len(query) > 512 {
		return TypeUnknown
	}

	if ip := net.ParseIP(query); ip != nil {
		if ip.To4() != nil {
			return TypeIPv4
		}
		return TypeIPv6
	}
	if _, _, err := net.ParseCIDR(query); err == nil {
		return TypeCIDR
	}
	if strings.HasSuffix(query, ".onion") && len(query) > 8 {
		return TypeOnion
	}
	if reURL.MatchString(query) {
		return TypeURL
	}
	if reEmail.MatchString(query) {
		return TypeEmail
	}
	if reEthAddr.MatchString(query) {
		return TypeEthereumAddress
	}
	if reBtcAddr.MatchString(query) {
		return TypeBitcoinAddress
	}
	if reSolAddr.MatchString(query) {
		return TypeSolanaAddress
	}
	if reMD5.MatchString(query) {
		return TypeHashMD5
	}
	if reSHA1.MatchString(query) {
		return TypeHashSHA1
	}
	if reSHA256.MatchString(query) {
		return TypeHashSHA256
	}
	if rePhone.MatchString(query) {
		return TypePhone
	}

	parts := strings.Fields(query)
	if len(parts) == 1 {
		s := parts[0]
		dots := strings.Count(s, ".")
		if dots >= 2 && hasLetter(s) {
			return TypeSubdomain
		}
		if dots == 1 && hasLetter(s) {
			return TypeDomain
		}
		if isAlphaNumericOnly(s) && len(s) >= 3 && !strings.ContainsAny(s, "@.") {
			return TypeUsername
		}
	}

	if len(parts) >= 2 {
		allAlpha := true
		for _, p := range parts {
			if !reAlpha.MatchString(p) {
				allAlpha = false
				break
			}
		}
		if allAlpha {
			combined := strings.Join(parts, " ")
			if reCompany.MatchString(combined) || (isTitleCase(parts) && len(parts) >= 3) {
				return TypeCompanyName
			}
			if isTitleCase(parts) {
				return TypePersonName
			}
			return TypePersonName
		}
		if reCompany.MatchString(query) {
			return TypeCompanyName
		}
	}

	return TypeUnknown
}

func isAlphaNumericOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return len(s) > 0
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func isTitleCase(parts []string) bool {
	for _, p := range parts {
		if len(p) > 0 && unicode.IsUpper(rune(p[0])) {
			continue
		}
		return false
	}
	return true
}
