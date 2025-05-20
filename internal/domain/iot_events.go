package domain

import (
	"encoding/json"
	"time"
)

// GenericIoTEvent dùng để parse bước đầu, lấy message_type và các trường chung
type GenericIoTEvent struct {
	DeviceID               string          `json:"device_id"` // ESP32 gửi là "device_id" (SECRET_AWS_THING_NAME)
	MessageType            string          `json:"message_type"`
	Timestamp              string          `json:"timestamp"`                          // ISO 8601 UTC string từ ESP32
	ReceivedMqttTopic      string          `json:"received_mqtt_topic,omitempty"`      // Do IoT Rule thêm vào
	IotProcessingTimestamp int64           `json:"iot_processing_timestamp,omitempty"` // Do IoT Rule thêm vào
	ClientIDFromIoT        string          `json:"client_id_iot,omitempty"`            // Do IoT Rule thêm vào
	RawPayload             json.RawMessage `json:"-"`                                  // Để lưu payload gốc nếu cần
}

type DeviceStartupInfoEvent struct {
	GenericIoTEvent
	FirmwareVersion string `json:"firmware_version"`
	StartupReason   string `json:"startup_reason"`
	CompileDate     string `json:"compile_date"`
	CompileTime     string `json:"compile_time"`
	ChipID          string `json:"chip_id"`      // ESP.getChipId() trả về uint32_t, ESP32 code nên gửi dạng string
	FlashSize       uint32 `json:"flash_size"`   // ESP.getFlashChipSize()
	CPUFreqMHz      uint32 `json:"cpu_freq_mhz"` // ESP.getCpuFreqMHz()
	WiFi            struct {
		SSID string `json:"ssid"`
		RSSI int    `json:"rssi"`
		IP   string `json:"ip"`
		MAC  string `json:"mac"`
	} `json:"wifi"`
	Config struct {
		GateSensors  int `json:"gate_sensors"`
		ParkingSlots int `json:"parking_slots"`
	} `json:"config"`
}

type DeviceBarrierStateEvent struct {
	GenericIoTEvent
	BarrierType  string       `json:"barrier_type"`  // "entry" hoặc "exit"
	BarrierState BarrierState `json:"barrier_state"` // "opened_command", "closed_command", "opened_auto", "closed_auto"
	BarrierID    string       `json:"barrier_id"`    // Ví dụ: "ESP32_ParkingController_01_entry"
	Location     string       `json:"location,omitempty"`
	Zone         string       `json:"zone,omitempty"`
	DeviceUptime int64        `json:"device_uptime,omitempty"` // millis() / 1000
	RSSI         int          `json:"rssi,omitempty"`
}

type DeviceGateSensorEvent struct {
	GenericIoTEvent
	SensorID            string `json:"sensor_id"`  // Ví dụ: "SENSOR_VAO_1"
	GateArea            string `json:"gate_area"`  // "entry_approach", "entry_passed", ...
	EventType           string `json:"event_type"` // "presence_detected", "vehicle_passed"
	EventID             string `json:"event_id"`   // String(millis())
	Location            string `json:"location,omitempty"`
	Zone                string `json:"zone,omitempty"`
	IsEntryArea         bool   `json:"is_entry_area,omitempty"`
	RequiresAction      bool   `json:"requires_action,omitempty"`
	RelatedBarrier      string `json:"related_barrier,omitempty"`
	RelatedBarrierState string `json:"related_barrier_state,omitempty"`
}

type DeviceParkingSlotEvent struct {
	GenericIoTEvent                // device_id từ đây là ThingName, message_type là "slot_status"
	SlotID              string     `json:"slot_id"` // "S1", "S2", "S3", "S4"
	IsOccupied          bool       `json:"is_occupied"`
	Status              SlotStatus `json:"status"`     // "occupied" hoặc "available" (ESP32 gửi là "available")
	ChangedAt           string     `json:"changed_at"` // Timestamp khi trạng thái thay đổi
	Location            string     `json:"location,omitempty"`
	Floor               string     `json:"floor,omitempty"`
	Zone                string     `json:"zone,omitempty"`
	TotalSlots          int        `json:"total_slots,omitempty"`
	TotalOccupied       int        `json:"total_occupied,omitempty"`
	AvailableSlots      int        `json:"available_slots,omitempty"`
	OccupancyPercentage float64    `json:"occupancy_percentage,omitempty"`
	IsFull              bool       `json:"is_full,omitempty"`
}

