package key_events

import (
	"log"
	"time"

	"github.com/SirGFM/midi-go-key/midi"
)

// The action taken (e.g., generate a key press) in response to a MIDI event.
type midiAction func(midi.MidiEvent)

// How many timed actions may be queued at once.
const timedActionQueueSize = 64

// Value used to identify an event bound to multiple keyboard keys.
const multiKey = 0x8000000000000000

// Controls the keyboard by pressing and releasing keys.
type KeyController interface {
	// Releases every resource associated with the key controller.
	Close() error

	// PressKeys presses the requested keys, by their keycode.
	PressKeys(...int)

	// ReleaseKeys releases the requested keys, by their keycode.
	ReleaseKeys(...int)
}

type KeyEvents interface {
	// Releases every resource associated with the key events generator
	Close() error

	// RegisterBasicPressAction registers the most basic action of pressing and
	// shortly thereafter (after releaseTime) releasing it.
	// The input is ignored if it's less than or equal to the threshold.
	RegisterBasicPressAction(
		evType midi.MidiEventType,
		channel,
		key uint8,
		keyCode int,
		threshold uint8,
		releaseTime time.Duration,
	)

	// RegisterVelocityAction registers the action of pressing a key
	// based on the velocity of the MIDI event.
	// The greater the velocity, the closer to maxPress that the key is held down.
	// The input is ignored if it's less than or equal to the threshold.
	RegisterVelocityAction(
		evType midi.MidiEventType,
		channel,
		key uint8,
		keyCode int,
		threshold uint8,
		minPress,
		maxPress time.Duration,
	)

	// RegisterToggleAction registers an action that toggles a key whenever
	// the MIDI event is generated.
	// To send quick presses, set a velocity toggleThreshold,
	// bellow which the action will be executed regularly, instead of toggling.
	// This also causes the toggle to get released, if it were pressed.
	// The input is ignored if it's less than or equal to the threshold.
	RegisterToggleAction(
		evType midi.MidiEventType,
		channel,
		key uint8,
		keyCode int,
		threshold,
		acceptThreshold uint8,
		quickPressDuration time.Duration,
	)

	// RegisterHoldAction registers an actions that stays pressed
	// as long as the MIDI event is repeated.
	// If the MIDI event hasn't been sent in maxRepeatDelayMs,
	// then the key will be released after shortRelease.
	// Otherwise, it will stay pressed for maxRepeatDelayMs
	// of the last event.
	// The first of a series of repeated inputs
	// is ignored if it's less than or equal to the threshold.
	RegisterHoldAction(
		evType midi.MidiEventType,
		channel,
		key uint8,
		keyCode int,
		threshold uint8,
		maxRepeatDelayMs int32,
		shortRelease time.Duration,
	)

	// RegisterSequenceHoldAction registers an action that stays pressed
	// as long as the MIDI event is repeated.
	// However, the actual pressed key is picked from keyCodes,
	// advancing to the next one on MIDI event nextKeyCode,
	// moving back to the previous one on MIDI event prevKeyCode,
	// and resetting back to the start on MIDI event resetKeyCode.
	// If the MIDI event hasn't been sent in maxRepeatDelayMs,
	// then the key will be released after shortRelease.
	// Otherwise, it will stay pressed for maxRepeatDelayMs
	// of the last event.
	// The first of a series of repeated inputs
	// is ignored if it's less than or equal to the threshold.
	RegisterSequenceHoldAction(
		evType midi.MidiEventType,
		channel,
		key uint8,
		keyCodes [][]int,
		threshold uint8,
		maxRepeatDelayMs int32,
		shortRelease time.Duration,
		prevKeyCode,
		nextKeyCode,
		resetKeyCode uint8,
	)

	// ReadConfig reads the configuration file in path and registers the listed actions.
	ReadConfig(path string) error
}

// A MIDI event generated for a given note,
// ignoring it's velocity.
type noteEvent [2]byte

// An timerAction generated by a timer.
type timerAction func()

