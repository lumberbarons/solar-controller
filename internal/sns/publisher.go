package sns

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	Enabled     bool   `yaml:"enabled"`
	Region      string `yaml:"region"`
	TopicArn    string `yaml:"topicArn"`
	TopicPrefix string `yaml:"topicPrefix"` // Optional: overrides global topicPrefix
}

type Publisher struct {
	client      *sns.Client
	config      Configuration
	topicPrefix string
}

func NewPublisher(cfg *Configuration, topicPrefix string) (*Publisher, error) {
	if !cfg.Enabled {
		log.Info("SNS publisher disabled via configuration")
		return &Publisher{}, nil
	}

	if cfg.TopicArn == "" {
		log.Warn("SNS enabled but no topic ARN provided, publisher disabled")
		return &Publisher{}, nil
	}

	if cfg.Region == "" {
		log.Warn("SNS enabled but no region provided, publisher disabled")
		return &Publisher{}, nil
	}

	// Load AWS SDK configuration
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SNS client
	client := sns.NewFromConfig(awsCfg)

	// Verify topic exists by getting its attributes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
		TopicArn: aws.String(cfg.TopicArn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify SNS topic: %w", err)
	}

	log.Infof("connected to SNS topic %s in region %s", cfg.TopicArn, cfg.Region)

	publisher := &Publisher{
		client:      client,
		config:      *cfg,
		topicPrefix: topicPrefix,
	}

	return publisher, nil
}

func (p *Publisher) Publish(topicSuffix, payload string) {
	if p.client == nil {
		return
	}

	topic := fmt.Sprintf("%s/%s", p.topicPrefix, topicSuffix)

	log.Infof("publishing for %s to SNS", topicSuffix)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish message with subject as the topic path
	_, err := p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.config.TopicArn),
		Subject:  aws.String(topic),
		Message:  aws.String(payload),
	})

	if err != nil {
		log.Errorf("failed to publish to SNS for %s: %s", topicSuffix, err)
	}
}

func (p *Publisher) Close() {
	// SNS client doesn't require explicit cleanup
	// Connection pooling is handled by the AWS SDK
}
