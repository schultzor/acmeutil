// linenum attempts to get the current selected line number for an open acme window
// use: linenum acme_win_id
//

package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"9fans.net/go/acme"
)

func main() {
	log.SetPrefix("linenum")
	displayPath := flag.Bool("path", false, "if set, print file /file/path:linenum instead of just line number")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("expected window ID argument")
	}
	winId, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	w, err := acme.Open(winId, nil)
	if err != nil {
		log.Fatal(err)
	}
	if _, _, err := w.ReadAddr(); err != nil {
		log.Fatal(err)
	}
	if err := w.Ctl("addr=dot"); err != nil {
		log.Fatal(err)
	}
	q0, _, _ := w.ReadAddr()
	contents, err := w.ReadAll("body")
	if err != nil {
		log.Fatal(err)
	}
	count := 1
	for i := 0; i < len(contents) && i < q0; i++ {
		if contents[i] == '\n' {
			count++
		}
	}
	if *displayPath {
		tb, _ := w.ReadAll("tag")
		f := strings.Fields(string(tb))
		fmt.Printf("%s:%d", f[0], count)
	} else {
		fmt.Println(count)
	}
}