type actionSet map[noteEvent]midiAction

type keyEvents struct {
	// The internal key controller.
	kc KeyController
	// The channel used to receive MIDI events.
	conn <-chan midi.MidiEvent
	// List actions taken in response to the registered actions.
	actions actionSet
	// List of named action sets taken in response to the registered actions.
	namedSets map[string]actionSet
	// The currently active named action set.
	curSet string
	// List actions responsible for pressing/releasing keys.
	keyActions map[uint64]*keyAction
	// Receive actions that should be generated based on a timer.
	timedAction chan timerAction
	// Whether unhandled events should be logged.
	logUnhandled bool
}

// NewKeyEvents creates and starts a new event generator.
// When conn is closed, the key event generator stops running.
func NewKeyEvents(kc KeyController, conn <-chan midi.MidiEvent, logUnhandled bool) (KeyEvents, error) {
	kbEv := &keyEvents{
		kc:           kc,
		conn:         conn,
		actions:      make(map[noteEvent]midiAction),
		namedSets:    make(map[string]actionSet),
		keyActions:   make(map[uint64]*keyAction),
		timedAction:  make(chan timerAction, timedActionQueueSize),
		logUnhandled: logUnhandled,
	}
	go kbEv.run()

	return kbEv, nil
}

func (kbEv *keyEvents) Close() error {
	return kbEv.kc.Close()
}

// run listens for MIDI events and generates key events.
func (kbEv *keyEvents) run() {
	for {
		select {
		case midiEv, hasMore := <-kbEv.conn:
			if !hasMore {
				return
			}
			kbEv.handleMidiEvent(midiEv)
		case action := <-kbEv.timedAction:
			// timedAction are queued as a response to MIDI events,
			// so just execute them as they are received.
			action()
		}
	}
}

// handleMidiEvent handles a given MIDI event,
// executing its registered action.
func (kbEv *keyEvents) handleMidiEvent(midiEv midi.MidiEvent) {
	var event noteEvent

	if len(midiEv.Source) < len(event) {
		log.Printf("invalid event received: %s\n", midiEv)
		return
	}
	copy(event[:], midiEv.Source)

	action, ok := kbEv.actions[event]
	if !ok {
		// If the action isn't on the default set,
		// check if it's in the currently active named set.
		var set map[noteEvent]midiAction
		if set, ok = kbEv.namedSets[kbEv.curSet]; ok {
			action, ok = set[event]
		}
	}

	if ok {
		action(midiEv)
	} else if kbEv.logUnhandled {
		log.Printf("unhandled: %s\n", midiEv)
	}
}

// generateNoteEvent generates noteEvent from the desired parameters.
func generateNoteEvent(evType midi.MidiEventType, channel, key uint8) noteEvent {
	var event noteEvent

	event[0] = evType.ToUint8() | channel
	event[1] = key

	return event
}

// removeAction removes an action associated with the give event, if any.
func (kbEv *keyEvents) removeAction(event noteEvent) {
	if _, ok := kbEv.actions[event]; ok {
		delete(kbEv.actions, event)
	}
}

// newKeyAction creates a new keyAction, with its timer already configured (but stopped).
// If an action has already been registered for that keyCode,
// then that first action will be returned instead.
//
// This function isn't thread safe and should be called before any event is received.
func (kbEv *keyEvents) newKeyAction(keyCode int, onTimeout timerAction) *keyAction {
	if action, ok := kbEv.keyActions[uint64(keyCode)]; ok {
		return action
	}

	action := newKeyAction(keyCode, kbEv.kc, kbEv.timedAction, onTimeout)
	kbEv.keyActions[uint64(keyCode)] = action
	return action
}

