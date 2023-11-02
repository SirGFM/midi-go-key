package main

import (
	"flag"
	"strings"
	"time"

	"github.com/SirGFM/midi-go-key/key_events/key_handler"
)

func main() {
	names := flag.String("keys", "SPACE,A", "the keys to be pressed, separated by commas")
	delay := flag.Duration("delay", 0, "delay until the key is pressed")
	hold := flag.Duration("hold", time.Second / 2, "for how long the key should be pressed")
	flag.Parse()

	var keys []int
	for _, name := range strings.Split(*names, ",") {
		key, found := keyNameToInt[name]
		if !found {
			panic("invalid key: " + name)
		}
		keys = append(keys, key)
	}

	kh, err := key_handler.New()
	if err != nil {
		panic(err)
	}
	defer kh.Close()

	time.Sleep(*delay)

	kh.PressKeys(keys...)
	time.Sleep(*hold)
	kh.ReleaseKeys(keys...)
}
