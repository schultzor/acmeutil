// parse a devcontainer.json file and launch docker with necessary things
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// using https://containers.dev/implementors/json_reference/ to guide this
type devcontainerConfig struct {
	Image string `json:"image"`
	Build struct {
		Dockerfile string         `json:"dockerfile"`
		Context    string         `json:"context"`
		Args       map[string]any `json:"args"`
		Target     string         `json:"target"`
		CacheFrom  string         `json:"cacheFrom"` // or array
	} `json:"build"`
	AppPort         int            `json:"appPort"` // or string or array
	ContainerEnv    map[string]any `json:"containerEnv"`
	ContainerUser   string         `json:"containerUser"`
	Mounts          []string       `json:"mounts"`
	WorkspaceMount  string         `json:"workspaceMount"`
	WorkspaceFolder string         `json:"workspaceFolder"`
	RunArgs         []string       `json:"runArgs"`
	Name            string         `json:"name"`
	ForwardPorts    []string       `json:"forwardPorts"`
}

func main() {
	fn := flag.String("file", ".devcontainer/devcontainer.json", "json devcontainer file")
	flag.Parse()

	f, err := os.Open(*fn)
	if err != nil {
		log.Fatal("error opening file", *fn, err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	var cfg devcontainerConfig
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatal("error decoding file", err)
	}
	log.Println("config:", devcontainerConfig)

}
