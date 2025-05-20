package iot

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"log"
	"smart_parking/internal/config"
	"smart_parking/internal/service"
	"time"
)

type SQSConsumer struct {
	sqsClient  *sqs.Client
	queueURL   string
	iotService *service.IoTService
}

func NewSQSConsumer(client *sqs.Client, cfg *config.Config, iotService *service.IoTService) *SQSConsumer {
	return &SQSConsumer{
		sqsClient:  client,
		queueURL:   cfg.SQSEventQueueURL,
		iotService: iotService,
	}
}

func (c *SQSConsumer) Start(ctx context.Context) {
	log.Printf("SQS Consumer đang bắt đầu lắng nghe queue: %s", c.queueURL)
	for {
		select {
		case <-ctx.Done():
			log.Println("SQS Consumer: context cancelled, stopping.")
			return
		default:
			receiveInput := &sqs.ReceiveMessageInput{
				QueueUrl:            &c.queueURL,
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20,
				VisibilityTimeout:   60,
			}

			result, err := c.sqsClient.ReceiveMessage(ctx, receiveInput)
			if err != nil {
				log.Printf("SQS Consumer: Lỗi khi nhận message: %v", err)
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					log.Println("SQS Consumer: context cancelled while waiting for retry.")
					return
				}
				continue
			}

			if len(result.Messages) == 0 {
				continue
			}

			log.Printf("SQS Consumer: Đã nhận %d message(s)", len(result.Messages))

			for _, message := range result.Messages {
				if message.Body == nil {
					log.Println("SQS Consumer: Nhận được message với body rỗng. Đang xóa...")
					c.deleteMessage(ctx, message.ReceiptHandle)
					continue
				}

				processingErr := c.iotService.HandleDeviceEvent(ctx, *message.Body)

				if processingErr == nil {
					c.deleteMessage(ctx, message.ReceiptHandle)
				} else {
					log.Printf("SQS Consumer: Lỗi khi xử lý message ID %s: %v. Message sẽ được xử lý lại sau visibility timeout.", *message.MessageId, processingErr)
				}
			}
		}
	}
}

func (c *SQSConsumer) deleteMessage(ctx context.Context, receiptHandle *string) {
	if receiptHandle == nil {
		log.Println("SQS Consumer: Receipt handle rỗng, không thể xóa message.")
		return
	}
	_, delErr := c.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &c.queueURL,
		ReceiptHandle: receiptHandle,
	})
	if delErr != nil {
		log.Printf("SQS Consumer: Lỗi khi xóa message: %v", delErr)
	}
}