// newKeyActionMulti creates a new keyAction for multiple keys, with its timer already configured (but stopped).
// If an action has already been registered for that keyCode,
// then that first action will be returned instead.
//
// This function isn't thread safe and should be called before any event is received.
// Also, keyCodes must have at most 4 keys.
func (kbEv *keyEvents) newKeyActionMulti(keyCodes []int, onTimeout timerAction) *keyAction {
	if len(keyCodes) > 4 {
		panic("multi actions can act on at most 4 keys")
	}

	code := uint64(0)
	if len(keyCodes) > 1 {
		code = multiKey
	}
	for i, key := range keyCodes {
		code |= uint64(key << (i * 16))
	}

	if action, ok := kbEv.keyActions[code]; ok {
		return action
	}

	action := newKeyActionMulti(keyCodes, kbEv.kc, kbEv.timedAction, onTimeout)
	kbEv.keyActions[code] = action
	return action
}

func (kbEv *keyEvents) RegisterBasicPressAction(
	evType midi.MidiEventType,
	channel,
	key uint8,
	keyCode int,
	threshold uint8,
	releaseTime time.Duration,
) {

	event := generateNoteEvent(evType, channel, key)

	kbEv.removeAction(event)

	// Create a new key handler and start its timer.
	keyAction := kbEv.newKeyAction(keyCode, nil)

	// Register the onPress function.
	action := func(ev midi.MidiEvent) {
		if ev.Type != midi.EventNoteOn || ev.Velocity <= threshold {
			return
		}

		keyAction.Press()

		keyAction.QueueTimedAction(releaseTime)
	}

	kbEv.actions[event] = action
}

func (kbEv *keyEvents) RegisterVelocityAction(
	evType midi.MidiEventType,
	channel,
	key uint8,
	keyCode int,
	threshold uint8,
	minPress,
	maxPress time.Duration,
) {
	event := generateNoteEvent(evType, channel, key)

	kbEv.removeAction(event)

	// Create a new key handler and start its timer.
	keyAction := kbEv.newKeyAction(keyCode, nil)

	// Register the onPress function.
	action := func(ev midi.MidiEvent) {
		if ev.Type != midi.EventNoteOn || ev.Velocity <= threshold {
			return
		}

		if ev.Velocity > midi.MaxVelocity {
			log.Printf("max velocity was reached! %d\n", ev.Velocity)
		}

		// Calculate how long the key should be pressed based on the key velocity.
		releaseTime := maxPress - minPress
		releaseTime *= time.Duration(ev.Velocity)
		releaseTime /= midi.MaxVelocity
		releaseTime += minPress

		onPress := func() {
			keyAction.Press()

			keyAction.QueueTimedAction(releaseTime)
		}

		if keyAction.IsPressed() {
			// If the key was already pressed,
			// release it momentarily and then press it again.
			keyAction.Release()

			go func() {
				time.Sleep(time.Millisecond)
				onPress()
			}()
		} else {
			onPress()
		}
	}

	kbEv.actions[event] = action
}

func (kbEv *keyEvents) RegisterToggleAction(
	evType midi.MidiEventType,
	channel,
	key uint8,
	keyCode int,
	threshold,
	toggleThreshold uint8,
	quickPressDuration time.Duration,
) {
	event := generateNoteEvent(evType, channel, key)

	kbEv.removeAction(event)

	// Create a new key handler and start its timer.
	keyAction := kbEv.newKeyAction(keyCode, nil)

	// Register the onPress function.
	action := func(ev midi.MidiEvent) {
		if ev.Type != midi.EventNoteOn || ev.Velocity <= threshold {
			return
		}

		if keyAction.IsPressed() {
			// If it is pressed, simply release it.
			keyAction.Release()
		} else {
			// Otherwise, simply toggle it on.
			keyAction.Press()

			// If the hit was bellow the threshold,
			// simply do a quick press.
			if ev.Velocity < toggleThreshold {
				keyAction.QueueTimedAction(quickPressDuration)
			}
		}
	}

	kbEv.actions[event] = action
}

