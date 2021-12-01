package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"9fans.net/go/acme"
)

type handler struct {
	w *acme.Win
}

type delfunc func(w acme.WinInfo) bool

const (
	oldCutoff = time.Hour * 24 * 3
)

var (
	verbose = false

	// track the last time a window was updated
	lastmod = make(map[int]time.Time)

	// track all the paths that have been opened
	pathlog = make(map[string]bool)

	// some windows are usually ok to close
	deleters = []delfunc{
		func(w acme.WinInfo) bool { return strings.HasPrefix(w.Name, "/godocs/") },
		func(w acme.WinInfo) bool { return strings.HasSuffix(w.Name, "/+Errors") },
		func(w acme.WinInfo) bool {
			if stat, err := os.Stat(w.Name); err == nil {
				return stat.IsDir()
			}
			return false
		},
	}
)

func canDelete(w acme.WinInfo) bool {
	for _, f := range deleters {
		if f(w) {
			return true
		}
	}
	return false
}

func (h *handler) println(a ...interface{}) {
	h.w.Write("body", []byte(fmt.Sprint(a...)+"\n"))
	h.w.Ctl("clean")
}

func (h *handler) closeWin(winID int) {
	if winID == h.w.ID() {
		return // don't close ourself
	}
	if wp, err := acme.Open(winID, nil); err == nil {
		wp.Del(false) // ignore errors
	}
}

func (h *handler) ExecTidy(cmd string) {
	for k, v := range lastmod {
		if v.Before(time.Now().Add(-1 * oldCutoff)) {
			h.closeWin(k)
		}
	}
	allWin, err := acme.Windows()
	if err != nil {
		log.Fatal("error getting windows", err)
	}
	for _, wi := range allWin {
		if canDelete(wi) {
			h.closeWin(wi.ID)
		}
	}
}

func (h *handler) ExecVerbose(cmd string) {
	verbose = !verbose
}

func (h *handler) ExecLog(cmd string) {
	h.w.Clear()
	var s []string
	for k := range pathlog {
		s = append(s, k)
	}
	sort.Strings(s)
	for _, n := range s {
		h.println(n)
	}
}

func (h *handler) Execute(cmd string) bool {
	return false
}
func (h *handler) Look(arg string) bool {
	return false
}

func readlog(h *handler) {
	l, err := acme.Log()
	if err != nil {
		log.Fatal(err)
	}
	for {
		event, err := l.Read()
		if err != nil {
			log.Fatal(err)
		}
		if verbose {
			h.println(fmt.Sprint(event))
		}
		pathlog[event.Name] = true
		switch event.Op {
		case "del":
			delete(lastmod, event.ID)
		case "focus": // ignore em
		default:
			lastmod[event.ID] = time.Now()
		}
	}
}

func main() {
	log.SetPrefix("janitor ")
	w, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	w.Name("+janitor")
	w.Write("tag", []byte("Log Tidy Verbose"))
	h := handler{w: w}
	go readlog(&h)
	w.EventLoop(&h)
}
