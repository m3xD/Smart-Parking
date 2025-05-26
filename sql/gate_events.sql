-- File: sql/migrations/001_add_gate_events_table.sql
-- Migration để thêm gate_events table

-- Bảng lưu trữ gate events để track processing workflow
CREATE TABLE IF NOT EXISTS gate_events (
    id                SERIAL PRIMARY KEY,
    event_id          VARCHAR(255) NOT NULL UNIQUE,           -- UUID from ESP32
    lot_id            INT          NOT NULL REFERENCES parking_lots(id) ON DELETE CASCADE,
    device_id         VARCHAR(100) NOT NULL,                  -- ESP32 Thing Name
    gate_direction    VARCHAR(10)  NOT NULL CHECK (gate_direction IN ('entry', 'exit')),
    event_type        VARCHAR(50)  NOT NULL,                  -- 'vehicle_approaching', 'vehicle_at_gate', etc.
    status            VARCHAR(50)  NOT NULL DEFAULT 'pending', -- 'pending', 'awaiting_lpr', 'lpr_completed', 'session_created', 'timeout', 'error'
    sensor_id         VARCHAR(100),

    -- LPR Results
    detected_plate    VARCHAR(20),
    lpr_confidence    DECIMAL(5,4),                           -- 0.0000 to 1.0000
    is_manual_entry   BOOLEAN     DEFAULT FALSE,

    -- Session Linkage
    session_id        INT         REFERENCES parking_sessions(id) ON DELETE SET NULL,

    -- Processing Metadata
    processing_notes  TEXT,
    assigned_operator VARCHAR(100),                           -- Nếu cần manual intervention

    -- Timestamps
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at        TIMESTAMPTZ,                            -- Timeout threshold
    completed_at      TIMESTAMPTZ                             -- Khi hoàn thành xử lý
);

-- Indexes cho performance
CREATE INDEX IF NOT EXISTS idx_gate_events_status_created ON gate_events(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gate_events_lot_id_status ON gate_events(lot_id, status);
CREATE INDEX IF NOT EXISTS idx_gate_events_device_direction ON gate_events(device_id, gate_direction);
CREATE INDEX IF NOT EXISTS idx_gate_events_expires_at ON gate_events(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_gate_events_event_id ON gate_events(event_id);
CREATE INDEX IF NOT EXISTS idx_gate_events_session_id ON gate_events(session_id) WHERE session_id IS NOT NULL;

-- Trigger để auto-update updated_at
CREATE TRIGGER update_gate_events_updated_at
    BEFORE UPDATE ON gate_events
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at_column();

-- Function để cleanup expired events
CREATE OR REPLACE FUNCTION cleanup_expired_gate_events()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    UPDATE gate_events
    SET status = 'timeout',
        updated_at = CURRENT_TIMESTAMP,
        processing_notes = COALESCE(processing_notes || '; ', '') || 'Auto-expired due to timeout'
    WHERE status IN ('pending', 'awaiting_lpr')
      AND expires_at < CURRENT_TIMESTAMP;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    -- Log the cleanup activity
    INSERT INTO device_events_log (
        received_at,
        message_type,
        payload,
        processed_status,
        processing_notes
    ) VALUES (
        CURRENT_TIMESTAMP,
        'gate_event_cleanup',
        ('{"expired_events_count": ' || deleted_count || '}')::jsonb,
        'processed',
        'Automatic cleanup of expired gate events'
    );

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function để lấy thống kê gate events
CREATE OR REPLACE FUNCTION get_gate_event_stats(from_time TIMESTAMPTZ, to_time TIMESTAMPTZ)
RETURNS TABLE (
    total_events INTEGER,
    completed_events INTEGER,
    timeout_events INTEGER,
    error_events INTEGER,
    avg_processing_time_minutes NUMERIC,
    lpr_success_rate NUMERIC,
    auto_session_rate NUMERIC,
    manual_intervention_rate NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH stats AS (
        SELECT
            COUNT(*) as total,
            COUNT(CASE WHEN status = 'session_created' THEN 1 END) as completed,
            COUNT(CASE WHEN status = 'timeout' THEN 1 END) as timeout,
            COUNT(CASE WHEN status = 'error' THEN 1 END) as error,
            AVG(EXTRACT(EPOCH FROM (completed_at - created_at))/60) as avg_processing_min,
            COUNT(CASE WHEN detected_plate IS NOT NULL AND detected_plate != '' THEN 1 END) as lpr_success,
            COUNT(CASE WHEN session_id IS NOT NULL AND is_manual_entry = FALSE THEN 1 END) as auto_sessions,
            COUNT(CASE WHEN is_manual_entry = TRUE THEN 1 END) as manual_entries
        FROM gate_events
        WHERE created_at BETWEEN from_time AND to_time
    )
    SELECT
        total::INTEGER,
        completed::INTEGER,
        timeout::INTEGER,
        error::INTEGER,
        COALESCE(avg_processing_min, 0)::NUMERIC,
        CASE WHEN total > 0 THEN (lpr_success::NUMERIC / total::NUMERIC * 100) ELSE 0 END,
        CASE WHEN total > 0 THEN (auto_sessions::NUMERIC / total::NUMERIC * 100) ELSE 0 END,
        CASE WHEN total > 0 THEN (manual_entries::NUMERIC / total::NUMERIC * 100) ELSE 0 END
    FROM stats;
END;
$$ LANGUAGE plpgsql;

-- Insert sample data cho testing (optional - có thể xóa trong production)
/*
INSERT INTO parking_lots (name, address, total_slots) VALUES
('Test Parking Lot', 'Test Address', 10)
ON CONFLICT (name) DO NOTHING;

INSERT INTO devices (thing_name, lot_id, status) VALUES
('ESP32_TestDevice_01', 1, 'online')
ON CONFLICT (thing_name) DO NOTHING;
*/

-- Verify migration
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'gate_events') THEN
        RAISE NOTICE 'Gate events table created successfully';
    ELSE
        RAISE EXCEPTION 'Failed to create gate_events table';
    END IF;
END
$$;