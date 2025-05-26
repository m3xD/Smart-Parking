package service

import (
	"context"
	"gopkg.in/guregu/null.v4"

	// "encoding/json" // Không cần ở đây nữa
	"errors"
	"fmt"
	"log"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"strings"
	"time"
)

type ParkingService struct {
	lotRepo      repository.ParkingLotRepository
	slotRepo     repository.ParkingSlotRepository
	barrierRepo  repository.BarrierRepository
	sessionRepo  repository.ParkingSessionRepository
	deviceRepo   repository.DeviceRepository // Thêm deviceRepo
	eventLogRepo repository.DeviceEventsLogRepository
}

func NewParkingService(
	lotRepo repository.ParkingLotRepository,
	slotRepo repository.ParkingSlotRepository,
	barrierRepo repository.BarrierRepository,
	sessionRepo repository.ParkingSessionRepository,
	deviceRepo repository.DeviceRepository, // Thêm vào constructor
	eventLogRepo repository.DeviceEventsLogRepository,
) *ParkingService {
	return &ParkingService{
		lotRepo:      lotRepo,
		slotRepo:     slotRepo,
		barrierRepo:  barrierRepo,
		sessionRepo:  sessionRepo,
		deviceRepo:   deviceRepo, // Gán
		eventLogRepo: eventLogRepo,
	}
}

// --- ParkingLot ---
func (s *ParkingService) CreateParkingLot(ctx context.Context, dto domain.ParkingLotDTO) (*domain.ParkingLot, error) {
	lot := &domain.ParkingLot{
		Name:       dto.Name,
		Address:    dto.Address,
		TotalSlots: dto.TotalSlots,
	}
	return s.lotRepo.Create(ctx, lot)
}

func (s *ParkingService) GetParkingLotByID(ctx context.Context, id int) (*domain.ParkingLot, error) {
	return s.lotRepo.FindByID(ctx, id)
}

func (s *ParkingService) GetAllParkingLots(ctx context.Context) ([]domain.ParkingLot, error) {
	return s.lotRepo.FindAll(ctx)
}

func (s *ParkingService) UpdateParkingLot(ctx context.Context, id int, dto domain.ParkingLotDTO) (*domain.ParkingLot, error) {
	lot, err := s.lotRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	lot.Name = dto.Name
	lot.Address = dto.Address
	lot.TotalSlots = dto.TotalSlots
	return s.lotRepo.Update(ctx, lot)
}

func (s *ParkingService) DeleteParkingLot(ctx context.Context, id int) error {
	// TODO: Cân nhắc việc xóa các slots và barriers liên quan hoặc đặt FOREIGN KEY ON DELETE CASCADE
	// Hiện tại, nếu có slot hoặc barrier thuộc lot này, việc xóa lot sẽ thất bại do ràng buộc khóa ngoại.
	// Cần xóa các slot và barrier trước, hoặc DB phải có ON DELETE CASCADE.
	// Kiểm tra xem có slot nào thuộc lot này không
	slots, err := s.slotRepo.FindByLotID(ctx, id)
	if err != nil {
		return fmt.Errorf("lỗi khi kiểm tra các chỗ đỗ của bãi %d: %w", id, err)
	}
	if len(slots) > 0 {
		return fmt.Errorf("không thể xóa bãi đỗ %d vì vẫn còn các chỗ đỗ liên kết", id)
	}
	// Kiểm tra xem có barrier nào thuộc lot này không
	barriers, err := s.barrierRepo.FindByLotID(ctx, id)
	if err != nil {
		return fmt.Errorf("lỗi khi kiểm tra các rào chắn của bãi %d: %w", id, err)
	}
	if len(barriers) > 0 {
		return fmt.Errorf("không thể xóa bãi đỗ %d vì vẫn còn các rào chắn liên kết", id)
	}
	return s.lotRepo.Delete(ctx, id)
}

// --- ParkingSlot ---
func (s *ParkingService) CreateParkingSlot(ctx context.Context, dto domain.ParkingSlotDTO) (*domain.ParkingSlot, error) {
	// Kiểm tra LotID có tồn tại không
	lot, err := s.lotRepo.FindByID(ctx, dto.LotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("bãi đỗ xe với ID %d không tồn tại", dto.LotID)
		}
		return nil, fmt.Errorf("lỗi khi kiểm tra bãi đỗ xe: %w", err)
	}

	// Tùy chọn: Kiểm tra xem số lượng slot hiện tại có vượt quá lot.TotalSlots không
	if lot.TotalSlots > 0 {
		currentSlots, err := s.slotRepo.FindByLotID(ctx, dto.LotID)
		if err != nil {
			return nil, fmt.Errorf("lỗi khi lấy số lượng chỗ đỗ hiện tại: %w", err)
		}
		if len(currentSlots) >= lot.TotalSlots {
			return nil, fmt.Errorf("số lượng chỗ đỗ đã đạt tối đa (%d) cho bãi xe này", lot.TotalSlots)
		}
	}

	slot := &domain.ParkingSlot{
		LotID:                  dto.LotID,
		SlotIdentifier:         dto.SlotIdentifier,
		Esp32ThingName:         dto.Esp32ThingName,
		Status:                 domain.StatusVacant, // Mặc định
		LastStatusUpdateSource: "admin_creation",    // Hoặc "api_creation"
	}
	return s.slotRepo.Create(ctx, slot)
}

