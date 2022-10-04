// parse a devcontainer.json file and launch docker with necessary things
package main

import (
	"bufio"
	"bytes"
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

func (a args) Add(b args) {
	a = append(a, b...)
}
func (a args) String() string {
	return strings.Join(a, " ")
}

type object map[string]any
type stringOrArray any
type stringOrArrayOrInt any

// refer to https://containers.dev/implementors/json_reference/
type cfgType struct {
	Image string `json:"image"`
	Build struct {
		Dockerfile string        `json:"dockerfile"`
		Context    string        `json:"context"`
		Args       object        `json:"args"`
		Target     string        `json:"target"`
		CacheFrom  stringOrArray `json:"cacheFrom"`
	} `json:"build"`
	Settings        object             `json:"settings"`
	AppPort         stringOrArrayOrInt `json:"appPort"`
	ContainerEnv    object             `json:"containerEnv"`
	ContainerUser   string             `json:"containerUser"`
	Mounts          []string           `json:"mounts"`
	WorkspaceMount  string             `json:"workspaceMount"`
	WorkspaceFolder string             `json:"workspaceFolder"`
	RunArgs         []string           `json:"runArgs"`

	DockerComposeFile stringOrArray `json:"dockerComposeFile"`
	Service           string        `json:"service"`
	RunServices       []string      `json:"runServices"`

	Name                 string        `json:"name"`
	ForwardPorts         stringOrArray `json:"forwardPorts"`
	PortsAttributes      object        `json:"portsAttributes"`
	OtherPortsAttributes object        `json:"otherPortsAttributes"`
	RemoteEnv            object        `json:"remoteEnv"`
	RemoteUser           string        `json:"remoteUser"`
	UpdateRemoteUserUID  bool          `json:"updateRemoteUserUID"`
	UserEnvProbe         string        `json:"userEnvProbe"`
	OverrideCommand      bool          `json:"overrideCommand"`
	Features             object        `json:"features"`
	ShutdownAction       string        `json:"shutdownAction"`
	Customizations       object        `json:"customizations"`
	InitializeCommand    stringOrArray `json:"initializeCommand"`
	OnCreateCommand      stringOrArray `json:"onCreateCommand"`
	UpdateContentCommand stringOrArray `json:"updateContentCommand"`
	PostCreateCommand    stringOrArray `json:"postCreateCommand"`
	PostStartCommand     stringOrArray `json:"postStartCommand"`
	PostAttachCommand    stringOrArray `json:"postAttachCommand"`
	WaitFor              string        `json:"waitFor"`
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

func main() {
	// TODO: add local socket here that can listen for commands?
	dir := flag.String("dir", ".devcontainer", "directory with devcontainer.json and Dockerfile")
	wd := flag.String("l", getWd(), "local workspace to map into container")
	ws := flag.String("w", "/workspace", "Working directory inside the container")
	cmd := flag.String("cmd", "docker", "name of docker command")
	flag.Parse()
	containerFile := filepath.Join(*dir, "devcontainer.json")
	dockerFile := filepath.Join(*dir, "Dockerfile")
	log.Println("reading file", containerFile)
	f, err := os.Open(containerFile)
	if err != nil {
		log.Fatal("error opening file", containerFile, err)
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
		log.Fatal("error decoding file", err)
	}
	log.Println("config:", cfg)
	buildTag := fmt.Sprintf("localdevcon-%s:%d", strings.ToLower(cfg.Name), time.Now().Unix())
	buildArgs := args{"build", "-t", buildTag, "-f", dockerFile}
	buildArgs.Add(formatObject("--build-arg", cfg.Build.Args))
	buildArgs.Add(args{*dir})
	log.Println("running:", buildArgs)
	if err := exec.Command(*cmd, buildArgs...).Run(); err != nil {
		log.Fatal("error building container:", err)
	}
	shell := "sh" // TODO get from .settings.terminal.integrated.shell.linux ?
	runArgs := args{"run", "--rm", "-it", "-w", *ws}
	runArgs.Add(formatMount(*wd, *ws))
	runArgs.Add(formatPorts(cfg.AppPort))
	runArgs.Add(formatPorts(cfg.ForwardPorts))
	runArgs.Add(formatObject("-e", cfg.RemoteEnv))
	runArgs.Add(args{buildTag})
	runArgs.Add(args{shell})

	log.Println("running:", runArgs)
	runCmd := exec.Command(*cmd, runArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stdin = os.Stdin
	runCmd.Stderr = os.Stderr
	runCmd.Run()
}
