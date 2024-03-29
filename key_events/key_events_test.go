package key_events

import (
	"runtime/debug"
	"testing"
	"time"

	"github.com/SirGFM/midi-go-key/event_logger"
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

// The time when the test started, for calculating the event timestamp.
// This time is set to the past, so every first timestamp in a test
// shall be a large, non-zero value.
var startTime = time.Now().Add(-time.Minute)

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
	timestamp := now.Sub(startTime) / time.Millisecond
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
		debug.PrintStack()
		t.Fatalf("failed to detect that the keyCode was pressed in time")
	case pressed := <-kc[keyCode].newState:
		if !pressed {
			debug.PrintStack()
			t.Fatalf("keyCode wasn't pressed in time")
		}
	}

	// Check that the keyCode was held for long enough.
	select {
	case <-kc[keyCode].newState:
		debug.PrintStack()
		t.Fatalf("keyCode was released early")
	case <-held:
		// Key was held down for as long as desired!
	}

	// Check that the keyCode was release in time.
	select {
	case <-deadline:
		debug.PrintStack()
		t.Fatalf("failed to detect that the keyCode was released in time")
	case pressed := <-kc[keyCode].newState:
		if pressed {
			debug.PrintStack()
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

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
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

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
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

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
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

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
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

func TestHoldMultiKeys(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const badKey = 2
	const midiKey1 = 3
	const midiKey2 = 4
	const keyCode = 5
	const shortRelease = 10 * time.Millisecond
	const maxDelayMs = 100
	const eventDelay = 95 * time.Millisecond
	const threshold = 30

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := NewMockKeyController(keyCode)
	defer kc.Close()

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	ke.RegisterHoldAction(
		evType,
		channel,
		midiKey1,
		keyCode,
		threshold,
		maxDelayMs,
		shortRelease,
	)
	ke.RegisterHoldAction(
		evType,
		channel,
		midiKey2,
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
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey1,
		100,
		conn,
		shortRelease,
		time.Millisecond,
	)

	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey2,
		100,
		conn,
		shortRelease,
		time.Millisecond,
	)

	// Test that sending repeated MIDI events generates a long-lasting keyCode press.
	const alternatedDelay = eventDelay / 2
	const count = 10
	const maxTime = alternatedDelay * count
	go func() {
		midiKeys := []uint8{midiKey1, midiKey2}

		// Queue events roughly following the expected duration,
		// but alternating between the two MIDI events.
		for i := 0; i < count; i++ {
			midiKey := midiKeys[i&1]
			sendMidiEvent(evType, channel, midiKey, 100, conn)
			time.Sleep(alternatedDelay)
		}
	}()

	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		midiKey2,
		100,
		conn,
		maxTime,
		time.Millisecond*30,
	)
}

func TestRepeatSequence(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const midiPress = 2
	const midiNext = 3
	const midiPrev = 4
	const midiReset = 5
	const midiInvalid = 6
	keyCodes := [][]int{
		[]int{7},
		[]int{8, 9},
		[]int{10},
		[]int{11},
	}
	const shortRelease = 10 * time.Millisecond
	const maxDelayMs = 100
	const eventDelay = 95 * time.Millisecond
	const threshold = 30

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)

	var keys []int
	for _, kcs := range keyCodes {
		for _, key := range kcs {
			keys = append(keys, key)
		}
	}
	kc := NewMockKeyController(keys...)
	defer kc.Close()

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	ke.RegisterSequenceHoldAction(
		evType,
		channel,
		midiPress,
		keyCodes,
		threshold,
		maxDelayMs,
		shortRelease,
		midiPrev,
		midiNext,
		midiReset,
	)

	// Test that sending a MIDI event different from the expected doesn't set the keyCode.
	sendMidiEvent(evType, channel, midiInvalid, 100, conn)
	select {
	case <-kc[7].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-kc[8].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-kc[9].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-kc[10].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-kc[11].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-time.After(time.Millisecond):
		// Key wasn't pressed, as expected!
	}

	// testPress checks that the desired key was pressed.
	testPress := func(want int) {
		// Test that sending a quick MIDI event generates a quickly resolved keyCode press.
		assertKeyEvent(
			t,
			kc,
			want,
			evType,
			channel,
			midiPress,
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
				sendMidiEvent(evType, channel, midiPress, 100, conn)
				time.Sleep(eventDelay)
			}
		}()

		assertKeyEvent(
			t,
			kc,
			want,
			evType,
			channel,
			midiPress,
			100,
			conn,
			maxTime,
			time.Millisecond*10,
		)
	}

	// Try sending an event to press the initial key.
	testPress(keyCodes[0][0])

	// Move back to the last key and test again.
	sendMidiEvent(evType, channel, midiPrev, 100, conn)
	time.Sleep(eventDelay)
	testPress(keyCodes[3][0])

	// Move back another time and test again.
	sendMidiEvent(evType, channel, midiPrev, 100, conn)
	time.Sleep(eventDelay)
	testPress(keyCodes[2][0])

	// Reset back to the start and test again.
	sendMidiEvent(evType, channel, midiReset, 100, conn)
	time.Sleep(eventDelay)
	testPress(keyCodes[0][0])

	// Advance to the second step (that presses two keys).
	sendMidiEvent(evType, channel, midiNext, 100, conn)
	time.Sleep(eventDelay)
	sendMidiEvent(evType, channel, midiPress, 100, conn)

	// These checks must be manual
	// because otherwise the channel that reports that a key was pressed
	// ends up getting blocked.

	// Check that the keyCode were pressed.
	pressDeadline := time.After(time.Millisecond)
	releaseDeadline := time.After(shortRelease + 10*time.Millisecond)
	held := time.After(shortRelease - 10*time.Millisecond)
	select {
	case <-pressDeadline:
	case pressed := <-kc[keyCodes[1][0]].newState:
		if !pressed {
			debug.PrintStack()
			t.Fatalf("keyCode wasn't pressed in time")
		}
	}

	select {
	case <-pressDeadline:
	case pressed := <-kc[keyCodes[1][1]].newState:
		if !pressed {
			debug.PrintStack()
			t.Fatalf("keyCode wasn't pressed in time")
		}
	}

	// Check that the keyCode were released.
	select {
	case <-kc[keyCodes[1][0]].newState:
		debug.PrintStack()
		t.Fatalf("keyCode was released early")
	case <-kc[keyCodes[1][1]].newState:
		debug.PrintStack()
		t.Fatalf("keyCode was released early")
	case <-held:
		// Key was held down for as long as desired!
	}

	// Check that the keyCode was release in time.
	select {
	case <-releaseDeadline:
		debug.PrintStack()
		t.Fatalf("failed to detect that the keyCode was released in time")
	case pressed := <-kc[keyCodes[1][0]].newState:
		if pressed {
			debug.PrintStack()
			t.Fatalf("keyCode wasn't released in time")
		}
	}

	select {
	case <-releaseDeadline:
		debug.PrintStack()
		t.Fatalf("failed to detect that the keyCode was released in time")
	case pressed := <-kc[keyCodes[1][1]].newState:
		if pressed {
			debug.PrintStack()
			t.Fatalf("keyCode wasn't released in time")
		}
	}
}

