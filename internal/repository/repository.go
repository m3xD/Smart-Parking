package repository

import (
	"context"
	"errors"
	"smart_parking/internal/domain"
	"time"
)

var ErrNotFound = errors.New("không tìm thấy bản ghi")
var ErrDuplicateEntry = errors.New("bản ghi đã tồn tại")
var ErrNoActiveSession = errors.New("không tìm thấy phiên đỗ xe đang hoạt động cho thông tin cung cấp")

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id int) (*domain.User, error)
}

type ParkingLotRepository interface {
	Create(ctx context.Context, lot *domain.ParkingLot) (*domain.ParkingLot, error)
	FindByID(ctx context.Context, id int) (*domain.ParkingLot, error)
	FindAll(ctx context.Context) ([]domain.ParkingLot, error)
	Update(ctx context.Context, lot *domain.ParkingLot) (*domain.ParkingLot, error)
	Delete(ctx context.Context, id int) error
}

type ParkingSlotRepository interface {
	Create(ctx context.Context, slot *domain.ParkingSlot) (*domain.ParkingSlot, error)
	FindByID(ctx context.Context, id int) (*domain.ParkingSlot, error)
	FindByLotID(ctx context.Context, lotID int) ([]domain.ParkingSlot, error)
	FindByLotIDAndSlotIdentifier(ctx context.Context, lotID int, slotIdentifier string) (*domain.ParkingSlot, error)
	FindByThingAndSlotIdentifier(ctx context.Context, esp32ThingName string, slotIdentifier string) (*domain.ParkingSlot, error)
	UpdateStatus(ctx context.Context, id int, status domain.SlotStatus, lastEventTime *time.Time, source string) error
	Update(ctx context.Context, slot *domain.ParkingSlot) (*domain.ParkingSlot, error)
	Delete(ctx context.Context, id int) error
	FindFirstAvailableByLotID(ctx context.Context, lotID int) (*domain.ParkingSlot, error)
}

type BarrierRepository interface {
	Create(ctx context.Context, barrier *domain.Barrier) (*domain.Barrier, error)
	FindByID(ctx context.Context, id int) (*domain.Barrier, error)
	FindByLotID(ctx context.Context, lotID int) ([]domain.Barrier, error)
	FindByThingAndBarrierIdentifier(ctx context.Context, esp32ThingName string, barrierIdentifier string) (*domain.Barrier, error)
	UpdateState(ctx context.Context, id int, state domain.BarrierState, lastCommand string, lastCommandTime *time.Time, source string) error
	FindByThingName(ctx context.Context, esp32ThingName string) ([]domain.Barrier, error)                    // << THÊM HÀM MỚI
	FindByLotIDAndThingName(ctx context.Context, lotID int, esp32ThingName string) ([]domain.Barrier, error) // << THÊM HÀM MỚI VÀO INTERFACE
	Update(ctx context.Context, barrier *domain.Barrier) (*domain.Barrier, error)
	Delete(ctx context.Context, id int) error
}

type DeviceEventsLogRepository interface {
	Create(ctx context.Context, event *domain.DeviceEventLog) error
}

type ParkingSessionRepository interface {
	Create(ctx context.Context, session *domain.ParkingSession) (*domain.ParkingSession, error)
	FindByID(ctx context.Context, id int) (*domain.ParkingSession, error)
	FindActiveBySlotID(ctx context.Context, slotID int) (*domain.ParkingSession, error) // Tìm theo ID của slot
	FindActiveByVehicleIdentifier(ctx context.Context, lotID int, vehicleID string) (*domain.ParkingSession, error)
	// Tìm phiên active gần nhất cho một ESP32, có thể dùng để tìm phiên cần kết thúc khi xe ra mà không có vehicleID
	FindLatestActiveByThingName(ctx context.Context, esp32ThingName string) (*domain.ParkingSession, error)
	Update(ctx context.Context, session *domain.ParkingSession) (*domain.ParkingSession, error)
	GetActiveSessionsByLot(ctx context.Context, lotID int) ([]domain.ParkingSession, error)
	// Thêm các hàm tìm kiếm khác nếu cần (ví dụ: theo khoảng thời gian, theo trạng thái)
	Find(ctx context.Context, filter domain.ParkingSessionFilterDTO) ([]domain.ParkingSession, error)
}

// Thêm DeviceRepository interface
type DeviceRepository interface {
	CreateOrUpdate(ctx context.Context, device *domain.Device) (*domain.Device, error)
	FindByThingName(ctx context.Context, thingName string) (*domain.Device, error)
	FindAll(ctx context.Context) ([]domain.Device, error)
	UpdateStatus(ctx context.Context, thingName string, status domain.DeviceStatus, lastSeenAt time.Time) error
	UpdateDetails(ctx context.Context, device *domain.Device) (*domain.Device, error)
}