func (s *ParkingService) GetParkingSlotByID(ctx context.Context, slotID int) (*domain.ParkingSlot, error) {
	return s.slotRepo.FindByID(ctx, slotID)
}

func (s *ParkingService) GetSlotsByLotID(ctx context.Context, lotID int) ([]domain.ParkingSlot, error) {
	return s.slotRepo.FindByLotID(ctx, lotID)
}

func (s *ParkingService) UpdateParkingSlot(ctx context.Context, slotID int, dto domain.ParkingSlotDTO) (*domain.ParkingSlot, error) {
	slot, err := s.slotRepo.FindByID(ctx, slotID)
	if err != nil {
		return nil, err
	}

	if dto.LotID != 0 && dto.LotID != slot.LotID {
		targetLot, lotErr := s.lotRepo.FindByID(ctx, dto.LotID)
		if lotErr != nil {
			if errors.Is(lotErr, repository.ErrNotFound) {
				return nil, fmt.Errorf("bãi đỗ xe mới với ID %d không tồn tại", dto.LotID)
			}
			return nil, fmt.Errorf("lỗi khi kiểm tra bãi đỗ xe mới: %w", lotErr)
		}
		// Tùy chọn: Kiểm tra TotalSlots của bãi mới
		if targetLot.TotalSlots > 0 {
			currentSlotsInNewLot, _ := s.slotRepo.FindByLotID(ctx, dto.LotID)
			if len(currentSlotsInNewLot) >= targetLot.TotalSlots {
				return nil, fmt.Errorf("số lượng chỗ đỗ đã đạt tối đa cho bãi xe mới ID %d", dto.LotID)
			}
		}
		slot.LotID = dto.LotID
	}
	if dto.SlotIdentifier != "" {
		slot.SlotIdentifier = dto.SlotIdentifier
	}
	if dto.Esp32ThingName != "" {
		slot.Esp32ThingName = dto.Esp32ThingName
	}
	if dto.Status != "" {
		validStatus := false
		for _, valid_s := range []domain.SlotStatus{domain.StatusVacant, domain.StatusOccupied, domain.StatusMaintenance, domain.StatusReserved} {
			if domain.SlotStatus(dto.Status) == valid_s {
				validStatus = true
				break
			}
		}
		if !validStatus {
			return nil, fmt.Errorf("trạng thái slot không hợp lệ: %s", dto.Status)
		}
		slot.Status = domain.SlotStatus(dto.Status)
	}
	slot.LastStatusUpdateSource = "admin_update" // Hoặc "api_update"

	return s.slotRepo.Update(ctx, slot)
}

func (s *ParkingService) DeleteParkingSlot(ctx context.Context, slotID int) error {
	return s.slotRepo.Delete(ctx, slotID)
}

func (s *ParkingService) UpdateParkingSlotStatusFromDevice(ctx context.Context, event domain.DeviceParkingSlotEvent) error {
	status := domain.StatusOccupied

	if !event.IsOccupied {
		status = domain.StatusVacant
	}
	log.Printf("Service: Đang cập nhật trạng thái cho slot '%s' (ESP32: '%s') thành '%s'", event.SlotID, event.DeviceID, status)

	slot, err := s.slotRepo.FindByThingAndSlotIdentifier(ctx, event.DeviceID, event.SlotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Printf("Không tìm thấy slot cho ThingID '%s' và SlotIdentifier '%s'.", event.DeviceID, event.SlotID)
			// TODO: Nếu không tìm thấy, có thể tạo mới slot nếu logic cho phép, hoặc báo lỗi rõ ràng.
			// Ví dụ: tìm lot_id dựa trên event.DeviceID (nếu có bảng mapping device to lot)
			// rồi tạo slot mới. Hiện tại, chúng ta sẽ báo lỗi.
			return fmt.Errorf("%w: slot '%s' cho thiết bị '%s' không được đăng ký trong hệ thống", repository.ErrNotFound, event.SlotID, event.DeviceID)
		}
		log.Printf("Lỗi khi tìm slot bằng ThingID '%s' và SlotIdentifier '%s': %v", event.DeviceID, event.SlotID, err)
		return fmt.Errorf("lỗi tìm slot: %w", err)
	}

	// Chuyển đổi timestamp từ string sang time.Time
	parsedTime := time.Now()

	// Chỉ cập nhật nếu trạng thái thực sự thay đổi HOẶC nếu sự kiện này mới hơn sự kiện đã lưu cuối cùng
	// Điều này giúp xử lý các message đến không theo thứ tự hoặc message trùng lặp.
	if slot.Status != status || slot.LastEventTimestamp == nil || (slot.LastEventTimestamp != nil && parsedTime.After(*slot.LastEventTimestamp)) {
		err = s.slotRepo.UpdateStatus(ctx, slot.ID, status, &parsedTime, "device")
		if err != nil {
			log.Printf("Lỗi khi cập nhật trạng thái slot ID %d: %v", slot.ID, err)
			return fmt.Errorf("lỗi cập nhật trạng thái slot: %w", err)
		}
		log.Printf("Đã cập nhật trạng thái slot ID %d (Identifier: %s, LotID: %d) thành %s", slot.ID, slot.SlotIdentifier, slot.LotID, status)
	} else {
		log.Printf("Trạng thái slot ID %d (Identifier: %s) không thay đổi (%s) hoặc sự kiện cũ hơn (DB: %v, Event: %v). Bỏ qua cập nhật.",
			slot.ID, slot.SlotIdentifier, status, slot.LastEventTimestamp, parsedTime)
	}
	return nil
}

