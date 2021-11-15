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
	// track the last time a window was interacted with
	lastmod = make(map[int]time.Time)
	// track all the paths that have been opened
	pathlog = make(map[string]bool)

	// some windows are almost always ok to clean up
	deleters = []delfunc{
		func(w acme.WinInfo) bool { return w.Name == "Del" },
		func(w acme.WinInfo) bool { return strings.HasPrefix(w.Name, "/godocs/") },
		func(w acme.WinInfo) bool { return strings.HasSuffix(w.Name, "+Errors") && w.Name != "+Errors" },
		func(w acme.WinInfo) bool { return strings.HasSuffix(w.Name, "+pg") },
		func(w acme.WinInfo) bool { return strings.HasSuffix(w.Name, "+ff") },
		func(w acme.WinInfo) bool {
			if stat, err := os.Stat(w.Name); err == nil {
				return stat.IsDir()
			}
			return false
		},
	}
)

func (h *handler) println(a ...interface{}) {
	h.w.Write("body", []byte(fmt.Sprint(a...)+"\n"))
	h.w.Ctl("clean")
}

func (h *handler) closeWin(winID int) {
	wp, err := acme.Open(winID, nil)
	if err != nil {
		h.println("error opening window", winID, err)
	}
	wp.Del(false) // ignore errors :shrug:
}

func (h *handler) ExecTidy(cmd string) {
	shouldDelete := func(w acme.WinInfo) bool {
		for _, f := range deleters {
			if f(w) {
				return true
			}
		}
		return false
	}
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
		if shouldDelete(wi) {
			h.closeWin(wi.ID)
		}
	}
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
		switch event.Op {
		case "del":
			delete(lastmod, event.ID)
		case "get":
			pathlog[event.Name] = true
		case "focus":
		default:
			h.println(fmt.Sprint(event))
			lastmod[event.ID] = time.Now()
			pathlog[event.Name] = true
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
	w.Write("tag", []byte("Tidy Log"))
	h := handler{w: w}
	go readlog(&h)
	w.EventLoop(&h)
}
