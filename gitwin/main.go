// gitwin is a hacky way to interact with git via an acme window
//
// Usage:
//
//	gitwin -path /path/to/gitrepo/root
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
	"strings"
	"time"

	"9fans.net/go/acme"
)

// [go install .]

func debugf(format string, args ...interface{}) {
	if _, debug := os.LookupEnv("DEBUG_WIN"); debug {
		log.Printf(format, args...)
	}
}

type handler struct {
	w    *acme.Win
	path string
	buf  bytes.Buffer
}

var (
	branchTemplate string
)

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
	for {
		event, err := l.Read()
		if err != nil {
			log.Fatal(err)
		}
		if event.Name != "" && event.Op == "put" && strings.HasPrefix(event.Name, h.path) && event.Name != h.path {
			debugf("readLog handling %v\n", event)
			// TODO: should send on the channel being listened to by event loop below
			h.ExecGet("")
		}
	}
}

func main() {
	log.SetPrefix("gitwin ")
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	branchPrefix := os.Getenv("USER")
	if branchPrefix == "" {
		branchPrefix = "branch"
	}
	repoPath := flag.String("path", pwd, "path to repo root dir")
	flag.StringVar(&branchTemplate, "branchTemplate", os.Getenv("USER")+"-200601021504", "time.Format-compatible template for generating branch names")
	flag.Parse()
	if err := os.Chdir(*repoPath); err != nil {
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
	w.Name(*repoPath + "/+git")
	w.Write("tag", []byte("Get Diff Pull Rebase Push Ls Log Help"))
	h := handler{path: *repoPath, w: w}
	h.ExecGet("")
	go readLog(&h, l)
	w.EventLoop(&h)
}