// --- Barrier Service Logic ---
func (s *ParkingService) CreateBarrier(ctx context.Context, dto domain.BarrierDTO) (*domain.Barrier, error) {
	_, err := s.lotRepo.FindByID(ctx, dto.LotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("bãi đỗ xe với ID %d không tồn tại", dto.LotID)
		}
		return nil, fmt.Errorf("lỗi khi kiểm tra bãi đỗ xe: %w", err)
	}
	barrier := &domain.Barrier{
		LotID:                 dto.LotID,
		BarrierIdentifier:     dto.BarrierIdentifier,
		Esp32ThingName:        dto.Esp32ThingName,
		BarrierType:           dto.BarrierType,
		CurrentState:          domain.StateClosed, // Mặc định khi tạo
		LastStateUpdateSource: "admin_creation",
	}
	if dto.CurrentState != "" {
		// Validate barrier state
		validState := false
		for _, vs := range []domain.BarrierState{domain.StateOpenedAuto, domain.StateOpenedCommand, domain.StateClosedAuto, domain.StateClosedCommand, domain.StateError, domain.StateUnknown} {
			if domain.BarrierState(dto.CurrentState) == vs {
				validState = true
				break
			}
		}
		if !validState {
			return nil, fmt.Errorf("trạng thái rào chắn không hợp lệ: %s", dto.CurrentState)
		}
		barrier.CurrentState = domain.BarrierState(dto.CurrentState)
	}
	return s.barrierRepo.Create(ctx, barrier)
}

func (s *ParkingService) GetBarrierByID(ctx context.Context, barrierID int) (*domain.Barrier, error) {
	return s.barrierRepo.FindByID(ctx, barrierID)
}

func (s *ParkingService) GetBarriersByLotID(ctx context.Context, lotID int) ([]domain.Barrier, error) {
	return s.barrierRepo.FindByLotID(ctx, lotID)
}

func (s *ParkingService) UpdateBarrier(ctx context.Context, barrierID int, dto domain.BarrierDTO) (*domain.Barrier, error) {
	barrier, err := s.barrierRepo.FindByID(ctx, barrierID)
	if err != nil {
		return nil, err
	}

	if dto.LotID != 0 && dto.LotID != barrier.LotID {
		_, lotErr := s.lotRepo.FindByID(ctx, dto.LotID)
		if lotErr != nil {
			return nil, fmt.Errorf("bãi đỗ xe mới ID %d không tồn tại", dto.LotID)
		}
		barrier.LotID = dto.LotID
	}
	if dto.BarrierIdentifier != "" {
		barrier.BarrierIdentifier = dto.BarrierIdentifier
	}
	if dto.Esp32ThingName != "" {
		barrier.Esp32ThingName = dto.Esp32ThingName
	}
	if dto.BarrierType != "" {
		if dto.BarrierType != "entry" && dto.BarrierType != "exit" {
			return nil, fmt.Errorf("loại rào chắn không hợp lệ: %s", dto.BarrierType)
		}
		barrier.BarrierType = dto.BarrierType
	}
	if dto.CurrentState != "" {
		// Validate barrier state
		barrier.CurrentState = domain.BarrierState(dto.CurrentState)
	}
	barrier.LastStateUpdateSource = "admin_update"

	return s.barrierRepo.Update(ctx, barrier)
}

func (s *ParkingService) DeleteBarrier(ctx context.Context, barrierID int) error {
	return s.barrierRepo.Delete(ctx, barrierID)
}

