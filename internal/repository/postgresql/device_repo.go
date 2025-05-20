package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"
)

type pgDeviceRepository struct {
	db *sql.DB
}

func NewPgDeviceRepository(db *sql.DB) repository.DeviceRepository {
	return &pgDeviceRepository{db: db}
}

// CreateOrUpdate tạo mới device nếu chưa có (dựa trên thing_name) hoặc cập nhật nếu đã có.
func (r *pgDeviceRepository) CreateOrUpdate(ctx context.Context, device *domain.Device) (*domain.Device, error) {
	// Thử tìm device trước
	existingDevice, err := r.FindByThingName(ctx, device.ThingName)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("DeviceRepository.CreateOrUpdate (finding existing): %w", err)
	}

	if existingDevice != nil { // Device đã tồn tại, cập nhật nó
		device.ID = existingDevice.ID               // Giữ lại ID cũ
		device.CreatedAt = existingDevice.CreatedAt // Không cập nhật CreatedAt
		return r.UpdateDetails(ctx, device)
	}

	// Device chưa tồn tại, tạo mới
	query := `INSERT INTO devices (thing_name, lot_id, firmware_version, last_seen_at, status, ip_address, mac_address, last_rssi, last_free_heap, last_uptime_seconds, notes, created_at, updated_at) 
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
	           RETURNING id, created_at, updated_at`

	var lotIDVal sql.NullInt64
	if device.LotID.Valid {
		lotIDVal = sql.NullInt64{Int64: device.LotID.Int64, Valid: true}
	}
	var lastSeenAtVal sql.NullTime
	if device.LastSeenAt.Valid {
		lastSeenAtVal = sql.NullTime{Time: device.LastSeenAt.Time, Valid: true}
	}
	var lastRssiVal sql.NullInt64
	if device.LastRssi.Valid {
		lastRssiVal = sql.NullInt64{Int64: device.LastRssi.Int64, Valid: true}
	}
	var lastFreeHeapVal sql.NullInt64
	if device.LastFreeHeap.Valid {
		lastFreeHeapVal = sql.NullInt64{Int64: device.LastFreeHeap.Int64, Valid: true}
	}
	var lastUptimeVal sql.NullInt64
	if device.LastUptimeSeconds.Valid {
		lastUptimeVal = sql.NullInt64{Int64: device.LastUptimeSeconds.Int64, Valid: true}
	}

	err = r.db.QueryRowContext(ctx, query,
		device.ThingName, lotIDVal, sql.NullString{String: device.FirmwareVersion, Valid: device.FirmwareVersion != ""},
		lastSeenAtVal, device.Status, sql.NullString{String: device.IPAddress, Valid: device.IPAddress != ""},
		sql.NullString{String: device.MacAddress, Valid: device.MacAddress != ""},
		lastRssiVal, lastFreeHeapVal, lastUptimeVal,
		sql.NullString{String: device.Notes, Valid: device.Notes != ""},
	).Scan(&device.ID, &device.CreatedAt, &device.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" && pqErr.Constraint == "devices_thing_name_key" {
				// Trường hợp này ít xảy ra nếu đã kiểm tra FindByThingName, nhưng để phòng ngừa race condition
				return nil, fmt.Errorf("%w: thiết bị với thing_name '%s' đã tồn tại", repository.ErrDuplicateEntry, device.ThingName)
			}
		}
		return nil, fmt.Errorf("DeviceRepository.Create: %w", err)
	}
	device.CreatedAt = device.CreatedAt.In(time.UTC)
	device.UpdatedAt = device.UpdatedAt.In(time.UTC)
	return device, nil
}

func (r *pgDeviceRepository) FindByThingName(ctx context.Context, thingName string) (*domain.Device, error) {
	device := &domain.Device{}
	query := `SELECT id, thing_name, lot_id, firmware_version, last_seen_at, status, ip_address, mac_address, 
	                 last_rssi, last_free_heap, last_uptime_seconds, notes, created_at, updated_at 
	           FROM devices WHERE thing_name = $1`

	err := r.db.QueryRowContext(ctx, query, thingName).Scan(
		&device.ID, &device.ThingName, &device.LotID, &device.FirmwareVersion, &device.LastSeenAt, &device.Status,
		&device.IPAddress, &device.MacAddress, &device.LastRssi, &device.LastFreeHeap, &device.LastUptimeSeconds,
		&device.Notes, &device.CreatedAt, &device.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("DeviceRepository.FindByThingName: %w", err)
	}
	// ... (chuẩn hóa thời gian và nullables) ...
	if device.LastSeenAt.Valid {
		device.LastSeenAt.Time = device.LastSeenAt.Time.In(time.UTC)
	}
	device.CreatedAt = device.CreatedAt.In(time.UTC)
	device.UpdatedAt = device.UpdatedAt.In(time.UTC)
	return device, nil
}

