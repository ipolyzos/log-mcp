package normalize

// Apply returns a redacted, normalized copy of s according to cfg.
//
// Steps run in this fixed order (deterministic):
//  1. Mask tokens (JWT, Bearer, api_key=/token=/password=, sk-*, gh*_*).
//  2. Mask UUIDs.
//  3. Mask IPv6 addresses (validated with net.ParseIP; includes IPv4-mapped IPv6).
//  4. Mask IPv4 addresses.
//  5. Normalize timestamp substrings (RFC3339-like and 10-digit unix seconds in 2000–2100 UTC).
//  6. Normalize dynamic IDs (15–22 digit decimals, 16–40 hex runs).
func Apply(s string, cfg Config) string {
	if cfg.empty() {
		return s
	}
	out := s
	if cfg.MaskTokens {
		out = maskTokens(out)
	}
	if cfg.MaskUUIDs {
		out = maskUUIDs(out)
	}
	if cfg.MaskIPv6 {
		out = maskIPv6(out)
	}
	if cfg.MaskIPv4 {
		out = maskIPv4(out)
	}
	if cfg.NormalizeTimestamps {
		out = normalizeTimestamps(out)
	}
	if cfg.NormalizeDynamicIDs {
		out = normalizeDynamicIDs(out)
	}
	return out
}
