package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"9fans.net/go/acme"
)

func (h *handler) ExecTrackOrigin(cmd string) {
	status, err := h.gitPorcelain()
	if err != nil {
		log.Fatalf("error getting status: %v", err)
	}
	switch status.branch {
	case "master":
	case "main":
	default:
		h.git("branch", "--set-upstream-to=origin/"+status.branch, status.branch)
	}
	h.flush()
}

func (h *handler) ExecRemote(cmd string) {
	h.git("remote", "-v")
	h.flush()
}

func (h *handler) ExecRevert(cmd string) {
	debugf("doing ExecRevert [%s]\n", cmd)
	args := []string{"checkout", "--"}
	files := strings.Fields(cmd)
	args = append(args, files...)
	h.git(args...)
	allWindows, _ := acme.Windows()
	for _, filename := range files {
		for _, w := range allWindows {
			if strings.HasPrefix(w.Name, h.path) && strings.HasSuffix(w.Name, filename) {
				if win, err := acme.Open(w.ID, nil); win != nil && err == nil {
					// see acme(4) for ctl commands here
					win.Ctl("clean")
					win.Ctl("get")
					fmt.Fprintln(&h.buf, "reverted", w.Name)
				}
			}
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

func (h *handler) ExecUnstage(cmd string) {
	if h.git("restore", "--staged", cmd) != nil {
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
	h.git("difftool", "-y")
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
	status, err := h.gitPorcelain()
	if err != nil {
		log.Fatalf("error getting status: %v", err)
	}
	remote := status.branch
	if status.branch == "master" || status.branch == "main" {
		remote = tsbranch()
		fmt.Fprintf(&h.buf, "pushing to remote branch %s instead of main/master\n", remote)
	}
	h.git("push", "origin", status.branch+":"+remote)
	h.flush()
}

func (h *handler) ExecStatus(cmd string) {
	h.git("status")
	h.flush()
}
func (h *handler) ExecGet(cmd string) {
	debugf("doing ExecGet [%s]\n", cmd)
	status, err := h.gitPorcelain()
	if err != nil {
		log.Fatalf("error getting status: %v", err)
	}
	debugf("status: %v", status)
	coName := "master"
	if status.branch == "master" {
		coName = tsbranch()
	}
	fmt.Fprintf(&h.buf, "on %s tracking %s\nCheckout %s\nCommit fix: something\n", status.branch, status.upstream, coName)
	formatStatus(&h.buf, status)
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
