// godocs runs the 'go doc' command for a given argument, writing the output to a new acme window.
//
// Usage:
//
// 	godocs [search_string]
//
// Button 3-clicking in a godocs window will spawn a child window for the symbol name that's searched for,
// so you can "drill down" for docs on particular functions or types within a single go package.
// Calling this with no arguments will run `go list ...` to list available go packages, which make take some time.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"9fans.net/go/acme"
	"golang.org/x/mod/module"
)

// state and handlers for each /godocs/ window
type handler struct {
	path   []string
	useAll bool
	useSrc bool
	win    *acme.Win
	cwd    string
}

// for parsing output of `go list -m`
type modOutput struct {
	Path     string
	Version  string
	Info     string
	GoMod    string
	Zip      string
	Dir      string
	Sum      string
	GoModSum string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("godocs: ")
	flag.Parse()
	args := flag.Args()
	var docPath []string
	var modDir string
	if len(args) > 0 {
		d, err := moduleDir(args[0])
		if err != nil {
			log.Fatalf("error looking up module: %v", err)
		}
		modDir = d
		docPath = []string{args[0]}
	}
	newwin(docPath, modDir)
}

func run(cwd string, name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("error running [%v] in [%s]: %v", cmd, cwd, err)
	}
	return out, err
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("exiting on error: %v - %s", err, msg)
	}
}

// attempt to get module information for a go package path, this is imperfect
func moduleDir(pkgName string) (string, error) {
	// see if a package belongs to a module we need to fetch
	// assume if we get an error here that it's a built-in module
	if err := module.CheckPath(pkgName); err != nil {
		return "", nil
	}
	// create a fake module in a temp dir
	tmpDir, err := os.MkdirTemp("", "godocs")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	_, err = run(tmpDir, "go", "mod", "init", "godocs.foo/bar")
	if err != nil {
		return "", err
	}
	// do a `go get` to resolve the package to a module name in our fake module
	_, err = run(tmpDir, "go", "get", "-d", pkgName)
	if err != nil {
		return "", err
	}
	// do a `go list -m -u -json all` to see what modules we grabbed
	output, err := run(tmpDir, "go", "list", "-m", "-u", "-json", "all")
	if err != nil {
		return "", err
	}

	// decode the json output from the above command
	decoder := json.NewDecoder(bytes.NewBuffer(output))
	var target modOutput
	for decoder.More() {
		var current modOutput
		if err := decoder.Decode(&current); err != nil {
			log.Fatalf("error decoding module output: %v", err)
		}
		// find the longest module prefix that matches our package name
		if strings.HasPrefix(pkgName, current.Path) && len(current.Path) > len(target.Path) {
			target = current
		}
	}
	return target.Dir, nil
}

// make a new window, searching the package and symbol in names
func newwin(docPath []string, pkgCwd string) {
	w, err := acme.New()
	check(err, "error creating window")
	h := &handler{
		path: docPath,
		win:  w,
		cwd:  pkgCwd,
	}
	if len(h.path) < 1 {
		h.win.Name("/godocs/+list")
		h.win.Write("body", []byte("doing 'go list ...' to show available standard packages"))
		h.runcmd("go", []string{"list", "..."})
	} else {
		h.win.Name("/godocs/" + strings.Join(h.path, "."))
		h.win.Write("tag", []byte("All Src"))
		h.godoc()
	}
	w.EventLoop(h)
}

// write the output from a command to the window for a handler instance
func (h *handler) runcmd(prog string, args []string) {
	out, err := run(h.cwd, prog, args...)
	h.win.Clear()
	if err != nil {
		h.win.Write("body", []byte(fmt.Sprint(err)))
	} else {
		h.win.Write("body", out)
	}
	h.win.Ctl("clean")
}

// call `go doc` with appopriate flags for the window's state
func (h *handler) godoc() {
	args := []string{"doc"}
	if h.useAll {
		args = append(args, "-all")
	}
	if h.useSrc {
		args = append(args, "-src")
	}
	args = append(args, strings.Join(h.path, "."))
	h.runcmd("go", args)
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

func (h *handler) Look(arg string) bool {
	go newwin(append(h.path, arg), h.cwd)
	return true
}
