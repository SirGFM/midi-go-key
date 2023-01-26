package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/SirGFM/midi-go-key/key_events"
	"github.com/SirGFM/midi-go-key/midi"
)

// How many events may be queued
const defaulEventQueueSize = 64

func main() {
	defer midi.Cleanup()

	eventQueueSize := flag.Int("queueSize", defaulEventQueueSize, "how many events may be queued")
	port := flag.Int("port", 0, "the device's port")
	list := flag.Bool("list", false, "whether the application should list the devices and exit")
	path := flag.String("config", "./config.txt", "the path to the configuration file")
	logUnhandled := flag.Bool("log-unhandled", false, "whether unhandled events should be logged")
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
	if path == nil {
		path = new(string)
		*path = "./config.txt"
	}
	if logUnhandled == nil {
		logUnhandled = new(bool)
		*logUnhandled = false
	}

	conn := make(chan midi.MidiEvent, *eventQueueSize)
	kb, err := key_events.NewKeyEvents(conn, *logUnhandled)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	defer kb.Close()

	if len(*path) > 0 {
		err = kb.ReadConfig(*path)
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
	}

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
