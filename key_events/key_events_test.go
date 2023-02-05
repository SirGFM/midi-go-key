package key_events

import (
	"runtime/debug"
	"testing"
	"time"

	"github.com/SirGFM/midi-go-key/midi"
)

// A single mocked key code.
type mockKeyCode struct {
	// Signal that the keyCode changed to a new state.
	newState chan bool
	// The keyCode's current state.
	state bool
}

// Set switches the key code to the new state, only if not set.
func (kc *mockKeyCode) Set(state bool) {
	if kc.state != state {
		kc.newState <- state
		kc.state = state
	}
}

// Mocks KeyController.
type mockKeyController map[int]*mockKeyCode

func NewMockKeyController(keyCodes ...int) mockKeyController {
	kc := make(mockKeyController)

	for _, keyCode := range keyCodes {
		kc[keyCode] = &mockKeyCode{
			newState: make(chan bool, 1),
			state:    false,
		}
	}

	return kc
}

func (kc mockKeyController) Close() error {
	return nil
}

func (kc mockKeyController) PressKeys(keyCodes ...int) {
	for _, keyCode := range keyCodes {
		kc[keyCode].Set(true)
	}
}

func (kc mockKeyController) ReleaseKeys(keyCodes ...int) {
	for _, keyCode := range keyCodes {
		kc[keyCode].Set(false)
	}
}

// assert tests whether the given condition is true,
// printing the message and marking the test as having failed otherwise.
func assert(t *testing.T, condition bool, fmt string, args ...interface{}) {
	if !condition {
		debug.PrintStack()
		t.Fatalf(fmt, args...)
	}
}

// The last time an event was sent, for the event timestamp.
var lastSend = time.Now()

// sendMidiEvent sends a dummy MIDI event to conn,
// returning the expected event deadline.
func sendMidiEvent(
	evType midi.MidiEventType,
	channel,
	midiKey,
	velocity uint8,
	conn chan midi.MidiEvent,
) {
	now := time.Now()
	timestamp := now.Sub(lastSend) / time.Millisecond
	if timestamp > 0xffffffff {
		panic("timestamp extrapolated an int32")
	}

	source := generateNoteEvent(evType, channel, midiKey)

	conn <- midi.MidiEvent{
		Source:    source[:],
		Timestamp: int32(timestamp),
		Type:      evType,
		Channel:   channel,
		Key:       midiKey,
		Velocity:  velocity,
	}

	// Update the last send date.
	lastSend = now
}

// assertKeyEvent sends a MIDI event of the supplied on conn,
// checking that the keyCode was held down for duration every step.
// Since properly sync'ing this to the key event generator is somewhat tricky,
// this only fails if the key isn't released within graceTime of the duration.
func assertKeyEvent(
	t *testing.T,
	kc mockKeyController,
	keyCode int,
	evType midi.MidiEventType,
	channel,
	midiKey,
	velocity uint8,
	conn chan midi.MidiEvent,
	duration,
	graceTime time.Duration,
) {
	held := time.After(duration - graceTime)
	deadline := time.After(duration + 2*graceTime)

	start := time.Now()
	sendMidiEvent(evType, channel, midiKey, velocity, conn)

	// Check that the keyCode was pressed.
	select {
	case <-time.After(time.Millisecond / 2):
		t.Fatalf("failed to detect that the keyCode was pressed in time")
	case pressed := <-kc[keyCode].newState:
		if !pressed {
			t.Fatalf("keyCode wasn't pressed in time")
		}
	}

	// Check that the keyCode was held for long enough.
	select {
	case <-kc[keyCode].newState:
		t.Fatalf("keyCode was released early")
	case <-held:
		// Key was held down for as long as desired!
	}

	// Check that the keyCode was release in time.
	select {
	case <-deadline:
		t.Fatalf("failed to detect that the keyCode was released in time")
	case pressed := <-kc[keyCode].newState:
		if pressed {
			t.Fatalf("keyCode wasn't released in time")
		}
		return
	}

	// If the keyCode wasn't released in time,
	// wait for a little longer to check if it's a simple timming thing.
	elapsed := time.Now().Sub(start)
	select {
	case <-time.After(time.Millisecond * 200):
		afterRetest := time.Now().Sub(start)
		t.Fatalf("keyCode release wasn't detected even after %s; elapsed=%s", afterRetest, elapsed)
	case pressed := <-kc[keyCode].newState:
		afterRetest := time.Now().Sub(start)
		if pressed {
			t.Fatalf("keyCode wasn't released even after %s; elapsed=%s", afterRetest, elapsed)
		} else {
			t.Fatalf("keyCode was released only after %s; elapsed=%s", afterRetest, elapsed)
		}
	}
}

