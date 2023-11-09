package event_logger

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// The number of events that may be queued
const queueSize = 10

// How often updates are sent to the remote server.
const cacheTime = time.Millisecond * 50

type EventLogger interface {
	// Close closes the event logger.
	Close()

	// SendRegisterEvent queues a register event,
	// which associates the tuple (channel,key) with keyboard,
	// which may be various keyboard keys separated by commas.
	SendRegisterEvent(channel, key uint8, keyboard string)

	// SendMIDIEvent queues a MIDI event,
	// which registers that the given MIDI event was received.
	SendMIDIEvent(channel, key uint8)

	// SendKeyboardEvent queues a keyboard event,
	// which registers that the given keyboard key(s) was(were) pressed/released.
	SendKeyboardEvent(keys []string, isPressed bool)
}

type keyboardEvent struct {
	// The keys that were pressed by this event.
	Keys []string
	// The state of the keys.
	IsPressed bool
}

type midiEvent struct {
	// The MIDI event channel.
	Channel uint8
	// The MIDI event key.
	Key uint8
}

// MarshalText implements encoding.TextMarshaler,
// so midiEvent may be used as a key to a map.
func (me midiEvent) MarshalText() ([]byte, error) {
	data := []byte{me.Channel, me.Key}
	encoded := "0x" + hex.EncodeToString(data)
	return []byte(encoded), nil
}

type registerEvent struct {
	// The MIDI event.
	Event midiEvent
	// The keyboard key(s) associated with this event.
	// If multiple keys are to be pressed by a single event,
	// those should be comma-separated.
	Key string
}

type eventLogger struct {
	// The HTTP client used to log the event ot the remote endpoint.
	client *http.Client
	// The endpoint to which events are sent.
	endpoint string
	// The channel used to queue the prepared message to be set to the remote endpoint.
	sender chan []byte
	// The channel used to receive events.
	queue chan any
	// Timer used to buffer a few events together.
	timer *time.Ticker
	// Association between a MIDI event and the key(s) that it presses.
	midiMap map[midiEvent]string
	// When each MIDI event was last pressed.
	midiPresses map[midiEvent]time.Time
	// Which keyboard keys are currently pressed.
	keyPresses map[string]bool
	// Whether there were any updates to the maps.
	didUpdate bool
}

type message struct {
	// Association between a MIDI event and the key(s) that it presses.
	MidiMap map[midiEvent]string `json:"map"`
	// When each MIDI event was last pressed.
	MidiPresses map[midiEvent]time.Time `json:"midi"`
	// Which keyboard keys are currently pressed.
	KeysPresses map[string]bool `json:"keys"`
}

// New starts a new event logger,
// which may send the current state of the pressed keys/received events
// to an endpoint.
func New(endpoint *string) EventLogger {
	el := &eventLogger{
		queue:       make(chan any, queueSize),
		midiMap:     make(map[midiEvent]string),
		midiPresses: make(map[midiEvent]time.Time),
		keyPresses:  make(map[string]bool),
		timer:       time.NewTicker(cacheTime),
	}

	if endpoint != nil && *endpoint != "" {
		el.client = &http.Client{}
		el.endpoint = *endpoint
		el.sender = make(chan []byte, 1)
		go el.send()
	}
	go el.run()

	return el
}

func (el *eventLogger) Close() {
	close(el.queue)
	if el.client != nil {
		el.timer.Stop()
		close(el.sender)
	}
	return
}

func (el *eventLogger) SendRegisterEvent(channel, key uint8, keyboard string) {
	el.queue <- registerEvent{
		Event: midiEvent{
			Channel: channel,
			Key:     key,
		},
		Key: keyboard,
	}
}

func (el *eventLogger) SendMIDIEvent(channel, key uint8) {
	el.queue <- midiEvent{
		Channel: channel,
		Key:     key,
	}
}

func (el *eventLogger) SendKeyboardEvent(keys []string, isPressed bool) {
	el.queue <- keyboardEvent{
		Keys:      keys,
		IsPressed: isPressed,
	}
}

// run is the event logger's mainloop,
// waiting and handling events.
func (el *eventLogger) run() {
	for {
		select {
		case event, more := <-el.queue:
			if !more {
				return
			}

			el.handleEvent(event)
		case <-el.timer.C:
			if el.client == nil || !el.didUpdate {
				continue
			}

			el.queueMessage()
		}
	}
}

// handleEvent registers the event in the appropriate map.
func (el *eventLogger) handleEvent(event any) {
	switch value := event.(type) {
	case keyboardEvent:
		for _, key := range value.Keys {
			el.keyPresses[key] = value.IsPressed
		}
	case midiEvent:
		el.midiPresses[value] = time.Now()
	case registerEvent:
		el.midiMap[value.Event] = value.Key
	default:
		log.Printf("event_logger: unknown event type '%T'", event)
	}

	el.didUpdate = true
}

// queueMessage prepares a message and queue it to be sent to the remote server.
func (el *eventLogger) queueMessage() {
	msg := message{
		MidiMap:     el.midiMap,
		MidiPresses: el.midiPresses,
		KeysPresses: el.keyPresses,
	}

	data, err := json.Marshal(&msg)
	if err != nil {
		log.Printf("event_logger: failed to encode the message: %+v", err)
		return
	}

	el.sender <- data
	el.didUpdate = false
}

// send sends the message to the remote endpoint.
func (el *eventLogger) send() {
	var buf bytes.Buffer

	for msg := range el.sender {
		buf.Reset()
		buf.Write(msg)

		resp, err := el.client.Post(el.endpoint, "application/json", &buf)
		if err != nil {
			log.Printf("event_logger: failed to send a message: %+v", err)
			continue
		} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			log.Printf("event_logger: request failed with code '%s'", resp.Status)
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
