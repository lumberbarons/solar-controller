package main

import (
	"embed"
	"flag"
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
	configFilePath  *string
	debugMode   *bool
)

type Config struct {
	EpeverController EpeverController `yaml:"epeverController"`
}

type EpeverController struct {
	SerialPort string `yaml:"serialPort"`
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

	r.StaticFS("/", staticFS())

	err = r.Run()

	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func staticFS() http.FileSystem {
	sub, err := fs.Sub(site, "site/build")

	if err != nil {
		log.Fatalf("Failed to load static site: %v", err)
	}

	return http.FS(sub)
}