package maestro

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
)

var (
	config         MaestroConfig
	username       MaestroUser
	flagDebug      bool
	domain         string
	fleetAddress   string
	fleetEndpoints string
	fleetOptions   []string
	configFile     string
	maestroDir     string
	userFile       string
	volumesDir     string
)

// Setup directory ($CWD/.maestro) used to store user informations and
// temporary build file before submitting to coreos.
// Directory will be created if not existent.
func SetupMaestroDir(dir string) {
	if dir == "" {
		user, err := user.Current()
		lg.Fatal(err)
		dir = user.HomeDir
	}
	maestroDir = path.Join(dir, ".maestro")
	if _, err := os.Stat(maestroDir); err != nil {
		lg.Debug(maestroDir + " not found, creating")
		os.Mkdir(maestroDir, 0755)
	}
}

func Init(maestroDir, domainName, address, volumes, endpoints string, options []string, debug bool) {
	FleetCheckExec()
	EtcdCheckExec()
	fleetAddress = address
	flagDebug = debug
	if debug {
		lg.SetupDebug()
	}
	fleetOptions = options
	fleetEndpoints = endpoints
	domain = domainName
	volumesDir = volumes
	SetupMaestroDir(maestroDir)
}

// Public function used in the main to load the configuration.
func BuildMaestroConfig(cfg string) MaestroConfig {
	configFile = cfg
	config = config.LoadMaestroConfig(configFile)
	lg.SetupBase()
	config.SetupUsername()
	config.SetupMaestroAppDirs()
	config.SetMaestroComponentConfig()
	return config
}

type MaestroUser struct {
	Name string `json:"name"`
}

// MaestroComponent structure
type MaestroComponent struct {
	After         string   `json:"after"`
	App           string   `json:"app"`
	BuildUnitPath string   `json:"build_unitpath"`
	Cmd           string   `json:"cmd"`
	ContainerName string   `json:"container_name"`
	DNS           string   `json:"dns"`
	DockerArgs    string   `json:"docker_args"`
	Env           []string `json:"env"`
	Frontend      bool     `json:"frontend"`
	GitSrc        string   `json:"gitsrc"`
	Global        bool     `json:"global"`
	InternalDNS   string   `json:"internal_dns"`
	KeepOnExit    bool     `json:"keep_on_exit"`
	Name          string   `json:"name"`
	Ports         []int    `json:"ports"`
	Scale         int      `json:"scale"`
	Single        bool     `json:"single"`
	Src           string   `json:"src"`
	Stage         string   `json:"stage"`
	UnitName      string   `json:"unitname"`
	UnitPath      string   `json:"unitpath"`
	Username      string   `json:"username"`
	Volumes       []string `json:"volumes"`
	VolumesDir    string   `json:"volumes_dir"`
}

// MaestroConfig structure
type MaestroConfig struct {
	App    string `json:"app"`
	Stages []struct {
		Components []MaestroComponent `json:"components"`
		Name       string             `json:"name"`
	} `json:"stages"`
	Username string `json:"username"`
}

// Simple repr for MaestroConfig struct.
func (c *MaestroConfig) Print() {
	configJson, _ := json.MarshalIndent(c, "", "    ")
	userJson, _ := json.MarshalIndent(username, "", "    ")

	lg.Out(lg.b("config and build dir: ") + maestroDir)
	lg.Out(lg.b("user config path: ") + maestroDir + "/user.json")
	lg.Out(string(userJson))
	lg.Out(lg.b("app config path: ") + configFile)
	lg.Out(string(configJson))
}

// Parses JSON config file into MaestroConfig struct.
func (c *MaestroConfig) LoadMaestroConfig(path string) MaestroConfig {
	lg.Debug2("maestro json config file is ", path)
	file, err := ioutil.ReadFile(path)
	lg.Fatal(err)
	lg.Debug("maestro json config file found, loading json")
	err = json.Unmarshal(file, c)
	lg.Fatal(err)
	return *c
}

