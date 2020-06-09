package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"9fans.net/go/acme"
)

// [go install .]

func debugf(format string, args ...interface{}) {
	if _, debug := os.LookupEnv("DEBUG_WIN"); debug {
		fmt.Printf(format, args...)
	}
}

type handler struct {
	w    *acme.Win
	path string
	buf  bytes.Buffer
}

func tsbranch() string {
	now := time.Now()
	return fmt.Sprintf("patch-%4d%02d%02d-%02d%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
}

func (h *handler) branch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = h.path
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("error reading git branch: %v", err)
	}
	return strings.TrimSpace(string(out))
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
	return cmd.Run()
}

func (h *handler) ExecGet(cmd string) {
	debugf("doing ExecGet [%s]\n", cmd)
	currBranch := h.branch()
	coName := "master"
	if currBranch == "master" {
		coName = tsbranch()
	}
	fmt.Fprintf(&h.buf, "on %s\nCheckout %s\nCommit fix: something broken\n", currBranch, coName)
	h.git("status", "--porcelain", "-uall")
	h.flush()
}

func (h *handler) ExecHelp(cmd string) {
	debugf("doing ExecHelp [%s]\n", cmd)
	t := reflect.TypeOf(h)
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if strings.HasPrefix(m.Name, "Exec") && m.Name != "Execute" {
			fmt.Fprintf(&h.buf, "command %v\n", m.Name[4:])
		}
	}
	h.flush()
}

func (h *handler) ExecAdd(cmd string) {
	if h.git("add", cmd) != nil {
		h.flush()
	} else {
		h.ExecGet("")
	}
}

func (h *handler) ExecBranches(cmd string) {
	h.git("branch")
	h.flush()
}

func (h *handler) ExecLs(cmd string) {
	h.git("ls-files")
	h.flush()
}

func (h *handler) ExecCheckout(cmd string) {
	debugf("doing ExecCheckout [%s]\n", cmd)
	args := []string{"checkout"}
	if cmd != "master" {
		args = append(args, "-B")
	}
	args = append(args, cmd)
	if h.git(args...) != nil {
		h.flush()
	} else {
		h.ExecGet("")
	}
}

func (h *handler) ExecCommit(cmd string) {
	// check for an "all:" prefix on the commit message
	args := []string{"commit"}
	msg := cmd
	if strings.HasPrefix(cmd, "all:") {
		args = append(args, "-a")
		msg = cmd[4:]
	}
	args = append(args, "-m", msg)
	if h.git(args...) != nil {
		h.flush()
	} else {
		h.buf.WriteString("\n")
		h.ExecGet("")
	}
}

func (h *handler) ExecCommitAll(cmd string) {
	h.ExecCommit("all:" + cmd)
}

func (h *handler) ExecDiff(cmd string) {
	h.git("diff")
	h.flush()
}

func (h *handler) ExecDifftool(cmd string) {
	h.git("difftool")
	h.flush()
}

func (h *handler) ExecFetch(cmd string) {
	h.git("fetch")
	h.flush()
}

func (h *handler) ExecLog(cmd string) {
	if cmd == "" {
		cmd = "-10"
	}
	h.git("log", cmd)
	h.flush()
}

func (h *handler) ExecPull(cmd string) {
	h.git("pull")
	h.flush()
}

func (h *handler) ExecPush(cmd string) {
	local := h.branch()
	remote := local
	if local == "master" {
		remote = tsbranch()
		fmt.Fprintf(&h.buf, "pushing to remote branch %s instead of master\n", remote)
	}
	h.git("push", "origin", local+":"+remote)
	h.flush()
}

func (h *handler) ExecRemote(cmd string) {
	h.git("remote", "-v")
	h.flush()
}

func (h *handler) ExecReset(cmd string) {}

func (h *handler) ExecRevert(cmd string) {
	debugf("doing ExecRevert [%s]\n", cmd)
	// callgit("checkout", "--", argVal)
	// refresh any acme windows for the file
	// wins, _ := acme.Windows()
	// for _, w := range wins {
	// if strings.HasPrefix(w.Name, repoDir) &&
	// strings.HasSuffix(w.Name, argVal) {
	// if tgt, err := acme.Open(w.ID, nil); tgt != nil && err == nil {
	// tgt.Ctl("get")
	// tgt.CloseFiles()
	// }
	// }
	// }
}

func (h *handler) ExecTrackOrigin(cmd string) {
	local := h.branch()
	if local != "master" {
		h.git("branch", "--set-upstream-to=origin/"+local, local)
	}
	h.flush()
}

func (h *handler) Look(arg string) bool {
	//fmt.Printf("handling Look: %s\n", arg)
	return false
}

var addPrefixes = []string{
	"?? ",
	" M ",
	"M ", // acme EventHandler interface swallows leading space
	"AM ",
	"MM ",
	" D ",
}

func (h *handler) Execute(cmd string) bool {
	debugf("handling Execute: [%s]\n", cmd)
	if cmd == "Del" {
		return false
	}
	for _, p := range addPrefixes {
		if strings.HasPrefix(cmd, p) {
			debugf("doing ExecAdd for [%s]\n", cmd)
			argVal := strings.TrimSpace(cmd[len(p):])
			h.ExecAdd(argVal)
			return true
		}
	}
	return false
}

func readLog(h *handler, l *acme.LogReader) {
	for {
		event, err := l.Read()
		if err != nil {
			log.Fatal(err)
		}
		// file was put for somewhere in our repo,
		if event.Name != "" && event.Op == "put" && strings.HasPrefix(event.Name, h.path) && event.Name != h.path {
			debugf("readLog handling %v\n", event)
			h.ExecGet("")
		}
	}
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	repoPath := flag.String("path", pwd, "path to repo root dir")
	flag.Parse()
	w, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	l, err := acme.Log()
	if err != nil {
		log.Fatal(err)
	}
	w.Name(*repoPath + "/+git")
	w.Write("tag", []byte("Get Diff Pull Push Ls Log Help"))
	h := handler{path: *repoPath, w: w}
	h.ExecGet("")
	go readLog(&h, l)
	w.EventLoop(&h)
}