type DeviceParkingSummaryEvent struct {
	GenericIoTEvent
	TotalSlots          int     `json:"total_slots"`
	OccupiedSlots       int     `json:"occupied_slots"`
	AvailableSlots      int     `json:"available_slots"`
	OccupancyPercentage float64 `json:"occupancy_percentage"`
	IsFull              bool    `json:"is_full"`
	IsEmpty             bool    `json:"is_empty"`
	EntryBarrierOpen    bool    `json:"entry_barrier_open"`
	ExitBarrierOpen     bool    `json:"exit_barrier_open"`
	Location            string  `json:"location,omitempty"`
	Slots               []struct {
		ID       string `json:"id"`
		Occupied bool   `json:"occupied"`
	} `json:"slots,omitempty"`
}

type DeviceSystemStatusEvent struct {
	GenericIoTEvent
	FirmwareVersion     string  `json:"firmware_version"`
	UptimeSeconds       int64   `json:"uptime_seconds"`
	FreeHeap            uint32  `json:"free_heap"`
	HeapFragmentation   uint8   `json:"heap_fragmentation"` // Thường là %
	CPUFreqMHz          uint32  `json:"cpu_freq_mhz"`
	WifiSSID            string  `json:"wifi_ssid"`
	WifiRSSI            int     `json:"wifi_rssi"`
	WifiIP              string  `json:"wifi_ip"`
	WifiMAC             string  `json:"wifi_mac"`
	MqttConnected       bool    `json:"mqtt_connected"`
	MqttReconnectCount  int     `json:"mqtt_reconnect_count"`
	PowerMode           string  `json:"power_mode"` // "low" hoặc "normal"
	LastActivitySecAgo  int64   `json:"last_activity_seconds_ago"`
	TotalSlots          int     `json:"total_slots"`
	OccupiedSlots       int     `json:"occupied_slots"`
	AvailableSlots      int     `json:"available_slots"`
	OccupancyPercentage float64 `json:"occupancy_percentage"`
	EntryBarrierOpen    bool    `json:"entry_barrier_open"`
	ExitBarrierOpen     bool    `json:"exit_barrier_open"`
}

type DeviceErrorEvent struct {
	GenericIoTEvent
	ErrorCode     int    `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
	ErrorID       string `json:"error_id"` // String(millis())
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`
	FreeHeap      uint32 `json:"free_heap,omitempty"`
	WifiRSSI      int    `json:"wifi_rssi,omitempty"`
	MqttConnected bool   `json:"mqtt_connected,omitempty"`
	PowerMode     string `json:"power_mode,omitempty"`
}

type DeviceCommandAckEvent struct {
	GenericIoTEvent        // message_type nên là "command_acknowledgement"
	Status          string `json:"status"` // "acknowledged"
	RequestID       string `json:"request_id,omitempty"`
	ReceivedAction  string `json:"received_action,omitempty"`
}

// --- Struct cho Lệnh Điều khiển (Gửi từ Go Backend -> ESP32) ---
type BarrierControlCommandPayload struct {
	Command   string `json:"command"`              // "open" hoặc "close"
	RequestID string `json:"request_id,omitempty"` // ID để theo dõi lệnh (tùy chọn)
	// BarrierTargetID string `json:"barrier_target_id,omitempty"` // Có thể không cần nếu topic đã đủ rõ ràng
}

// Struct để lưu log sự kiện vào DB (tùy chọn)
type DeviceEventLog struct {
	ID              int64           `json:"id"`
	ReceivedAt      time.Time       `json:"received_at"`
	Esp32ThingName  string          `json:"esp32_thing_name"`
	MqttTopic       string          `json:"mqtt_topic"`
	MessageType     string          `json:"message_type"`
	Payload         json.RawMessage `json:"payload"`          // Lưu payload gốc dạng JSONB
	ProcessedStatus string          `json:"processed_status"` // "pending", "processed", "error"
	ProcessingNotes string          `json:"processing_notes,omitempty"`
}
