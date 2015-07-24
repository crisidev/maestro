package maestro

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
)

//func ValidateJsonSchema(jsonString string) bool {
//data, err := Asset("files/json.schema")
//if err != nil {
//PrintE(err)
//}
//schemaLoader := gojsonschema.NewStringLoader(string(data))
//documentLoader := gojsonschema.NewReferenceLoader("file://" + *flagConfigFile)
//result, err := gojsonschema.Validate(schemaLoader, documentLoader)
//if err != nil {
//PrintF(err)
//}
//if !result.Valid() {
//PrintE(errors.New("invalid json config"))
//for _, desc := range result.Errors() {
//PrintE(errors.New(fmt.Sprintf("- %s\n", desc)))
//}
//return false
//}
//return true
//}
var (
	config             MaestroConfig
	flagFleetEndpoints string
	flagFleetOptions   []string
	flagConfigFile     string
	flagDebug          bool
)

func init() {
	SetupMaestroDir()
	SetupUsername()
}

// parse json config file
func LoadMaestroConfig(path string) MaestroConfig {
	jsonConfig := MaestroConfig{}
	PrintD("maestro json config file is " + path)
	file, err := ioutil.ReadFile(path)
	if err != nil {
		PrintF(err)
	}
	PrintD("maestro json config file found, loading json")
	err = json.Unmarshal(file, &jsonConfig)
	if err != nil {
		PrintF(err)
	}
	//PrintD("config loaded, starting validation")
	//if ValidateJsonSchema(string(file)) {
	//PrintD("json config valid")
	//} else {
	//PrintF(errors.New("json config not valid"))
	//}
	return jsonConfig
}

func ParseMaestroConfig(configFile, domain, volumesDir, fleetEndpoints string, fleetOptions []string, debug bool) MaestroConfig {
	flagDebug = debug
	flagFleetOptions = fleetOptions
	flagFleetEndpoints = fleetEndpoints
	config = LoadMaestroConfig(configFile)
	config.SetMaestroConfig(domain)
	config.SetMaestroComponentConfig(domain, volumesDir)
	return config
}

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

// set defaults
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

func (c *MaestroConfig) GetPrefix() (prefix string) {
	prefix = "0"
	if config.Scalable {
		prefix = "%i"
	}
	return
}

func (c *MaestroConfig) GetUnitDNS(stage, component, domain string) string {
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s", c.GetPrefix(), component, c.App, stage, c.Username, domain)
}

func (c *MaestroConfig) GetStagePath(stage string) string {
	return path.Join(maestroDir, username.Username, stage)
}

func (c *MaestroConfig) GetAppPath(stage string) string {
	return path.Join(c.GetStagePath(stage), c.App)
}

func (c *MaestroConfig) GetUnitPath(stage, component string) string {
	ret := path.Join(c.GetAppPath(stage), c.GetUnitName(stage, component, "run"))
	return ret
}

func (c *MaestroConfig) GetBuildUnitPath(stage, component string) string {
	ret := path.Join(c.GetAppPath(stage), c.GetUnitName(stage, component, "build"))
	return ret
}

func (c *MaestroConfig) GetVolumePath(volume, volumesDir string) string {
	return fmt.Sprintf("%s:%s", path.Join(volumesDir, c.Username, c.App, volume), volume)
}

func (c *MaestroConfig) GetContainerName(stage, component string) string {
	return fmt.Sprintf("%s%%i", c.GetUnitName(stage, component, ""))
}

func (c *MaestroConfig) GetUnitDNSame(stage, component string) string {
	return fmt.Sprintf("%s%%i", c.GetUnitName(stage, component, ""))
}

func (c *MaestroConfig) GetNumberedUnitPath(path, number string) string {
	return strings.Replace(path, "@", fmt.Sprintf("@%s", number), 1)
}
