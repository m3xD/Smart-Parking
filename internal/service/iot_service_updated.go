// File: internal/service/iot_service.go - ENHANCED VERSION
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"smart_parking/internal/config"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
)

// Interface cho WebSocket Manager để tránh circular dependency
type WebSocketManager interface {
	BroadcastGateEvent(event domain.GateEventNotification)
}

type IoTService struct {
	parkingService   *ParkingService
	iotDataClient    *iotdataplane.Client
	cfg              *config.Config
	eventLogRepo     repository.DeviceEventsLogRepository
	gateEventRepo    repository.GateEventRepository // NEW
	webSocketManager WebSocketManager               // NEW
}

func NewIoTService(
	ps *ParkingService,
	iotDataClient *iotdataplane.Client,
	cfg *config.Config,
	eventLogRepo repository.DeviceEventsLogRepository,
) *IoTService {
	return &IoTService{
		parkingService: ps,
		iotDataClient:  iotDataClient,
		cfg:            cfg,
		eventLogRepo:   eventLogRepo,
	}
}

// NEW: Constructor với Gate Event support
func NewIoTServiceUpdated(
	ps *ParkingService,
	iotDataClient *iotdataplane.Client,
	cfg *config.Config,
	eventLogRepo repository.DeviceEventsLogRepository,
	gateEventRepo repository.GateEventRepository,
	wsManager WebSocketManager,
) *IoTService {
	return &IoTService{
		parkingService:   ps,
		iotDataClient:    iotDataClient,
		cfg:              cfg,
		eventLogRepo:     eventLogRepo,
		gateEventRepo:    gateEventRepo,
		webSocketManager: wsManager,
	}
}

func (s *IoTService) HandleDeviceEvent(ctx context.Context, sqsMessageBody string) error {
	log.Printf("IoTService: Xử lý sự kiện từ SQS: %s", sqsMessageBody)

	var rawPayload json.RawMessage
	if err := json.Unmarshal([]byte(sqsMessageBody), &rawPayload); err != nil {
		log.Printf("Lỗi unmarshal raw payload: %v. Body: %s", err, sqsMessageBody)
		if s.eventLogRepo != nil {
			logEntry := &domain.DeviceEventLog{
				ReceivedAt:      time.Now().UTC(),
				Payload:         json.RawMessage(sqsMessageBody),
				ProcessedStatus: "error",
				ProcessingNotes: fmt.Sprintf("Failed to unmarshal raw payload: %v", err),
			}
			s.eventLogRepo.Create(context.Background(), logEntry)
		}
		return fmt.Errorf("lỗi unmarshal raw payload: %w", err)
	}

	var genericEvent domain.GenericIoTEvent
	if err := json.Unmarshal(rawPayload, &genericEvent); err != nil {
		log.Printf("Lỗi unmarshal generic IoT event: %v. Body: %s", err, sqsMessageBody)
		if s.eventLogRepo != nil {
			logEntry := &domain.DeviceEventLog{
				ReceivedAt:      time.Now().UTC(),
				Esp32ThingName:  genericEvent.ClientIDFromIoT,
				MqttTopic:       genericEvent.ReceivedMqttTopic,
				Payload:         rawPayload,
				ProcessedStatus: "error",
				ProcessingNotes: fmt.Sprintf("Failed to unmarshal generic event: %v", err),
			}
			s.eventLogRepo.Create(context.Background(), logEntry)
		}
		return err
	}
	genericEvent.RawPayload = rawPayload

	logEntry := &domain.DeviceEventLog{
		ReceivedAt:      time.Now().UTC(),
		Esp32ThingName:  genericEvent.ClientIDFromIoT,
		MqttTopic:       genericEvent.ReceivedMqttTopic,
		MessageType:     genericEvent.MessageType,
		Payload:         genericEvent.RawPayload,
		ProcessedStatus: "pending",
	}
	if s.eventLogRepo != nil {
		if err := s.eventLogRepo.Create(context.Background(), logEntry); err != nil {
			log.Printf("Lỗi khi ghi log sự kiện vào DB (pending): %v", err)
		}
	}

	var processingError error

	switch genericEvent.MessageType {
	case "startup":
		//Handle device startup events
		var event domain.DeviceStartupInfoEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.HandleDeviceStartup(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal startup event: %w", err)
		}
	case "barrier_state":
		var event domain.DeviceBarrierStateEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.UpdateBarrierStateFromDevice(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal barrier_state event: %w", err)
		}

	case "gate_event":
		// NEW: Enhanced gate event processing
		var event domain.DeviceGateSensorEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.handleGateEventEnhanced(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal gate_event: %w", err)
		}

	case "slot_status":
		var event domain.DeviceParkingSlotEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.UpdateParkingSlotStatusFromDevice(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal slot_status event: %w", err)
		}

	case "parking_summary":
		var event domain.DeviceParkingSummaryEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.HandleParkingSummary(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal parking_summary event: %w", err)
		}

	case "system_status":
		var event domain.DeviceSystemStatusEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.HandleSystemStatus(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal system_status event: %w", err)
		}
	case "error":
		var event domain.DeviceErrorEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.HandleDeviceError(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal error event: %w", err)
		}

	case "command_acknowledgement":
		var event domain.DeviceCommandAckEvent
		if err := json.Unmarshal(genericEvent.RawPayload, &event); err == nil {
			event.GenericIoTEvent = genericEvent
			processingError = s.parkingService.HandleCommandAck(ctx, event)
		} else {
			processingError = fmt.Errorf("lỗi unmarshal command_ack event: %w", err)
		}

	default:
		log.Printf("IoTService: Loại message không được xử lý: '%s'", genericEvent.MessageType)
		processingError = nil // Không coi là lỗi, chỉ log
	}

	if processingError != nil {
		log.Printf("Lỗi khi xử lý sự kiện loại '%s' (Device: %s, Topic: %s): %v",
			genericEvent.MessageType, genericEvent.ClientIDFromIoT, genericEvent.ReceivedMqttTopic, processingError)
	}

	return processingError
}

