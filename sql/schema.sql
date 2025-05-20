-- Bảng lưu thông tin người dùng (cho Authentication)
CREATE TABLE IF NOT EXISTS users
(
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'operator', -- Ví dụ: 'admin', 'operator'
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Bảng lưu thông tin các thiết bị ESP32
CREATE TABLE IF NOT EXISTS devices
(
    id                  SERIAL PRIMARY KEY,
    thing_name          VARCHAR(100) NOT NULL UNIQUE,                                 -- Khớp với SECRET_AWS_THING_NAME của ESP32
    lot_id              INT          REFERENCES parking_lots (id) ON DELETE SET NULL, -- Bãi đỗ mà thiết bị này quản lý (nếu có, và 1 ESP32 chỉ thuộc 1 bãi)
    firmware_version    VARCHAR(50),
    last_seen_at        TIMESTAMPTZ,
    status              VARCHAR(20)           DEFAULT 'unknown',                      -- 'online', 'offline', 'error', 'maintenance'
    ip_address          VARCHAR(45),
    mac_address         VARCHAR(17),
    last_rssi           INT,
    last_free_heap      INT,
    last_uptime_seconds BIGINT,
    notes               TEXT,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Bảng lưu thông tin các bãi đỗ xe
CREATE TABLE IF NOT EXISTS parking_lots
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    address     TEXT,
    total_slots INT                   DEFAULT 0, -- Tổng số chỗ có thể có (cấu hình)
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Bảng lưu thông tin các chỗ đỗ xe cụ thể trong một bãi
CREATE TABLE IF NOT EXISTS parking_slots
(
    id                        SERIAL PRIMARY KEY,
    lot_id                    INT         NOT NULL REFERENCES parking_lots (id) ON DELETE CASCADE,
    slot_identifier           VARCHAR(50) NOT NULL,                  -- Định danh của chỗ đỗ trong bãi, ví dụ: "S1", "A01" (sẽ khớp với slotMqttId từ ESP32)
    esp32_thing_name          VARCHAR(100),                          -- Thing Name của ESP32 quản lý chỗ đỗ này
    status                    VARCHAR(20) NOT NULL DEFAULT 'vacant', -- 'vacant', 'occupied', 'maintenance', 'reserved'
    last_status_update_source VARCHAR(50),                           -- Nguồn cập nhật cuối cùng: 'device', 'admin', 'system'
    last_event_timestamp      TIMESTAMPTZ,                           -- Thời gian của sự kiện cuối cùng từ thiết bị
    created_at                TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (lot_id, slot_identifier)                                 -- Đảm bảo mỗi chỗ đỗ là duy nhất trong một bãi
    -- Cân nhắc thêm UNIQUE (esp32_thing_name, slot_identifier) nếu một ESP32 có thể quản lý nhiều chỗ với slot_identifier giống nhau ở các bãi khác nhau (ít khả năng)
    -- Hoặc UNIQUE (esp32_thing_name, esp32_sensor_id_on_device) nếu ESP32 gửi một ID cảm biến riêng.
    -- Hiện tại, slot_identifier từ ESP32 ("S1", "S2") được dùng để map.
);

-- Bảng lưu thông tin các rào chắn
CREATE TABLE IF NOT EXISTS barriers
(
    id                       SERIAL PRIMARY KEY,
    lot_id                   INT          NOT NULL REFERENCES parking_lots (id) ON DELETE CASCADE,
    barrier_identifier       VARCHAR(100) NOT NULL,                  -- Định danh của rào chắn, ví dụ: "entry_barrier_1", "ESP32_ParkingController_01_entry"
    esp32_thing_name         VARCHAR(100) NOT NULL,                  -- Thing Name của ESP32 điều khiển rào chắn này
    barrier_type             VARCHAR(20)  NOT NULL,                  -- 'entry' hoặc 'exit'
    current_state            VARCHAR(50)           DEFAULT 'closed', -- 'opened_command', 'closed_command', 'opened_auto', 'closed_auto', 'error', 'unknown'
    last_state_update_source VARCHAR(50),
    last_command_sent        VARCHAR(50),                            -- Lệnh cuối cùng được gửi: 'open', 'close'
    last_command_timestamp   TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (lot_id, barrier_identifier)                              -- Đảm bảo mỗi rào chắn là duy nhất trong một bãi
    -- Cân nhắc UNIQUE (esp32_thing_name, barrier_type) nếu một ESP32 chỉ quản lý một rào vào và một rào ra.
);

-- Bảng lưu trữ thông tin các phiên đỗ xe
CREATE TABLE IF NOT EXISTS parking_sessions
(
    id                  SERIAL PRIMARY KEY,
    lot_id              INT          NOT NULL REFERENCES parking_lots (id) ON DELETE CASCADE,
    slot_id             INT          REFERENCES parking_slots (id) ON DELETE SET NULL, -- Chỗ đỗ cụ thể, có thể NULL
    esp32_thing_name    VARCHAR(100) NOT NULL,                                         -- ESP32 ghi nhận sự kiện
    vehicle_identifier  VARCHAR(100),                                                  -- ID xe (ví dụ: biển số xe)
    entry_time          TIMESTAMPTZ  NOT NULL,
    exit_time           TIMESTAMPTZ,
    duration_minutes    INT,
    calculated_fee      DECIMAL(10, 2),
    payment_status      VARCHAR(20)           DEFAULT 'pending',                       -- 'pending', 'paid', 'failed', 'waived'
    status              VARCHAR(20)  NOT NULL DEFAULT 'active',                        -- 'active', 'completed', 'cancelled'
    entry_gate_event_id VARCHAR(255),                                                  -- ID của sự kiện cảm biến cổng vào từ ESP32
    exit_gate_event_id  VARCHAR(255),                                                  -- ID của sự kiện cảm biến cổng ra từ ESP32
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Bảng lưu trữ lịch sử sự kiện từ thiết bị (rất hữu ích cho việc gỡ lỗi và phân tích)
CREATE TABLE IF NOT EXISTS device_events_log
(
    id               BIGSERIAL PRIMARY KEY,
    received_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Thời gian backend nhận được
    esp32_thing_name VARCHAR(100),
    mqtt_topic       TEXT,
    message_type     VARCHAR(100),                                   -- "startup", "barrier_state", "gate_event", "slot_status", "parking_summary", "system_status", "error", "command_ack"
    payload          JSONB,                                          -- Toàn bộ payload JSON từ ESP32 (bao gồm cả metadata từ IoT Rule)
    processed_status VARCHAR(20)          DEFAULT 'pending',         -- 'pending', 'processed', 'error'
    processing_notes TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP  -- Thêm created_at cho bảng log
);

-- Hàm trigger để tự động cập nhật updated_at
CREATE OR REPLACE FUNCTION set_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Áp dụng trigger cho các bảng
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE set_updated_at_column();

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE
    ON devices
    FOR EACH ROW
EXECUTE PROCEDURE set_updated_at_column();

CREATE TRIGGER update_parking_lots_updated_at
    BEFORE UPDATE
    ON parking_lots
    FOR EACH ROW
EXECUTE PROCEDURE set_updated_at_column();

CREATE TRIGGER update_parking_slots_updated_at
    BEFORE UPDATE
    ON parking_slots
    FOR EACH ROW
EXECUTE PROCEDURE set_updated_at_column();

CREATE TRIGGER update_barriers_updated_at
    BEFORE UPDATE
    ON barriers
    FOR EACH ROW
EXECUTE PROCEDURE set_updated_at_column();

CREATE TRIGGER update_parking_sessions_updated_at
    BEFORE UPDATE
    ON parking_sessions
    FOR EACH ROW
EXECUTE PROCEDURE set_updated_at_column();

-- Không cần trigger updated_at cho device_events_log vì nó thường chỉ được insert.

-- Thêm một số index để tăng tốc độ truy vấn
CREATE INDEX IF NOT EXISTS idx_parking_slots_lot_id ON parking_slots (lot_id);
CREATE INDEX IF NOT EXISTS idx_parking_slots_esp32_thing_name_slot_identifier ON parking_slots (esp32_thing_name, slot_identifier);
CREATE INDEX IF NOT EXISTS idx_barriers_lot_id ON barriers (lot_id);
CREATE INDEX IF NOT EXISTS idx_barriers_esp32_thing_name ON barriers (esp32_thing_name);
CREATE INDEX IF NOT EXISTS idx_parking_sessions_lot_id_status ON parking_sessions (lot_id, status);
CREATE INDEX IF NOT EXISTS idx_parking_sessions_esp32_thing_name_status ON parking_sessions (esp32_thing_name, status);
CREATE INDEX IF NOT EXISTS idx_parking_sessions_vehicle_id_status ON parking_sessions (vehicle_identifier, status);
CREATE INDEX IF NOT EXISTS idx_device_events_log_time_type_thing ON device_events_log (received_at DESC, message_type, esp32_thing_name);

