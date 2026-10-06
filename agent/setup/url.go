package setup

import (
	"errors"
	"net/url"
	"path"
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

// NormalizeServerURL canonicalizes user input so the same server always
// compares equal and path joins never produce "//" artifacts:
//
//   - surrounding whitespace trimmed
//   - missing scheme defaulted (https; http for loopback so bare
//     "localhost:7541/..." works in dev)
//   - scheme and host lowercased; default ports stripped (:443/:80)
//   - userinfo, query, and fragment dropped (a server base URL carries no
//     credentials; stripping avoids persisting them into config.json)
//   - path cleaned (duplicate slashes collapsed, trailing slash removed)
//
// It reports syntax problems only; use ValidateServerURL for policy
// (e.g. the loopback-only HTTP rule).
func NormalizeServerURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("server URL is required")
	}
	if !strings.Contains(s, "://") {
		host := s
		if i := strings.Index(host, "/"); i >= 0 {
			host = host[:i]
		}
		if i := strings.Index(host, ":"); i >= 0 {
			host = host[:i]
		}
		if isLoopback(strings.ToLower(host)) {
			s = "http://" + s
		} else {
			s = "https://" + s
		}
	}
	u, err := url.Parse(s)
	if err != nil || u == nil {
		return "", errors.New("server URL is not a valid URL")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errInvalidScheme
	}
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return "", errMissingHost
	}
	// Read the port BEFORE rebuilding Host (assignment would drop it).
	port := u.Port()
	u.Host = hostname
	if port != "" {
		if !((u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80")) {
			u.Host = hostname + ":" + port
		}
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	p := path.Clean(strings.TrimSpace(u.EscapedPath()))
	if p == "." || p == "/" {
		p = ""
	}
	u.RawPath = ""
	u.Path = p
	return u.String(), nil
}

// ValidateServerURL enforces connection policy on top of normalization:
// the URL must be syntactically canonicalizable, and plain HTTP is allowed
// only for loopback (lab) hosts. Production servers must be https://.
func ValidateServerURL(raw string) error {
	norm, err := NormalizeServerURL(raw)
	if err != nil {
		return err
	}
	p, err := parseURL(norm)
	if err != nil {
		return err
	}
	if p.Scheme == "http" && !isLoopback(p.Hostname()) {
		return errInsecureRemote
	}
	if strings.TrimSpace(p.Host) == "" {
		return errMissingHost
	}
	return nil
}

// JoinURL appends path segments to a (normalized) base URL with exactly one
// slash between parts — never "https://host//vaultguard/api//agents/enroll".
func JoinURL(base string, parts ...string) string {
	base = strings.TrimSuffix(base, "/")
	var b strings.Builder
	b.WriteString(base)
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p == "" {
			continue
		}
		b.WriteByte('/')
		b.WriteString(p)
	}
	return b.String()
}