// NEW: Enhanced gate event processing với WebSocket notification
func (s *IoTService) handleGateEventEnhanced(ctx context.Context, event domain.DeviceGateSensorEvent) error {
	log.Printf("IoTService: Xử lý gate event: Device='%s', Sensor='%s', Area='%s', EventType='%s'",
		event.DeviceID, event.SensorID, event.GateArea, event.EventType)

	// Backward compatibility: vẫn gọi existing logic
	err := s.parkingService.RecordGateSensorEvent(ctx, event)
	if err != nil {
		log.Printf("Lỗi trong RecordGateSensorEvent: %v", err)
	}

	// NEW: Enhanced processing với WebSocket notification
	if s.gateEventRepo != nil && s.webSocketManager != nil {
		return s.processGateEventWithNotification(ctx, event)
	}

	return err
}

func (s *IoTService) processGateEventWithNotification(ctx context.Context, event domain.DeviceGateSensorEvent) error {
	// Chỉ xử lý các events quan trọng cần intervention
	if !s.shouldTriggerNotification(event) {
		log.Printf("Gate event không cần notification: %s", event.EventType)
		return nil
	}

	// Tìm thông tin lot từ device
	device, err := s.parkingService.GetDeviceByThingName(ctx, event.DeviceID)
	if err != nil {
		log.Printf("Không tìm thấy device %s: %v", event.DeviceID, err)
		return fmt.Errorf("device not found: %w", err)
	}

	var lotID int
	var lotName string
	if device.LotID.Valid {
		lotID = int(device.LotID.Int64)
		lot, err := s.parkingService.GetParkingLotByID(ctx, lotID)
		if err == nil {
			lotName = lot.Name
		}
	} else {
		// Fallback: tìm từ barriers
		barriers, _ := s.parkingService.barrierRepo.FindByThingName(ctx, event.DeviceID)
		if len(barriers) > 0 {
			lotID = barriers[0].LotID
			lot, _ := s.parkingService.GetParkingLotByID(ctx, lotID)
			if lot != nil {
				lotName = lot.Name
			}
		}
	}

	if lotID == 0 {
		return fmt.Errorf("không thể xác định lot_id cho device %s", event.DeviceID)
	}

	// Tạo gate event record
	eventRecord := &domain.GateEventRecord{
		EventID:       event.EventID,
		LotID:         lotID,
		DeviceID:      event.DeviceID,
		SensorID:      event.SensorID,
		GateDirection: s.determineGateDirection(event),
		EventType:     s.mapEventType(event),
		Status:        domain.StatusPending,
		ExpiresAt:     timePtr(time.Now().Add(time.Duration(s.cfg.GateEventTimeoutMinutes) * time.Minute)),
	}

	// Lưu vào DB
	err = s.gateEventRepo.Create(ctx, eventRecord)
	if err != nil {
		log.Printf("Lỗi lưu gate event record: %v", err)
		return err
	}

	// Tạo notification cho frontend
	notification := domain.GateEventNotification{
		EventID:           event.EventID,
		LotID:             lotID,
		LotName:           lotName,
		DeviceID:          event.DeviceID,
		GateDirection:     eventRecord.GateDirection,
		EventType:         eventRecord.EventType,
		Timestamp:         time.Now(),
		SensorID:          event.SensorID,
		RequiresLPR:       s.requiresLPR(event),
		RequiresUserInput: s.requiresUserInput(event),
		Message:           s.generateUserMessage(event, lotName),
		SuggestedCameraID: s.getSuggestedCameraID(event),
	}

	// Push đến frontend
	s.webSocketManager.BroadcastGateEvent(notification)

	// Cập nhật status
	s.gateEventRepo.UpdateStatus(ctx, eventRecord.EventID, domain.StatusAwaitingLPR, "")

	log.Printf("Đã gửi gate event notification: EventID=%s, LotID=%d", event.EventID, lotID)
	return nil
}

