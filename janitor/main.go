package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"9fans.net/go/acme"
)

type handler struct {
	w *acme.Win
}

func (h *handler) println(a ...interface{}) {
	h.w.Write("body", []byte(fmt.Sprint(a...)+"\n"))
	h.w.Ctl("clean")
}

func shouldDelete(w acme.WinInfo) bool {
	if w.Name == "Del" {
		return true
	}
	if strings.HasPrefix(w.Name, "/godocs/") {
		return true
	}
	if strings.HasSuffix(w.Name, "+Errors") && w.Name != "+Errors" {
		return true
	}
	if strings.HasSuffix(w.Name, "+pg") {
		return true
	}
	if strings.HasSuffix(w.Name, "+ff") {
		return true
	}
	stat, err := os.Stat(w.Name)
	if err != nil {
		return false
	}
	if stat.IsDir() {
		return true
	}
	return false
}

// track the last time a window was interacted with
var lastMod = make(map[int]time.Time)

const oldCutoff = time.Hour * 24 * 5 // if I haven't touched a buffer in 5 days...

func (h *handler) closeWin(winID int) {
	wp, err := acme.Open(winID, nil)
	if err != nil {
		h.println("error opening window", winID, err)
	}
	h.println("deleting ", wp)
	if err := wp.Del(false); err == nil {
		h.println("error deleting window", winID, err)
	}
}

func (h *handler) ExecTidy(cmd string) {
	h.println("tidying up")
	for k, v := range lastMod {
		if v.Before(time.Now().Add(-1 * oldCutoff)) {
			h.println("deleting old window ", k, v)
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

func (h *handler) ExecDebug(cmd string) {
	for k, v := range lastMod {
		h.println("lastMod: ", k, v)
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
			//h.println(fmt.Sprint(event))
			delete(lastMod, event.ID)
		case "get":
		case "focus":
		// ignored
		default:
			h.println(fmt.Sprint(event))
			lastMod[event.ID] = time.Now()
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
	w.Write("tag", []byte("Tidy Debug"))
	h := handler{w: w}
	go readlog(&h)
	w.EventLoop(&h)
}