func TestBasicPress(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const midiKey = 2
	const badKey = 3
	const keyCode = 3
	const releaseTime = 10 * time.Millisecond
	const threshold = 30

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := NewMockKeyController(keyCode)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	ke.RegisterBasicPressAction(
		evType,
		channel,
		midiKey,
		keyCode,
		threshold,
		releaseTime,
	)

	// Test that sending a MIDI event different from the expected doesn't set the keyCode.
	sendMidiEvent(evType, channel, badKey, 100, conn)
	select {
	case <-kc[keyCode].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-time.After(time.Millisecond):
		// Key wasn't pressed, as expected!
	}

	// Test that sending the event keeps the keyCode pressed for the desired time.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		100,
		conn,
		releaseTime,
		time.Millisecond,
	)
}

func TestVelocityPress(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const midiKey = 2
	const badKey = 3
	const keyCode = 3
	const minTime = 10 * time.Millisecond
	const maxTime = 100 * time.Millisecond
	const threshold = 0

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := NewMockKeyController(keyCode)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	ke.RegisterVelocityAction(
		evType,
		channel,
		midiKey,
		keyCode,
		threshold,
		minTime,
		maxTime,
	)

	// Test that sending a MIDI event different from the expected doesn't set the keyCode.
	sendMidiEvent(evType, channel, badKey, 100, conn)
	select {
	case <-kc[keyCode].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-time.After(time.Millisecond):
		// Key wasn't pressed, as expected!
	}

	// Test that sending a quick MIDI event generates a quickly resolved keyCode press.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		1,
		conn,
		minTime,
		time.Millisecond,
	)

	// Test that sending a long MIDI event generates a long-lasting keyCode press.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		128,
		conn,
		maxTime,
		time.Millisecond,
	)
}

func TestHoldPress(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const midiKey = 2
	const badKey = 3
	const keyCode = 3
	const shortRelease = 10 * time.Millisecond
	const maxDelayMs = 100
	const eventDelay = 95 * time.Millisecond
	const threshold = 30

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := NewMockKeyController(keyCode)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	ke.RegisterHoldAction(
		evType,
		channel,
		midiKey,
		keyCode,
		threshold,
		maxDelayMs,
		shortRelease,
	)

	// Test that sending a MIDI event different from the expected doesn't set the keyCode.
	sendMidiEvent(evType, channel, badKey, 100, conn)
	select {
	case <-kc[keyCode].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-time.After(time.Millisecond):
		// Key wasn't pressed, as expected!
	}

	// Test that sending a quick MIDI event generates a quickly resolved keyCode press.
	lastSend = time.Now().Add(-shortRelease - maxDelayMs*time.Millisecond)
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		100,
		conn,
		shortRelease,
		time.Millisecond,
	)

	// Test that sending repeated MIDI events generates a long-lasting keyCode press.
	const count = 5
	const maxTime = eventDelay * count
	go func() {
		// Queue events roughly following the expected duration.
		for i := 0; i < count; i++ {
			sendMidiEvent(evType, channel, midiKey, 100, conn)
			time.Sleep(eventDelay)
		}
	}()

	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		100,
		conn,
		maxTime,
		time.Millisecond*10,
	)
}

func TestTogglePress(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const midiKey = 2
	const badKey = 3
	const keyCode = 3
	const shortRelease = 10 * time.Millisecond
	const threshold = 30
	const toggleThreshold = 80

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := NewMockKeyController(keyCode)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	ke.RegisterToggleAction(
		evType,
		channel,
		midiKey,
		keyCode,
		threshold,
		toggleThreshold,
		shortRelease,
	)

	// Test that sending a MIDI event different from the expected doesn't set the keyCode.
	sendMidiEvent(evType, channel, badKey, 100, conn)
	select {
	case <-kc[keyCode].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-time.After(time.Millisecond):
		// Key wasn't pressed, as expected!
	}

	// Test that sending a quick MIDI event generates a quickly resolved keyCode press.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		toggleThreshold-1,
		conn,
		shortRelease,
		time.Millisecond,
	)

	// Test that sending a long MIDI event holds the keyCode down
	// until the event is sent again.
	const maxTime = time.Millisecond * 100
	go func() {
		// Queue an event to be sent after maxTime.
		time.Sleep(maxTime)
		sendMidiEvent(evType, channel, midiKey, toggleThreshold, conn)
	}()

	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey,
		toggleThreshold+1,
		conn,
		maxTime,
		time.Millisecond*5,
	)
}
