package tor

import (
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

func NewClient(socksHost string, socksPort int) (*http.Client, error) {
	proxyURL := &url.URL{
		Scheme: "socks5",
		Host:   socksHost,
	}
	if socksPort != 0 {
		proxyURL.Host = socksHost
		portStr := "9050"
		if socksPort > 0 {
			portStr = itoa(socksPort)
		}
		proxyURL.Host = socksHost + ":" + portStr
	}

	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
