// listens to the acme event log and tracks the last time each window was touched/updated
// allows for deleting windows that haven't been updated in the last N hours
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"9fans.net/go/acme"
	// lru "github.com/hashicorp/golang-lru"
)

const janitorname = "+janitor"

type mapval struct {
	ts   time.Time
	name string
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

type handler struct {
	win      *acme.Win
	acmeLogs chan acme.LogEvent
	windows  map[int]mapval
}

func initHandler(lr *acme.LogReader) *handler {
	w, err := acme.New()
	if err != nil {
		log.Fatal("error creating acme window", err)
	}
	h := &handler{
		win:      w,
		acmeLogs: readLogs(lr),
		windows:  make(map[int]mapval),
	}
	h.win.Name(janitorname)
	h.win.Write("tag", []byte("List Tidy Expire48 "))

	// add existing windows
	if wl, err := acme.Windows(); err == nil {
		for _, w := range wl {
			h.touchWin(w.ID, w.Name)
		}
	} else {
		log.Fatal("error listing acme windows:", err)
	}
	return h
}

func (h *handler) expire(cutoff time.Time) {
	var ids []int
	for k, v := range h.windows {
		switch {
		case v.name == janitorname:
			// ignore ourself
		case v.ts.Before(cutoff):
			ids = append(ids, k)
		}
	}
	for _, i := range ids {
		h.deleteWin(i)
	}
}

func (h *handler) tidy() {
	var ids []int
	for k, v := range h.windows {
		switch {
		case v.name == janitorname:
			// ignore ourself
		case strings.TrimSpace(v.name) == "":
			ids = append(ids, k)
		case strings.HasPrefix(v.name, "/godocs/"):
			ids = append(ids, k)
		case strings.HasSuffix(v.name, "+Errors"):
			ids = append(ids, k)
		case isDir(v.name):
			ids = append(ids, k)
		}
	}
	for _, i := range ids {
		h.deleteWin(i)
	}
}

func (h *handler) list() {
	var lst []mapval
	for _, v := range h.windows {
		lst = append(lst, v)
	}
	slices.SortFunc(lst, func(a, b mapval) int { return a.ts.Compare(b.ts) })
	var bb bytes.Buffer
	for _, v := range lst {
		fmt.Fprintf(&bb, "%s,%s\n", v.name, fmtTime(v.ts))
	}
	h.win.Clear()
	h.win.Write("body", bb.Bytes())
	h.win.Ctl("clean")
}

func (h *handler) touchWin(id int, name string) {
	if id == h.win.ID() {
		name = janitorname
	}
	h.windows[id] = mapval{ts: time.Now(), name: name}
}

func (h *handler) deleteWin(id int) {
	if w, err := acme.Open(id, nil); err == nil {
		if err := w.Del(false); err != nil {
			log.Println("error deleting window:", id)
		}
	}
	delete(h.windows, id)
}

func (h *handler) run() {
	uiEvents := h.win.EventChan()
	for {
		select {
		case e := <-uiEvents:
			switch e.C2 {
			case 'l', 'L': // look
				h.win.WriteEvent(e)
			case 'x', 'X': // execute
				cmd := strings.TrimSpace(string(e.Text))
				switch {
				case strings.HasPrefix(cmd, "Expire"):
					if hours, err := strconv.Atoi(strings.TrimPrefix(cmd, "Expire")); err == nil {
						cutoff := time.Now().Add(-1 * time.Hour * time.Duration(hours))
						h.expire(cutoff)
					} else {
						log.Println("error doing command", cmd, err)
					}
				case cmd == "List":
					h.list()
				case cmd == "Tidy":
					h.tidy()
				case cmd == "Del":
					h.win.WriteEvent(e)
					return
				default:
					h.win.WriteEvent(e)
				}
			}

		case e := <-h.acmeLogs:
			switch e.Op {
			case "focus":
			case "del":
				h.deleteWin(e.ID)
			default:
				h.touchWin(e.ID, e.Name)
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
