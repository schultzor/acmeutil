// try to bring some order to the chaos that is my acme workspace
// delete all clean directory windows

package main

import (
	"log"
	"strings"

	"os"

	"9fans.net/go/acme"
)

func shouldDelete(w acme.WinInfo) bool {
	log.Printf("checking window '%s'", w.Name)
	if w.Name == "Del" {
		return true
	}
	if strings.HasPrefix(w.Name, "/godocs/") {
		return true
	}
	if strings.HasSuffix(w.Name, "+Errors") && w.Name != "+Errors" {
		return true
	}
	if strings.HasSuffix(w.Name, "+pg") {
		return true
	}
	if strings.HasSuffix(w.Name, "+ff") {
		return true
	}
	stat, err := os.Stat(w.Name)
	if err != nil {
		return false
	}
	if stat.IsDir() {
		return true
	}
	return false
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("cleanup: ")
	allWindows, err := acme.Windows()
	if err != nil {
		log.Fatal("error getting windows", err)
	}
	for _, wi := range allWindows {
		if shouldDelete(wi) {
			wp, err := acme.Open(wi.ID, nil)
			if err != nil {
				log.Println("error opening window", wi, err)
			}
			if err := wp.Del(false); err != nil {
				log.Println("error deleting window", wi, err)
			}
		}
	}

}
