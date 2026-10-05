package setup

import (
	"errors"
	"net/url"
	"strings"
)

var (
	errInvalidScheme  = errors.New("server URL must start with https:// (http:// is allowed only for localhost)")
	errInsecureRemote = errors.New("plain HTTP to a non-loopback host is not allowed — use https://")
	errMissingHost    = errors.New("server URL has no host")
)

type parsedURL struct {
	Scheme   string
	Host     string
	hostname string
}

func (p parsedURL) Hostname() string { return p.hostname }

func parseURL(raw string) (parsedURL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil {
		return parsedURL{}, errors.New("server URL is not a valid URL")
	}
	return parsedURL{Scheme: strings.ToLower(u.Scheme), Host: u.Host, hostname: strings.ToLower(u.Hostname())}, nil
}

func isLoopback(host string) bool {
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}
