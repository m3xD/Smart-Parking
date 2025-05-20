package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"

	"github.com/lib/pq"
)

type pgParkingSlotRepository struct {
	db *sql.DB
}

func NewPgParkingSlotRepository(db *sql.DB) repository.ParkingSlotRepository {
	return &pgParkingSlotRepository{db: db}
}

func (r *pgParkingSlotRepository) Create(ctx context.Context, slot *domain.ParkingSlot) (*domain.ParkingSlot, error) {
	query := `INSERT INTO parking_slots (lot_id, slot_identifier, esp32_thing_name, status, last_status_update_source, created_at, updated_at) 
	           VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
	           RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query,
		slot.LotID, slot.SlotIdentifier, sql.NullString{String: slot.Esp32ThingName, Valid: slot.Esp32ThingName != ""},
		slot.Status, sql.NullString{String: slot.LastStatusUpdateSource, Valid: slot.LastStatusUpdateSource != ""},
	).Scan(&slot.ID, &slot.CreatedAt, &slot.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				// Tên constraint cho UNIQUE (lot_id, slot_identifier) thường là parking_slots_lot_id_slot_identifier_key
				if pqErr.Constraint == "parking_slots_lot_id_slot_identifier_key" {
					return nil, fmt.Errorf("%w: chỗ đỗ '%s' đã tồn tại trong bãi %d", repository.ErrDuplicateEntry, slot.SlotIdentifier, slot.LotID)
				}
				// Nếu có constraint UNIQUE (lot_id, esp32_thing_name, slot_identifier) hoặc tương tự
				// thì cần kiểm tra tên constraint đó. Hiện tại schema chỉ có UNIQUE (lot_id, slot_identifier).
				if pqErr.Constraint == "parking_slots_lot_id_esp32_sensor_id_key" {
					return nil, fmt.Errorf("%w: định danh cảm biến (slot_identifier) '%s' đã được sử dụng trong bãi %d", repository.ErrDuplicateEntry, slot.SlotIdentifier, slot.LotID)
				}
			}
		}
		return nil, fmt.Errorf("ParkingSlotRepository.Create: %w", err)
	}
	slot.CreatedAt = slot.CreatedAt.In(time.UTC)
	slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
	return slot, nil
}

func (r *pgParkingSlotRepository) FindByID(ctx context.Context, id int) (*domain.ParkingSlot, error) {
	slot := &domain.ParkingSlot{}
	query := `SELECT id, lot_id, slot_identifier, esp32_thing_name, status, last_status_update_source, last_event_timestamp, created_at, updated_at 
	           FROM parking_slots WHERE id = $1`
	var esp32ThingName, lastStatusSource sql.NullString
	var lastEventTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&slot.ID, &slot.LotID, &slot.SlotIdentifier, &esp32ThingName, &slot.Status,
		&lastStatusSource, &lastEventTime, &slot.CreatedAt, &slot.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("ParkingSlotRepository.FindByID: %w", err)
	}
	if esp32ThingName.Valid {
		slot.Esp32ThingName = esp32ThingName.String
	}
	if lastStatusSource.Valid {
		slot.LastStatusUpdateSource = lastStatusSource.String
	}
	if lastEventTime.Valid {
		t := lastEventTime.Time.In(time.UTC)
		slot.LastEventTimestamp = &t
	}
	slot.CreatedAt = slot.CreatedAt.In(time.UTC)
	slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
	return slot, nil
}

func (r *pgParkingSlotRepository) FindByLotID(ctx context.Context, lotID int) ([]domain.ParkingSlot, error) {
	query := `SELECT id, lot_id, slot_identifier, esp32_thing_name, status, last_status_update_source, last_event_timestamp, created_at, updated_at 
	           FROM parking_slots WHERE lot_id = $1 ORDER BY slot_identifier`
	rows, err := r.db.QueryContext(ctx, query, lotID)
	if err != nil {
		return nil, fmt.Errorf("ParkingSlotRepository.FindByLotID: %w", err)
	}
	defer rows.Close()

	var slots []domain.ParkingSlot
	for rows.Next() {
		var slot domain.ParkingSlot
		var esp32ThingName, lastStatusSource sql.NullString
		var lastEventTime sql.NullTime
		if err := rows.Scan(
			&slot.ID, &slot.LotID, &slot.SlotIdentifier, &esp32ThingName, &slot.Status,
			&lastStatusSource, &lastEventTime, &slot.CreatedAt, &slot.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ParkingSlotRepository.FindByLotID (scanning row): %w", err)
		}
		if esp32ThingName.Valid {
			slot.Esp32ThingName = esp32ThingName.String
		}
		if lastStatusSource.Valid {
			slot.LastStatusUpdateSource = lastStatusSource.String
		}
		if lastEventTime.Valid {
			t := lastEventTime.Time.In(time.UTC)
			slot.LastEventTimestamp = &t
		}
		slot.CreatedAt = slot.CreatedAt.In(time.UTC)
		slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
		slots = append(slots, slot)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ParkingSlotRepository.FindByLotID (rows error): %w", err)
	}
	return slots, nil
}

func (r *pgParkingSlotRepository) FindByLotIDAndSlotIdentifier(ctx context.Context, lotID int, slotIdentifier string) (*domain.ParkingSlot, error) {
	slot := &domain.ParkingSlot{}
	query := `SELECT id, lot_id, slot_identifier, esp32_thing_name, status, last_status_update_source, last_event_timestamp, created_at, updated_at 
	           FROM parking_slots 
	           WHERE lot_id = $1 AND slot_identifier = $2`
	var esp32ThingName, lastStatusSource sql.NullString
	var lastEventTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, lotID, slotIdentifier).Scan(
		&slot.ID, &slot.LotID, &slot.SlotIdentifier, &esp32ThingName, &slot.Status,
		&lastStatusSource, &lastEventTime, &slot.CreatedAt, &slot.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("ParkingSlotRepository.FindByLotIDAndSlotIdentifier: %w", err)
	}
	if esp32ThingName.Valid {
		slot.Esp32ThingName = esp32ThingName.String
	}
	if lastStatusSource.Valid {
		slot.LastStatusUpdateSource = lastStatusSource.String
	}
	if lastEventTime.Valid {
		t := lastEventTime.Time.In(time.UTC)
		slot.LastEventTimestamp = &t
	}
	slot.CreatedAt = slot.CreatedAt.In(time.UTC)
	slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
	return slot, nil
}

func (r *pgParkingSlotRepository) FindByThingAndSlotIdentifier(ctx context.Context, esp32ThingName string, slotIdentifier string) (*domain.ParkingSlot, error) {
	slot := &domain.ParkingSlot{}
	query := `SELECT id, lot_id, slot_identifier, esp32_thing_name, status, last_status_update_source, last_event_timestamp, created_at, updated_at 
	           FROM parking_slots 
	           WHERE esp32_thing_name = $1 AND slot_identifier = $2`
	var lastEventTime sql.NullTime
	var lastStatusSource sql.NullString
	var dbEsp32ThingName sql.NullString

	err := r.db.QueryRowContext(ctx, query, esp32ThingName, slotIdentifier).Scan(
		&slot.ID, &slot.LotID, &slot.SlotIdentifier, &dbEsp32ThingName, &slot.Status,
		&lastStatusSource, &lastEventTime, &slot.CreatedAt, &slot.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("ParkingSlotRepository.FindByThingAndSlotIdentifier: %w", err)
	}
	if dbEsp32ThingName.Valid {
		slot.Esp32ThingName = dbEsp32ThingName.String
	}
	if lastEventTime.Valid {
		t := lastEventTime.Time.In(time.UTC)
		slot.LastEventTimestamp = &t
	}
	if lastStatusSource.Valid {
		slot.LastStatusUpdateSource = lastStatusSource.String
	}
	slot.CreatedAt = slot.CreatedAt.In(time.UTC)
	slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
	return slot, nil
}

func (r *pgParkingSlotRepository) UpdateStatus(ctx context.Context, id int, status domain.SlotStatus, lastEventTime *time.Time, source string) error {
	query := `UPDATE parking_slots 
	           SET status = $1, last_event_timestamp = $2, last_status_update_source = $3, updated_at = CURRENT_TIMESTAMP 
	           WHERE id = $4`
	var eventTime sql.NullTime
	if lastEventTime != nil {
		eventTime = sql.NullTime{Time: *lastEventTime, Valid: true}
	}
	var statusSource sql.NullString
	if source != "" {
		statusSource = sql.NullString{String: source, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query, status, eventTime, statusSource, id)
	if err != nil {
		return fmt.Errorf("ParkingSlotRepository.UpdateStatus: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ParkingSlotRepository.UpdateStatus (checking rows affected): %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *pgParkingSlotRepository) Update(ctx context.Context, slot *domain.ParkingSlot) (*domain.ParkingSlot, error) {
	query := `UPDATE parking_slots 
               SET lot_id = $1, slot_identifier = $2, esp32_thing_name = $3, status = $4, 
                   last_status_update_source = $5, last_event_timestamp = $6, updated_at = CURRENT_TIMESTAMP 
               WHERE id = $7 
               RETURNING updated_at`

	var esp32ThingName sql.NullString
	if slot.Esp32ThingName != "" {
		esp32ThingName = sql.NullString{String: slot.Esp32ThingName, Valid: true}
	}
	var lastStatusSource sql.NullString
	if slot.LastStatusUpdateSource != "" {
		lastStatusSource = sql.NullString{String: slot.LastStatusUpdateSource, Valid: true}
	}
	var lastEventTime sql.NullTime
	if slot.LastEventTimestamp != nil {
		lastEventTime = sql.NullTime{Time: *slot.LastEventTimestamp, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		slot.LotID, slot.SlotIdentifier, esp32ThingName, slot.Status,
		lastStatusSource, lastEventTime, slot.ID,
	).Scan(&slot.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				if pqErr.Constraint == "parking_slots_lot_id_slot_identifier_key" {
					return nil, fmt.Errorf("%w: chỗ đỗ '%s' đã tồn tại trong bãi %d", repository.ErrDuplicateEntry, slot.SlotIdentifier, slot.LotID)
				}
			}
		}
		return nil, fmt.Errorf("ParkingSlotRepository.Update: %w", err)
	}
	slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
	return slot, nil
}

func (r *pgParkingSlotRepository) FindFirstAvailableByLotID(ctx context.Context, lotID int) (*domain.ParkingSlot, error) {
	slot := &domain.ParkingSlot{}
	query := `SELECT id, lot_id, slot_identifier, esp32_thing_name, status, last_status_update_source, last_event_timestamp, created_at, updated_at 
	           FROM parking_slots 
	           WHERE lot_id = $1 AND status = $2 
	           ORDER BY slot_identifier ASC LIMIT 1` // Lấy slot trống đầu tiên theo thứ tự slot_identifier
	var esp32ThingName, lastStatusSource sql.NullString
	var lastEventTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, lotID, domain.StatusVacant).Scan(
		&slot.ID, &slot.LotID, &slot.SlotIdentifier, &esp32ThingName, &slot.Status,
		&lastStatusSource, &lastEventTime, &slot.CreatedAt, &slot.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound // Không có slot nào trống
		}
		return nil, fmt.Errorf("ParkingSlotRepository.FindFirstAvailableByLotID: %w", err)
	}
	if esp32ThingName.Valid {
		slot.Esp32ThingName = esp32ThingName.String
	}
	if lastStatusSource.Valid {
		slot.LastStatusUpdateSource = lastStatusSource.String
	}
	if lastEventTime.Valid {
		t := lastEventTime.Time.In(time.UTC)
		slot.LastEventTimestamp = &t
	}
	slot.CreatedAt = slot.CreatedAt.In(time.UTC)
	slot.UpdatedAt = slot.UpdatedAt.In(time.UTC)
	return slot, nil
}

func (r *pgParkingSlotRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM parking_slots WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ParkingSlotRepository.Delete: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ParkingSlotRepository.Delete (checking rows affected): %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}
