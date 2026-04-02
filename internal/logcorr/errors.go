package logcorr

import "errors"

// ErrEmptyValue is returned when correlate_events is called without a value.
var ErrEmptyValue = errors.New("correlate_events: value is required")
