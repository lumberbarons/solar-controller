package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/epever-controller/configuration"
	"github.com/lumberbarons/epever-controller/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"net/http"
	"time"
)

//go:embed site/build
var site embed.FS

var (
	configFilePath *string
	debugMode  	   *bool
)

type Config struct {
	EpeverController EpeverController `yaml:"epeverController"`
}

type EpeverController struct {
	SerialPort string `yaml:"serialPort"`
	HttpPort   int    `yaml:"httpPort"`
}

func init() {
	configFilePath = flag.String("config", "", "Config file path")
	debugMode = flag.Bool("debug", false, "Debug mode")
}

func loadConfigFile() Config {
	if *configFilePath == "" {
		log.Fatalf("Must specify config file path")
	}

	configFile, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatalf("Failed to load configuration file: %v", err)
	}

	config := Config{}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func main() {
	log.Info("Starting epever-controller")

	flag.Parse()

	if *debugMode {
		log.SetLevel(log.DebugLevel)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	controllerConfig := loadConfigFile()
	handler := modbus.NewRTUClientHandler(controllerConfig.EpeverController.SerialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 5 * time.Second

	err := handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to serial port: %v", err)
	}

	defer handler.Close()

	client := modbus.NewClient(handler)
	prometheus.MustRegister()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	collector := metrics.NewSolarCollector(client)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	r.GET("/metrics", func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/api/metrics", collector.MetricsGet())

	configurer := configuration.NewSolarConfigurer(client)

	r.GET("/api/configuration", configurer.ConfigGet())
	r.PATCH("/api/configuration", configurer.ConfigPatch())
	r.POST("/api/query", configurer.QueryPost())

	siteFS := EmbedFolder(site, "site/build", true)
	r.Use(static.Serve("/", siteFS))

	err = r.Run(fmt.Sprintf(":%v", controllerConfig.EpeverController.HttpPort))

	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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