func (s *ParkingService) UpdateBarrierStateFromDevice(ctx context.Context, event domain.DeviceBarrierStateEvent) error {
	log.Printf("Service: Đang cập nhật trạng thái cho rào chắn '%s' (ESP32: '%s') thành '%s'", event.BarrierID, event.DeviceID, event.BarrierState)

	// event.BarrierID từ ESP32 có dạng "ESP32_ParkingController_01_entry"
	// Chúng ta cần tìm barrier dựa trên Esp32ThingName và BarrierIdentifier (phần cuối của event.BarrierID)
	// Hoặc nếu BarrierIdentifier trong DB lưu đầy đủ "ESP32_ParkingController_01_entry" thì dùng trực tiếp.
	// Dựa trên schema, barrier_identifier trong DB là "entry_barrier_1", "exit_barrier_1"
	// Và esp32_thing_name là Thing Name.
	// => event.BarrierID từ ESP32 nên là "entry_barrier_1" hoặc "exit_barrier_1"
	// và event.DeviceID là Thing Name.
	barrier, err := s.barrierRepo.FindByThingAndBarrierIdentifier(ctx, event.DeviceID, event.BarrierID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Printf("Không tìm thấy rào chắn cho ThingID '%s' và BarrierIdentifier '%s'.", event.DeviceID, event.BarrierID)
			return fmt.Errorf("%w: rào chắn '%s' cho thiết bị '%s' không được đăng ký", repository.ErrNotFound, event.BarrierID, event.DeviceID)
		}
		log.Printf("Lỗi khi tìm rào chắn bằng ThingID '%s' và BarrierIdentifier '%s': %v", event.DeviceID, event.BarrierID, err)
		return fmt.Errorf("lỗi tìm rào chắn: %w", err)
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
	if err != nil {
		log.Printf("Lỗi parse timestamp '%s' cho sự kiện rào chắn: %v. Sử dụng thời gian hiện tại.", event.Timestamp, err)
		parsedTime = time.Now().UTC()
	}

	var commandFromState string
	if strings.HasSuffix(string(event.BarrierState), "_command") {
		commandFromState = strings.TrimSuffix(string(event.BarrierState), "_command")
	} else if strings.HasSuffix(string(event.BarrierState), "_auto") {
		commandFromState = strings.TrimSuffix(string(event.BarrierState), "_auto")
	}

	if barrier.CurrentState != event.BarrierState || barrier.LastCommandTimestamp == nil || (barrier.LastCommandTimestamp != nil && parsedTime.After(*barrier.LastCommandTimestamp)) {
		err = s.barrierRepo.UpdateState(ctx, barrier.ID, event.BarrierState, commandFromState, &parsedTime, "device")
		if err != nil {
			log.Printf("Lỗi khi cập nhật trạng thái rào chắn ID %d: %v", barrier.ID, err)
			return fmt.Errorf("lỗi cập nhật trạng thái rào chắn: %w", err)
		}
		log.Printf("Đã cập nhật trạng thái rào chắn ID %d (Identifier: %s) thành %s", barrier.ID, barrier.BarrierIdentifier, event.BarrierState)
	} else {
		log.Printf("Trạng thái rào chắn ID %d (Identifier: %s) không thay đổi (%s) hoặc sự kiện cũ hơn. Bỏ qua cập nhật.", barrier.ID, barrier.BarrierIdentifier, event.BarrierState)
	}
	return nil
}

func (s *ParkingService) RecordGateSensorEvent(ctx context.Context, event domain.DeviceGateSensorEvent) error {
	log.Printf("Service: Ghi nhận sự kiện cảm biến cổng: Device='%s', Sensor='%s', Area='%s', EventType='%s'",
		event.DeviceID, event.SensorID, event.GateArea, event.EventType)

	// TODO: Logic nghiệp vụ chi tiết cho sự kiện cảm biến cổng
	// Ví dụ:
	// - Ghi log sự kiện này vào bảng device_events_log (đã được thực hiện ở IoTService).
	// - Nếu event.EventType == "vehicle_passed" và event.GateArea == "entry_passed":
	//   - Tìm ParkingSlot trống gần nhất (logic phức tạp hơn cho Giai đoạn sau).
	//   - Tạo một "phiên đỗ xe" (ParkingSession) mới, liên kết với slot đó (nếu tìm được) và vehicle_id (nếu có LPR).
	// - Nếu event.EventType == "vehicle_passed" và event.GateArea == "exit_passed":
	//   - Tìm ParkingSession đang mở của xe đó.
	//   // - Kết thúc ParkingSession, tính toán thời gian đỗ, phí.
	//   - Cập nhật trạng thái ParkingSlot thành "vacant".

	// Hiện tại, chúng ta có thể chỉ ghi nhận sự kiện.
	// Nếu có bảng parking_sessions, đây là nơi để tương tác với nó.
	return nil
}

