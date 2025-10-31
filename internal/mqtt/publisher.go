package mqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	Enabled       bool   `yaml:"enabled"`
	Host          string `yaml:"host"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	TopicPrefix   string `yaml:"topicPrefix"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type Publisher struct {
	client mqtt.Client
	config Configuration
}

func NewPublisher(config *Configuration) (*Publisher, error) {
	if !config.Enabled {
		log.Info("MQTT publisher disabled via configuration")
		return &Publisher{}, nil
	}

	if config.Host == "" {
		log.Warn("MQTT enabled but no host provided, publisher disabled")
		return &Publisher{}, nil
	}

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

	publisher := &Publisher{client: client, config: *config}

	return publisher, nil
}

func (p *Publisher) Publish(topicSuffix, payload string) {
	if p.client == nil {
		return
	}

	topic := fmt.Sprintf("%s/%s", p.config.TopicPrefix, topicSuffix)

	log.Infof("publishing for %s to %s", topicSuffix, topic)

	token := p.client.Publish(topic, 0, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		log.Errorf("timeout waiting for publish for %s collector", topicSuffix)
	} else if token.Error() != nil {
		log.Errorf("failed to publish: %s", token.Error())
	}
}

func (p *Publisher) Close() {
	p.client.Disconnect(250)
}
