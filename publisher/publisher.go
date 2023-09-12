package publisher

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"time"
)

type MqttConfiguration struct {
	Host          string `yaml:"host"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	TopicPrefix   string `yaml:"topicPrefix"`
	PublishPeriod int    `yaml:"publishPeriod"`
}

type MqttPublisher struct {
	client mqtt.Client
	config MqttConfiguration
}

func NewMqttPublisher(config MqttConfiguration) (*MqttPublisher, error) {
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

	publisher := &MqttPublisher{client: client, config: config}

	return publisher, nil
}

func (p *MqttPublisher) Publish(topicSuffix string, payload string) {
	if p.client == nil {
		return
	}

	topic := fmt.Sprintf("%s/%s", p.config.TopicPrefix, topicSuffix)

	log.Infof("publishing for %s to %s", topicSuffix, topic)

	token := p.client.Publish(topic, 0, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		log.Error("timeout waiting for publish for %s collector", topicSuffix)
	} else if token.Error() != nil {
		log.Errorf("failed to publish: %s", token.Error())
	}
}

func (p *MqttPublisher) Close() {
	p.client.Disconnect(250)
}
