package postgresql

import (
	"database/sql"
	"fmt"
	"smart_parking/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDB(cfg *config.Config) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSslMode)

	db, err := sql.Open("pgx", psqlInfo) // Sử dụng "pgx" nếu import pgx/stdlib, "postgres" nếu dùng lib/pq
	if err != nil {
		return nil, fmt.Errorf("lỗi mở kết nối database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("lỗi ping database: %w", err)
	}
	return db, nil
}
