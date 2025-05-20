package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"strings"
	"time"
)

type pgParkingSessionRepository struct {
	db *sql.DB
}

func NewPgParkingSessionRepository(db *sql.DB) repository.ParkingSessionRepository {
	return &pgParkingSessionRepository{db: db}
}

func (r *pgParkingSessionRepository) Create(ctx context.Context, session *domain.ParkingSession) (*domain.ParkingSession, error) {
	query := `INSERT INTO parking_sessions 
	           (lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, payment_status, status, entry_gate_event_id, created_at, updated_at) 
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
	           RETURNING id, created_at, updated_at`

	var slotIDVal sql.NullInt64
	if session.SlotID.Valid {
		slotIDVal = sql.NullInt64{Int64: session.SlotID.Int64, Valid: true}
	}
	var vehicleIDVal sql.NullString
	if session.VehicleIdentifier.Valid {
		vehicleIDVal = sql.NullString{String: session.VehicleIdentifier.String, Valid: true}
	}
	var entryGateEventIDVal sql.NullString
	if session.EntryGateEventID.Valid {
		entryGateEventIDVal = sql.NullString{String: session.EntryGateEventID.String, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		session.LotID, slotIDVal, session.Esp32ThingName, vehicleIDVal, session.EntryTime,
		session.PaymentStatus, session.Status, entryGateEventIDVal,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("ParkingSessionRepository.Create: %w", err)
	}
	session.CreatedAt = session.CreatedAt.In(time.UTC)
	session.UpdatedAt = session.UpdatedAt.In(time.UTC)
	return session, nil
}

func (r *pgParkingSessionRepository) FindByID(ctx context.Context, id int) (*domain.ParkingSession, error) {
	session := &domain.ParkingSession{}
	query := `SELECT id, lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, exit_time, 
	                 duration_minutes, calculated_fee, payment_status, status, 
	                 entry_gate_event_id, exit_gate_event_id, created_at, updated_at 
	           FROM parking_sessions WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.LotID, &session.SlotID, &session.Esp32ThingName, &session.VehicleIdentifier,
		&session.EntryTime, &session.ExitTime, &session.DurationMinutes, &session.CalculatedFee,
		&session.PaymentStatus, &session.Status, &session.EntryGateEventID, &session.ExitGateEventID,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("ParkingSessionRepository.FindByID: %w", err)
	}
	session.EntryTime = session.EntryTime.In(time.UTC)
	if session.ExitTime.Valid {
		session.ExitTime.Time = session.ExitTime.Time.In(time.UTC)
	}
	session.CreatedAt = session.CreatedAt.In(time.UTC)
	session.UpdatedAt = session.UpdatedAt.In(time.UTC)
	return session, nil
}

func (r *pgParkingSessionRepository) FindActiveBySlotID(ctx context.Context, slotID int) (*domain.ParkingSession, error) {
	session := &domain.ParkingSession{}
	query := `SELECT id, lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, exit_time, 
	                 duration_minutes, calculated_fee, payment_status, status, 
	                 entry_gate_event_id, exit_gate_event_id, created_at, updated_at 
	           FROM parking_sessions 
	           WHERE slot_id = $1 AND status = $2 
	           ORDER BY entry_time DESC LIMIT 1`

	err := r.db.QueryRowContext(ctx, query, slotID, domain.SessionActive).Scan(
		&session.ID, &session.LotID, &session.SlotID, &session.Esp32ThingName, &session.VehicleIdentifier,
		&session.EntryTime, &session.ExitTime, &session.DurationMinutes, &session.CalculatedFee,
		&session.PaymentStatus, &session.Status, &session.EntryGateEventID, &session.ExitGateEventID,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNoActiveSession
		}
		return nil, fmt.Errorf("ParkingSessionRepository.FindActiveBySlotID: %w", err)
	}
	// ... (chuẩn hóa thời gian) ...
	return session, nil
}

func (r *pgParkingSessionRepository) FindActiveByVehicleIdentifier(ctx context.Context, lotID int, vehicleID string) (*domain.ParkingSession, error) {
	session := &domain.ParkingSession{}
	query := `SELECT id, lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, exit_time, 
	                 duration_minutes, calculated_fee, payment_status, status, 
	                 entry_gate_event_id, exit_gate_event_id, created_at, updated_at 
	           FROM parking_sessions 
	           WHERE lot_id = $1 AND vehicle_identifier = $2 AND status = $3 
	           ORDER BY entry_time DESC LIMIT 1`

	err := r.db.QueryRowContext(ctx, query, lotID, vehicleID, domain.SessionActive).Scan(
		&session.ID, &session.LotID, &session.SlotID, &session.Esp32ThingName, &session.VehicleIdentifier,
		&session.EntryTime, &session.ExitTime, &session.DurationMinutes, &session.CalculatedFee,
		&session.PaymentStatus, &session.Status, &session.EntryGateEventID, &session.ExitGateEventID,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNoActiveSession
		}
		return nil, fmt.Errorf("ParkingSessionRepository.FindActiveByVehicleIdentifier: %w", err)
	}
	// ... (chuẩn hóa thời gian) ...
	return session, nil
}

func (r *pgParkingSessionRepository) FindLatestActiveByThingName(ctx context.Context, esp32ThingName string) (*domain.ParkingSession, error) {
	session := &domain.ParkingSession{}
	query := `SELECT id, lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, exit_time, 
                     duration_minutes, calculated_fee, payment_status, status, 
                     entry_gate_event_id, exit_gate_event_id, created_at, updated_at 
               FROM parking_sessions 
               WHERE esp32_thing_name = $1 
                 AND status = $2 
                 AND exit_time IS NULL 
               ORDER BY entry_time DESC LIMIT 1`

	err := r.db.QueryRowContext(ctx, query, esp32ThingName, domain.SessionActive).Scan(
		&session.ID, &session.LotID, &session.SlotID, &session.Esp32ThingName, &session.VehicleIdentifier,
		&session.EntryTime, &session.ExitTime, &session.DurationMinutes, &session.CalculatedFee,
		&session.PaymentStatus, &session.Status, &session.EntryGateEventID, &session.ExitGateEventID,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNoActiveSession
		}
		return nil, fmt.Errorf("ParkingSessionRepository.FindLatestActiveByThingName: %w", err)
	}
	session.EntryTime = session.EntryTime.In(time.UTC)
	if session.ExitTime.Valid {
		session.ExitTime.Time = session.ExitTime.Time.In(time.UTC)
	}
	session.CreatedAt = session.CreatedAt.In(time.UTC)
	session.UpdatedAt = session.UpdatedAt.In(time.UTC)
	return session, nil
}

func (r *pgParkingSessionRepository) Update(ctx context.Context, session *domain.ParkingSession) (*domain.ParkingSession, error) {
	query := `UPDATE parking_sessions 
	           SET lot_id = $1, slot_id = $2, esp32_thing_name = $3, vehicle_identifier = $4, 
	               entry_time = $5, exit_time = $6, duration_minutes = $7, calculated_fee = $8, 
	               payment_status = $9, status = $10, entry_gate_event_id = $11, exit_gate_event_id = $12, 
	               updated_at = CURRENT_TIMESTAMP 
	           WHERE id = $13 
	           RETURNING updated_at`

	var slotIDVal sql.NullInt64
	if session.SlotID.Valid {
		slotIDVal = sql.NullInt64{Int64: session.SlotID.Int64, Valid: true}
	}
	var vehicleIDVal sql.NullString
	if session.VehicleIdentifier.Valid {
		vehicleIDVal = sql.NullString{String: session.VehicleIdentifier.String, Valid: true}
	}
	var exitTimeVal sql.NullTime
	if session.ExitTime.Valid {
		exitTimeVal = sql.NullTime{Time: session.ExitTime.Time, Valid: true}
	}
	var durationVal sql.NullInt64
	if session.DurationMinutes.Valid {
		durationVal = sql.NullInt64{Int64: session.DurationMinutes.Int64, Valid: true}
	}
	var feeVal sql.NullFloat64
	if session.CalculatedFee.Valid {
		feeVal = sql.NullFloat64{Float64: session.CalculatedFee.Float64, Valid: true}
	}
	var entryGateEventIDVal sql.NullString
	if session.EntryGateEventID.Valid {
		entryGateEventIDVal = sql.NullString{String: session.EntryGateEventID.String, Valid: true}
	}
	var exitGateEventIDVal sql.NullString
	if session.ExitGateEventID.Valid {
		exitGateEventIDVal = sql.NullString{String: session.ExitGateEventID.String, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		session.LotID, slotIDVal, session.Esp32ThingName, vehicleIDVal,
		session.EntryTime, exitTimeVal, durationVal, feeVal,
		session.PaymentStatus, session.Status, entryGateEventIDVal, exitGateEventIDVal,
		session.ID,
	).Scan(&session.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("ParkingSessionRepository.Update: %w", err)
	}
	session.UpdatedAt = session.UpdatedAt.In(time.UTC)
	return session, nil
}

func (r *pgParkingSessionRepository) GetActiveSessionsByLot(ctx context.Context, lotID int) ([]domain.ParkingSession, error) {
	query := `SELECT id, lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, exit_time, 
	                 duration_minutes, calculated_fee, payment_status, status, 
	                 entry_gate_event_id, exit_gate_event_id, created_at, updated_at 
	           FROM parking_sessions 
	           WHERE lot_id = $1 AND status = $2 
	           ORDER BY entry_time DESC`
	rows, err := r.db.QueryContext(ctx, query, lotID, domain.SessionActive)
	if err != nil {
		return nil, fmt.Errorf("ParkingSessionRepository.GetActiveSessionsByLot: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ParkingSession
	for rows.Next() {
		var session domain.ParkingSession
		if err := rows.Scan(
			&session.ID, &session.LotID, &session.SlotID, &session.Esp32ThingName, &session.VehicleIdentifier,
			&session.EntryTime, &session.ExitTime, &session.DurationMinutes, &session.CalculatedFee,
			&session.PaymentStatus, &session.Status, &session.EntryGateEventID, &session.ExitGateEventID,
			&session.CreatedAt, &session.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ParkingSessionRepository.GetActiveSessionsByLot (scanning row): %w", err)
		}
		session.EntryTime = session.EntryTime.In(time.UTC)
		if session.ExitTime.Valid {
			session.ExitTime.Time = session.ExitTime.Time.In(time.UTC)
		}
		session.CreatedAt = session.CreatedAt.In(time.UTC)
		session.UpdatedAt = session.UpdatedAt.In(time.UTC)
		sessions = append(sessions, session)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ParkingSessionRepository.GetActiveSessionsByLot (rows error): %w", err)
	}
	return sessions, nil
}

func (r *pgParkingSessionRepository) Find(ctx context.Context, filter domain.ParkingSessionFilterDTO) ([]domain.ParkingSession, error) {
	baseQuery := `SELECT id, lot_id, slot_id, esp32_thing_name, vehicle_identifier, entry_time, exit_time, 
                         duration_minutes, calculated_fee, payment_status, status, 
                         entry_gate_event_id, exit_gate_event_id, created_at, updated_at 
                   FROM parking_sessions`

	var conditions []string
	var args []interface{}
	argID := 1

	if filter.LotID != nil {
		conditions = append(conditions, fmt.Sprintf("lot_id = $%d", argID))
		args = append(args, *filter.LotID)
		argID++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argID))
		args = append(args, *filter.Status)
		argID++
	}
	// Thêm các điều kiện filter khác nếu cần

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY entry_time DESC" // Sắp xếp theo thời gian vào gần nhất

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ParkingSessionRepository.Find: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ParkingSession
	for rows.Next() {
		var session domain.ParkingSession
		// ... (scan tương tự như các hàm Find khác) ...
		if err := rows.Scan(
			&session.ID, &session.LotID, &session.SlotID, &session.Esp32ThingName, &session.VehicleIdentifier,
			&session.EntryTime, &session.ExitTime, &session.DurationMinutes, &session.CalculatedFee,
			&session.PaymentStatus, &session.Status, &session.EntryGateEventID, &session.ExitGateEventID,
			&session.CreatedAt, &session.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ParkingSessionRepository.Find (scanning row): %w", err)
		}
		session.EntryTime = session.EntryTime.In(time.UTC)
		if session.ExitTime.Valid {
			session.ExitTime.Time = session.ExitTime.Time.In(time.UTC)
		}
		session.CreatedAt = session.CreatedAt.In(time.UTC)
		session.UpdatedAt = session.UpdatedAt.In(time.UTC)
		sessions = append(sessions, session)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ParkingSessionRepository.Find (rows error): %w", err)
	}
	return sessions, nil
}