// Helper methods cho gate event processing
func (s *IoTService) shouldTriggerNotification(event domain.DeviceGateSensorEvent) bool {
	return event.EventType == "vehicle_at_gate" ||
		event.EventType == "presence_detected" ||
		(event.EventType == "vehicle_passed" && event.IsEntryArea)
}

func (s *IoTService) determineGateDirection(event domain.DeviceGateSensorEvent) domain.GateDirection {
	if event.IsEntryArea ||
		event.GateArea == "entry_approach" ||
		event.GateArea == "entry_passed" {
		return domain.GateDirectionEntry
	}
	return domain.GateDirectionExit
}

func (s *IoTService) mapEventType(event domain.DeviceGateSensorEvent) domain.GateEventType {
	switch event.EventType {
	case "presence_detected":
		return domain.GateEventVehicleApproaching
	case "vehicle_at_gate":
		return domain.GateEventVehicleAtGate
	case "vehicle_passed":
		return domain.GateEventVehiclePassed
	default:
		return domain.GateEventVehicleAtGate
	}
}

func (s *IoTService) requiresLPR(event domain.DeviceGateSensorEvent) bool {
	return event.IsEntryArea || event.GateArea == "entry_approach"
}

func (s *IoTService) requiresUserInput(event domain.DeviceGateSensorEvent) bool {
	return false // Có thể cấu hình dựa trên policy
}

func (s *IoTService) generateUserMessage(event domain.DeviceGateSensorEvent, lotName string) string {
	if event.IsEntryArea {
		return fmt.Sprintf("Xe đang đến cổng vào bãi %s. Vui lòng chụp ảnh biển số.", lotName)
	}
	return fmt.Sprintf("Xe đang đến cổng ra bãi %s.", lotName)
}

