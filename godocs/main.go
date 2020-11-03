// godocs runs the 'go doc' command for a given argument, writing the output to a new acme window.
//
// Usage:
//
// 	godocs [-all] [-get] search_string
//
// Button 3 clicks in a godocs window will spawn child windows for the symbol name that's search for,
// so you can "drill down" for docs on particular functions or types within a single go package.

package main

import (
	"flag"
	"log"
	"os/exec"
	"strings"

	"9fans.net/go/acme"
)

var useAll bool
var doGet bool

func main() {
	log.SetFlags(0)
	log.SetPrefix("godocs: ")
	flag.BoolVar(&useAll, "all", false, "pass -all flag to go doc when searching for module/symbol")
	flag.BoolVar(&doGet, "get", true, "do 'go get' for the package before running 'go doc' on it")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("no search string provided")
	}
	newwin([]string{args[0]})
}

type handler struct {
	path []string
}

func newwin(names []string) {
	if len(names) < 1 {
		log.Printf("no search strings provided")
		return
	}
	if doGet && len(names) == 1 {
		getCmd := exec.Command("go", "get", names[0])
		log.Println("running", getCmd)
		_, err := getCmd.CombinedOutput()
		if err != nil {
			log.Printf("error doing '%s': %v", getCmd, err)
			return
		}
	}
	args := []string{"doc"}
	if useAll {
		args = append(args, "-all")
	}
	searchFor := strings.Join(names, ".")
	args = append(args, searchFor)
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("error running %v: %v", cmd, err)
		return
	}
	w, err := acme.New()
	if err != nil {
		log.Println("error creating window", err)
		return
	}
	w.Name("/godocs/" + searchFor)
	w.Write("body", out)
	w.Ctl("clean")
	w.EventLoop(&handler{names})
}

func (h *handler) Execute(cmd string) bool {
	return false
}

func (h *handler) Look(arg string) bool {
	// be smarter here
	go newwin(append(h.path, arg))
	return true
}
