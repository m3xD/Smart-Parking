package service

//
//import (
//	"context"
//	"encoding/json"
//	"fmt"
//	"log"
//	"smart_parking/internal/config"
//	"smart_parking/internal/domain"
//	"smart_parking/internal/repository"
//	"time"
//
//	"github.com/aws/aws-sdk-go-v2/aws"
//	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
//)
//
//type IoTService struct {
//	parkingService *ParkingService
//	iotDataClient  *iotdataplane.Client
//	cfg            *config.Config
//	eventLogRepo   repository.DeviceEventsLogRepository
//}
//
//func NewIoTService(
//	ps *ParkingService,
//	iotDataClient *iotdataplane.Client,
//	cfg *config.Config,
//	eventLogRepo repository.DeviceEventsLogRepository,
//) *IoTService {
//	return &IoTService{
//		parkingService: ps,
//		iotDataClient:  iotDataClient,
//		cfg:            cfg,
//		eventLogRepo:   eventLogRepo,
//	}
//}
//
//func (s *IoTService) HandleDeviceEvent(ctx context.Context, sqsMessageBody string) error {
//	log.Printf("IoTService: Xử lý sự kiện từ SQS: %s", sqsMessageBody)
//
//	var rawPayload json.RawMessage
//	if err := json.Unmarshal([]byte(sqsMessageBody), &rawPayload); err != nil {
//		log.Printf("Lỗi unmarshal raw payload: %v. Body: %s", err, sqsMessageBody)
//		if s.eventLogRepo != nil {
//			logEntry := &domain.DeviceEventLog{
//				ReceivedAt:      time.Now().UTC(),
//				Payload:         json.RawMessage(sqsMessageBody),
//				ProcessedStatus: "error",
//				ProcessingNotes: fmt.Sprintf("Failed to unmarshal raw payload: %v", err),
//			}
//			s.eventLogRepo.Create(context.Background(), logEntry)
//		}
//		return fmt.Errorf("lỗi unmarshal raw payload: %w", err)
//	}
//
//	var genericEvent domain.GenericIoTEvent
//	if err := json.Unmarshal(rawPayload, &genericEvent); err != nil {
//		log.Printf("Lỗi unmarshal generic IoT event: %v. Body: %s", err, sqsMessageBody)
//		if s.eventLogRepo != nil {
//			logEntry := &domain.DeviceEventLog{
//				ReceivedAt:      time.Now().UTC(),
//				Esp32ThingName:  genericEvent.ClientIDFromIoT,
//				MqttTopic:       genericEvent.ReceivedMqttTopic,
//				Payload:         rawPayload,
//				ProcessedStatus: "error",
//				ProcessingNotes: fmt.Sprintf("Failed to unmarshal generic event: %v", err),
//			}
//			s.eventLogRepo.Create(context.Background(), logEntry)
//		}
//		return err
//	}
//	genericEvent.RawPayload = rawPayload
//
//	logEntry := &domain.DeviceEventLog{
//		ReceivedAt:      time.Now().UTC(),
//		Esp32ThingName:  genericEvent.ClientIDFromIoT, // Sử dụng ClientIDFromIoT vì nó được thêm bởi Rule
//		MqttTopic:       genericEvent.ReceivedMqttTopic,
//		MessageType:     genericEvent.MessageType,
//		Payload:         genericEvent.RawPayload,
//		ProcessedStatus: "pending",
//	}
//	if s.eventLogRepo != nil {
//		if err := s.eventLogRepo.Create(context.Background(), logEntry); err != nil {
//			log.Printf("Lỗi khi ghi log sự kiện vào DB (pending): %v", err)
//		}
//	}
//
//	var processingError error
//	// var processingNotes string
//
//	switch genericEvent.MessageType {
//	case "startup":
//		//var event domain.DeviceStartupInfoEvent
//		//if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//		//	event.GenericIoTEvent = genericEvent // Chép các trường chung
//		//	processingError = s.parkingService.HandleDeviceStartup(ctx, event)
//		//} else {
//		//	processingError = fmt.Errorf("lỗi unmarshal startup event: %w", err)
//		//}
//	case "barrier_state":
//		var event domain.DeviceBarrierStateEvent
//		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//			event.GenericIoTEvent = genericEvent
//			processingError = s.parkingService.UpdateBarrierStateFromDevice(ctx, event)
//		} else {
//			processingError = fmt.Errorf("lỗi unmarshal barrier_state event: %w", err)
//		}
//	case "gate_event":
//		var event domain.DeviceGateSensorEvent
//		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//			event.GenericIoTEvent = genericEvent
//			processingError = s.parkingService.RecordGateSensorEvent(ctx, event)
//		} else {
//			processingError = fmt.Errorf("lỗi unmarshal gate_event: %w", err)
//		}
//	case "slot_status":
//		var event domain.DeviceParkingSlotEvent
//		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//			event.GenericIoTEvent = genericEvent
//			processingError = s.parkingService.UpdateParkingSlotStatusFromDevice(ctx, event)
//		} else {
//			processingError = fmt.Errorf("lỗi unmarshal slot_status event: %w", err)
//		}
//	case "parking_summary":
//		var event domain.DeviceParkingSummaryEvent
//		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//			event.GenericIoTEvent = genericEvent
//			processingError = s.parkingService.HandleParkingSummary(ctx, event)
//		} else {
//			processingError = fmt.Errorf("lỗi unmarshal parking_summary event: %w", err)
//		}
//	case "system_status":
//		//var event domain.DeviceSystemStatusEvent
//		//if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//		//	event.GenericIoTEvent = genericEvent
//		//	processingError = s.parkingService.HandleSystemStatus(ctx, event)
//		//} else {
//		//	processingError = fmt.Errorf("lỗi unmarshal system_status event: %w", err)
//		//}
//	case "error":
//		var event domain.DeviceErrorEvent
//		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//			event.GenericIoTEvent = genericEvent
//			processingError = s.parkingService.HandleDeviceError(ctx, event)
//		} else {
//			processingError = fmt.Errorf("lỗi unmarshal error event: %w", err)
//		}
//	case "command_acknowledgement":
//		var event domain.DeviceCommandAckEvent
//		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//			event.GenericIoTEvent = genericEvent
//			processingError = s.parkingService.HandleCommandAck(ctx, event)
//		} else {
//			processingError = fmt.Errorf("lỗi unmarshal command_ack event: %w", err)
//		}
//	default:
//		//if strings.Contains(genericEvent.ReceivedMqttTopic, "smart_parking/command/barriers") {
//		//	//var event domain.DeviceCommandAckEvent
//		//	//if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
//		//	//	event.GenericIoTEvent = genericEvent
//		//	//	processingError = s.parkingService.HandleCommandAck(ctx, event)
//		//	//} else {
//		//	//	processingError = fmt.Errorf("lỗi unmarshal command_ack (từ topic) event: %w", err)
//		//	//}
//		//} else {
//		//	//processingError = fmt.Errorf("loại tin nhắn không xác định: '%s' từ topic '%s'", genericEvent.MessageType, genericEvent.ReceivedMqttTopic)
//		//}
//	}
//
//	if s.eventLogRepo != nil && logEntry.ID != 0 { // Chỉ cập nhật nếu đã tạo log entry thành công
//		if processingError != nil {
//			logEntry.ProcessedStatus = "error"
//			logEntry.ProcessingNotes = processingError.Error()
//		} else {
//			logEntry.ProcessedStatus = "processed"
//			logEntry.ProcessingNotes = "Successfully processed"
//		}
//		// TODO: Cần hàm Update trong DeviceEventsLogRepository để cập nhật log entry
//		//errUpdateLog := s.eventLogRepo.Update(context.Background(), logEntry)
//		//if errUpdateLog != nil {
//		//    log.Printf("Lỗi khi cập nhật log sự kiện ID %d: %v", logEntry.ID, errUpdateLog)
//		//}
//	} else if processingError != nil {
//		log.Printf("Lỗi khi xử lý sự kiện loại '%s' (Device: %s, Topic: %s): %v",
//			genericEvent.MessageType, genericEvent.ClientIDFromIoT, genericEvent.ReceivedMqttTopic, processingError)
//	}
//
//	return processingError
//}
//
//func (s *IoTService) SendBarrierControlCommand(ctx context.Context, esp32ControllerID string, barrierType string, command string, requestID string) error {
//	topic := fmt.Sprintf("smart_parking/command/barriers/%s", barrierType)
//
//	payload := domain.BarrierControlCommandPayload{
//		Command:   command,
//		RequestID: requestID,
//	}
//	payloadBytes, err := json.Marshal(payload)
//	if err != nil {
//		return fmt.Errorf("lỗi marshal payload lệnh rào chắn: %w", err)
//	}
//
//	log.Printf("IoTService: Đang publish lệnh '%s' (ReqID: %s) tới topic %s cho ESP32 %s", command, requestID, topic, esp32ControllerID)
//	_, err = s.iotDataClient.Publish(ctx, &iotdataplane.PublishInput{
//		Topic:   aws.String(topic),
//		Qos:     1,
//		Payload: payloadBytes,
//	})
//	if err != nil {
//		return fmt.Errorf("lỗi publish lệnh MQTT: %w", err)
//	}
//
//	log.Printf("Đã gửi lệnh '%s' (ReqID: %s) thành công tới rào chắn %s của ESP32 %s", command, requestID, barrierType, esp32ControllerID)
//	// TODO: Cập nhật trạng thái lệnh là "sent" trong DB (ví dụ, trong bảng command_log)
//	// Cập nhật last_command_sent và last_command_timestamp cho barrier tương ứng
//	barrier, err := s.parkingService.barrierRepo.FindByThingAndBarrierIdentifier(ctx, esp32ControllerID, fmt.Sprintf("%s_%s", esp32ControllerID, barrierType)) // Giả sử barrier_identifier có dạng này
//	if err == nil && barrier != nil {
//		now := time.Now().UTC()
//		err := s.parkingService.barrierRepo.UpdateState(ctx, barrier.ID, barrier.CurrentState, command, &now, "server_command_sent")
//		if err != nil {
//			log.Printf("Lỗi khi cập nhật trạng thái lệnh cho rào chắn %s: %v", barrierType, err)
//		}
//	} else {
//		log.Printf("Không tìm thấy rào chắn %s cho ESP32 %s để cập nhật last_command_sent", barrierType, esp32ControllerID)
//	}
//
//	return nil
//}