// --- ParkingSession Logic ---
func (s *ParkingService) StartParkingSession(ctx context.Context, event domain.DeviceGateSensorEvent) (*domain.ParkingSession, error) {
	log.Printf("Service: Bắt đầu phiên đỗ xe dựa trên sự kiện cổng: Device='%s', Sensor='%s', Area='%s'",
		event.DeviceID, event.SensorID, event.GateArea)

	// Tìm thiết bị để lấy lot_id (nếu có)
	device, err := s.deviceRepo.FindByThingName(ctx, event.DeviceID)
	if err != nil || device == nil || !device.LotID.Valid {
		log.Printf("Không tìm thấy thiết bị hoặc lot_id cho thiết bị %s. Sử dụng lot_id mặc định hoặc báo lỗi.", event.DeviceID)
		// return nil, fmt.Errorf("không thể xác định bãi đỗ xe cho thiết bị %s", event.DeviceID)
		// Tạm thời dùng một lot_id cố định nếu không tìm thấy, hoặc bạn cần logic khác
		// Ví dụ, nếu không có device.LotID, có thể tìm barrier thuộc device này và lấy lot_id từ barrier
	}

	var lotID int
	if device != nil && device.LotID.Valid {
		lotID = int(device.LotID.Int64)
	} else {
		// Cần logic dự phòng để xác định lotID nếu device không có lot_id
		// Ví dụ: Tìm barrier do device này quản lý và có type là "entry"
		barriers, bErr := s.barrierRepo.FindByLotIDAndThingName(ctx, 0, event.DeviceID) // Cần hàm FindByThingName
		if bErr == nil && len(barriers) > 0 {
			for _, b := range barriers {
				if b.BarrierType == "entry" { // Giả sử sự kiện vào cổng là từ rào vào
					lotID = b.LotID
					break
				}
			}
		}
		if lotID == 0 { // Vẫn không tìm được lotID
			log.Printf("Không thể xác định LotID cho thiết bị %s từ barrier. Cần cấu hình LotID cho thiết bị hoặc rào chắn.", event.DeviceID)
			return nil, fmt.Errorf("không thể xác định bãi đỗ xe cho thiết bị %s", event.DeviceID)
		}
	}

	// Tùy chọn: Tìm một chỗ đỗ trống tự động
	availableSlot, err := s.slotRepo.FindFirstAvailableByLotID(ctx, lotID)
	var sessionSlotID null.Int
	if err == nil && availableSlot != nil {
		sessionSlotID = null.IntFrom(int64(availableSlot.ID))
		parsedEventTime, _ := time.Parse(time.RFC3339Nano, event.Timestamp)
		if errTime := s.slotRepo.UpdateStatus(ctx, availableSlot.ID, domain.StatusOccupied, &parsedEventTime, "session_start"); errTime != nil {
			log.Printf("Lỗi khi cập nhật trạng thái slot %d thành occupied: %v", availableSlot.ID, errTime)
			// Không block việc tạo session, nhưng cần log lại
		} else {
			log.Printf("Đã gán chỗ đỗ %s (ID: %d) cho phiên mới.", availableSlot.SlotIdentifier, availableSlot.ID)
		}
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("Lỗi khi tìm chỗ đỗ trống: %v. Phiên sẽ không có slot_id cụ thể.", err)
	} else {
		log.Printf("Không tìm thấy chỗ đỗ trống tự động. Phiên sẽ không có slot_id cụ thể.")
	}

	entryTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
	if err != nil {
		log.Printf("Lỗi parse entry time: %v. Sử dụng thời gian hiện tại.", err)
		entryTime = time.Now().UTC()
	}

	session := &domain.ParkingSession{
		LotID:            lotID,
		SlotID:           sessionSlotID,
		Esp32ThingName:   event.DeviceID,
		EntryTime:        entryTime,
		PaymentStatus:    "pending",
		Status:           domain.SessionActive,
		EntryGateEventID: null.StringFrom(event.EventID),
	}

	createdSession, err := s.sessionRepo.Create(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("lỗi tạo phiên đỗ xe: %w", err)
	}
	log.Printf("Đã tạo phiên đỗ xe mới ID: %d cho thiết bị %s tại bãi %d", createdSession.ID, event.DeviceID, lotID)
	return createdSession, nil
}

func (s *ParkingService) EndParkingSession(ctx context.Context, event domain.DeviceGateSensorEvent) (*domain.ParkingSession, error) {
	log.Printf("Service: Kết thúc phiên đỗ xe dựa trên sự kiện cổng: Device='%s', Sensor='%s', Area='%s'",
		event.DeviceID, event.SensorID, event.GateArea)

	activeSession, err := s.sessionRepo.FindLatestActiveByThingName(ctx, event.DeviceID)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveSession) {
			log.Printf("Không tìm thấy phiên đỗ xe đang hoạt động cho thiết bị %s (có thể xe vào bằng cổng khác hoặc đã ra)", event.DeviceID)
			return nil, repository.ErrNoActiveSession
		}
		return nil, fmt.Errorf("lỗi tìm phiên đỗ xe đang hoạt động: %w", err)
	}

	exitTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
	if err != nil {
		log.Printf("Lỗi parse exit time: %v. Sử dụng thời gian hiện tại.", err)
		exitTime = time.Now().UTC()
	}

	activeSession.ExitTime = null.TimeFrom(exitTime)
	activeSession.Status = domain.SessionCompleted
	activeSession.ExitGateEventID = null.StringFrom(event.EventID)

	duration := exitTime.Sub(activeSession.EntryTime)
	activeSession.DurationMinutes = null.IntFrom(int64(duration.Minutes()))

	// TODO: Tính toán phí dựa trên biểu phí (ví dụ: từ bảng `tariffs` liên kết với `parking_lots`)
	if activeSession.DurationMinutes.Valid {
		// Ví dụ: 1000 VND/phút, tối thiểu 5000 VND
		fee := float64(activeSession.DurationMinutes.Int64) * 1000.0
		if fee < 5000.0 && activeSession.DurationMinutes.Int64 > 0 { // Phí tối thiểu nếu có đỗ
			fee = 5000.0
		} else if activeSession.DurationMinutes.Int64 <= 0 { // Không tính phí nếu thời gian < 0 hoặc = 0
			fee = 0
		}
		activeSession.CalculatedFee = null.FloatFrom(fee)
	} else {
		activeSession.CalculatedFee = null.FloatFrom(0)
	}
	// activeSession.PaymentStatus sẽ được cập nhật sau

	updatedSession, err := s.sessionRepo.Update(ctx, activeSession)
	if err != nil {
		return nil, fmt.Errorf("lỗi cập nhật phiên đỗ xe: %w", err)
	}

	if activeSession.SlotID.Valid {
		err = s.slotRepo.UpdateStatus(ctx, int(activeSession.SlotID.Int64), domain.StatusVacant, &exitTime, "session_end")
		if err != nil {
			log.Printf("Lỗi cập nhật trạng thái chỗ đỗ %d thành trống: %v", activeSession.SlotID.Int64, err)
		} else {
			log.Printf("Đã cập nhật chỗ đỗ ID %d thành trống.", activeSession.SlotID.Int64)
		}
	}

	log.Printf("Đã kết thúc phiên đỗ xe ID: %d. Thời gian đỗ: %d phút. Phí (tạm tính): %.2f",
		updatedSession.ID, updatedSession.DurationMinutes.Int64, updatedSession.CalculatedFee.Float64)
	return updatedSession, nil
}

