//go:build integration

package sns

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lumberbarons/solar-controller/internal/testing/containers"
)

func TestSNSPublisherIntegration(t *testing.T) {
	// Start LocalStack container
	localStack := containers.StartLocalStack(t)

	ctx := context.Background()

	// Configure AWS SDK for LocalStack
	endpoint := localStack.GetSNSEndpoint(t)
	region := localStack.GetRegion()
	accessKey, secretKey := localStack.GetCredentials()

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               endpoint,
					HostnameImmutable: true,
				}, nil
			},
		)),
	)
	require.NoError(t, err)

	// Create SNS and SQS clients
	snsClient := sns.NewFromConfig(awsCfg)
	sqsClient := sqs.NewFromConfig(awsCfg)

	// Create SNS topic
	topicResp, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String("test-solar-metrics"),
	})
	require.NoError(t, err)
	topicArn := *topicResp.TopicArn

	// Create SQS queue to receive SNS messages
	queueResp, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("test-solar-metrics-queue"),
	})
	require.NoError(t, err)
	queueURL := *queueResp.QueueUrl

	// Get queue ARN
	attrResp, err := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	require.NoError(t, err)
	queueArn := attrResp.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]

	// Subscribe SQS queue to SNS topic
	_, err = snsClient.Subscribe(ctx, &sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		TopicArn: aws.String(topicArn),
		Endpoint: aws.String(queueArn),
	})
	require.NoError(t, err)

	// Create publisher configuration
	cfg := &Configuration{
		Enabled:     true,
		Region:      region,
		TopicArn:    topicArn,
		TopicPrefix: "solar",
	}

	// Create publisher with custom AWS config
	publisher, err := newPublisherWithConfig(cfg, "solar", awsCfg)
	require.NoError(t, err)
	require.NotNil(t, publisher.client)
	defer publisher.Close()

	t.Run("PublishSingleMessage", func(t *testing.T) {
		// Publish a message
		topicSuffix := "controller-1/epever/battery-voltage"
		payload := `{"value": 12.8, "unit": "volts", "timestamp": 1699000000}`

		publisher.Publish(topicSuffix, payload)

		// Wait a bit for message to be delivered
		time.Sleep(2 * time.Second)

		// Receive message from SQS
		msgResp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     5,
		})
		require.NoError(t, err)
		require.Len(t, msgResp.Messages, 1)

		// Parse SNS notification
		var snsNotification struct {
			Type      string `json:"Type"`
			MessageId string `json:"MessageId"`
			TopicArn  string `json:"TopicArn"`
			Subject   string `json:"Subject"`
			Message   string `json:"Message"`
			Timestamp string `json:"Timestamp"`
		}
		err = json.Unmarshal([]byte(*msgResp.Messages[0].Body), &snsNotification)
		require.NoError(t, err)

		// Verify SNS message content
		assert.Equal(t, "Notification", snsNotification.Type)
		assert.Equal(t, topicArn, snsNotification.TopicArn)
		assert.Equal(t, "solar/controller-1/epever/battery-voltage", snsNotification.Subject)
		assert.Equal(t, payload, snsNotification.Message)

		// Verify payload is valid JSON
		var metricPayload map[string]interface{}
		err = json.Unmarshal([]byte(snsNotification.Message), &metricPayload)
		require.NoError(t, err)
		assert.Equal(t, 12.8, metricPayload["value"])
		assert.Equal(t, "volts", metricPayload["unit"])

		// Clean up message
		_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(queueURL),
			ReceiptHandle: msgResp.Messages[0].ReceiptHandle,
		})
		require.NoError(t, err)
	})

	t.Run("PublishMultipleMessages", func(t *testing.T) {
		// Publish multiple messages
		metrics := []struct {
			topicSuffix string
			payload     string
		}{
			{"controller-1/epever/array-voltage", `{"value": 18.5, "unit": "volts", "timestamp": 1699000001}`},
			{"controller-1/epever/battery-soc", `{"value": 85, "unit": "percent", "timestamp": 1699000002}`},
			{"controller-1/epever/charging-power", `{"value": 120, "unit": "watts", "timestamp": 1699000003}`},
		}

		for _, m := range metrics {
			publisher.Publish(m.topicSuffix, m.payload)
		}

		// Wait for messages to be delivered
		time.Sleep(2 * time.Second)

		// Receive all messages from SQS
		msgResp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     5,
		})
		require.NoError(t, err)
		assert.Len(t, msgResp.Messages, 3, "should receive all 3 messages")

		// Verify each message
		subjects := make(map[string]string)
		for _, msg := range msgResp.Messages {
			var snsNotification struct {
				Subject string `json:"Subject"`
				Message string `json:"Message"`
			}
			err = json.Unmarshal([]byte(*msg.Body), &snsNotification)
			require.NoError(t, err)

			subjects[snsNotification.Subject] = snsNotification.Message

			// Clean up
			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			})
			require.NoError(t, err)
		}

		// Verify all expected subjects were received
		assert.Contains(t, subjects, "solar/controller-1/epever/array-voltage")
		assert.Contains(t, subjects, "solar/controller-1/epever/battery-soc")
		assert.Contains(t, subjects, "solar/controller-1/epever/charging-power")
	})

	t.Run("CustomTopicPrefix", func(t *testing.T) {
		// Create publisher with custom topic prefix
		customCfg := &Configuration{
			Enabled:     true,
			Region:      region,
			TopicArn:    topicArn,
			TopicPrefix: "custom-prefix",
		}

		customPublisher, err := newPublisherWithConfig(customCfg, "default-prefix", awsCfg)
		require.NoError(t, err)
		defer customPublisher.Close()

		// Publish message
		topicSuffix := "controller-1/epever/device-temp"
		payload := `{"value": 35, "unit": "celsius", "timestamp": 1699000004}`

		customPublisher.Publish(topicSuffix, payload)

		// Wait and receive
		time.Sleep(2 * time.Second)

		msgResp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     5,
		})
		require.NoError(t, err)
		require.Len(t, msgResp.Messages, 1)

		// Parse and verify
		var snsNotification struct {
			Subject string `json:"Subject"`
		}
		err = json.Unmarshal([]byte(*msgResp.Messages[0].Body), &snsNotification)
		require.NoError(t, err)

		// Should use default prefix since TopicPrefix in config is set
		assert.Equal(t, "custom-prefix/controller-1/epever/device-temp", snsNotification.Subject)

		// Clean up
		_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(queueURL),
			ReceiptHandle: msgResp.Messages[0].ReceiptHandle,
		})
		require.NoError(t, err)
	})
}
