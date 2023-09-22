// gitwin is a hacky way to interact with git via an acme window
//
// Usage:
//
//	gitwin /path/to/gitrepo/root
//
// Available commands are defined in the commands.go, they can be enumerated by doing
// a button 2 click on the "Help" command in the gitwin window.

package main

import (
	"bytes"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"9fans.net/go/acme"
)

// [go install .]

type handler struct {
	w    *acme.Win
	path string
	buf  bytes.Buffer
}

var (
	branchTemplate string
	debugLogs      bool
)

func debugf(format string, args ...interface{}) {
	if debugLogs {
		log.Printf(format, args...)
	}
}

func tsbranch() string {
	return time.Now().Format(branchTemplate)
}

func (h *handler) flush() {
	h.w.Clear()
	h.w.Write("body", h.buf.Bytes())
	h.w.Ctl("clean")
	h.buf = bytes.Buffer{}
}

func (h *handler) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = h.path
	cmd.Stdout = &h.buf
	cmd.Stderr = &h.buf
	debugf("running: %v", cmd)
	err := cmd.Run()
	if err != nil {
		debugf("git error for %v: %v", cmd, err)
	}
	return err
}

func (h *handler) Look(arg string) bool {
	return false
}

func (h *handler) Execute(cmd string) bool {
	return false
}

func readLog(h *handler, l *acme.LogReader) {
	pfx := filepath.Clean(h.path + "/")
	for {
		event, err := l.Read()
		if err != nil {
			log.Fatal(err)
		}
		// update the git status output when a file in the repo is put/written by acme
		if event.Name != "" && event.Op == "put" && strings.HasPrefix(event.Name, pfx) && event.Name != h.path {
			debugf("readLog handling %v\n", event)
			// no current way to send an event on the internal channel that EventLoop()
			// uses to dispatch events, so call our Get method directly here :/
			h.ExecGet("")
		}
	}
}

func main() {
	log.SetPrefix("gitwin ")
	repoPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	brPfx := os.Getenv("USER")
	if brPfx == "" {
		brPfx = "branch"
	}
	flag.BoolVar(&debugLogs, "debug", false, "true to enable debug logging")
	flag.StringVar(&branchTemplate, "branchTemplate", brPfx+"-200601021504", "template for default branch names, populated with time.Format")
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		repoPath = args[0]
	}
	if err := os.Chdir(repoPath); err != nil {
		log.Fatal("error doing chdir to repo:", err)
	}
	w, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	l, err := acme.Log()
	if err != nil {
		log.Fatal(err)
	}
	w.Name(repoPath + "/+git")
	w.Write("tag", []byte("Get Diff Fetch Pull Branches Push Ls Log Help"))
	h := handler{path: repoPath, w: w}
	h.ExecGet("")
	go readLog(&h, l)
	w.EventLoop(&h)
}
