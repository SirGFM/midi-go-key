package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/SirGFM/midi-go-key/event_logger"
	"github.com/SirGFM/midi-go-key/key_events"
	"github.com/SirGFM/midi-go-key/key_events/key_handler"
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
	endpoint := flag.String("endpoint", "http://localhost:8080/ram_store/drums", "(optional) the overlay endpoint")
	logUnhandled := flag.Bool("log-unhandled", false, "whether unhandled events should be logged")
	flag.Parse()

	// List the devices and exit.
	if list != nil && *list {
		devs, err := midi.ListDevices()
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}

		log.Println("device(s):")
		for _, dev := range devs {
			log.Printf("%s: port=%d", dev.Name, dev.Port)
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

	el := event_logger.New(endpoint)
	defer el.Close()

	conn := make(chan midi.MidiEvent, *eventQueueSize)

	kc, err := key_handler.New()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	kb, err := key_events.NewKeyEvents(kc, conn, *logUnhandled)
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
	log.Println("listening to device...")
	intHndlr := make(chan os.Signal, 1)
	signal.Notify(intHndlr, os.Interrupt)
	<-intHndlr
	log.Printf("exiting...")
}
