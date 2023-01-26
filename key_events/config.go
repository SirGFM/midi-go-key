package key_events

import (
	"bufio"
	"os"
	"time"
	"strings"
	"strconv"

	"github.com/SirGFM/midi-go-key/err_wrap"
	"github.com/SirGFM/midi-go-key/midi"
)

// List how many arguments each action has
var actionsToArgCount = map[string]int {
	"BASIC": 1,
	"VELOCITY": 2,
	"TOGGLE": 2,
	"REPEAT": 2,
}

// The minimum number of arguments in a line.
const minArgs = 4

// getInt reads an integer from arg, removing the prefix from the start.
// badTokeErr is returned if the prefix is invalid, and invalidErr
// if the value isn't an integer.
func getInt(arg, prefix string, badTokenErr, invalidErr error) (int, error) {
	if !strings.HasPrefix(arg, prefix) {
		return 0, badTokenErr
	}

	strValue := arg[len(prefix):]
	val, err := strconv.ParseInt(strValue, 0, 16)
	if err != nil {
		return 0, err_wrap.Wrap(err, invalidErr)
	}

	return int(val), nil
}

func (kbEv *keyEvents) ReadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err_wrap.Wrap(err, ErrOpenConfig)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and lines starting on # (i.e., comments).
		if len(line) <= 0 || line[0] == '#' {
			continue
		}

		// Break each line into space-separated components.
		args := strings.Split(line, " ")
		if len(args) < minArgs {
			return ErrConfigArgsBad
		}

		// Parse the arguments
		intCh, err := getInt(args[0], "ch=", ErrConfigChannelTokenMissing, ErrConfigChannelTokenInvalid)
		if err != nil {
			return err
		} else if intCh < 0 || intCh > 15 {
			return ErrConfigChannelTokenInvalid
		}

		intEv, err := getInt(args[1], "ev=", ErrConfigEventTokenMissing, ErrConfigEventInvalid)
		if err != nil {
			return err
		} else if intEv < 0 || intEv > 255 {
			return ErrConfigEventInvalid
		}

		if !strings.HasPrefix(args[2], "key=") {
			return ErrConfigKeyTokenMissing
		}
		strKey := args[2][len("key="):]
		key, ok := keyNameToInt[strings.ToUpper(strKey)]
		if !ok {
			return ErrConfigKeyInvalid
		}

		// Check that there are enough arguments for the action.
		action := args[3]
		wantArgs, ok := actionsToArgCount[action]
		if !ok {
			return ErrConfigActionInvalid
		} else if len(args) != wantArgs + minArgs {
			return ErrConfigArgsBad
		}

		// Parse every argument as a simple non-zero integer.
		var numArgs []int
		for _, arg := range args[minArgs:] {
			num, err := getInt(arg, "", nil, ErrConfigActionArgumentInvalid)
			if err != nil {
				return err
			} else if num <= 0 {
				return ErrConfigActionArgumentInvalid
			}

			numArgs = append(numArgs, num)
		}
		if len(numArgs) != wantArgs {
			return ErrConfigArgsBad
		}

		ch := uint8(intCh)
		ev := uint8(intEv)

		switch action {
		case "BASIC":
			releaseTime := time.Duration(numArgs[0]) * time.Millisecond

			kbEv.RegisterBasicPressAction(
				midi.EventNoteOn,
				ch,
				ev,
				key,
				releaseTime,
			)
		case "VELOCITY":
			minPress := time.Duration(numArgs[0]) * time.Millisecond
			maxPress := time.Duration(numArgs[1]) * time.Millisecond

			kbEv.RegisterVelocityAction(
				midi.EventNoteOn,
				ch,
				ev,
				key,
				minPress,
				maxPress,
			)
		case "TOGGLE":
			if numArgs[0] > 128 {
				return ErrConfigActionArgumentInvalid
			}
			threshold := uint8(numArgs[0])
			quickPressDuration := time.Duration(numArgs[1]) * time.Millisecond

			kbEv.RegisterToggleAction(
				midi.EventNoteOn,
				ch,
				ev,
				key,
				threshold,
				quickPressDuration,
			)
		case "REPEAT":
			maxRepeatDelayMs := int32(numArgs[0])
			shortRelease := time.Duration(numArgs[1]) * time.Millisecond

			kbEv.RegisterHoldAction(
				midi.EventNoteOn,
				ch,
				ev,
				key,
				maxRepeatDelayMs,
				shortRelease,
			)
		default:
			return ErrConfigActionInvalid
		}
	}

	if err := scanner.Err(); err != nil {
		return err_wrap.Wrap(err, ErrReadFile)
	}

	return nil
}
