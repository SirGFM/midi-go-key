package key_events

// Represents errors in this package.
type errCode int

const (
	// Failed to start the key generator.
	ErrGetKeyGenerator errCode = iota
)

// Implements the 'error' interface for 'errCode'.
func (e errCode) Error() string {
	switch e {
	case ErrGetKeyGenerator:
		return "(key_events) failed to start the key generator."
	default:
		return "(key_events) unknown error"
	}
}
