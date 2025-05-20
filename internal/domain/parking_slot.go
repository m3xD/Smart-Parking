package domain

import "time"

type SlotStatus string

const (
	StatusVacant      SlotStatus = "vacant"
	StatusOccupied    SlotStatus = "occupied"
	StatusMaintenance SlotStatus = "maintenance"
	StatusReserved    SlotStatus = "reserved"
)

type ParkingSlot struct {
	ID                     int        `json:"id"`
	LotID                  int        `json:"lot_id"`
	SlotIdentifier         string     `json:"slot_identifier"`
	Esp32ThingName         string     `json:"esp32_thing_name,omitempty"`
	Status                 SlotStatus `json:"status"`
	LastStatusUpdateSource string     `json:"last_status_update_source,omitempty"`
	LastEventTimestamp     *time.Time `json:"last_event_timestamp,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type ParkingSlotDTO struct {
	LotID          int    `json:"lot_id" binding:"required"`
	SlotIdentifier string `json:"slot_identifier" binding:"required"`
	Esp32ThingName string `json:"esp32_thing_name"`
	Status         string `json:"status,omitempty"`
}
