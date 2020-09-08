// hacky git porcelain parsing stuff
// see https://git-scm.com/docs/git-status#_porcelain_format_version_2
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type changedEntry struct {
	xy           string
	sub          string
	modeHead     string // octal
	modeIndex    string // octal
	modeWorktree string // octal
	objNameHead  string
	objNameIndex string
	path         string
}

func (e changedEntry) hasChangesToStage() bool {
	// not sure how correct this is :shrug:
	return e.xy[0] == '.' || e.xy[1] == 'M'
}

type renameCopiedEntry struct {
	xy              string
	sub             string
	modeHead        string // octal
	modeIndex       string // octal
	modeWorktree    string // octal
	objNameHead     string
	objNameIndex    string
	renameCopyScore string
	path            string
	origPath        string
}

type unmergedEntry struct {
	xy            string
	sub           string
	modeStage1    string // octal
	modeStage2    string // octal
	modeStage3    string // octal
	modeWorktree  string // octal
	objNameStage1 string
	objNameStage2 string
	objNameStage3 string
	path          string
}

type gitStatus struct {
	branch    string
	upstream  string
	changed   []changedEntry
	renamed   []renameCopiedEntry
	unmerged  []unmergedEntry
	untracked []string
}

func formatStatus(w io.Writer, s *gitStatus) {
	unstaged := []string{}
	staged := []string{}
	for _, x := range s.changed {
		if x.hasChangesToStage() {
			unstaged = append(unstaged, x.path)
		} else {
			staged = append(staged, x.path)
		}
	}
	if len(unstaged) > 0 {
		fmt.Fprint(w, "UNSTAGED\n")
		for _, x := range unstaged {
			fmt.Fprintf(w, "\tAdd %s\n", x)
		}
	}
	if len(staged) > 0 {
		fmt.Fprint(w, "STAGED\n")
		for _, x := range staged {
			fmt.Fprintf(w, "\tUnstage %s\n", x)
		}
	}
	if len(s.untracked) > 0 {
		fmt.Fprint(w, "UNTRACKED\n")
		for _, x := range s.untracked {
			fmt.Fprintf(w, "\tAdd %s\n", x)
		}
	}
}

type porcelainLineParser func(line string, s *gitStatus)

type xysub struct {
	xy   string
	sub  string
	rest []string
}

func parseXy(line string) xysub {
	return xysub{
		xy:   line[2:4],
		sub:  line[5:9],
		rest: strings.Fields(line[9:len(line)]),
	}
}

func parsePorcelainHeader(line string, s *gitStatus) {
	if line[0] == '#' {
		f := strings.Fields(line)
		switch f[1] {
		case "branch.head":
			s.branch = f[2]
		case "branch.upstream":
			s.upstream = f[2]
		}
	}
}
func parsePorcelainChanged(line string, s *gitStatus) {
	if line[0] == '1' {
		v := parseXy(line)
		s.changed = append(s.changed, changedEntry{
			xy:           v.xy,
			sub:          v.sub,
			modeHead:     v.rest[0],
			modeIndex:    v.rest[1],
			modeWorktree: v.rest[2],
			objNameHead:  v.rest[3],
			objNameIndex: v.rest[4],
			path:         v.rest[5],
		})
	}
}
func parsePorcelainRenamed(line string, s *gitStatus) {
	if line[0] == '2' {
		v := parseXy(line)
		s.renamed = append(s.renamed, renameCopiedEntry{
			xy:              v.xy,
			sub:             v.sub,
			modeHead:        v.rest[0],
			modeIndex:       v.rest[1],
			modeWorktree:    v.rest[2],
			objNameHead:     v.rest[3],
			objNameIndex:    v.rest[4],
			renameCopyScore: v.rest[5],
			path:            v.rest[6],
			origPath:        v.rest[7],
		})

	}
}
func parsePorcelainUnmerged(line string, s *gitStatus) {
	if line[0] == 'u' {
		v := parseXy(line)
		s.unmerged = append(s.unmerged, unmergedEntry{
			xy:            v.xy,
			sub:           v.sub,
			modeStage1:    v.rest[0],
			modeStage2:    v.rest[1],
			modeStage3:    v.rest[2],
			modeWorktree:  v.rest[3],
			objNameStage1: v.rest[4],
			objNameStage2: v.rest[5],
			objNameStage3: v.rest[6],
			path:          v.rest[7],
		})
	}
}
func parsePorcelainUntracked(line string, s *gitStatus) {
	if line[0] == '?' {
		s.untracked = append(s.untracked, line[2:len(line)])
	}
}

func (h *handler) gitPorcelain() (*gitStatus, error) {
	cmd := exec.Command("git", "status", "--branch", "--porcelain=v2", "-uall")
	cmd.Dir = h.path
	var pb bytes.Buffer
	cmd.Stdout = &pb
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(&pb)
	funcs := []porcelainLineParser{
		parsePorcelainHeader,
		parsePorcelainChanged,
		parsePorcelainRenamed,
		parsePorcelainUnmerged,
		parsePorcelainUntracked,
	}
	status := &gitStatus{}
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 1 {
			continue
		}
		for _, pf := range funcs {
			pf(line, status)
		}
	}
	return status, scanner.Err()
}