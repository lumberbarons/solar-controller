package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lumberbarons/solar-controller/epever"
	"github.com/lumberbarons/solar-controller/exporter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"net/http"
)

//go:embed site/build
var site embed.FS

var (
	configFilePath *string
	debugMode  	   *bool
)

type Config struct {
	SolarController SolarController `yaml:"solarController"`
}

type Mqtt struct {
	Host             string `yaml:"host"`
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	TopicPrefix      string `yaml:"topicPrefix"`
}

type SolarController struct {
	HttpPort int    `yaml:"httpPort"`
	Mqtt     Mqtt   `yaml:"mqtt"`
	Epever   Epever`yaml:"epever"`
}

type Epever struct {
	SerialPort  string `yaml:"serialPort"`
	CacheExpiry int64  `yaml:"cacheExpiry"`
}

func init() {
	configFilePath = flag.String("config", "", "Config file path")
	debugMode = flag.Bool("debug", false, "Debug mode")
}

func main() {
	log.Info("Starting solar-controller")

	flag.Parse()

	if *debugMode {
		log.SetLevel(log.DebugLevel)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	controllerConfig := loadConfigFile()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	epeverController := epever.NewEpeverCollector(
		controllerConfig.SolarController.Epever.SerialPort,
		controllerConfig.SolarController.Epever.CacheExpiry)

	defer epeverController.Close()

	prometheusExporter := exporter.NewPrometheusEndpoint(epeverController.EpeverCollector)

	r.GET("/collector", func(c *gin.Context) {
		prometheusExporter.Handler.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/api/metrics", epeverController.EpeverCollector.MetricsGet())
	r.GET("/api/config", epeverController.EpeverConfigurer.ConfigGet())
	r.PATCH("/api/config", epeverController.EpeverConfigurer.ConfigPatch())
	r.POST("/api/query", epeverController.EpeverConfigurer.QueryPost())

	siteFS := EmbedFolder(site, "site/build", true)
	r.Use(static.Serve("/", siteFS))

	log.Infof("Starting server on port %v", controllerConfig.SolarController.HttpPort)
	err := r.Run(fmt.Sprintf(":%v", controllerConfig.SolarController.HttpPort))

	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func loadConfigFile() Config {
	if *configFilePath == "" {
		log.Fatalf("Must specify config file path")
	}

	configFile, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatalf("Failed to load configurer file: %v", err)
	}

	config := Config{}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}



type embedFileSystem struct {
	http.FileSystem
	indexes bool
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	f, err := e.Open(path)
	if err != nil {
		return false
	}

	s, _ := f.Stat()
	if s.IsDir() && !e.indexes {
		return false
	}

	return true
}

func EmbedFolder(fsEmbed embed.FS, targetPath string, index bool) static.ServeFileSystem {
	subFS, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		log.Fatalf("Failed to load static site: %v", err)
	}

	return embedFileSystem{
		FileSystem: http.FS(subFS),
		indexes:    index,
	}
}