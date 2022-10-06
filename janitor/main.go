package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"9fans.net/go/acme"
)

var verbose bool

func logf(fmt string, v ...any) {
	if verbose {
		log.Printf(fmt, v...)
	}
}

func logv(v ...any) {
	if verbose {
		log.Println(v...)
	}
}

func main() {
	freqSecs := flag.Int("f", 60, "frequency to tidy things up, in seconds")
	deleteAgeHours := flag.Int("a", 32, "stale window deletion age, in hours")
	flag.Parse()
	log.SetPrefix("janitor:")
	log.SetFlags(0)

	// track the last time each window was touched
	windows := make(map[int]time.Time)

	l, err := acme.Log()
	if err != nil {
		log.Fatal("couldn't open acme log", err)
	}
	defer l.Close()
	window, err := acme.New()
	if err != nil {
		log.Fatal("couldn't create acme window", err)
	}
	window.Name("+janitor")
	uiEvents := window.EventChan()
	acmeLogs := make(chan acme.LogEvent)
	go func() {
		for {
			if e, err := l.Read(); err == nil {
				acmeLogs <- e
			} else {
				log.Println(e)
			}
		}
	}()

	ticker := time.NewTicker(time.Duration(*freqSecs) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case e := <-uiEvents:
			switch e.C2 {
			case 'x', 'X': // execute
				cmd := strings.TrimSpace(string(e.Text))
				logv("got window command", cmd)
				window.WriteEvent(e)
			}

		case e := <-acmeLogs:
			switch e.Op {
			case "focus":
			case "del":
				delete(windows, e.ID)
			default:
				windows[e.ID] = time.Now()
			}

		case <-ticker.C:
			cutoff := time.Now().Add(-1 * time.Duration(*deleteAgeHours) * time.Hour)
			logv("cleaning up stale windows before", cutoff.Format(time.Stamp))
			for id, v := range windows {
				if id == window.ID() {
					continue // don't mess with our window
				}
				if v.Before(cutoff) {
					logv("removing window", id, "last touched at", v.Format(time.Stamp))
					if w, err := acme.Open(id, nil); err == nil {
						w.Del(false)
					}
				}
			}
		}
	}
}
