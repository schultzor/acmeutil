package main

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
