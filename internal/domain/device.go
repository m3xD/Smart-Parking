package domain

import (
	"gopkg.in/guregu/null.v4"
	"time"
)

type DeviceStatus string

const (
	DeviceOnline      DeviceStatus = "online"
	DeviceOffline     DeviceStatus = "offline"
	DeviceErrorStatus DeviceStatus = "error" // Phân biệt với domain.StateError của Barrier
	DeviceMaintenance DeviceStatus = "maintenance"
	DeviceUnknown     DeviceStatus = "unknown"
)

type Device struct {
	ID                int          `json:"id"`
	ThingName         string       `json:"thing_name"` // SECRET_AWS_THING_NAME
	LotID             null.Int     `json:"lot_id"`     // Bãi đỗ mà thiết bị này quản lý (nếu có)
	FirmwareVersion   string       `json:"firmware_version,omitempty"`
	LastSeenAt        null.Time    `json:"last_seen_at"`
	Status            DeviceStatus `json:"status"`
	IPAddress         string       `json:"ip_address,omitempty"`
	MacAddress        string       `json:"mac_address,omitempty"`
	LastRssi          null.Int     `json:"last_rssi"`
	LastFreeHeap      null.Int     `json:"last_free_heap"`
	LastUptimeSeconds null.Int     `json:"last_uptime_seconds"`
	Notes             string       `json:"notes,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`

	ParkingLot *ParkingLot `json:"parking_lot,omitempty" gorm:"-"`
}

type UpdateDeviceStatusDTO struct {
	ThingName       string
	FirmwareVersion string
	LastSeenAt      time.Time
	Status          DeviceStatus
	IPAddress       string
	MacAddress      string
	RSSI            *int
	FreeHeap        *uint32
	UptimeSeconds   *int64
}
