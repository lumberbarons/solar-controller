package publisher

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"time"
)

type SolarCollector interface {
	GetTopicSuffix() string
	GetStatusString() (string, error)
}

type MqttConfiguration struct {
	Host          string `yaml:"host"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	TopicPrefix   string `yaml:"topicPrefix"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type MqttPublisher struct {
	client mqtt.Client
	solarCollectors []SolarCollector
	config MqttConfiguration
}

func NewMqttPublisher(config MqttConfiguration, solarCollectors ...SolarCollector) (*MqttPublisher, error) {
	if config.Host == "" {
		log.Info("publisher disabled, no host provided")
		return &MqttPublisher{}, nil
	}

	//mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New()

	opts := mqtt.NewClientOptions().
		AddBroker(config.Host).
		SetUsername(config.Username).
		SetPassword(config.Password).
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to broker: %w", token.Error())
	}

	log.Infof("connected to broker %s", config.Host)

	publisher := &MqttPublisher{client: client, solarCollectors: solarCollectors, config: config}

	log.Infof("starting periodic publisher with period of %d seconds", config.PublishPeriod)

	s := gocron.NewScheduler(time.UTC)

	_, err := s.Every(config.PublishPeriod).Seconds().WaitForSchedule().Do(publisher.publish)
	if err != nil {
		return nil, fmt.Errorf("failed to start publisher %w", err)
	}

	s.StartAsync()

	return publisher, nil
}

func (p *MqttPublisher) publish() {
	for _, collector := range p.solarCollectors {
		topicSuffix := collector.GetTopicSuffix()
		topic := fmt.Sprintf("%s/%s", p.config.TopicPrefix, topicSuffix)

		log.Infof("publishing for %s to %s", topicSuffix, topic)

		payload, err := collector.GetStatusString()
		if err != nil {
			log.Errorf("failed to get status from %s collector:  %s", topicSuffix, err)
			continue
		}

		token := p.client.Publish(topic, 0, false, payload)
		if !token.WaitTimeout(5 * time.Second) {
			log.Error("timeout waiting for publish for %s collector", topicSuffix)
		} else if token.Error() != nil {
			log.Errorf("failed to publish: %s", token.Error())
		}
	}
}

func (p *MqttPublisher) Close() {
	p.client.Disconnect(250)
}
