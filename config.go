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
	config             MaestroConfig
	username           MaestroUser
	flagDebug          bool
	domain             string
	flagFleetEndpoints string
	flagConfigFile     string
	maestroDir         string
	userFile           string
	volumesDir         string
	flagFleetOptions   []string
)

// Setup directory ($CWD/.maestro) used to store user informations and
// temporary build file before submitting to coreos.
// Directory will be created if not existent.
func SetupMaestroDir(dir string) {
	if dir == "" {
		user, err := user.Current()
		if err != nil {
			PrintF(err)
		}
		dir = user.HomeDir
	}
	maestroDir = path.Join(dir, ".maestro")
	if _, err := os.Stat(maestroDir); err != nil {
		PrintD(maestroDir + " not found, creating")
		os.Mkdir(maestroDir, 0755)
	}
}

// Public function used in the main to load the configuration.
func BuildMaestroConfig(configFile, maestroDir, domainName, volumesPath, fleetEndpoints string, fleetOptions []string, debug bool) MaestroConfig {
	flagDebug = debug
	flagFleetOptions = fleetOptions
	flagFleetEndpoints = fleetEndpoints
	domain = domainName
	volumesDir = volumesPath
	SetupMaestroDir(maestroDir)
	config = config.LoadMaestroConfig(configFile)
	config.SetupUsername()
	//config.SetMaestroConfig(domain)
	config.SetupMaestroAppDirs()
	config.SetMaestroComponentConfig()
	return config
}

type MaestroUser struct {
	Name string `json:"name"`
}

// MaestroComponent structure
type MaestroComponent struct {
	App           string   `json:"app"`
	BuildUnitPath string   `json:"build_unitpath"`
	Cmd           string   `json:"cmd"`
	ContainerName string   `json:"container_name"`
	DNS           string   `json:"dns"`
	Env           []string `json:"env"`
	Frontend      bool     `json:"frontend"`
	GitSrc        string   `json:"gitsrc"`
	Global        bool     `json:"global"`
	InternalDNS   string   `json:"internal_dns"`
	KeepOnExit    bool     `json:"keep_on_exit"`
	Name          string   `json:"name"`
	Ports         []int    `json:"ports"`
	Scale         int      `json:"scale"`
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

	fmt.Printf("fleet endpoints: %s\n", flagFleetEndpoints)
	fmt.Printf("config and build dir: %s\n", maestroDir)
	fmt.Printf("user config path: %s/user.json\n", maestroDir)
	fmt.Println(string(userJson))
	fmt.Printf("\napp config path: %s\n", flagConfigFile)
	fmt.Println(string(configJson))
}

// Parses JSON config file into MaestroConfig struct.
func (c *MaestroConfig) LoadMaestroConfig(path string) MaestroConfig {
	PrintD("maestro json config file is " + path)
	file, err := ioutil.ReadFile(path)
	if err != nil {
		PrintF(err)
	}
	PrintD("maestro json config file found, loading json")
	err = json.Unmarshal(file, c)
	if err != nil {
		PrintF(err)
	}
	return *c
}

// Creates directories for unit file building. Schema: $CWD/.maestro/$username/$stage/$app
func (c *MaestroConfig) SetupMaestroAppDirs() {
	PrintD("creating build dirs for app and components")
	os.Mkdir(fmt.Sprintf("%s/%s", maestroDir, c.Username), 0755)
	for _, stage := range c.Stages {
		os.Mkdir(path.Join(maestroDir, c.Username, stage.Name), 0755)
		os.Mkdir(path.Join(maestroDir, c.Username, stage.Name, c.App), 0755)
	}
}

func (c *MaestroConfig) GetUsername() {
	Print(c.Username)
}

