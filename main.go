package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/SirGFM/midi-go-key/key_events"
	"github.com/SirGFM/midi-go-key/midi"

	"time"
)

// How many events may be queued
const defaulEventQueueSize = 64

func main() {
	defer midi.Cleanup()

	eventQueueSize := flag.Int("queueSize", defaulEventQueueSize, "how many events may be queued")
	port := flag.Int("port", 0, "the device's port")
	list := flag.Bool("list", false, "whether the application should list the devices and exit")
	flag.Parse()

	// List the devices and exit.
	if list != nil && *list {
		devs, err := midi.ListDevices()
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}

		fmt.Println("Device(s):")
		for _, dev := range devs {
			fmt.Printf("%s: Port=%d\n", dev.Name, dev.Port)
		}

		return
	}

	if port == nil {
		panic("a port must be supplied!")
	}
	if eventQueueSize == nil {
		eventQueueSize = new(int)
		*eventQueueSize = defaulEventQueueSize
	}

	conn := make(chan midi.MidiEvent, *eventQueueSize)
	kb, err := key_events.NewKeyEvents(conn)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	defer kb.Close()

	// Press 'A' after a tom4.
	kb.RegisterBasicPressAction(midi.EventNoteOn, 9, 41, 30, time.Second)
	// Press 'B' after a tom3
	kb.RegisterVelocityAction(midi.EventNoteOn, 9, 43, 48, time.Millisecond * 10, time.Second)
	// Press 'C' after a hi-hat
	kb.RegisterToggleAction(midi.EventNoteOn, 9, 44, 46, 75, time.Millisecond * 10)

	midiDev, err := midi.NewMidi(*port, conn)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	defer midiDev.Close()

	// Register a signal handler, so the application may sleep until it's done.
	fmt.Println("Listening to device...")
	intHndlr := make(chan os.Signal, 1)
	signal.Notify(intHndlr, os.Interrupt)
	<-intHndlr
	fmt.Printf("Exiting...")
}