func TestSwapNamedSet(t *testing.T) {
	const evType = midi.EventNoteOn
	const channel = 1
	const midiUnnamedKey = 2
	const midiSwapSet = 3
	const midiNamedKey = 4
	const badKey = 5
	const unnamedKeyCode = 6
	const namedKeyCodeA = 7
	const namedKeyCodeB = 8
	const releaseTime = 10 * time.Millisecond
	const threshold = 30

	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := NewMockKeyController(unnamedKeyCode, namedKeyCodeA, namedKeyCodeB)
	defer kc.Close()

	el := event_logger.New(nil)
	defer el.Close()

	ke, err := NewKeyEvents(kc, conn, false, el)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	// Register an event in the unnamed namespace,
	// activating unnamedKeyCode on midiUnnamedKey,
	// and the same event in two different namespaces,
	// one activating namedKeyCodeA (on namespace SET_A),
	// and the other activating namedKeyCodeB (on namespace SET_B).

	ke.RegisterBasicPressAction(
		evType,
		channel,
		midiUnnamedKey,
		unnamedKeyCode,
		threshold,
		releaseTime,
	)

	ke.RegisterMapSwap(
		evType,
		channel,
		midiSwapSet,
		threshold,
		[]string{"SET_A", "SET_B"},
	)

	ke.RegisterNamedSet("SET_A")

	ke.RegisterBasicPressAction(
		evType,
		channel,
		midiNamedKey,
		namedKeyCodeA,
		threshold,
		releaseTime,
	)

	ke.RegisterNamedSet("SET_B")

	ke.RegisterBasicPressAction(
		evType,
		channel,
		midiNamedKey,
		namedKeyCodeB,
		threshold,
		releaseTime,
	)

	ke.SetNamedSet("SET_A")

	// Test that sending a MIDI event different from the expected doesn't set the keyCode.
	sendMidiEvent(evType, channel, badKey, 100, conn)
	select {
	case <-kc[unnamedKeyCode].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-kc[namedKeyCodeA].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-kc[namedKeyCodeB].newState:
		t.Fatalf("keyCode was pressed by an invalid MIDI event")
	case <-time.After(time.Millisecond):
		// Key wasn't pressed, as expected!
	}

	// Test that sending the event keeps the keyCode pressed for the desired time.
	assertKeyEvent(
		t,
		kc,
		unnamedKeyCode,
		evType,
		channel,
		midiUnnamedKey,
		100,
		conn,
		releaseTime,
		time.Millisecond,
	)

	// Test that the first named set is active.
	assertKeyEvent(
		t,
		kc,
		namedKeyCodeA,
		evType,
		channel,
		midiNamedKey,
		100,
		conn,
		releaseTime,
		time.Millisecond,
	)

	// Ensure that the second named set (SET_B) wasn't activated.
	select {
	case <-kc[namedKeyCodeB].newState:
		t.Fatalf("keyCode was activated when its namespace should have been inactive")
	default:
		// Key wasn't activated, as expected.
	}

	// Swap and ensure that key B gets activated.
	sendMidiEvent(evType, channel, midiSwapSet, 100, conn)
	time.Sleep(time.Millisecond)

	assertKeyEvent(
		t,
		kc,
		namedKeyCodeB,
		evType,
		channel,
		midiNamedKey,
		100,
		conn,
		releaseTime,
		time.Millisecond,
	)

	// Ensure that the first named set (SET_A) wasn't activated.
	select {
	case <-kc[namedKeyCodeA].newState:
		t.Fatalf("keyCode was activated when its namespace should have been inactive")
	default:
		// Key wasn't activated, as expected.
	}

	// Swap back and ensure that both key A keys activated,
	// and the key B doesn't get activated.
	sendMidiEvent(evType, channel, midiSwapSet, 100, conn)
	time.Sleep(time.Millisecond)

	assertKeyEvent(
		t,
		kc,
		namedKeyCodeA,
		evType,
		channel,
		midiNamedKey,
		100,
		conn,
		releaseTime,
		time.Millisecond,
	)
	select {
	case <-kc[namedKeyCodeB].newState:
		t.Fatalf("keyCode was activated when its namespace should have been inactive")
	default:
		// Key wasn't activated, as expected.
	}
}
