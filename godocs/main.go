// godocs runs the 'go doc' command for a given argument, writing the output to a new acme window.
//
// Usage:
//
// 	godocs [pkg]
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
	"path/filepath"
	"strings"

	"9fans.net/go/acme"
	"golang.org/x/mod/module"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("exiting on error: %v - %s", err, msg)
	}
}

// state and handlers for each /godocs/ window
type handler struct {
	win    *acme.Win
	pkg    []string
	dotpkg string // replace "." in pkg name with this for display
	sym    []string
	useAll bool
	useSrc bool
}

func newhandler() *handler {
	w, err := acme.New()
	check(err, "error creating acme window")
	h := &handler{win: w}
	w.Name("/godocs/")
	w.Write("tag", []byte("Info Get All Src Pkg packageName Look lookFor"))
	return h
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("godocs: ")
	flag.Parse()
	args := flag.Args()

	parent := newhandler()
	if len(args) == 0 {
		parent.run("go", "list", "all")
	} else {
		parent.chdir(args[0])
		parent.pkg = []string{args[0]}
		// try doing `go doc pkgName` first, if that fails try `go doc .` in the module dir instead
		if err := parent.godoc(); err != nil {
			parent.pkg = []string{"."}
			parent.dotpkg = args[0]
			parent.godoc()
		}
	}
	parent.listen()
}

// for parsing output of `go list -m`
type modInfo struct {
	Path     string
	Version  string
	Info     string
	GoMod    string
	Zip      string
	Dir      string
	Sum      string
	GoModSum string
}

// try to chdir to the appropriate module directory for a given package path
func (h *handler) chdir(pkg string) bool {
	// see if a package belongs to a module we need to fetch
	// assume if we get an error here that it's a builtin package
	if err := module.CheckPath(pkg); err != nil {
		return false
	}
	// create a fake module in a temp dir
	tmpDir, err := os.MkdirTemp("", "godocs")
	check(err, "making temp dir")

	defer os.RemoveAll(tmpDir)
	os.Chdir(tmpDir)

	_, err = h.run("go", "mod", "init", "example/godocs")
	check(err, "go mod init")

	// do a `go get` to resolve the package to a module name in our fake module
	_, err = h.run("go", "get", "-d", pkg)
	check(err, "go get -d")

	// do a `go list -m -u -json all` to see what modules we grabbed
	output, err := h.run("go", "list", "-m", "-u", "-json", "all")
	check(err, "go list -m")

	// decode the json output from the above command
	decoder := json.NewDecoder(bytes.NewBuffer(output))
	var target modInfo
	for decoder.More() {
		var current modInfo
		check(decoder.Decode(&current), "decoding go list -m output")
		// find the longest module prefix that matches our package name
		if strings.HasPrefix(pkg, current.Path) && len(current.Path) > len(target.Path) {
			target = current
		}
	}
	os.Chdir(target.Dir)
	return true
}

func (h *handler) listen() {
	h.win.EventLoop(h)
}

func (h *handler) Look(arg string) bool {
	if arg[0] == ':' { // allow for selecting buffer, e.g. ':,'
		return false
	}
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return false
	}
	// launch a child window for the symbol/package name that was looked for
	go func() {
		child := newhandler()
		child.dotpkg = h.dotpkg
		if len(h.pkg) == 0 { // case where parent window did a "go list ..."
			child.pkg = []string{arg}
		} else {
			child.pkg = h.pkg
			child.sym = append(h.sym, arg)
		}
		child.godoc()
		child.listen()
	}()
	return true
}

func (h *handler) ExecInfo(arg string) {
	h.win.Clear()
	cwd, _ := os.Getwd()
	h.win.Write("body", []byte(fmt.Sprintf("current package %s in %s", h.pkg, cwd)))
	h.win.Ctl("clean")
	// TODO: see if there's a way to list child packages in mod dir?
}

// launch a new window for a child package
func (h *handler) ExecPkg(arg string) {
	if arg != "" {
		// launch a child for the subpackage
		go func() {
			child := newhandler()
			child.dotpkg = h.dotpkg
			child.pkg = append(h.pkg, arg)
			child.godoc()
			child.listen()
		}()
	}
}

func (h *handler) run(prog string, args ...string) ([]byte, error) {
	h.win.Clear()
	cmd := exec.Command(prog, args...)
	h.win.Write("body", []byte(fmt.Sprintf("running '%v' \n", cmd)))
	//log.Println("running", cmd)
	out, err := cmd.Output()
	if err != nil {
		wd, _ := os.Getwd()
		h.win.Write("body", []byte(fmt.Sprintf("error running [%v] in %s: %v\n", cmd, wd, err)))
	} else {
		h.win.Clear()
		h.win.Write("body", out)
	}
	h.win.Ctl("clean")
	return out, err
}

// call `go doc` with appopriate flags for the window's state
func (h *handler) godoc() error {
	args := []string{"doc"}
	wn := "/godocs/"
	if h.useAll {
		args = append(args, "-all")
	}
	if h.useSrc {
		args = append(args, "-src")
	}
	if len(h.pkg) > 0 {
		dpn := make([]string, len(h.pkg))
		copy(dpn, h.pkg)
		if h.dotpkg != "" {
			dpn[0] = h.dotpkg
		}
		wn += filepath.Join(dpn...)
		args = append(args, strings.Join(h.pkg, "/"))
	}
	if len(h.sym) > 0 {
		s := strings.Join(h.sym, ".")
		wn += "." + s
		args = append(args, s)
	}
	h.win.Name(wn)
	_, err := h.run("go", args...)
	return err
}

func (h *handler) ExecGet(cmd string) {
	h.godoc()
}

func (h *handler) Execute(cmd string) bool {
	return false
}

// toggle the `-all` flag
func (h *handler) ExecAll(cmd string) {
	h.useAll = !h.useAll
	h.godoc()
}

// toggle the `-src` flag
func (h *handler) ExecSrc(cmd string) {
	h.useSrc = !h.useSrc
	h.godoc()
}
