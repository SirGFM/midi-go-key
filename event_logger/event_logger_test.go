package event_logger

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEncodeMidiEventToStr(t *testing.T) {
	m := make(map[midiEvent]string)

	ev := midiEvent{
		Channel: 1,
		Key:     2,
	}
	m[ev] = "test"

	data, err := json.Marshal(&m)
	if err != nil {
		t.Fatalf("failed to encode the map: %+v", err)
	}

	out := make(map[string]string)
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Fatalf("failed to decode the message: %+v", err)
	} else if want, got := m[ev], out["0x0102"]; want != got {
		t.Fatalf("invalid message decoded - want: '%s', got: '%s'", want, got)
	}
}

func TestEncodeMidiEventToTime(t *testing.T) {
	m := make(map[midiEvent]time.Time)

	ev := midiEvent{
		Channel: 1,
		Key:     2,
	}
	date, err := time.Parse(time.RFC3339, "2023-11-09T21:12:20-03:00")
	if err != nil {
		t.Fatalf("failed to parse the time: %+v", err)
	}
	m[ev] = date

	data, err := json.Marshal(&m)
	if err != nil {
		t.Fatalf("failed to encode the map: %+v", err)
	}

	out := make(map[string]time.Time)
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Fatalf("failed to decode the message: %+v", err)
	} else if want, got := m[ev].String(), out["0x0102"].String(); want != got {
		t.Fatalf("invalid message decoded - want: '%s', got: '%s'", want, got)
	}
}

func TestEncodeMidiEventToBool(t *testing.T) {
	m := make(map[midiEvent]bool)

	ev1 := midiEvent{
		Channel: 1,
		Key:     2,
	}
	m[ev1] = true
	ev2 := midiEvent{
		Channel: 3,
		Key:     4,
	}
	m[ev2] = true

	data, err := json.Marshal(&m)
	if err != nil {
		t.Fatalf("failed to encode the map: %+v", err)
	}

	out := make(map[string]bool)
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Fatalf("failed to decode the message: %+v", err)
	} else if want, got := m[ev1], out["0x0102"]; want != got {
		t.Fatalf("invalid message decoded - want: '%v', got: '%v'", want, got)
	} else if want, got := m[ev2], out["0x0304"]; want != got {
		t.Fatalf("invalid message decoded - want: '%v', got: '%v'", want, got)
	}
}