// Creates directories for unit file building. Schema: $CWD/.maestro/$username/$stage/$app
func (c *MaestroConfig) SetupMaestroAppDirs() {
	lg.Debug("creating build dirs for app and components")
	os.Mkdir(fmt.Sprintf("%s/%s", maestroDir, c.Username), 0755)
	for _, stage := range c.Stages {
		os.Mkdir(path.Join(maestroDir, c.Username, stage.Name), 0755)
		appDir := path.Join(maestroDir, c.Username, stage.Name, c.App)
		lg.Debug2("creating app dir", appDir, stage.Name)
		os.Mkdir(appDir, 0755)
	}
}

func (c *MaestroConfig) GetUsername() {
	lg.Out(c.Username)
	os.Exit(0)
}

// Manages username creation, loading and saving to file.
func (c *MaestroConfig) SetupUsername() {
	if c.Username == "" {
		userFile = path.Join(maestroDir, "user.json")
		lg.Out("username missing in config. using default from " + lg.b(userFile))
		file, err := ioutil.ReadFile(userFile)
		// user config file do not exist. starting wizard.
		if err != nil {
			lg.Debug("user json config file not found, starting wizard")
			c.UsernameWizard()
			c.WriteUsernameFile()
		} else {
			lg.Debug("user json config file found, loading json")
			// load username details
			err = json.Unmarshal(file, &username)
			lg.Fatal(err)
		}
		c.Username = username.Name
	} else {
		username.Name = c.Username
	}
	lg.Debug("username " + lg.b(c.Username))
}

// Wizard to create a new username.
func (c *MaestroConfig) UsernameWizard() {
	reader := bufio.NewReader(os.Stdin)
	lg.Out("maestro setup wizard")
	lg.OutRaw("username: ")
	text, _ := reader.ReadString('\n')
	username.Name = strings.Split(text, "\n")[0]
}

// Writes username JSON informations onto user file
func (c *MaestroConfig) WriteUsernameFile() {
	lg.Debug("writing username details for " + username.Name)
	data, err := json.Marshal(username)
	lg.Fatal(err)
	err = ioutil.WriteFile(userFile, data, 0644)
	lg.Fatal(err)
	lg.Debug("username details saved into " + userFile)
}

// Sets components config values.
func (c *MaestroConfig) SetMaestroComponentConfig() {
	// stages and components ids and systemd units names
	for i, _ := range c.Stages {
		stage := &c.Stages[i]
		// stage id
		lg.Debug("setting up components config for stage ", stage.Name)
		for k, _ := range stage.Components {
			component := &stage.Components[k]
			// scale
			if component.Scale == 0 {
				lg.Debug("setting scale to 1, is has to be at least 1", stage.Name, component.Name)
				component.Scale = 1
			}
			// dns
			if component.DNS != "" {
				lg.Debug("dns is set, component will be published to "+component.DNS, stage.Name, component.Name)
			}

			// global
			if component.Global {
				lg.Debug("component is global, setting scale to 1", stage.Name, component.Name)
				component.Scale = 1
				if component.DNS != "" {
					component.DNS = fmt.Sprintf("%s-%%H", component.DNS)
					lg.Debug2("dns is set with global, overriding to", component.DNS, stage.Name, component.Name)
				}
			}

			// namespaces info
			component.Username = c.Username
			component.App = c.App
			component.Stage = stage.Name

			// names
			component.UnitName = c.GetUnitName(component, "@")
			lg.Debug2("unit name", component.UnitName, stage.Name, component.Name)
			component.ContainerName = c.GetContainerName(component)
			lg.Debug2("container name", component.ContainerName, stage.Name, component.Name)

			// paths
			component.UnitPath = c.GetUnitPath(component, "run")
			lg.Debug2("unit local path", component.UnitPath, stage.Name, component.Name)
			if component.GitSrc != "" {
				component.BuildUnitPath = c.GetUnitPath(component, "build")
				lg.Debug2("gitsrc is set, build unit local path", component.BuildUnitPath, stage.Name, component.Name)
			}

			// dns
			component.InternalDNS = c.GetUnitInternalDNS(component, domain)
			lg.Debug2("internal dns", component.InternalDNS, stage.Name, component.Name)

			// volumes
			component.VolumesDir = path.Join(volumesDir, c.Username, c.App)
			for j, volume := range component.Volumes {
				component.Volumes[j] = c.GetVolumePath(stage.Name, volume, volumesDir)
				lg.Debug2("volume", component.Volumes[j], stage.Name, component.Name)
			}

			// after info
			if component.After != "" {
				component.After = c.GetAfterUnit(component.After)
				lg.Debug2("component will run after, "+component.After, stage.Name, component.Name)
			}
		}
	}
}

