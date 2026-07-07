//go:build integration

package sns

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

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

// snsTestFixture holds per-subtest SNS/SQS resources
type snsTestFixture struct {
	topicArn  string
	queueURL  string
	snsClient *sns.Client
	sqsClient *sqs.Client
	awsCfg    aws.Config
}

// setupSNSFixture creates an isolated SNS topic + SQS queue + subscription for a single subtest.
func setupSNSFixture(t *testing.T, localStack *containers.LocalStackContainer, name string) *snsTestFixture {
	t.Helper()

	ctx := context.Background()

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

	snsClient := sns.NewFromConfig(awsCfg)
	sqsClient := sqs.NewFromConfig(awsCfg)

	topicResp, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String(fmt.Sprintf("test-%s", name)),
	})
	require.NoError(t, err)
	topicArn := *topicResp.TopicArn

	queueResp, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(fmt.Sprintf("test-%s-queue", name)),
	})
	require.NoError(t, err)
	queueURL := *queueResp.QueueUrl

	attrResp, err := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	require.NoError(t, err)
	queueArn := attrResp.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]

	_, err = snsClient.Subscribe(ctx, &sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		TopicArn: aws.String(topicArn),
		Endpoint: aws.String(queueArn),
	})
	require.NoError(t, err)

	return &snsTestFixture{
		topicArn:  topicArn,
		queueURL:  queueURL,
		snsClient: snsClient,
		sqsClient: sqsClient,
		awsCfg:    awsCfg,
	}
}

func TestSNSPublisherIntegration(t *testing.T) {
	// Start LocalStack container (shared across subtests)
	localStack := containers.StartLocalStack(t)

	t.Run("PublishSingleMessage", func(t *testing.T) {
		f := setupSNSFixture(t, localStack, "single-message")
		ctx := context.Background()

		cfg := &Configuration{
			Enabled:     true,
			Region:      f.awsCfg.Region,
			TopicArn:    f.topicArn,
			TopicPrefix: "solar",
		}

		publisher, err := newPublisherWithConfig(cfg, "solar", &f.awsCfg)
		require.NoError(t, err)
		defer publisher.Close()

		topicSuffix := "controller-1/epever/battery-voltage"
		payload := `{"value": 12.8, "unit": "volts", "timestamp": 1699000000}`

		publisher.Publish(topicSuffix, payload)

		msgResp, err := f.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(f.queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     5,
		})
		require.NoError(t, err)
		require.Len(t, msgResp.Messages, 1)

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

		assert.Equal(t, "Notification", snsNotification.Type)
		assert.Equal(t, f.topicArn, snsNotification.TopicArn)
		assert.Equal(t, "solar/controller-1/epever/battery-voltage", snsNotification.Subject)
		assert.Equal(t, payload, snsNotification.Message)

		var metricPayload map[string]interface{}
		err = json.Unmarshal([]byte(snsNotification.Message), &metricPayload)
		require.NoError(t, err)
		assert.Equal(t, 12.8, metricPayload["value"])
		assert.Equal(t, "volts", metricPayload["unit"])
	})

	t.Run("PublishMultipleMessages", func(t *testing.T) {
		f := setupSNSFixture(t, localStack, "multiple-messages")
		ctx := context.Background()

		cfg := &Configuration{
			Enabled:     true,
			Region:      f.awsCfg.Region,
			TopicArn:    f.topicArn,
			TopicPrefix: "solar",
		}

		publisher, err := newPublisherWithConfig(cfg, "solar", &f.awsCfg)
		require.NoError(t, err)
		defer publisher.Close()

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

		msgResp, err := f.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(f.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     5,
		})
		require.NoError(t, err)
		assert.Len(t, msgResp.Messages, 3, "should receive all 3 messages")

		subjects := make(map[string]string)
		for _, msg := range msgResp.Messages {
			var snsNotification struct {
				Subject string `json:"Subject"`
				Message string `json:"Message"`
			}
			err = json.Unmarshal([]byte(*msg.Body), &snsNotification)
			require.NoError(t, err)
			subjects[snsNotification.Subject] = snsNotification.Message
		}

		assert.Contains(t, subjects, "solar/controller-1/epever/array-voltage")
		assert.Contains(t, subjects, "solar/controller-1/epever/battery-soc")
		assert.Contains(t, subjects, "solar/controller-1/epever/charging-power")
	})

	t.Run("CustomTopicPrefix", func(t *testing.T) {
		f := setupSNSFixture(t, localStack, "custom-prefix")
		ctx := context.Background()

		customCfg := &Configuration{
			Enabled:     true,
			Region:      f.awsCfg.Region,
			TopicArn:    f.topicArn,
			TopicPrefix: "custom-prefix",
		}

		customPublisher, err := newPublisherWithConfig(customCfg, "default-prefix", &f.awsCfg)
		require.NoError(t, err)
		defer customPublisher.Close()

		topicSuffix := "controller-1/epever/device-temp"
		payload := `{"value": 35, "unit": "celsius", "timestamp": 1699000004}`

		customPublisher.Publish(topicSuffix, payload)

		msgResp, err := f.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(f.queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     5,
		})
		require.NoError(t, err)
		require.Len(t, msgResp.Messages, 1)

		var snsNotification struct {
			Subject string `json:"Subject"`
		}
		err = json.Unmarshal([]byte(*msgResp.Messages[0].Body), &snsNotification)
		require.NoError(t, err)

		assert.Equal(t, "custom-prefix/controller-1/epever/device-temp", snsNotification.Subject)
	})
}
