// godocs runs the 'go doc' command for a given argument, writing the output to a new acme window.
//
// Usage:
//
// 	godocs search_string
//
// Button 3 clicks in a godocs window will spawn child windows for the symbol name that's search for,
// so you can "drill down" for docs on particular functions or types within a single go package.

package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"9fans.net/go/acme"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("godocs: ")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("no search string provided")
	}
	newwin([]string{args[0]})
}

// tracks state and has event handlers for each /godocs/ window
type handler struct {
	path   []string
	useAll bool
	useSrc bool
	win    *acme.Win
}

// call `go doc` with appopriate flags for the given window state
func (h *handler) godoc() {
	args := []string{"doc"}
	if h.useAll {
		args = append(args, "-all")
	}
	if h.useSrc {
		args = append(args, "-src")
	}
	args = append(args, strings.Join(h.path, "."))
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	h.win.Clear()
	if err != nil {
		h.win.Write("body", []byte(fmt.Sprintf("error for '%v': %v", cmd, err)))
	} else {
		h.win.Write("body", out)
	}
	h.win.Ctl("clean")
}

// make a new window for the given package/symbol names to search for
func newwin(names []string) {
	if len(names) < 1 {
		log.Printf("no search strings provided")
		return
	}
	w, err := acme.New()
	if err != nil {
		log.Println("error creating window", err)
		return
	}
	h := &handler{
		path: names,
		win:  w,
	}
	h.win.Name("/godocs/" + strings.Join(h.path, "."))
	h.win.Write("tag", []byte("Get All Src"))
	h.godoc()
	w.EventLoop(h)
}

func (h *handler) Execute(cmd string) bool {
	return false
}

func (h *handler) ExecAll(cmd string) {
	h.useAll = !h.useAll
	h.godoc()
}

func (h *handler) ExecSrc(cmd string) {
	h.useSrc = !h.useSrc
	h.godoc()
}

func (h *handler) ExecGet(cmd string) {
	getCmd := exec.Command("go", "get", h.path[0])
	_, err := getCmd.CombinedOutput()
	if err != nil {
		log.Printf("error doing '%s': %v", getCmd, err)
		return
	}
	h.godoc()
}

func (h *handler) Look(arg string) bool {
	// be smarter here
	go newwin(append(h.path, arg))
	return true
}
