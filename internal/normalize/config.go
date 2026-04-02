package normalize

// Config selects which transforms run in [Apply]. All zero values mean "skip
// that step"; use [DefaultConfig] for the standard pipeline.
type Config struct {
	MaskUUIDs           bool
	MaskIPv4            bool
	MaskIPv6            bool
	MaskTokens          bool
	NormalizeTimestamps bool
	NormalizeDynamicIDs bool
}

// DefaultConfig enables every masking and normalization step in the canonical
// order documented on [Apply].
func DefaultConfig() Config {
	return Config{
		MaskUUIDs:           true,
		MaskIPv4:            true,
		MaskIPv6:            true,
		MaskTokens:          true,
		NormalizeTimestamps: true,
		NormalizeDynamicIDs: true,
	}
}

func (c Config) empty() bool {
	return !c.MaskUUIDs && !c.MaskIPv4 && !c.MaskIPv6 && !c.MaskTokens &&
		!c.NormalizeTimestamps && !c.NormalizeDynamicIDs
}
