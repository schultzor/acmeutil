package main

import (
	"flag"
	"log"
	"os/exec"

	"9fans.net/go/acme"
)

// [go install .]

type handler struct {
	search string
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("no search string provided")
	}
	newwin(args[0])
}

func newwin(search string) {
	w, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	w.Name("/godocs/" + search)
	cmd := exec.Command("go", "doc", search)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("error running %v: %v", cmd, err)
	}
	w.Write("body", out)
	w.Ctl("clean")
	w.EventLoop(&handler{search})
}

func (h *handler) Execute(cmd string) bool {
	return false
}

func (h *handler) Look(arg string) bool {
	go newwin(h.search + "." + arg)
	return true
}