// Manages username creation, loading and saving to file.
func (c *MaestroConfig) SetupUsername() {
	if c.Username == "" {
		userFile = path.Join(maestroDir, "user.json")
		Print("username missing in config. using default from " + userFile)
		file, err := ioutil.ReadFile(userFile)
		// user config file do not exist. starting wizard.
		if err != nil {
			PrintD("user json config file not found, starting wizard")
			c.UsernameWizard()
			c.WriteUsernameFile()
		} else {
			PrintD("user json config file found, loading json")
			// load username details
			err = json.Unmarshal(file, &username)
			if err != nil {
				PrintF(err)
			}
		}
		c.Username = username.Name
	}
	PrintD("username " + c.Username)
}

// Wizard to create a new username.
func (c *MaestroConfig) UsernameWizard() {
	reader := bufio.NewReader(os.Stdin)
	Print("maestro setup wizard")
	PrintR("username: ")
	text, _ := reader.ReadString('\n')
	username.Name = strings.Split(text, "\n")[0]
}

// Writes username JSON informations onto user file
func (c *MaestroConfig) WriteUsernameFile() {
	PrintD("writing username details")
	data, err := json.Marshal(username)
	if err != nil {
		PrintF(err)
	}
	err = ioutil.WriteFile(userFile, data, 0644)
	if err != nil {
		PrintF(err)
	}
	PrintD("username details saved into " + userFile)
}

// Sets components config values.
func (c *MaestroConfig) SetMaestroComponentConfig() {
	// stages and components ids and systemd units names
	for i, _ := range c.Stages {
		stage := &c.Stages[i]
		// stage id
		PrintD("stage " + stage.Name + ", id:" + strconv.Itoa(i))
		for k, _ := range stage.Components {
			component := &stage.Components[k]
			// scale
			if component.Scale == 0 {
				component.Scale = 1
			}

			// namespaces info
			component.Username = c.Username
			component.App = c.App
			component.Stage = stage.Name

			// names
			component.UnitName = c.GetUnitName(component, "")
			component.ContainerName = c.GetContainerName(component)

			// paths
			component.UnitPath = c.GetUnitPath(component, "run")
			if component.GitSrc != "" {
				component.BuildUnitPath = c.GetUnitPath(component, "build")
			}

			// dns
			component.InternalDNS = c.GetUnitInternalDNS(component, domain)
			if component.Frontend {
				if component.DNS == "" {
					component.DNS = c.GetUnitPublicDNS(component, domain)
				}
			}

			// volumes
			component.VolumesDir = path.Join(volumesDir, c.Username, c.App)
			for j, volume := range component.Volumes {
				component.Volumes[j] = c.GetVolumePath(volume, volumesDir)
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
	} else {
		suffix = "0"
		if component.Scale > 1 {
			suffix = "%i"
		}
		return fmt.Sprintf("%s_%s_%s_%s%s", c.Username, component.Stage, c.App, component.Name, suffix)
	}
}

// Returns an inernal DNS name for a unit (mainly for debugging purposes).
func (c *MaestroConfig) GetUnitInternalDNS(component *MaestroComponent, domain string) string {
	prefix := "0"
	if component.Scale > 1 {
		prefix = "%i"
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s", prefix, component.Name, c.App, component.Stage, c.Username, domain)
}

// Returns an public DNS name for a unit (mainly for debugging purposes).
func (c *MaestroConfig) GetUnitPublicDNS(component *MaestroComponent, domain string) string {
	return fmt.Sprintf("%s-%s-%s-%s.%s", c.Username, component.Stage, c.App, component.Name, domain)
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
func (c *MaestroConfig) GetVolumePath(volume, volumesDir string) string {
	return fmt.Sprintf("%s:%s", path.Join(volumesDir, c.Username, c.App, volume), volume)
}

// Returns the container name for a component (mainly for debugging purposes).
func (c *MaestroConfig) GetContainerName(component *MaestroComponent) string {
	return fmt.Sprintf("%s", c.GetUnitName(component, ""))
}

// Return a path for a numbered unit, used to start scalable components.
func (c *MaestroConfig) GetNumberedUnitPath(path, number string) string {
	return strings.Replace(path, "@", fmt.Sprintf("@%s", number), 1)
}
