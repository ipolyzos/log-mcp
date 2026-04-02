package normalize

import (
	"net"
	"regexp"
	"strings"
)

var (
	uuidRE = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)

	ipv4RE = regexp.MustCompile(`\b(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}\b`)

	// Token runs that may contain colons / hex; validated with net.ParseIP.
	ipv6TokenRE = regexp.MustCompile(`[0-9a-fA-F:.]{3,}`)

	jwtRE = regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`)

	bearerRE = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)

	authBearerRE = regexp.MustCompile(`(?i)(Authorization:\s*Bearer\s+)([^\s]+)`)

	apiishRE = regexp.MustCompile(`(?i)\b(?:api[_-]?key|access[_-]?token|auth[_-]?token|secret|password|token)\s*[=:]\s*[^\s&;,]+`)

	opaquePrefixRE = regexp.MustCompile(`(?i)\b(?:sk-[A-Za-z0-9]{20,}|gh[ps]_[A-Za-z0-9]{20,})\b`)
)

const (
	placeholderUUID  = "<uuid>"
	placeholderIPv4  = "<ipv4>"
	placeholderIPv6  = "<ipv6>"
	placeholderToken = "<token>"
)

func maskUUIDs(s string) string {
	return uuidRE.ReplaceAllString(s, placeholderUUID)
}

func maskIPv4(s string) string {
	return ipv4RE.ReplaceAllString(s, placeholderIPv4)
}

func maskIPv6(s string) string {
	return ipv6TokenRE.ReplaceAllStringFunc(s, func(m string) string {
		colons := strings.Count(m, ":")
		if colons < 2 && !strings.HasPrefix(m, "::") {
			return m
		}
		ip := net.ParseIP(m)
		if ip == nil {
			return m
		}
		if ip4 := ip.To4(); ip4 != nil {
			// Mapped IPv4 (::ffff:x.x.x.x) or IPv4-in-IPv6 context.
			if strings.Contains(m, ":") {
				return placeholderIPv6
			}
			return m
		}
		return placeholderIPv6
	})
}

func maskTokens(s string) string {
	s = jwtRE.ReplaceAllString(s, placeholderToken)
	s = authBearerRE.ReplaceAllString(s, "${1}"+placeholderToken)
	s = bearerRE.ReplaceAllString(s, "Bearer "+placeholderToken)
	s = apiishRE.ReplaceAllStringFunc(s, func(m string) string {
		eq := strings.IndexByte(m, '=')
		co := strings.IndexByte(m, ':')
		sep := -1
		switch {
		case eq >= 0 && (co < 0 || eq < co):
			sep = eq
		case co >= 0:
			sep = co
		default:
			return m
		}
		return m[:sep+1] + placeholderToken
	})
	s = opaquePrefixRE.ReplaceAllString(s, placeholderToken)
	return s
}
