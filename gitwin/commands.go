package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"9fans.net/go/acme"
)

func (h *handler) getMainName() string {
	h.git("branch", "-l")
	branchWords := strings.Fields(h.buf.String())
	h.buf = bytes.Buffer{}
	debugf("branch output: %v", branchWords)
	for _, w := range branchWords {
		switch w {
		case "main":
			return "main"
		case "master":
			return "master"
		}
	}
	return ""
}

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

type wordFilterFunc func(string) bool

func regularFile(p string) bool {
	debugf("regularFile checking %s\n", p)
	if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() {
		return true
	}
	return false
}

func filter(words []string, filt wordFilterFunc) []string {
	ret := []string{}
	for _, w := range words {
		if filt(w) {
			ret = append(ret, w)
		}
	}
	return ret
}

func (h *handler) repoWindows(winCmd string) {
	allWindows, _ := acme.Windows()
	for _, w := range allWindows {
		if strings.HasPrefix(w.Name, h.path) && regularFile(w.Name) {
			if win, err := acme.Open(w.ID, nil); win != nil && err == nil {
				// see acme(4) for ctl commands here
				debugf("doing '%s' on window %d: %s", winCmd, w.ID, w.Name)
				win.Ctl(winCmd)
			}
		}
	}
}

func (h *handler) ExecDelWindows() {
	h.repoWindows("del")
}

func (h *handler) ExecRevert(cmd string) {
	debugf("doing ExecRevert [%s]\n", cmd)
	args := []string{"checkout", "--"}
	files := filter(strings.Fields(cmd), func(w string) bool { return w != "Revert" })
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
	files := filter(strings.Fields(cmd), func(w string) bool { return w != "Add" })
	args := []string{"add"}
	args = append(args, files...)
	if h.git(args...) != nil {
		h.flush()
	} else {
		h.ExecGet("")
	}
}

func (h *handler) ExecUnstage(cmd string) {
	files := filter(strings.Fields(cmd), func(w string) bool { return w != "Unstage" })
	args := []string{"restore", "--staged"}
	args = append(args, files...)
	if h.git(args...) != nil {
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
	if cmd != "master" && cmd != "main" {
		args = append(args, "-B")
	}
	args = append(args, cmd)
	if h.git(args...) != nil {
		h.flush()
	} else {
		h.ExecGet("")
	}
	h.repoWindows("get")
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

func (h *handler) ExecDiff(cmd string) {
	h.git("diff")
	h.flush()
}

func (h *handler) ExecDifftool(cmd string) {
	h.git("difftool", "-y")
	h.ExecGet("")
}

func (h *handler) ExecMergetool(cmd string) {
	h.git("mergetool", "-y")
	h.ExecGet("")
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
	h.repoWindows("get")
	h.flush()
}

func (h *handler) ExecRebase(cmd string) {
	h.git("pull", "--rebase")
	h.repoWindows("get")
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

func (h *handler) ExecGetWindows(cmd string) {
	h.repoWindows("get")
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
	coName := h.getMainName()
	if status.branch == "master" || status.branch == "main" {
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
