package file

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Configuration struct {
	Enabled    bool   `yaml:"enabled"`
	Filename   string `yaml:"filename"`
	MaxSizeMB  int    `yaml:"maxSizeMB"`
	MaxBackups int    `yaml:"maxBackups"`
	Compress   bool   `yaml:"compress"`
}

type Publisher struct {
	logger      *lumberjack.Logger
	config      Configuration
	topicPrefix string
}

func NewPublisher(config *Configuration, topicPrefix string) (*Publisher, error) {
	if !config.Enabled {
		log.Info("File publisher disabled via configuration")
		return &Publisher{}, nil
	}

	if config.Filename == "" {
		log.Warn("File publisher enabled but no filename provided, publisher disabled")
		return &Publisher{}, nil
	}

	// Set defaults if not provided
	maxSize := config.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 10 // 10MB default
	}

	maxBackups := config.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 10 // Keep 10 old files default
	}

	logger := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		Compress:   config.Compress,
		LocalTime:  true, // Use local time for timestamps in filenames
	}

	log.Infof("File publisher initialized: %s (maxSize: %dMB, maxBackups: %d, compress: %t)",
		config.Filename, maxSize, maxBackups, config.Compress)

	publisher := &Publisher{
		logger:      logger,
		config:      *config,
		topicPrefix: topicPrefix,
	}

	return publisher, nil
}

func (p *Publisher) Publish(topicSuffix, payload string) {
	if p.logger == nil {
		return
	}

	// Create full topic with prefix
	topic := fmt.Sprintf("%s/%s", p.topicPrefix, topicSuffix)

	// Write the message as a single line with timestamp
	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf(`{"timestamp":"%s","topic":"%s","payload":%s}`+"\n", timestamp, topic, payload)

	if _, err := p.logger.Write([]byte(line)); err != nil {
		log.Errorf("failed to write to log file: %v", err)
	}
}

func (p *Publisher) Close() {
	if p.logger != nil {
		if err := p.logger.Close(); err != nil {
			log.Errorf("failed to close log file: %v", err)
		}
	}
}