func (s *ParkingService) GetParkingSessionByID(ctx context.Context, sessionID int) (*domain.ParkingSession, error) {
	return s.sessionRepo.FindByID(ctx, sessionID)
}

func (s *ParkingService) GetActiveSessionsByLot(ctx context.Context, lotID int) ([]domain.ParkingSession, error) {
	return s.sessionRepo.GetActiveSessionsByLot(ctx, lotID)
}

func (s *ParkingService) FindParkingSessions(ctx context.Context, filter domain.ParkingSessionFilterDTO) ([]domain.ParkingSession, error) {
	return s.sessionRepo.Find(ctx, filter)
}

// --- Device Monitoring Logic ---
func (s *ParkingService) HandleDeviceStartup(ctx context.Context, event domain.DeviceStartupInfoEvent) error {
	log.Printf("Service: Xử lý thông tin khởi động từ thiết bị '%s', Firmware: %s", event.ClientIDFromIoT, event.FirmwareVersion)

	//now := time.Now().UTC()
	//var rssiVal null.Int
	//if event.Rssi != 0 { // Kiểm tra giá trị mặc định của int
	//	rssiVal = null.IntFrom(int64(event.Rssi))
	//}

	//device := &domain.Device{
	//	ThingName:       event.ClientIDFromIoT,
	//	FirmwareVersion: event.FirmwareVersion,
	//	LastSeenAt:      null.TimeFrom(now),
	//	Status:          domain.DeviceOnline,
	//	IPAddress:       event.Ip,
	//	MacAddress:      event.Mac,
	//	LastRssi:        rssiVal,
	//	// LotID: Cần logic để xác định LotID nếu ESP32 này quản lý một bãi cụ thể
	//}
	//_, err := s.deviceRepo.CreateOrUpdate(ctx, device)
	//if err != nil {
	//	log.Printf("Lỗi khi cập nhật/tạo thông tin thiết bị '%s': %v", event.ClientIDFromIoT, err)
	//	return err
	//}
	log.Printf("Đã cập nhật trạng thái khởi động cho thiết bị '%s'", event.ClientIDFromIoT)
	return nil
}

func (s *ParkingService) HandleParkingSummary(ctx context.Context, event domain.DeviceParkingSummaryEvent) error {
	log.Printf("Service: Xử lý thông tin tóm tắt bãi đỗ từ thiết bị '%s': %d/%d chỗ có xe",
		event.DeviceID, event.OccupiedSlots, event.TotalSlots)
	// TODO: Logic nghiệp vụ
	// - So sánh với trạng thái hiện tại trong DB để phát hiện bất đồng bộ (nếu có).
	// - Cập nhật last_seen_at cho thiết bị.
	s.deviceRepo.UpdateStatus(ctx, event.ClientIDFromIoT, domain.DeviceOnline, time.Now().UTC())
	return nil
}

func (s *ParkingService) HandleSystemStatus(ctx context.Context, event domain.DeviceSystemStatusEvent) error {
	log.Printf("Service: Xử lý thông tin trạng thái hệ thống từ thiết bị '%s': Uptime %ds, Free Heap %d, RSSI %d, MQTT Connected: %t",
		event.DeviceID, event.UptimeSeconds, event.FreeHeap, event.WifiRSSI, event.MqttConnected)

	now := time.Now().UTC()
	device, err := s.deviceRepo.FindByThingName(ctx, event.DeviceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) { // Nếu thiết bị chưa có, tạo mới
			device = &domain.Device{ThingName: event.DeviceID}
		} else {
			log.Printf("Lỗi tìm thiết bị '%s': %v", event.DeviceID, err)
			return err
		}
	}

	device.LastSeenAt = null.TimeFrom(now)
	device.Status = domain.DeviceOnline
	if event.FirmwareVersion != "" {
		device.FirmwareVersion = event.FirmwareVersion
	}
	if event.WifiIP != "" {
		device.IPAddress = event.WifiIP
	}
	if event.WifiMAC != "" {
		device.MacAddress = event.WifiMAC
	}
	if event.WifiRSSI != 0 {
		device.LastRssi = null.IntFrom(int64(event.WifiRSSI))
	}
	if event.FreeHeap != 0 {
		device.LastFreeHeap = null.IntFrom(int64(event.FreeHeap))
	} // uint32 to int64
	if event.UptimeSeconds != 0 {
		device.LastUptimeSeconds = null.IntFrom(event.UptimeSeconds)
	}

	_, err = s.deviceRepo.CreateOrUpdate(ctx, device)
	if err != nil {
		log.Printf("Lỗi khi cập nhật chi tiết thiết bị '%s' từ system_status: %v", event.DeviceID, err)
		return err
	}
	log.Printf("Đã cập nhật trạng thái hệ thống cho thiết bị '%s'", event.DeviceID)
	return nil
}

func (s *ParkingService) HandleDeviceError(ctx context.Context, event domain.DeviceErrorEvent) error {
	log.Printf("Service: Xử lý LỖI từ thiết bị '%s': Code %d, Message: '%s', ErrorID: '%s'",
		event.DeviceID, event.ErrorCode, event.ErrorMessage, event.ErrorID)

	s.deviceRepo.UpdateStatus(ctx, event.DeviceID, domain.DeviceErrorStatus, time.Now().UTC())
	// TODO: Gửi thông báo cho admin.
	// TODO: Ghi nhận lỗi vào hệ thống theo dõi lỗi chi tiết hơn (ngoài device_events_log).
	return nil
}

