package maestro_test

import (
	"testing"

	"github.com/crisidev/maestro"
	"github.com/stretchr/testify/assert"
)

var config maestro.MaestroConfig

func init() {
	maestro.SetupUsername()
	maestro.SetupMaestroAppDirs()
	config = maestro.ParseMaestroConfig("maestro.json", "maestro.io", "/var/maestro", "http://localhost:2379", []string{}, false)
}

func TestSetMaestroConfig(t *testing.T) {
	assert.False(t, config.Published, "app should not be published")
	assert.False(t, config.Global, "app should not be global")
	assert.False(t, config.Scalable, "app should not be scalable")
	assert.False(t, config.Single, "app should not be single")
}

func TestSetMaestroComponentConfig(t *testing.T) {
	for _, stage := range config.Stages {
		assert.Equal(t, stage.Name, "prod", "stage name should be prod")
		assert.Equal(t, stage.ID, 0, "stage ID should be 0")
		for _, component := range stage.Components {
			assert.Equal(t, component.Username, "crisidev", "component username should be crisidev")
			assert.False(t, component.Published, "component should not be published")
			assert.Equal(t, component.App, "pinger", "component app should be pinger")
			assert.Equal(t, component.Stage, "prod", "component stage should be prod")
			assert.False(t, component.Scalable, "component should not be scalable")
			assert.False(t, component.Single, "component should not be single")
			assert.Equal(t, component.DNS, "0.pinger.pinger.prod.crisidev.maestro.io", "component dns should be 0.pinger.pinger.prod.crisidev.maestro.io")
			assert.Equal(t, component.ID, 0, "component ID should be 0")
			assert.Equal(t, component.Cmd, "ping google.com", "component cmd should be ping google.com")
			assert.False(t, component.DontRmOnExit, "component should not be removed on exit")
			assert.Equal(t, component.Name, "pinger", "component name should be pinger")
			assert.Nil(t, component.Env)
			assert.False(t, component.Frontend, "component is not a frontend")
			assert.Nil(t, component.Ports)
			assert.Equal(t, component.Volumes[0], "/var/maestro/crisidev/pinger/data/mytest:/data/mytest", "component volume[0] should be /var/maestro/crisidev/pinger/data/mytest:/data/mytest")
			assert.Equal(t, component.Volumes[1], "/var/maestro/crisidev/pinger/data/mytest2:/data/mytest2", "component volume[0] should be /var/maestro/crisidev/pinger/data/mytest2:/data/mytest2")
			assert.Equal(t, component.Scale, 1, "component scale should be 1")
			assert.Equal(t, component.Src, "hub.maestro.io:5000/crisidev/debian", "component src should be hub.maestro.io:5000/crisidev/debian")
			assert.Equal(t, component.UnitName, "crisidev_prod_pinger_pinger@", "component unit name should be crisidev_prod_pinger_pinger@")
			assert.Equal(t, component.VolumesDir, "/var/maestro/crisidev/pinger", "component volumes dir should be /var/maestro/crisidev/pinger")
			assert.Equal(t, component.ContainerName, "crisidev_prod_pinger_pinger%i", "component container name should be crisidev_prod_pinger_pinger%i")
		}
	}
}

func TestGetUnitName(t *testing.T) {
	assert.Equal(t, config.GetUnitName("prod", "pinger", ""), "crisidev_prod_pinger_pinger", "unit name should be crisidev_prod_pinger_pinger")
	assert.Equal(t, config.GetUnitName("prod", "pinger", "run"), "crisidev_prod_pinger_pinger.service", "unit name should be crisidev_prod_pinger_pinger.service")
	assert.Equal(t, config.GetUnitName("prod", "pinger", "build"), "crisidev_prod_pinger_pinger-build.service", "unit name should be crisidev_prod_pinger_pinger-build.service")
}

func TestGetPrefix(t *testing.T) {
	assert.Equal(t, config.GetPrefix(), "0", "prefix should be 0")
}

func TestGetUnitDNS(t *testing.T) {
	assert.Equal(t, config.GetUnitDNS("prod", "pinger", "maestro.io"), "0.pinger.pinger.prod.crisidev.maestro.io", "dns should be 0.pinger.pinger.prod.crisidev.maestro.io")
}

func TestGetVolumePath(t *testing.T) {
	assert.Equal(t, config.GetVolumePath("test_volume", "/var/maestro"), "/var/maestro/crisidev/pinger/test_volume:test_volume", "volume path should be /var/maestro/crisidev/pinger/test_volume:test_volume")
}

func TestGetContainerName(t *testing.T) {
	assert.Equal(t, config.GetContainerName("prod", "pinger"), "crisidev_prod_pinger_pinger%i", "container name should be crisidev_prod_pinger_pinger%i")
}
