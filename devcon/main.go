// parse a devcontainer.json file and launch docker with necessary things
package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type args []string

func (a *args) Add(b args) {
	*a = append(*a, b...)
}
func (a *args) AddString(s string) {
	a.Add(args{s})
}
func (a args) String() string {
	return strings.Join(a, " ")
}

func formatPorts(v any) []string {
	switch v.(type) {
	case string:
	case int:
		return []string{fmt.Sprintf("-p %v:%v", v, v)}
	case []string:
		v := v.([]string)
		var buf []string
		for _, p := range v {
			buf = append(buf, fmt.Sprintf("-p %v:%v", p, p))
		}
		return buf
	case []int:
		v := v.([]int)
		var buf []string
		for _, p := range v {
			buf = append(buf, fmt.Sprintf("-p %v:%v", p, p))
		}
		return buf
	}
	return nil
}

func formatObject(flag string, o object) []string {
	var buf []string
	for k, v := range o {
		buf = append(buf, fmt.Sprintf("%s %s=%v", flag, k, v))
	}
	return buf
}

func formatMount(src, dst string) []string {
	return []string{"--mount", fmt.Sprintf("type=bind,source=%s,target=%s", src, dst)}
}

func getWd() string {
	if d, err := os.Getwd(); err == nil {
		return d
	}
	return ""
}

func execCommand(dockerCmd, containerName string, cmd []string) {
	execArgs := args{"exec", "-it", containerName, "sh", "-c"} // TODO: use shell defined in devcontainer.json instead
	execArgs.AddString(strings.Join(cmd, " "))

	log.Println("running:", execArgs)
	runCmd := exec.Command(dockerCmd, execArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stdin = os.Stdin
	runCmd.Stderr = os.Stderr
	runCmd.Run() // hold the container open until the command exits
}

func parseConfig(path string) (*cfgType, error) {
	log.Println("reading file", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// strip out comments, then decode
	var bb bytes.Buffer
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		beforeComment, _, _ := bytes.Cut(line, []byte("//"))
		bb.Write(beforeComment)
	}
	decoder := json.NewDecoder(&bb)
	var cfg cfgType
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getNamePath(tmpDir, containerDir string) string {
	if d, err := filepath.Abs(containerDir); err == nil {
		containerDir = d
	}
	return filepath.Join(tmpDir, fmt.Sprintf("devcon.%x", md5.Sum([]byte(containerDir))))
}

func getName(namePath string) (string, bool) {
	log.Println("checking for container name in", namePath)
	if b, err := os.ReadFile(namePath); err == nil {
		s := strings.TrimSpace(string(b))
		if len(s) > 0 {
			return s, true
		}
	}
	return "", false
}

func main() {
	tmp := flag.String("tmp", "/tmp", "temp directory for storing running container tags")
	containerDir := flag.String("d", ".devcontainer", "directory with devcontainer.json and Dockerfile")
	wd := flag.String("l", getWd(), "local workspace to map into container")
	ws := flag.String("w", "/workspace", "Working directory inside the container")
	cmd := flag.String("docker", "docker", "name of docker command")
	flag.Parse()

	namePath := getNamePath(*tmp, *containerDir)
	if containerName, ok := getName(namePath); ok {
		execCommand(*cmd, containerName, flag.Args())
		return
	}

	log.Println("no container name found in", namePath, "- starting container instead...")
	cfgFile := filepath.Join(*containerDir, "devcontainer.json")
	dockerFile := filepath.Join(*containerDir, "Dockerfile")
	cfg, err := parseConfig(cfgFile)
	if err != nil {
		log.Fatal("error parsing config file", err)
	}

	ts := time.Now().Unix()
	containerName := fmt.Sprintf("localdevcon_%s_%d", strings.ToLower(cfg.Name), ts)
	buildTag := fmt.Sprintf("localdevcon-%s:%d", strings.ToLower(cfg.Name), ts)
	buildArgs := args{"build", "-t", buildTag, "-f", dockerFile}
	buildArgs.Add(formatObject("--build-arg", cfg.Build.Args))
	buildArgs.Add(args{*containerDir})
	log.Println("running:", buildArgs)
	if err := exec.Command(*cmd, buildArgs...).Run(); err != nil {
		log.Fatal("error building container:", err)
	}
	shell := "sh" // TODO get from .settings.terminal.integrated.shell.linux ?
	runArgs := args{"run", "--rm", "-it", "-w", *ws, "--name", containerName}
	runArgs.Add(formatMount(*wd, *ws))
	runArgs.Add(formatPorts(cfg.AppPort))
	runArgs.Add(formatPorts(cfg.ForwardPorts))
	runArgs.Add(formatObject("-e", cfg.RemoteEnv))
	runArgs.Add(args{buildTag})
	runArgs.Add(args{shell})

	if err := os.WriteFile(namePath, []byte(containerName+"\n"), 0644); err != nil {
		log.Fatal("error writing to tag file", namePath, err)
	}
	defer os.Remove(namePath)
	log.Println("running:", runArgs)
	runCmd := exec.Command(*cmd, runArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stdin = os.Stdin
	runCmd.Stderr = os.Stderr
	runCmd.Run() // hold the container open until the shell exits?
}
