// Setup project for appengine/go
//
// 1) setup basic directory
// 2) setup swagger.yml
// 3) setup dependencies using glide

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/tools/imports"
)

var root = flag.String("r", "./", "server root directory")

const defaultYaml = `runtime: go
api_version: go1
service: default

instance_class: F1
automatic_scaling:
  min_idle_instances: 1
  max_idle_instances: automatic  # default value
  min_pending_latency: 250ms  # default value
  max_pending_latency: automatic
  max_concurrent_requests: 50
	
inbound_services:
- warmup
- mail

includes:
- ../endpoints.yaml
`

const backendYaml = `runtime: go
api_version: go1
service: backend

instance_class: B1
basic_scaling:
  max_instances: 1
  idle_timeout: 10m
	
inbound_services:
- warmup

includes:
- ../endpoints.yaml
`

const endpointsYaml = `handlers:
- url: /_ah/warmup
  script: _go_app
  login: admin

- url: /task/.*
  script: _go_app
  login: admin

- url: /cron/.*
  script: _go_app
  login: admin

- url: /_ah/push-handlers/.*
  script: _go_app
  login: admin

- url: /.*
  script: _go_app
  secure: always
`

const initFile = `package %s

func init() {
	router := gin.New()

	http.Handle("/", router)
}
`

func main() {
	flag.Parse()
	rp, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}

	makeDir(rp)

	err = os.Chdir(rp)
	if err != nil {
		log.Fatal(err)
	}

	setupSwagger()
	setupGlide()
	setupEndpoints()
	setupServer()
}

func setupEndpoints() {
	base := "endpoints"
	// Create api dir.
	api := filepath.Join(base, "api")
	makeDir(api)
	// Create admin dir.
	admin := filepath.Join(base, "admin")
	makeDir(admin)
	// Create taskqueue dir.
	tq := filepath.Join(base, "task")
	makeDir(tq)
	// Create cron dir.
	cron := filepath.Join(base, "cron")
	makeDir(cron)
	// Create pubsub dir.
	pubsub := filepath.Join(base, "pubsub")
	makeDir(pubsub)
}

func setupServer() {
	base := "modules"
	// Create modules
	makeDir(base)
	//Create endpoints.yaml
	ey := filepath.Join(base, "endpoints.yaml")
	makeFile(ey, endpointsYaml)
	// Create default module.
	df := filepath.Join(base, "defaultapp")
	makeDir(df)
	// Create default app.yaml.
	dy := filepath.Join(df, "app.yaml")
	makeFile(dy, defaultYaml)
	// Create defaultapp init.go
	dg := filepath.Join(df, "init.go")
	makeFile(dg, fmt.Sprintf(initFile, "defaultapp"))
	// Create backend module.
	bk := filepath.Join(base, "backend")
	makeDir(bk)
	// Create backend app.yaml
	by := filepath.Join(bk, "app.yaml")
	makeFile(by, backendYaml)
	// Create backend init.go
	bg := filepath.Join(bk, "init.go")
	makeFile(bg, fmt.Sprintf(initFile, "backend"))
	// Format go files
	formatGoFile(dg)
	formatGoFile(bg)
}

func setupSwagger() {
	runCmd("go", "get", "-u", "github.com/go-swagger/go-swagger/cmd/swagger")
	runCmd("swagger", "init", "spec")
}

func setupGlide() {
	runCmd("glide", "init", "--non-interactive")
	runCmd("glide", "get", "cloud.google.com/go", "--skip-test", "--non-interactive")
	runCmd("glide", "get", "google.golang.org/appengine", "--skip-test", "--non-interactive")
	runCmd("glide", "get", "gopkg.in/gin-gonic/gin.v1", "--skip-test", "--non-interactive")
	runCmd("glide", "update")
}

func makeDir(path string) {
	_, err := os.Open(path)
	if err == nil {
		// If file is already exist, then skip.
		return
	}
	err = os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func makeFile(path, data string) {
	_, err := os.Open(path)
	if err == nil {
		// If file is already exist, then skip.
	}
	err = ioutil.WriteFile(path, []byte(data), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func formatGoFile(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	formated, err := imports.Process(filename, data, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filename, formated, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