func (kbEv *keyEvents) RegisterHoldAction(
	evType midi.MidiEventType,
	channel,
	key uint8,
	keyCode int,
	threshold uint8,
	maxRepeatDelayMs int32,
	shortRelease time.Duration,
) {
	event := generateNoteEvent(evType, channel, key)

	kbEv.removeAction(event)

	// Create a new key handler and start its timer.
	keyAction := kbEv.newKeyAction(keyCode, nil)

	// Stores the last time the MIDI event was received.
	var lastTimestamp int32

	// Register the onPress function.
	action := func(ev midi.MidiEvent) {
		wasPressed := (ev.Timestamp-lastTimestamp > maxRepeatDelayMs)

		if ev.Type != midi.EventNoteOn || ev.Velocity == 0 || (!wasPressed && ev.Velocity <= threshold) {
			return
		}

		keyAction.Press()

		// If the event was sent quickly enough,
		// requeue the action for a longer time.
		// Otherwise, simply send a quick action.
		if wasPressed {
			keyAction.QueueTimedAction(shortRelease)
		} else {
			keyAction.QueueTimedAction(time.Duration(maxRepeatDelayMs) * time.Millisecond)
		}

		lastTimestamp = ev.Timestamp
	}

	kbEv.actions[event] = action
}

func (kbEv *keyEvents) RegisterSequenceHoldAction(
	evType midi.MidiEventType,
	channel,
	key uint8,
	keyCodes [][]int,
	threshold uint8,
	maxRepeatDelayMs int32,
	shortRelease time.Duration,
	prevKeyCode,
	nextKeyCode,
	resetKeyCode uint8,
) {
	pressEvent := generateNoteEvent(evType, channel, key)
	kbEv.removeAction(pressEvent)

	prevEvent := generateNoteEvent(evType, channel, prevKeyCode)
	kbEv.removeAction(prevEvent)

	nextEvent := generateNoteEvent(evType, channel, nextKeyCode)
	kbEv.removeAction(nextEvent)

	resetEvent := generateNoteEvent(evType, channel, resetKeyCode)
	kbEv.removeAction(resetEvent)

	// Create a new key handler for each step in the sequence and start their timers.
	var actions []*keyAction
	for _, keys := range keyCodes {
		actions = append(actions, kbEv.newKeyActionMulti(keys, nil))
	}
	// Set the first action as the active one.
	cur := 0

	// Stores the last time the MIDI event was received.
	var lastTimestamp int32

	// Register the onPress function for activating the current key.
	pressAction := func(ev midi.MidiEvent) {
		keyAction := actions[cur]

		wasPressed := (ev.Timestamp-lastTimestamp > maxRepeatDelayMs)

		if ev.Type != midi.EventNoteOn || ev.Velocity == 0 || (!wasPressed && ev.Velocity <= threshold) {
			return
		}

		keyAction.Press()

		// If the event was sent quickly enough,
		// requeue the action for a longer time.
		// Otherwise, simply send a quick action.
		if wasPressed {
			keyAction.QueueTimedAction(shortRelease)
		} else {
			keyAction.QueueTimedAction(time.Duration(maxRepeatDelayMs) * time.Millisecond)
		}

		lastTimestamp = ev.Timestamp
	}

	// Register the onPress function for going back to the previous input.
	prevAction := func(ev midi.MidiEvent) {
		if ev.Type != midi.EventNoteOn || ev.Velocity <= threshold {
			return
		}

		cur--
		if cur < 0 {
			cur = len(actions) - 1
		}
	}

	// Register the onPress function for advancing to the next input.
	nextAction := func(ev midi.MidiEvent) {
		if ev.Type != midi.EventNoteOn || ev.Velocity <= threshold {
			return
		}

		cur++
		if cur >= len(actions) {
			cur = 0
		}
	}

	// Register the onPress function for going to the start of the sequence.
	resetAction := func(ev midi.MidiEvent) {
		if ev.Type != midi.EventNoteOn || ev.Velocity <= threshold {
			return
		}

		cur = 0
	}

	kbEv.actions[pressEvent] = pressAction
	kbEv.actions[nextEvent] = nextAction
	kbEv.actions[prevEvent] = prevAction
	kbEv.actions[resetEvent] = resetAction
}
