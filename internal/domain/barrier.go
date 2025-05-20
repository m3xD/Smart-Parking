package domain

import "time"

type BarrierState string

const (
	StateOpenedCommand BarrierState = "opened_command"
	StateClosedCommand BarrierState = "closed_command"
	StateOpenedAuto    BarrierState = "opened_auto"
	StateClosedAuto    BarrierState = "closed_auto"
	StateError         BarrierState = "error"
	StateUnknown       BarrierState = "unknown" // Trạng thái không xác định
	StateClosed        BarrierState = "closed"  // Trạng thái đóng
)

type Barrier struct {
	ID                    int          `json:"id"`
	LotID                 int          `json:"lot_id"`
	BarrierIdentifier     string       `json:"barrier_identifier"` // Ví dụ: "entry_barrier_1" (khớp với barrier_id từ ESP32)
	Esp32ThingName        string       `json:"esp32_thing_name"`   // Thing Name của ESP32 điều khiển
	BarrierType           string       `json:"barrier_type"`       // "entry" hoặc "exit"
	CurrentState          BarrierState `json:"current_state"`
	LastStateUpdateSource string       `json:"last_state_update_source,omitempty"`
	LastCommandSent       string       `json:"last_command_sent,omitempty"` // "open", "close"
	LastCommandTimestamp  *time.Time   `json:"last_command_timestamp,omitempty"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
}

type BarrierDTO struct {
	LotID             int    `json:"lot_id" binding:"required"`
	BarrierIdentifier string `json:"barrier_identifier" binding:"required"`
	Esp32ThingName    string `json:"esp32_thing_name" binding:"required"`
	BarrierType       string `json:"barrier_type" binding:"required,oneof=entry exit"`
	CurrentState      string `json:"current_state,omitempty"`
}