func (s *IoTService) getSuggestedCameraID(event domain.DeviceGateSensorEvent) string {
	if event.IsEntryArea {
		return "entry_camera_1"
	}
	return "exit_camera_1"
}

// NEW: ProcessLPRResult - Xử lý kết quả LPR từ frontend
func (s *IoTService) ProcessLPRResult(ctx context.Context, request domain.LPRTriggerRequest, detectedPlate string, confidence float32) error {
	if s.gateEventRepo == nil {
		return fmt.Errorf("gate event repository not configured")
	}

	log.Printf("IoTService: Xử lý kết quả LPR cho EventID=%s, Plate=%s, Confidence=%.2f",
		request.EventID, detectedPlate, confidence)

	// Cập nhật gate event record
	err := s.gateEventRepo.UpdateLPRResult(ctx, request.EventID, detectedPlate, confidence)
	if err != nil {
		return fmt.Errorf("lỗi cập nhật LPR result: %w", err)
	}

	// Tự động tạo session nếu confidence đủ cao hoặc có manual override
	if confidence >= s.cfg.LPRConfidenceThreshold || request.ManualOverride != "" {
		return s.autoCreateSession(ctx, request.EventID, detectedPlate, request.ManualOverride != "")
	}

	log.Printf("Confidence thấp (%.2f), chờ manual confirmation", confidence)
	return nil
}

func (s *IoTService) autoCreateSession(ctx context.Context, eventID string, plate string, isManual bool) error {
	// Lấy gate event record
	gateEvent, err := s.gateEventRepo.FindByEventID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("không tìm thấy gate event: %w", err)
	}

	// Chỉ tạo session cho entry events
	if gateEvent.GateDirection != domain.GateDirectionEntry {
		log.Printf("Không tạo session cho exit event: %s", eventID)
		return nil
	}

	// Tạo parking session
	sessionDTO := domain.VehicleCheckInDTO{
		LotID:             gateEvent.LotID,
		Esp32ThingName:    gateEvent.DeviceID,
		VehicleIdentifier: plate,
		EntryTime:         time.Now().Format(time.RFC3339),
	}

	session, err := s.parkingService.VehicleCheckIn(ctx, sessionDTO)
	if err != nil {
		s.gateEventRepo.UpdateStatus(ctx, eventID, domain.StatusError, err.Error())
		return fmt.Errorf("lỗi tạo parking session: %w", err)
	}

	// Cập nhật gate event với session ID
	err = s.gateEventRepo.UpdateWithSession(ctx, eventID, session.ID)
	if err != nil {
		log.Printf("Lỗi cập nhật gate event với session ID: %v", err)
	}

	log.Printf("Đã tạo parking session ID=%d cho gate event=%s", session.ID, eventID)
	return nil
}

// Existing barrier control method
func (s *IoTService) SendBarrierControlCommand(ctx context.Context, esp32ControllerID string, barrierType string, command string, requestID string) error {
	topic := fmt.Sprintf("smart_parking/command/barriers/%s", barrierType)

	payload := domain.BarrierControlCommandPayload{
		Command:   command,
		RequestID: requestID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("lỗi marshal payload lệnh rào chắn: %w", err)
	}

	log.Printf("IoTService: Đang publish lệnh '%s' (ReqID: %s) tới topic %s cho ESP32 %s", command, requestID, topic, esp32ControllerID)
	_, err = s.iotDataClient.Publish(ctx, &iotdataplane.PublishInput{
		Topic:   aws.String(topic),
		Qos:     1,
		Payload: payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("lỗi publish lệnh MQTT: %w", err)
	}

	log.Printf("Đã gửi lệnh '%s' (ReqID: %s) thành công tới rão chắn %s của ESP32 %s", command, requestID, barrierType, esp32ControllerID)
	return nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
