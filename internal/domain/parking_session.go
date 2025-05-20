package domain

import (
	"gopkg.in/guregu/null.v4"
	"time"
)

type ParkingSessionStatus string

const (
	SessionActive    ParkingSessionStatus = "active"
	SessionCompleted ParkingSessionStatus = "completed"
	SessionCancelled ParkingSessionStatus = "cancelled" // Ví dụ: nếu admin hủy
)

type ParkingSession struct {
	ID                int                  `json:"id"`
	LotID             int                  `json:"lot_id"`
	SlotID            null.Int             `json:"slot_id"`
	Esp32ThingName    string               `json:"esp32_thing_name"`
	VehicleIdentifier null.String          `json:"vehicle_identifier"` // Sẽ dùng cho LPR
	EntryTime         time.Time            `json:"entry_time"`
	ExitTime          null.Time            `json:"exit_time"`
	DurationMinutes   null.Int             `json:"duration_minutes"`
	CalculatedFee     null.Float           `json:"calculated_fee"`
	PaymentStatus     string               `json:"payment_status"` // "pending", "paid", "failed", "waived"
	Status            ParkingSessionStatus `json:"status"`
	EntryGateEventID  null.String          `json:"entry_gate_event_id,omitempty"`
	ExitGateEventID   null.String          `json:"exit_gate_event_id,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`

	ParkingLot  *ParkingLot  `json:"parking_lot,omitempty" gorm:"-"`  // Không map vào DB, dùng để trả về API
	ParkingSlot *ParkingSlot `json:"parking_slot,omitempty" gorm:"-"` // Không map vào DB
}

type CreateParkingSessionDTO struct {
	LotID             int       `json:"lot_id" binding:"required"`
	SlotID            *int      `json:"slot_id"`
	Esp32ThingName    string    `json:"esp32_thing_name" binding:"required"`
	VehicleIdentifier string    `json:"vehicle_identifier,omitempty"`
	EntryTime         time.Time `json:"entry_time" binding:"required"`
	EntryGateEventID  string    `json:"entry_gate_event_id,omitempty"`
}

type EndParkingSessionDTO struct {
	ExitTime        time.Time `json:"exit_time" binding:"required"`
	ExitGateEventID string    `json:"exit_gate_event_id,omitempty"`
	// Có thể cần sessionID hoặc vehicleIdentifier để xác định phiên cần kết thúc
}

type ParkingSessionFilterDTO struct {
	LotID  *int    `form:"lotId"`
	Status *string `form:"status"`
	// Thêm các trường filter khác nếu cần (ví dụ: ngày, vehicle_id)
}

// DTO cho API Check-in (frontend gửi lên)
type VehicleCheckInDTO struct {
	LotID             int    `json:"lot_id" binding:"required"`
	Esp32ThingName    string `json:"esp32_thing_name" binding:"required"`
	VehicleIdentifier string `json:"vehicle_identifier" binding:"required"`
	EntryTime         string `json:"entry_time,omitempty"`
	// EntryImageBase64  string `json:"entry_image_base64,omitempty"` // Bỏ qua nếu LPR đã xử lý ở frontend hoặc 1 API riêng
}

// DTO cho API Check-out (frontend gửi lên)
type VehicleCheckOutDTO struct {
	LotID             int    `json:"lot_id" binding:"required"`
	Esp32ThingName    string `json:"esp32_thing_name" binding:"required"`
	VehicleIdentifier string `json:"vehicle_identifier" binding:"required"`
	ExitTime          string `json:"exit_time,omitempty"`
	// ExitImageBase64   string `json:"exit_image_base64,omitempty"`
}
