package mqtt

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	Enabled     bool   `yaml:"enabled"`
	Host        string `yaml:"host"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	TopicPrefix string `yaml:"topicPrefix"` // Optional: overrides global topicPrefix
}

type Publisher struct {
	client      mqtt.Client
	config      Configuration
	topicPrefix string
}

// credentialsOverPlaintext reports whether credentials are configured while
// the broker URL uses an unencrypted transport scheme.
func credentialsOverPlaintext(config *Configuration) bool {
	if config.Username == "" && config.Password == "" {
		return false
	}
	host := strings.ToLower(config.Host)
	for _, scheme := range []string{"ssl://", "tls://", "mqtts://", "wss://"} {
		if strings.HasPrefix(host, scheme) {
			return false
		}
	}
	return true
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

	if credentialsOverPlaintext(config) {
		log.Warnf("MQTT credentials are configured with plaintext broker scheme (%s); use ssl://, tls://, mqtts://, or wss:// to protect them in transit", config.Host)
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

	publisher := &Publisher{
		client:      client,
		config:      *config,
		topicPrefix: resolveTopicPrefix(config.TopicPrefix),
	}

	return publisher, nil
}

// resolveTopicPrefix returns the effective topic prefix using the priority:
// configPrefix > "solar" default.
func resolveTopicPrefix(configPrefix string) string {
	if configPrefix != "" {
		return configPrefix
	}
	return "solar"
}

func (p *Publisher) Publish(topicSuffix, payload string) {
	if p.client == nil {
		return
	}

	topic := fmt.Sprintf("%s/%s", p.topicPrefix, topicSuffix)

	log.Infof("publishing for %s to %s", topicSuffix, topic)

	token := p.client.Publish(topic, 0, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		log.Errorf("timeout waiting for publish for %s collector", topicSuffix)
	} else if token.Error() != nil {
		log.Errorf("failed to publish: %s", token.Error())
	}
}

func (p *Publisher) Close() {
	if p.client == nil {
		return
	}
	p.client.Disconnect(250)
}