func (s *ParkingService) HandleCommandAck(ctx context.Context, event domain.DeviceCommandAckEvent) error {
	log.Printf("Service: Xử lý xác nhận lệnh từ thiết bị '%s': Action '%s', RequestID '%s', Status: '%s'",
		event.DeviceID, event.ReceivedAction, event.RequestID, event.Status)
	// TODO: Logic nghiệp vụ
	// - Cập nhật trạng thái của một lệnh đã gửi trước đó trong bảng command_log (nếu có).
	s.deviceRepo.UpdateStatus(ctx, event.DeviceID, domain.DeviceOnline, time.Now().UTC()) // Thiết bị online vì đã gửi ack
	return nil
}

// --- Device Management API Logic ---
func (s *ParkingService) GetAllDevices(ctx context.Context) ([]domain.Device, error) {
	return s.deviceRepo.FindAll(ctx)
}

func (s *ParkingService) GetDeviceByThingName(ctx context.Context, thingName string) (*domain.Device, error) {
	return s.deviceRepo.FindByThingName(ctx, thingName)
}

// --- ParkingSession Logic ---
func (s *ParkingService) VehicleCheckIn(ctx context.Context, dto domain.VehicleCheckInDTO) (*domain.ParkingSession, error) {
	log.Printf("Service: Ghi nhận xe vào cổng (API): LotID=%d, ESP32='%s', Biển số='%s'",
		dto.LotID, dto.Esp32ThingName, dto.VehicleIdentifier)

	// 1. Xác thực LotID
	lot, err := s.lotRepo.FindByID(ctx, dto.LotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("bãi đỗ xe với ID %d không tồn tại", dto.LotID)
		}
		return nil, fmt.Errorf("lỗi khi kiểm tra bãi đỗ xe: %w", err)
	}

	// 2. Kiểm tra xem có phiên active nào cho biển số này trong bãi này chưa
	// (Điều này quan trọng để tránh check-in trùng lặp)
	existingActiveSession, err := s.sessionRepo.FindActiveByVehicleIdentifier(ctx, dto.LotID, dto.VehicleIdentifier)
	if err != nil && !errors.Is(err, repository.ErrNoActiveSession) && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("lỗi kiểm tra phiên hoạt động: %w", err)
	}
	if existingActiveSession != nil {
		log.Printf("Xe '%s' đã có phiên đang hoạt động (ID: %d) trong bãi %d.", dto.VehicleIdentifier, existingActiveSession.ID, dto.LotID)
		return nil, fmt.Errorf("%w: xe '%s' đã ở trong bãi", repository.ErrDuplicateEntry, dto.VehicleIdentifier)
	}

	// 3. Xác định EntryTime
	var entryTime time.Time
	if dto.EntryTime != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, dto.EntryTime)
		if err != nil {
			log.Printf("Lỗi parse entry time từ DTO: %v. Sử dụng thời gian hiện tại của server.", err)
			entryTime = time.Now().UTC()
		} else {
			entryTime = parsedTime.UTC()
		}
	} else {
		entryTime = time.Now().UTC()
	}

	// 4. Tùy chọn: Tìm một chỗ đỗ trống tự động
	var sessionSlotID null.Int
	// Chỉ tìm slot nếu lot này có cấu hình total_slots > 0 (nghĩa là quản lý slot cụ thể)
	if lot.TotalSlots > 0 {
		availableSlot, err := s.slotRepo.FindFirstAvailableByLotID(ctx, dto.LotID)
		if err == nil && availableSlot != nil {
			sessionSlotID = null.IntFrom(int64(availableSlot.ID))
			// Cập nhật trạng thái slot này thành occupied
			if errTime := s.slotRepo.UpdateStatus(ctx, availableSlot.ID, domain.StatusOccupied, &entryTime, "session_check_in"); errTime != nil {
				log.Printf("Lỗi khi cập nhật trạng thái slot %d thành occupied: %v", availableSlot.ID, errTime)
				// Quyết định: Có block việc tạo session không? Hay chỉ log?
				// Hiện tại, vẫn cho tạo session nhưng slot có thể không được update.
			} else {
				log.Printf("Đã gán chỗ đỗ %s (ID: %d) cho phiên check-in mới.", availableSlot.SlotIdentifier, availableSlot.ID)
			}
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			log.Printf("Lỗi khi tìm chỗ đỗ trống cho bãi %d: %v. Phiên sẽ không có slot_id cụ thể.", dto.LotID, err)
		} else {
			log.Printf("Không tìm thấy chỗ đỗ trống tự động cho bãi %d. Phiên sẽ không có slot_id cụ thể.", dto.LotID)
		}
	} else {
		log.Printf("Bãi đỗ %d không quản lý chỗ đỗ cụ thể (total_slots=0). Phiên sẽ không có slot_id.", dto.LotID)
	}

	// 5. Tạo bản ghi ParkingSession mới
	session := &domain.ParkingSession{
		LotID:             dto.LotID,
		SlotID:            sessionSlotID,
		Esp32ThingName:    dto.Esp32ThingName,
		VehicleIdentifier: null.StringFrom(dto.VehicleIdentifier),
		EntryTime:         entryTime,
		PaymentStatus:     "pending",
		Status:            domain.SessionActive,
		// EntryGateEventID: dto.EntryGateEventID, // Nếu frontend gửi
	}

	createdSession, err := s.sessionRepo.Create(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("lỗi tạo phiên đỗ xe: %w", err)
	}
	log.Printf("Đã tạo phiên đỗ xe mới ID: %d cho xe '%s' tại bãi %d", createdSession.ID, dto.VehicleIdentifier, dto.LotID)
	return createdSession, nil
}

