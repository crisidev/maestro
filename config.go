package maestro

import (
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
	flagFleetEndpoints string
	flagFleetOptions   []string
	flagConfigFile     string
	flagDebug          bool
	maestroDir         string
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
func ParseMaestroConfig(configFile, maestroDir, domain, volumesDir, fleetEndpoints string, fleetOptions []string, debug bool) MaestroConfig {
	flagDebug = debug
	flagFleetOptions = fleetOptions
	flagFleetEndpoints = fleetEndpoints
	SetupMaestroDir(maestroDir)
	SetupUsername()
	config = config.LoadMaestroConfig(configFile)
	config.SetMaestroConfig(domain)
	config.SetMaestroComponentConfig(domain, volumesDir)
	return config
}

// MaestroComponent structure
type MaestroComponent struct {
	Username      string   `json:"username"`
	Scalable      bool     `json:"scalable"`
	Published     bool     `json:"published"`
	App           string   `json:"app"`
	Stage         string   `json:"stage"`
	Global        bool     `json:"global"`
	Single        bool     `json:"single"`
	DNS           string   `json:"dns"`
	ID            int      `json:"id"`
	Cmd           string   `json:"cmd"`
	DontRmOnExit  bool     `json:"dont_rm_on_exit"`
	Name          string   `json:"name"`
	Env           []string `json:"env"`
	Frontend      bool     `json:"frontend"`
	Ports         []int    `json:"ports"`
	Volumes       []string `json:"volumes"`
	Scale         int      `json:"scale"`
	Src           string   `json:"src"`
	GitSrc        string   `json:"gitsrc"`
	UnitName      string   `json:"unit_name"`
	UnitPath      string   `json:"unitpath"`
	BuildUnitPath string   `json:"build_unitpath"`
	VolumesDir    string   `json:"volumes_dir"`
	ContainerName string   `json:"container_name"`
}

// MaestroConfig structure
type MaestroConfig struct {
	Username  string `json:"username"`
	Scalable  bool   `json:"scalable"`
	Published bool   `json:"published"`
	App       string `json:"app"`
	PublicDNS string `json:"public_dns"`
	Global    bool   `json:"global"`
	Single    bool   `json:"sigle"`
	Stages    []struct {
		Components []MaestroComponent `json:"components"`
		ID         int                `json:"id"`
		Name       string             `json:"name"`
	} `json:"stages"`
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

// Sets default config values.
func (c *MaestroConfig) SetMaestroConfig(domain string) {
	PrintD("analysing maestro app " + c.App)
	c.Username = username.Username
	// no dns name, no party ;)
	if c.Published {
		c.Published = true
		c.PublicDNS = fmt.Sprintf("%s-%s.%s", c.Username, c.App, domain)
		PrintD("app will be published to " + c.PublicDNS)
	}
	if c.Global {
		PrintD("app is global (X-Fleet Global:true)")
	}
	if c.Single {
		PrintD("app is single (X-Fleet Conflicts)")
	}
}

// Sets components config values.
func (c *MaestroConfig) SetMaestroComponentConfig(domain, volumesDir string) {
	// stages and components ids and systemd units names
	for i, stage := range c.Stages {
		// stage id
		PrintD("stage " + stage.Name + ", id:" + strconv.Itoa(i))
		c.Stages[i].ID = i
		for k, component := range stage.Components {
			// dns
			unitDNS := c.GetUnitDNS(stage.Name, component.Name, domain)
			PrintDC(component.Name, "internal dns will be "+unitDNS)
			c.Stages[i].Components[k].DNS = unitDNS

			// app info
			c.Stages[i].Components[k].Username = username.Username
			c.Stages[i].Components[k].App = c.App
			c.Stages[i].Components[k].Stage = stage.Name
			c.Stages[i].Components[k].Global = c.Global
			c.Stages[i].Components[k].Single = c.Single
			c.Stages[i].Components[k].Published = c.Published
			c.Stages[i].Components[k].ID = k

			// container_name
			containerName := c.GetContainerName(stage.Name, component.Name)
			PrintDC(component.Name, "container name will be "+containerName)
			c.Stages[i].Components[k].ContainerName = c.GetContainerName(stage.Name, component.Name)

			// volumes
			c.Stages[i].Components[k].VolumesDir = path.Join(volumesDir, c.Username, c.App)
			for j, volume := range component.Volumes {
				c.Stages[i].Components[k].Volumes[j] = c.GetVolumePath(volume, volumesDir)
			}

			component.Name = fmt.Sprintf("%s@", component.Name)
			// scalable
			if c.Scalable {
				PrintDC(component.Name, "app is scalable, component scale set to "+strconv.Itoa(component.Scale))
				c.Stages[i].Components[k].Scalable = true
			} else {
				PrintDC(component.Name, "app is not scalable, component scale set to 1")
				c.Stages[i].Components[k].Scale = 1
			}

			// unit name
			unitName := c.GetUnitName(stage.Name, component.Name, "")
			PrintDC(component.Name, "unit name will be "+unitName)
			c.Stages[i].Components[k].UnitName = unitName

			// local unit files
			c.Stages[i].Components[k].UnitPath = c.GetUnitPath(stage.Name, component.Name)
			if component.GitSrc != "" {
				c.Stages[i].Components[k].BuildUnitPath = c.GetBuildUnitPath(stage.Name, component.Name)
			}
		}
	}
}

// Returns a name for a unit, starting from a `stage`, a `component` and a `suffix`.
func (c *MaestroConfig) GetUnitName(stage, component, suffix string) string {
	if suffix == "run" {
		return fmt.Sprintf("%s_%s_%s_%s.service", c.Username, stage, c.App, component)
	} else if suffix == "build" {
		return fmt.Sprintf("%s_%s_%s_%s-build.service", c.Username, stage, c.App, component)
	} else if _, err := strconv.Atoi(suffix); err == nil {
		return fmt.Sprintf("%s_%s_%s_%s@%s", c.Username, stage, c.App, component, suffix)
	} else {
		return fmt.Sprintf("%s_%s_%s_%s", c.Username, stage, c.App, component)
	}
}

// Returns the prefx for DNS. This is dependent on the scalability of the app.
func (c *MaestroConfig) GetPrefix() (prefix string) {
	prefix = "0"
	if config.Scalable {
		prefix = "%i"
	}
	return
}

// Returns a DNS name for a unit (mainly for debugging purposes).
func (c *MaestroConfig) GetUnitDNS(stage, component, domain string) string {
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s", c.GetPrefix(), component, c.App, stage, c.Username, domain)
}

// Returns the local path for a stage (prod, dev, etc).
func (c *MaestroConfig) GetStagePath(stage string) string {
	return path.Join(maestroDir, username.Username, stage)
}

// Returns the local path for an app.
func (c *MaestroConfig) GetAppPath(stage string) string {
	return path.Join(c.GetStagePath(stage), c.App)
}

// Returns the local path for a run unit. A run unit is a component of an application which will be run
// as docker container on the coreos cluster.
func (c *MaestroConfig) GetUnitPath(stage, component string) string {
	ret := path.Join(c.GetAppPath(stage), c.GetUnitName(stage, component, "run"))
	return ret
}

// Returns the local path for a build unit. A build unit is an unit used to build a new docker image
// and make it available on the local registry.
func (c *MaestroConfig) GetBuildUnitPath(stage, component string) string {
	ret := path.Join(c.GetAppPath(stage), c.GetUnitName(stage, component, "build"))
	return ret
}

// Returns a proper volume path, bound to the shared directory on the node.
func (c *MaestroConfig) GetVolumePath(volume, volumesDir string) string {
	return fmt.Sprintf("%s:%s", path.Join(volumesDir, c.Username, c.App, volume), volume)
}

// Returns the container name for a component (mainly for debugging purposes).
func (c *MaestroConfig) GetContainerName(stage, component string) string {
	return fmt.Sprintf("%s%%i", c.GetUnitName(stage, component, ""))
}

// Return a path for a numbered unit, used to start scalable components.
func (c *MaestroConfig) GetNumberedUnitPath(path, number string) string {
	return strings.Replace(path, "@", fmt.Sprintf("@%s", number), 1)
}
