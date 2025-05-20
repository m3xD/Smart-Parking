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

type pgBarrierRepository struct {
	db *sql.DB
}

func NewPgBarrierRepository(db *sql.DB) repository.BarrierRepository {
	return &pgBarrierRepository{db: db}
}

func (r *pgBarrierRepository) Create(ctx context.Context, barrier *domain.Barrier) (*domain.Barrier, error) {
	query := `INSERT INTO barriers (lot_id, barrier_identifier, esp32_thing_name, barrier_type, current_state, created_at, updated_at) 
	           VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
	           RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query,
		barrier.LotID, barrier.BarrierIdentifier, barrier.Esp32ThingName, barrier.BarrierType, barrier.CurrentState,
	).Scan(&barrier.ID, &barrier.CreatedAt, &barrier.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("BarrierRepository.Create: %w", err)
	}
	barrier.CreatedAt = barrier.CreatedAt.In(time.UTC)
	barrier.UpdatedAt = barrier.UpdatedAt.In(time.UTC)
	return barrier, nil
}

func (r *pgBarrierRepository) FindByID(ctx context.Context, id int) (*domain.Barrier, error) {
	//TODO implement me
	panic("implement me")
}

func (r *pgBarrierRepository) FindByLotID(ctx context.Context, lotID int) ([]domain.Barrier, error) {
	//TODO implement me
	panic("implement me")
}

func (r *pgBarrierRepository) FindByThingAndBarrierIdentifier(ctx context.Context, esp32ThingName string, barrierIdentifier string) (*domain.Barrier, error) {
	barrier := &domain.Barrier{}
	query := `SELECT id, lot_id, barrier_identifier, esp32_thing_name, barrier_type, current_state, 
	                 last_state_update_source, last_command_sent, last_command_timestamp, created_at, updated_at 
	           FROM barriers 
	           WHERE esp32_thing_name = $1 AND barrier_identifier = $2`

	var lastStateSource, lastCmdSent sql.NullString
	var lastCmdTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, esp32ThingName, barrierIdentifier).Scan(
		&barrier.ID, &barrier.LotID, &barrier.BarrierIdentifier, &barrier.Esp32ThingName, &barrier.BarrierType, &barrier.CurrentState,
		&lastStateSource, &lastCmdSent, &lastCmdTime, &barrier.CreatedAt, &barrier.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("BarrierRepository.FindByThingAndBarrierIdentifier: %w", err)
	}
	if lastStateSource.Valid {
		barrier.LastStateUpdateSource = lastStateSource.String
	}
	if lastCmdSent.Valid {
		barrier.LastCommandSent = lastCmdSent.String
	}
	if lastCmdTime.Valid {
		t := lastCmdTime.Time.In(time.UTC)
		barrier.LastCommandTimestamp = &t
	}
	barrier.CreatedAt = barrier.CreatedAt.In(time.UTC)
	barrier.UpdatedAt = barrier.UpdatedAt.In(time.UTC)
	return barrier, nil
}

func (r *pgBarrierRepository) FindByThingName(ctx context.Context, esp32ThingName string) ([]domain.Barrier, error) {
	query := `SELECT id, lot_id, barrier_identifier, esp32_thing_name, barrier_type, current_state, 
	                 last_state_update_source, last_command_sent, last_command_timestamp, created_at, updated_at 
	           FROM barriers WHERE esp32_thing_name = $1 ORDER BY barrier_type`
	rows, err := r.db.QueryContext(ctx, query, esp32ThingName)
	if err != nil {
		return nil, fmt.Errorf("BarrierRepository.FindByThingName: %w", err)
	}
	defer rows.Close()

	var barriers []domain.Barrier
	for rows.Next() {
		var barrier domain.Barrier
		var lastStateSource, lastCmdSent sql.NullString
		var lastCmdTime sql.NullTime
		if err := rows.Scan(
			&barrier.ID, &barrier.LotID, &barrier.BarrierIdentifier, &barrier.Esp32ThingName, &barrier.BarrierType, &barrier.CurrentState,
			&lastStateSource, &lastCmdSent, &lastCmdTime, &barrier.CreatedAt, &barrier.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("BarrierRepository.FindByThingName (scanning row): %w", err)
		}
		if lastStateSource.Valid {
			barrier.LastStateUpdateSource = lastStateSource.String
		}
		if lastCmdSent.Valid {
			barrier.LastCommandSent = lastCmdSent.String
		}
		if lastCmdTime.Valid {
			t := lastCmdTime.Time.In(time.UTC)
			barrier.LastCommandTimestamp = &t
		}
		barrier.CreatedAt = barrier.CreatedAt.In(time.UTC)
		barrier.UpdatedAt = barrier.UpdatedAt.In(time.UTC)
		barriers = append(barriers, barrier)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("BarrierRepository.FindByThingName (rows error): %w", err)
	}
	return barriers, nil
}

func (r *pgBarrierRepository) UpdateState(ctx context.Context, id int, state domain.BarrierState, lastCommand string, lastCommandTime *time.Time, source string) error {
	query := `UPDATE barriers 
	           SET current_state = $1, last_command_sent = $2, last_command_timestamp = $3, last_state_update_source = $4, updated_at = CURRENT_TIMESTAMP 
	           WHERE id = $5`
	var cmdTime sql.NullTime
	if lastCommandTime != nil {
		cmdTime = sql.NullTime{Time: *lastCommandTime, Valid: true}
	}
	var cmd sql.NullString
	if lastCommand != "" {
		cmd = sql.NullString{String: lastCommand, Valid: true}
	}
	var src sql.NullString
	if source != "" {
		src = sql.NullString{String: source, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query, state, cmd, cmdTime, src, id)
	if err != nil {
		return fmt.Errorf("BarrierRepository.UpdateState: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("BarrierRepository.UpdateState (checking rows affected): %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *pgBarrierRepository) Update(ctx context.Context, barrier *domain.Barrier) (*domain.Barrier, error) {
	query := `UPDATE barriers 
               SET lot_id = $1, barrier_identifier = $2, esp32_thing_name = $3, barrier_type = $4, 
                   current_state = $5, last_state_update_source = $6, last_command_sent = $7, last_command_timestamp = $8, 
                   updated_at = CURRENT_TIMESTAMP 
               WHERE id = $9 
               RETURNING updated_at`

	var lastStateSource, lastCmdSent sql.NullString
	if barrier.LastStateUpdateSource != "" {
		lastStateSource = sql.NullString{String: barrier.LastStateUpdateSource, Valid: true}
	}
	if barrier.LastCommandSent != "" {
		lastCmdSent = sql.NullString{String: barrier.LastCommandSent, Valid: true}
	}
	var lastCmdTime sql.NullTime
	if barrier.LastCommandTimestamp != nil {
		lastCmdTime = sql.NullTime{Time: *barrier.LastCommandTimestamp, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		barrier.LotID, barrier.BarrierIdentifier, barrier.Esp32ThingName, barrier.BarrierType,
		barrier.CurrentState, lastStateSource, lastCmdSent, lastCmdTime,
		barrier.ID,
	).Scan(&barrier.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" && pqErr.Constraint == "barriers_lot_id_barrier_identifier_key" {
				return nil, fmt.Errorf("%w: rào chắn '%s' đã tồn tại trong bãi %d", repository.ErrDuplicateEntry, barrier.BarrierIdentifier, barrier.LotID)
			}
		}
		return nil, fmt.Errorf("BarrierRepository.Update: %w", err)
	}
	barrier.UpdatedAt = barrier.UpdatedAt.In(time.UTC)
	return barrier, nil
}

func (r *pgBarrierRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM barriers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("BarrierRepository.Delete: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("BarrierRepository.Delete (checking rows affected): %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}
