// File: internal/domain/gate_notifications.go
package domain

import "time"

type GateEventType string

const (
	GateEventVehicleApproaching GateEventType = "vehicle_approaching"
	GateEventVehicleAtGate      GateEventType = "vehicle_at_gate"
	GateEventVehiclePassed      GateEventType = "vehicle_passed"
	GateEventGateTimeout        GateEventType = "gate_timeout"
)

type GateDirection string

const (
	GateDirectionEntry GateDirection = "entry"
	GateDirectionExit  GateDirection = "exit"
)

// GateEventNotification - Event được gửi đến frontend qua WebSocket
type GateEventNotification struct {
	EventID       string        `json:"event_id"`
	LotID         int           `json:"lot_id"`
	LotName       string        `json:"lot_name"`
	DeviceID      string        `json:"device_id"`
	GateDirection GateDirection `json:"gate_direction"` // "entry" hoặc "exit"
	EventType     GateEventType `json:"event_type"`
	Timestamp     time.Time     `json:"timestamp"`
	SensorID      string        `json:"sensor_id,omitempty"`

	// Metadata bổ sung
	RequiresLPR       bool   `json:"requires_lpr"`        // Frontend cần thực hiện LPR không
	RequiresUserInput bool   `json:"requires_user_input"` // Cần input từ user không (manual override)
	Message           string `json:"message,omitempty"`   // Thông báo hiển thị cho user

	// Camera info nếu cần
	SuggestedCameraID string `json:"suggested_camera_id,omitempty"`
}

// LPRTriggerRequest - Request từ frontend để trigger LPR
type LPRTriggerRequest struct {
	EventID        string `json:"event_id" binding:"required"`
	ImageBase64    string `json:"image_base64" binding:"required"`
	CameraID       string `json:"camera_id,omitempty"`
	ManualOverride string `json:"manual_override,omitempty"` // Nếu user muốn nhập biển số manual
}

// SessionCreationRequest - Request tạo session sau khi có biển số
type SessionCreationRequest struct {
	EventID         string  `json:"event_id" binding:"required"`
	LotID           int     `json:"lot_id" binding:"required"`
	DetectedPlate   string  `json:"detected_plate" binding:"required"`
	Confidence      float32 `json:"confidence,omitempty"`
	IsManualEntry   bool    `json:"is_manual_entry,omitempty"`
	Esp32ThingName  string  `json:"esp32_thing_name" binding:"required"`
	AdditionalNotes string  `json:"additional_notes,omitempty"`
}

// GateEventStatus - Trạng thái xử lý của gate event
type GateEventStatus string

const (
	StatusPending        GateEventStatus = "pending"         // Chờ xử lý
	StatusAwaitingLPR    GateEventStatus = "awaiting_lpr"    // Chờ LPR
	StatusLPRCompleted   GateEventStatus = "lpr_completed"   // LPR xong
	StatusSessionCreated GateEventStatus = "session_created" // Đã tạo session
	StatusTimeout        GateEventStatus = "timeout"         // Timeout
	StatusError          GateEventStatus = "error"           // Lỗi
	StatusManualOverride GateEventStatus = "manual_override" // Xử lý manual
)

// GateEventRecord - Lưu trữ trong DB để track progress
type GateEventRecord struct {
	ID               int             `json:"id"`
	EventID          string          `json:"event_id"`
	LotID            int             `json:"lot_id"`
	DeviceID         string          `json:"device_id"`
	GateDirection    GateDirection   `json:"gate_direction"`
	EventType        GateEventType   `json:"event_type"`
	Status           GateEventStatus `json:"status"`
	SensorID         string          `json:"sensor_id,omitempty"`
	DetectedPlate    string          `json:"detected_plate,omitempty"`
	LPRConfidence    *float32        `json:"lpr_confidence,omitempty"`
	IsManualEntry    bool            `json:"is_manual_entry,omitempty"`
	SessionID        *int            `json:"session_id,omitempty"`
	ProcessingNotes  string          `json:"processing_notes,omitempty"`
	AssignedOperator string          `json:"assigned_operator,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"` // Timeout threshold
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
}

// GateEventStats - Thống kê gate events
type GateEventStats struct {
	TotalEvents            int     `json:"total_events"`
	CompletedEvents        int     `json:"completed_events"`
	TimeoutEvents          int     `json:"timeout_events"`
	ErrorEvents            int     `json:"error_events"`
	AvgProcessingTimeMin   float64 `json:"avg_processing_time_minutes"`
	LPRSuccessRate         float64 `json:"lpr_success_rate"`         // % events có successful LPR
	AutoSessionRate        float64 `json:"auto_session_rate"`        // % events tự động tạo session
	ManualInterventionRate float64 `json:"manual_intervention_rate"` // % events cần manual
}
