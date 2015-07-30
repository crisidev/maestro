# Maestro
![maestro](https://raw.githubusercontent.com/crisidev/maestro/master/maestro.jpg)
## A smart deploy system for lazy software developers
Maestro is a development system able to build and deploy applications (docker images) on a distrubuted CoreOS cluster. Applications are described using a simple JSON configuration and converted into runnable Fleet units and all the burden about namespaces, DNS, container linking, etc is automatically handled by Maestro

#### Maestro is based on
* [Vagrant](https://www.vagrantup.com/)
* [CoreOS](https://coreos.com/)
* [Systemd](http://www.freedesktop.org/wiki/Software/systemd/)
* [Etcd2](https://github.com/coreos/etcd)
* [Docker](https://www.docker.com/)
* [Fleet / Fleetctl](https://github.com/coreos/fleet)
* [Flanneld](https://github.com/coreos/flannel)
* [SkyDNS](https://github.com/skynetservices/skydns)
* [DockerGen](https://github.com/jwilder/docker-gen)
* [Confd](https://github.com/kelseyhightower/confd)
* [Nginx](http://nginx.org/)

#### Maestro is a set of tools written in
* Golang
* BASH
* Make
* HTML

#### Maestro components / tools / docker containers
* [Spartito Container](https://github.com/crisidev/maestro-spartito)
* [Violino Container](https://github.com/crisidev/maestro-violino)
* [CoreOS Vagrant Registry](https://github.com/crisidev/maestro-vagrant-registry)
* [CoreOS Vagrant Cluster](https://github.com/crisidev/maestro-vagrant-cluster)
* [Registry Container](https://github.com/crisidev/maestro-registry)
* [Prometheus Container](https://github.com/crisidev/maestro-prometheus)
* [Grafana Container](https://github.com/crisidev/maestro-grafana)

#### [GoDoc](https://godoc.org/github.com/crisidev/maestro)

#### [Keynote](http://crisidev.org/maestro-keynote)

### Prerequisites
Install [Vagrant](https://www.vagrantup.com/) and [Fleetctl](https://github.com/coreos/fleet) and [Golang](https://golang.org/) for your architecture.

### Installation
```sh
$ git clone https://github.com/crisidev/maestro
$ cd maestro
$ make
go clean .
go clean ./cmd/maestro
rm -rf maestro bindata.go
ln -s config/maestro-metrics.json maestro.json || echo "continuing..."
ln -s vagrant/maestro-vagrant-registry/user-data.sample vagrant/maestro-vagrant-registry/user-data || echo "continuing..."
ln -s vagrant/maestro-vagrant-cluster/user-data.sample vagrant/maestro-vagrant-cluster/user-data || echo "continuing..."
go-bindata -pkg maestro templates
go build .
go build ./cmd/maestro
go install ./cmd/maestro
$ maestro
usage: maestro [<flags>] <command> [<args> ...]

friendly generates and deploy systemd unit files on a CoreOS maestro cluster https://github.com/crisidev/maestro

Flags:
  --help           Show help (also see --help-long and --help-man).
  -d, --debug      enable debug mode
  -c, --config="maestro.json"
                   configuration file
  -V, --volumesdir="/share/maestro"
                   directory on the coreos host for shared volumes
  -m, --maestrodir=MAESTRODIR
                   directory on the local host for configs and temporary files (default to $USER/.maestro)
  --domain="maestro.io"
                   domain used to deal with etcd, skydns, spartito and violino
  -e, --etcd=ETCD  etcd / fleet endpoints to connect
  -F, --fleetopts=FLEETOPTS
                   fleetctl options
  -A, --fleetaddr="172.17.8.101"
                   fleetctl tunnel address and port

Commands:
  help [<command>...]
    Show help.

  corestatus
    report coreos cluster status

  exec <args>...
    exec an arbitrary command through fleet, returning output as stdout and exit code

  etcd [<flags>] [<name>]
    get maestro related list of keys from etcd

  run [<name>]
    run current app on coreos (this will build unit files, submit and run them)

  stop [<name>]
    stop current app without cleaning unit files on coreos

  nuke [<flags>] [<name>]
    stop current app and clean unit files on coreos

  status [<name>]
    show the global app status (systemctl status unitfiles)

  journal [<flags>] [<name>]
    show the journal (journalctl -xu unit) of one app s component

  user
    get current user name

  config
    print json configuration for current app

  build
    locally build app units

  buildimages [<unit>]
    run a container build and registry push on the cluster

  buildstatus [<name>]
    check status of a container build and registry push on the cluster

  buildnuke [<name>]
    check status of a container build and registry push on the cluster
```
### Usage

### DNS Resolution In Details

#### Configuration
##### A Basic Example
Let's say we want to run docker container which will ping google.com. Maestro config:
```json
{
  "username": "crisidev",
  "app": "pinger",
  "stages": [
    {
      "name": "prod",
      "components": [
        {
          "name": "pinger",
          "src": "hub.maestro.io:5000/crisidev/busybox",
          "gitsrc": "https://github.com/crisidev/maestro-busybox",
          "cmd": "ping google.com",
        }
      ]
    }
  ]
}
```

##### A Complex Example
Let's say we want to run a complete monitoring system for Maestro, using [Prometheus](http://prometheus.io) as timeseries database and [Grafana](http://grafana.org/) as visualiser. DNS metrics will be gathered from SkyDNS, container metrics from [Cadvisor](https://github.com/google/cadvisor) and node metrics from [Prometheus Node Exporter](https://github.com/prometheus/node_exporter). Grafana and Prometheus the components will share a volume (MacOSX only).
```json
{
  "username": "crisidev",
  "app": "metrics",
  "stages": [
    {
      "name": "prod",
      "components": [
        {
          "name": "prometheus",
          "dns": "prometheus",
          "frontend": true,
          "src": "hub.maestro.io:5000/crisidev/prometheus",
          "gitsrc": "https://github.com/crisidev/maestro-prometheus",
          "ports": [9090],
          "volumes": [
            "/sharedvol"
          ]
        },
        {
          "name": "grafana",
          "dns": "grafana",
          "frontend": true,
          "src": "hub.maestro.io:5000/crisidev/grafana",
          "gitsrc": "https://github.com/crisidev/maestro-grafana",
          "ports": [3001],
           "volumes": [
            "/sharedvol"
           ]
        },
        {
          "name": "cadvisor",
          "frontend": true,
          "global": true,
          "dns": "cadvisor",
          "src": "google/cadvisor:latest",
          "ports": [8080],
          "docker_args": "-v /:/rootfs:ro -v /var/run:/var/run:rw -v /sys:/sys:ro -v /var/lib/docker/:/var/lib/docker:ro"
        },
        {
          "name": "exporter",
          "global": true,
          "src": "prom/node-exporter",
          "docker_args": "--net=host"
        }
      ]
    }
  ]
}
```
Our application will be reachable at http://prometheus.maestro.io and http://grafana.maestro.io

###### Configuration Structure
```go
// MaestroComponent structure
type MaestroComponent struct {
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
```
