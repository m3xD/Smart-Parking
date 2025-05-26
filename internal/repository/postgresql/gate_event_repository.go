package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"
)

type pgGateEventRepository struct {
	db *sql.DB
}

func NewPgGateEventRepository(db *sql.DB) repository.GateEventRepository {
	return &pgGateEventRepository{db: db}
}

func (r *pgGateEventRepository) Create(ctx context.Context, event *domain.GateEventRecord) error {
	query := `INSERT INTO gate_events.sql 
		(event_id, lot_id, device_id, gate_direction, event_type, status, sensor_id, expires_at, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
		RETURNING id, created_at, updated_at`

	var expiresAt sql.NullTime
	if event.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *event.ExpiresAt, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		event.EventID, event.LotID, event.DeviceID, event.GateDirection,
		event.EventType, event.Status,
		sql.NullString{String: event.SensorID, Valid: event.SensorID != ""},
		expiresAt,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		return fmt.Errorf("GateEventRepository.Create: %w", err)
	}

	event.CreatedAt = event.CreatedAt.In(time.UTC)
	event.UpdatedAt = event.UpdatedAt.In(time.UTC)
	return nil
}

func (r *pgGateEventRepository) FindByEventID(ctx context.Context, eventID string) (*domain.GateEventRecord, error) {
	event := &domain.GateEventRecord{}
	query := `SELECT id, event_id, lot_id, device_id, gate_direction, event_type, status, 
		sensor_id, detected_plate, lpr_confidence, is_manual_entry, session_id, 
		processing_notes, assigned_operator, created_at, updated_at, expires_at, completed_at
		FROM gate_events.sql WHERE event_id = $1`

	var sensorID, detectedPlate, processingNotes, assignedOperator sql.NullString
	var lprConfidence sql.NullFloat64
	var isManualEntry sql.NullBool
	var sessionID sql.NullInt64
	var expiresAt, completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID, &event.EventID, &event.LotID, &event.DeviceID, &event.GateDirection,
		&event.EventType, &event.Status, &sensorID, &detectedPlate, &lprConfidence,
		&isManualEntry, &sessionID, &processingNotes, &assignedOperator,
		&event.CreatedAt, &event.UpdatedAt, &expiresAt, &completedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("GateEventRepository.FindByEventID: %w", err)
	}

	// Handle nullable fields
	if sensorID.Valid {
		event.SensorID = sensorID.String
	}
	if detectedPlate.Valid {
		event.DetectedPlate = detectedPlate.String
	}
	if lprConfidence.Valid {
		conf := float32(lprConfidence.Float64)
		event.LPRConfidence = &conf
	}
	if sessionID.Valid {
		sid := int(sessionID.Int64)
		event.SessionID = &sid
	}
	if processingNotes.Valid {
		event.ProcessingNotes = processingNotes.String
	}
	if expiresAt.Valid {
		t := expiresAt.Time.In(time.UTC)
		event.ExpiresAt = &t
	}

	event.CreatedAt = event.CreatedAt.In(time.UTC)
	event.UpdatedAt = event.UpdatedAt.In(time.UTC)
	return event, nil
}

func (r *pgGateEventRepository) UpdateStatus(ctx context.Context, eventID string, status domain.GateEventStatus, notes string) error {
	query := `UPDATE gate_events.sql 
		SET status = $1, processing_notes = COALESCE(processing_notes, '') || $2, updated_at = CURRENT_TIMESTAMP 
		WHERE event_id = $3`

	notesToAppend := ""
	if notes != "" {
		notesToAppend = fmt.Sprintf("; %s", notes)
	}

	result, err := r.db.ExecContext(ctx, query, status, notesToAppend, eventID)
	if err != nil {
		return fmt.Errorf("GateEventRepository.UpdateStatus: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("GateEventRepository.UpdateStatus (checking rows): %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *pgGateEventRepository) UpdateLPRResult(ctx context.Context, eventID string, plate string, confidence float32) error {
	query := `UPDATE gate_events.sql 
		SET detected_plate = $1, lpr_confidence = $2, status = $3, updated_at = CURRENT_TIMESTAMP 
		WHERE event_id = $4`

	result, err := r.db.ExecContext(ctx, query, plate, confidence, domain.StatusLPRCompleted, eventID)
	if err != nil {
		return fmt.Errorf("GateEventRepository.UpdateLPRResult: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("GateEventRepository.UpdateLPRResult (checking rows): %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *pgGateEventRepository) UpdateWithSession(ctx context.Context, eventID string, sessionID int) error {
	query := `UPDATE gate_events.sql 
		SET session_id = $1, status = $2, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
		WHERE event_id = $3`

	result, err := r.db.ExecContext(ctx, query, sessionID, domain.StatusSessionCreated, eventID)
	if err != nil {
		return fmt.Errorf("GateEventRepository.UpdateWithSession: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("GateEventRepository.UpdateWithSession (checking rows): %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *pgGateEventRepository) FindPendingEvents(ctx context.Context, limit int) ([]domain.GateEventRecord, error) {
	query := `SELECT id, event_id, lot_id, device_id, gate_direction, event_type, status, 
		created_at, expires_at 
		FROM gate_events.sql 
		WHERE status IN ('pending', 'awaiting_lpr') 
		ORDER BY created_at ASC 
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("GateEventRepository.FindPendingEvents: %w", err)
	}
	defer rows.Close()

	var events []domain.GateEventRecord
	for rows.Next() {
		var event domain.GateEventRecord
		var expiresAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.EventID, &event.LotID, &event.DeviceID,
			&event.GateDirection, &event.EventType, &event.Status,
			&event.CreatedAt, &expiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("GateEventRepository.FindPendingEvents (scanning): %w", err)
		}

		if expiresAt.Valid {
			t := expiresAt.Time.In(time.UTC)
			event.ExpiresAt = &t
		}
		event.CreatedAt = event.CreatedAt.In(time.UTC)
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("GateEventRepository.FindPendingEvents (rows error): %w", err)
	}

	return events, nil
}

func (r *pgGateEventRepository) FindExpiredEvents(ctx context.Context) ([]domain.GateEventRecord, error) {
	query := `SELECT id, event_id, lot_id, device_id, gate_direction, event_type, status, 
		created_at, expires_at 
		FROM gate_events.sql 
		WHERE status IN ('pending', 'awaiting_lpr') 
		  AND expires_at < CURRENT_TIMESTAMP 
		ORDER BY expires_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GateEventRepository.FindExpiredEvents: %w", err)
	}
	defer rows.Close()

	var events []domain.GateEventRecord
	for rows.Next() {
		var event domain.GateEventRecord
		var expiresAt sql.NullTime

		err := rows.Scan(
			&event.ID, &event.EventID, &event.LotID, &event.DeviceID,
			&event.GateDirection, &event.EventType, &event.Status,
			&event.CreatedAt, &expiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("GateEventRepository.FindExpiredEvents (scanning): %w", err)
		}

		if expiresAt.Valid {
			t := expiresAt.Time.In(time.UTC)
			event.ExpiresAt = &t
		}
		event.CreatedAt = event.CreatedAt.In(time.UTC)
		events = append(events, event)
	}

	return events, nil
}

func (r *pgGateEventRepository) CleanupExpiredEvents(ctx context.Context) (int, error) {
	query := `SELECT cleanup_expired_gate_events()`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GateEventRepository.CleanupExpiredEvents: %w", err)
	}

	return count, nil
}