func (s *ParkingService) VehicleCheckOut(ctx context.Context, dto domain.VehicleCheckOutDTO) (*domain.ParkingSession, error) {
	log.Printf("Service: Ghi nhận xe ra cổng (API): LotID=%d, ESP32='%s', Biển số='%s'",
		dto.LotID, dto.Esp32ThingName, dto.VehicleIdentifier)

	// 1. Xác thực LotID
	_, err := s.lotRepo.FindByID(ctx, dto.LotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("bãi đỗ xe với ID %d không tồn tại", dto.LotID)
		}
		return nil, fmt.Errorf("lỗi khi kiểm tra bãi đỗ xe: %w", err)
	}

	// 2. Tìm phiên đang active cho biển số này trong bãi này
	activeSession, err := s.sessionRepo.FindActiveByVehicleIdentifier(ctx, dto.LotID, dto.VehicleIdentifier)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveSession) || errors.Is(err, repository.ErrNotFound) {
			log.Printf("Không tìm thấy phiên đỗ xe đang hoạt động cho xe '%s' tại bãi %d.", dto.VehicleIdentifier, dto.LotID)
			return nil, fmt.Errorf("%w: không có xe '%s' đang đỗ tại bãi này", repository.ErrNoActiveSession, dto.VehicleIdentifier)
		}
		return nil, fmt.Errorf("lỗi tìm phiên đỗ xe đang hoạt động: %w", err)
	}

	// 3. Xác định ExitTime
	var exitTime time.Time
	if dto.ExitTime != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, dto.ExitTime)
		if err != nil {
			log.Printf("Lỗi parse exit time từ DTO: %v. Sử dụng thời gian hiện tại của server.", err)
			exitTime = time.Now().UTC()
		} else {
			exitTime = parsedTime.UTC()
		}
	} else {
		exitTime = time.Now().UTC()
	}

	// Đảm bảo exitTime không sớm hơn entryTime
	if exitTime.Before(activeSession.EntryTime) {
		log.Printf("Thời gian ra (%v) sớm hơn thời gian vào (%v) của phiên %d. Sử dụng thời gian vào làm thời gian ra.", exitTime, activeSession.EntryTime, activeSession.ID)
		exitTime = activeSession.EntryTime
	}

	// 4. Cập nhật thông tin cho phiên
	activeSession.ExitTime = null.TimeFrom(exitTime)
	activeSession.Status = domain.SessionCompleted
	// activeSession.ExitGateEventID = null.StringFrom(dto.ExitGateEventID) // Nếu frontend gửi

	duration := exitTime.Sub(activeSession.EntryTime)
	activeSession.DurationMinutes = null.IntFrom(int64(duration.Minutes()))

	// 5. Tính toán phí (ví dụ đơn giản)
	if activeSession.DurationMinutes.Valid {
		fee := float64(activeSession.DurationMinutes.Int64) * 1000.0 // 1000 VND/phút
		if fee < 5000.0 && activeSession.DurationMinutes.Int64 > 0 {
			fee = 5000.0
		} else if activeSession.DurationMinutes.Int64 <= 0 {
			fee = 0
		}
		activeSession.CalculatedFee = null.FloatFrom(fee)
	} else {
		activeSession.CalculatedFee = null.FloatFrom(0)
	}
	// activeSession.PaymentStatus sẽ được cập nhật bởi một quy trình thanh toán riêng

	// 6. Lưu cập nhật phiên
	updatedSession, err := s.sessionRepo.Update(ctx, activeSession)
	if err != nil {
		return nil, fmt.Errorf("lỗi cập nhật phiên đỗ xe: %w", err)
	}

	// 7. Cập nhật trạng thái chỗ đỗ (nếu có) thành vacant
	if activeSession.SlotID.Valid {
		err = s.slotRepo.UpdateStatus(ctx, int(activeSession.SlotID.Int64), domain.StatusVacant, &exitTime, "session_check_out")
		if err != nil {
			log.Printf("Lỗi cập nhật trạng thái chỗ đỗ %d thành trống: %v", activeSession.SlotID.Int64, err)
			// Không block việc kết thúc session nếu chỉ lỗi cập nhật slot status
		} else {
			log.Printf("Đã cập nhật chỗ đỗ ID %d thành trống.", activeSession.SlotID.Int64)
		}
	}

	log.Printf("Đã kết thúc phiên đỗ xe ID: %d cho xe '%s'. Thời gian đỗ: %d phút. Phí (tạm tính): %.2f",
		updatedSession.ID, dto.VehicleIdentifier, updatedSession.DurationMinutes.Int64, updatedSession.CalculatedFee.Float64)
	return updatedSession, nil
}
