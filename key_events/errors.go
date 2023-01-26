package key_events

// Represents errors in this package.
type errCode int

const (
	// Failed to start the key generator
	ErrGetKeyGenerator errCode = iota
	// Failed to open the config file
	ErrOpenConfig
	// Failed to read the config file
	ErrReadFile
	// Config line is missing some arguments (or has too many arguments)
	ErrConfigArgsBad
	// Missing token "ch=" for channel
	ErrConfigChannelTokenMissing
	// Invalid channel, must be a value between 0 and 15
	ErrConfigChannelTokenInvalid
	// Missing token "ev=" for event
	ErrConfigEventTokenMissing
	// Invalid event, must be a value between 0 and 255
	ErrConfigEventInvalid
	// Missing token "key=" for key
	ErrConfigKeyTokenMissing
	// Invalid key
	ErrConfigKeyInvalid
	// Invalid action, must one of BASIC, VELOCITY, TOGGLE, REPEAT
	ErrConfigActionInvalid
	// Invalid action argument, expect a positive integer
	ErrConfigActionArgumentInvalid
)

// Implements the 'error' interface for 'errCode'.
func (e errCode) Error() string {
	switch e {
	case ErrGetKeyGenerator:
		return "(key_events) failed to start the key generator"
	case ErrOpenConfig:
		return "(key_events) failed to open the config file"
	case ErrReadFile:
		return "(key_events) failed to read the config file"
	case ErrConfigArgsBad:
		return "(key_events) config line is missing some arguments (or has too many arguments)"
	case ErrConfigChannelTokenMissing:
		return `(key_events) missing token "ch=" for channel`
	case ErrConfigChannelTokenInvalid:
		return `(key_events) invalid channel, must be a value between 0 and 15`
	case ErrConfigEventTokenMissing:
		return `(key_events) missing token "ev=" for event`
	case ErrConfigEventInvalid:
		return `(key_events) invalid event, must be a value between 0 and 255`
	case ErrConfigKeyTokenMissing:
		return `(key_events) missing token "key=" for key`
	case ErrConfigKeyInvalid:
		return `(key_events) invalid key`
	case ErrConfigActionInvalid:
		return `(key_events) invalid action, must one of BASIC, VELOCITY, TOGGLE, REPEAT`
	case ErrConfigActionArgumentInvalid:
		return "(key_events) invalid action argument, expect a positive integer"
	default:
		return "(key_events) unknown error"
	}
}
