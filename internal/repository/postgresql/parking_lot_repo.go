package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"

	"github.com/lib/pq" // Import driver PostgreSQL
)

type pgParkingLotRepository struct {
	db *sql.DB
}

func NewPgParkingLotRepository(db *sql.DB) repository.ParkingLotRepository {
	return &pgParkingLotRepository{db: db}
}

func (r *pgParkingLotRepository) Create(ctx context.Context, lot *domain.ParkingLot) (*domain.ParkingLot, error) {
	query := `INSERT INTO parking_lots (name, address, total_slots) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, lot.Name, lot.Address, lot.TotalSlots).Scan(&lot.ID, &lot.CreatedAt, &lot.UpdatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				return nil, fmt.Errorf("%w: tên bãi đỗ xe '%s' đã tồn tại", repository.ErrDuplicateEntry, lot.Name)
			}
		}
		return nil, fmt.Errorf("ParkingLotRepository.Create: %w", err)
	}
	lot.CreatedAt = lot.CreatedAt.In(time.UTC)
	lot.UpdatedAt = lot.UpdatedAt.In(time.UTC)
	return lot, nil
}

func (r *pgParkingLotRepository) FindByID(ctx context.Context, id int) (*domain.ParkingLot, error) {
	lot := &domain.ParkingLot{}
	query := `SELECT id, name, address, total_slots, created_at, updated_at FROM parking_lots WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&lot.ID, &lot.Name, &lot.Address, &lot.TotalSlots, &lot.CreatedAt, &lot.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("ParkingLotRepository.FindByID: %w", err)
	}
	lot.CreatedAt = lot.CreatedAt.In(time.UTC)
	lot.UpdatedAt = lot.UpdatedAt.In(time.UTC)
	return lot, nil
}

func (r *pgParkingLotRepository) FindAll(ctx context.Context) ([]domain.ParkingLot, error) {
	query := `SELECT id, name, address, total_slots, created_at, updated_at FROM parking_lots ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ParkingLotRepository.FindAll: %w", err)
	}
	defer rows.Close()

	var lots []domain.ParkingLot
	for rows.Next() {
		var lot domain.ParkingLot
		if err := rows.Scan(&lot.ID, &lot.Name, &lot.Address, &lot.TotalSlots, &lot.CreatedAt, &lot.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ParkingLotRepository.FindAll (scanning row): %w", err)
		}
		lot.CreatedAt = lot.CreatedAt.In(time.UTC)
		lot.UpdatedAt = lot.UpdatedAt.In(time.UTC)
		lots = append(lots, lot)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ParkingLotRepository.FindAll (rows error): %w", err)
	}
	return lots, nil
}

func (r *pgParkingLotRepository) Update(ctx context.Context, lot *domain.ParkingLot) (*domain.ParkingLot, error) {
	query := `UPDATE parking_lots SET name = $1, address = $2, total_slots = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 RETURNING updated_at`
	err := r.db.QueryRowContext(ctx, query, lot.Name, lot.Address, lot.TotalSlots, lot.ID).Scan(&lot.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Nên là errors.Is
			return nil, repository.ErrNotFound
		}
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				return nil, fmt.Errorf("%w: tên bãi đỗ xe '%s' đã tồn tại", repository.ErrDuplicateEntry, lot.Name)
			}
		}
		return nil, fmt.Errorf("ParkingLotRepository.Update: %w", err)
	}
	lot.UpdatedAt = lot.UpdatedAt.In(time.UTC)
	return lot, nil
}

func (r *pgParkingLotRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM parking_lots WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ParkingLotRepository.Delete: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ParkingLotRepository.Delete (checking rows affected): %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}