// Returns a name for a unit, starting from a `stage`, a `component` and a `suffix`.
func (c *MaestroConfig) GetUnitName(component *MaestroComponent, suffix string) string {
	if suffix == "run" {
		return fmt.Sprintf("%s_%s_%s_%s@.service", c.Username, component.Stage, c.App, component.Name)
	} else if suffix == "build" {
		return fmt.Sprintf("%s_%s_%s_%s-build.service", c.Username, component.Stage, c.App, component.Name)
	} else if _, err := strconv.Atoi(suffix); err == nil {
		return fmt.Sprintf("%s_%s_%s_%s@%s", c.Username, component.Stage, c.App, component.Name, suffix)
	} else if suffix == "@" {
		return fmt.Sprintf("%s_%s_%s_%s%s", c.Username, component.Stage, c.App, component.Name, suffix)
	} else {
		suffix = "1"
		if component.Scale > 1 {
			suffix = "%i"
		}
		return fmt.Sprintf("%s_%s_%s_%s%s", c.Username, component.Stage, c.App, component.Name, suffix)
	}
}

// Returns an inernal DNS name for a unit (mainly for debugging purposes).
func (c *MaestroConfig) GetUnitInternalDNS(component *MaestroComponent, domain string) string {
	prefix := "1"
	if component.Scale > 1 {
		prefix = "%i"
	}
	if component.Global {
		prefix = "%H"
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s", prefix, component.Name, c.App, component.Stage, c.Username, domain)
}

// Returns the local path for an app.
func (c *MaestroConfig) GetAppPath(stage string) string {
	return path.Join(maestroDir, c.Username, stage, c.App)
}

// Returns the local path for a run unit. A run unit is a component of an application which will be run
// as docker container on the coreos cluster.
func (c *MaestroConfig) GetUnitPath(component *MaestroComponent, suffix string) string {
	ret := path.Join(c.GetAppPath(component.Stage), c.GetUnitName(component, suffix))
	return ret
}

// Returns a proper volume path, bound to the shared directory on the node.
func (c *MaestroConfig) GetVolumePath(stage, volume, volumesDir string) string {
	return fmt.Sprintf("%s:%s", path.Join(volumesDir, c.Username, stage, c.App, volume), volume)
}

// Returns the container name for a component (mainly for debugging purposes).
func (c *MaestroConfig) GetContainerName(component *MaestroComponent) string {
	return fmt.Sprintf("%s", c.GetUnitName(component, ""))
}

// Return a path for a numbered unit, used to start scalable components.
func (c *MaestroConfig) GetNumberedUnitPath(path, number string) string {
	return strings.Replace(path, "@", fmt.Sprintf("@%s", number), 1)
}

func (c *MaestroConfig) GetAfterUnit(name string) (after string) {
	for _, stage := range c.Stages {
		for _, component := range stage.Components {
			if component.Name == name {
				after = fmt.Sprintf("%s%%i.service", component.UnitName)
			}
		}
	}
	return
}
