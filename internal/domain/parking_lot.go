package domain

import "time"

type ParkingLot struct {
	ID         int       `json:"id"`
	Name       string    `json:"name" binding:"required"`
	Address    string    `json:"address,omitempty"`
	TotalSlots int       `json:"total_slots,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ParkingLotDTO struct {
	Name       string `json:"name" binding:"required"`
	Address    string `json:"address"`
	TotalSlots int    `json:"total_slots"`
}
