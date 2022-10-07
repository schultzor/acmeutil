package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"9fans.net/go/acme"
)

func fatal(v ...any) {
	log.Fatal(v...)
}

func fmtTime(t time.Time) string {
	return t.Format(time.Stamp)
}

func readLogs(lr *acme.LogReader) chan acme.LogEvent {
	c := make(chan acme.LogEvent)
	go func() {
		for {
			if e, err := lr.Read(); err == nil {
				c <- e
			} else {
				break
			}
		}
	}()
	return c
}

func isDir(p string) bool {
	if f, err := os.Stat(p); err == nil {
		return f.IsDir()
	}
	return false
}

func deleteWin(id int) {
	if w, err := acme.Open(id, nil); err == nil {
		w.Del(false)
	}
}

type winLog struct {
	ts time.Time
	e  acme.LogEvent
}

type handler struct {
	win      *acme.Win
	acmeLogs chan acme.LogEvent
	lastMod  map[int]winLog
	staleAge time.Duration
	debug    bool
}

func initHandler(lr *acme.LogReader) *handler {
	w, err := acme.New()
	if err != nil {
		fatal("error creating acme window", err)
	}
	h := &handler{win: w,
		acmeLogs: readLogs(lr),
		lastMod:  make(map[int]winLog),
		staleAge: time.Minute * 60,
	}
	h.win.Name("+janitor")
	h.win.Write("tag", []byte("Debug List Expire Tidy"))
	return h
}

func (h *handler) toggleDebug() {
	h.debug = !h.debug
}

func (h *handler) tidy() {
	wl, err := acme.Windows()
	if err != nil {
		log.Println("error listing acme windows:", err)
		return
	}
	for _, w := range wl {
		var deleteIt bool
		switch {
		case w.ID == h.win.ID():
			h.log("ignoring our window id", w.ID)

		case strings.HasPrefix(w.Name, "/godocs/"):
			deleteIt = true
		case strings.HasSuffix(w.Name, "+Errors"):
			deleteIt = true
		case isDir(w.Name):
			deleteIt = true
		}
		if deleteIt {
			h.log("deleting window", w.ID, "for", w.Name)
			deleteWin(w.ID)
		}
	}
}

func (h *handler) log(v ...any) {
	if h.debug {
		h.win.Write("body", []byte(fmt.Sprintln(v...)))
		h.win.Ctl("clean")
	}
}

func (h *handler) logf(format string, v ...any) {
	h.log(fmt.Sprintf(format, v...))
}

func (h *handler) deleteStaleWindows() {
	cutoff := time.Now().Add(-1 * h.staleAge)
	h.log("deleting windows that haven't changed since", fmtTime(cutoff))
	for id, v := range h.lastMod {
		if id == h.win.ID() {
			continue
		}
		if v.ts.Before(cutoff) {
			h.log("deleting window", id, v.e.Name, "last mod", fmtTime(v.ts))
			deleteWin(id)
		}
	}
}

func (h *handler) list() {
	var bb bytes.Buffer
	fmt.Fprintf(&bb, "staleAge: %v, debug: %v\n", h.staleAge, h.debug)
	for k, v := range h.lastMod {
		fmt.Fprintf(&bb, "%d -> %s, last mod at %s\n", k, v.e.Name, fmtTime(v.ts))
	}
	h.win.Clear()
	h.win.Write("body", bb.Bytes())
	h.win.Ctl("clean")
}

func (h *handler) run() {
	uiEvents := h.win.EventChan()
	for {
		select {
		case e := <-uiEvents:
			switch e.C2 {
			case 'x', 'X': // execute
				cmd := strings.TrimSpace(string(e.Text))
				switch cmd {
				case "Debug":
					h.toggleDebug()
				case "Tidy":
					h.tidy()
				case "Expire":
					h.deleteStaleWindows()
				case "List":
					h.list()
				case "Del":
					h.win.WriteEvent(e)
					return
				default:
					h.win.WriteEvent(e)
				}
			case 'l', 'L': // look
				h.win.WriteEvent(e)
			}

		case e := <-h.acmeLogs:
			switch e.Op {
			case "focus":
			case "del":
				h.log("deleting entry for", e.ID)
				delete(h.lastMod, e.ID)
			default:
				h.log("updating timestamp for", e.ID)
				h.lastMod[e.ID] = winLog{e: e, ts: time.Now()}
			}
		}
	}
}

func main() {
	log.SetPrefix("janitor:")
	log.SetFlags(0)

	acmeLog, err := acme.Log()
	if err != nil {
		log.Fatal("couldn't open acme log", err)
	}
	defer acmeLog.Close()
	ui := initHandler(acmeLog)

	ui.run()
}
