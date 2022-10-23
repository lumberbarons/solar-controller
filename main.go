package main

import (
	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/lumberbarons/epever-controller/config"
	"github.com/lumberbarons/epever-controller/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"gopkg.in/yaml.v3"
	"embed"
)

//go:embed site/build
var site embed.FS

type Config struct {
	EpeverController EpeverController `yaml:"epeverController"`
}

type EpeverController struct {
	SerialPort string `yaml:"serialPort"`
}

func init() {
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}
}

func loadConfigFile(configFilePath string) Config {
	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	config := Config{}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func main() {
	log.Println("starting epever-controller")

	serialPort := os.Getenv("SERIAL_PORT")
	if serialPort == "" {
		log.Fatal("serial port not set")
	}

	handler := modbus.NewRTUClientHandler(serialPort)

	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 5 * time.Second

	err := handler.Connect()
	if err != nil {
		log.Fatal("failed to connect to serial port")
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

	configurer := config.NewSolarConfigurer(client)

	r.GET("/api/config", configurer.ConfigGet())
	r.PATCH("/api/config", configurer.ConfigPatch())
	r.POST("/api/query", configurer.QueryPost())

	r.StaticFS("/", staticFS())

	r.Run()
}

func staticFS() http.FileSystem {
	sub, err := fs.Sub(site, "site/build")

	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}