func (r *pgDeviceRepository) FindAll(ctx context.Context) ([]domain.Device, error) {
	query := `SELECT id, thing_name, lot_id, firmware_version, last_seen_at, status, ip_address, mac_address, 
	                 last_rssi, last_free_heap, last_uptime_seconds, notes, created_at, updated_at 
	           FROM devices ORDER BY thing_name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("DeviceRepository.FindAll: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		var device domain.Device
		if err := rows.Scan(
			&device.ID, &device.ThingName, &device.LotID, &device.FirmwareVersion, &device.LastSeenAt, &device.Status,
			&device.IPAddress, &device.MacAddress, &device.LastRssi, &device.LastFreeHeap, &device.LastUptimeSeconds,
			&device.Notes, &device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("DeviceRepository.FindAll (scanning row): %w", err)
		}
		// ... (chuẩn hóa thời gian và nullables) ...
		if device.LastSeenAt.Valid {
			device.LastSeenAt.Time = device.LastSeenAt.Time.In(time.UTC)
		}
		device.CreatedAt = device.CreatedAt.In(time.UTC)
		device.UpdatedAt = device.UpdatedAt.In(time.UTC)
		devices = append(devices, device)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("DeviceRepository.FindAll (rows error): %w", err)
	}
	return devices, nil
}

func (r *pgDeviceRepository) UpdateStatus(ctx context.Context, thingName string, status domain.DeviceStatus, lastSeenAt time.Time) error {
	query := `UPDATE devices SET status = $1, last_seen_at = $2, updated_at = CURRENT_TIMESTAMP WHERE thing_name = $3`
	result, err := r.db.ExecContext(ctx, query, status, lastSeenAt, thingName)
	if err != nil {
		return fmt.Errorf("DeviceRepository.UpdateStatus: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeviceRepository.UpdateStatus (checking rows affected): %w", err)
	}
	if rowsAffected == 0 {
		// Nếu không tìm thấy device để update, có thể tạo mới hoặc báo lỗi tùy logic
		// Hiện tại, hàm này chỉ update, nếu muốn create or update thì dùng CreateOrUpdate
		return fmt.Errorf("%w: không tìm thấy thiết bị '%s' để cập nhật trạng thái", repository.ErrNotFound, thingName)
	}
	return nil
}

func (r *pgDeviceRepository) UpdateDetails(ctx context.Context, device *domain.Device) (*domain.Device, error) {
	query := `UPDATE devices 
	           SET lot_id = $1, firmware_version = $2, last_seen_at = $3, status = $4, 
	               ip_address = $5, mac_address = $6, last_rssi = $7, last_free_heap = $8, 
	               last_uptime_seconds = $9, notes = $10, updated_at = CURRENT_TIMESTAMP 
	           WHERE id = $11 
	           RETURNING updated_at`

	var lotIDVal sql.NullInt64
	if device.LotID.Valid {
		lotIDVal = sql.NullInt64{Int64: device.LotID.Int64, Valid: true}
	}
	var lastSeenAtVal sql.NullTime
	if device.LastSeenAt.Valid {
		lastSeenAtVal = sql.NullTime{Time: device.LastSeenAt.Time, Valid: true}
	}
	var lastRssiVal sql.NullInt64
	if device.LastRssi.Valid {
		lastRssiVal = sql.NullInt64{Int64: device.LastRssi.Int64, Valid: true}
	}
	var lastFreeHeapVal sql.NullInt64
	if device.LastFreeHeap.Valid {
		lastFreeHeapVal = sql.NullInt64{Int64: device.LastFreeHeap.Int64, Valid: true}
	}
	var lastUptimeVal sql.NullInt64
	if device.LastUptimeSeconds.Valid {
		lastUptimeVal = sql.NullInt64{Int64: device.LastUptimeSeconds.Int64, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		lotIDVal, sql.NullString{String: device.FirmwareVersion, Valid: device.FirmwareVersion != ""},
		lastSeenAtVal, device.Status, sql.NullString{String: device.IPAddress, Valid: device.IPAddress != ""},
		sql.NullString{String: device.MacAddress, Valid: device.MacAddress != ""},
		lastRssiVal, lastFreeHeapVal, lastUptimeVal,
		sql.NullString{String: device.Notes, Valid: device.Notes != ""},
		device.ID,
	).Scan(&device.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("DeviceRepository.UpdateDetails: %w", err)
	}
	device.UpdatedAt = device.UpdatedAt.In(time.UTC)
	return device, nil
}
