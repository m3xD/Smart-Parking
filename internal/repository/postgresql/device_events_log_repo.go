package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
)

type pgDeviceEventsLogRepository struct {
	db *sql.DB
}

func NewPgDeviceEventsLogRepository(db *sql.DB) repository.DeviceEventsLogRepository {
	return &pgDeviceEventsLogRepository{db: db}
}

func (r *pgDeviceEventsLogRepository) Create(ctx context.Context, event *domain.DeviceEventLog) error {
	query := `INSERT INTO device_events_log 
                (received_at, esp32_thing_name, mqtt_topic, message_type, payload, processed_status, processing_notes) 
               VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	var payloadToStore []byte
	if event.Payload != nil {
		payloadToStore = event.Payload // Giả sử event.Payload đã là json.RawMessage
	}

	var id int64 // Để nhận ID trả về nếu cần
	err := r.db.QueryRowContext(ctx, query,
		event.ReceivedAt,
		sql.NullString{String: event.Esp32ThingName, Valid: event.Esp32ThingName != ""},
		sql.NullString{String: event.MqttTopic, Valid: event.MqttTopic != ""},
		sql.NullString{String: event.MessageType, Valid: event.MessageType != ""},
		payloadToStore,
		sql.NullString{String: event.ProcessedStatus, Valid: event.ProcessedStatus != ""},
		sql.NullString{String: event.ProcessingNotes, Valid: event.ProcessingNotes != ""},
	).Scan(&id) // Scan ID trả về

	if err != nil {
		return fmt.Errorf("DeviceEventsLogRepository.Create: %w", err)
	}
	event.ID = id // Gán ID vào struct nếu cần sử dụng sau này
	// log.Printf("Sự kiện từ thiết bị đã được ghi log với ID: %d", event.ID)
	return nil
}
