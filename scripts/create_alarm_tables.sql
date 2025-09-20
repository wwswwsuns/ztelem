-- 创建告警上报表
CREATE TABLE IF NOT EXISTS telemetry.alarm_report (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id VARCHAR(255) NOT NULL,
    flow_id BIGINT NOT NULL,
    code BIGINT NOT NULL,
    occurrence_time BIGINT,
    update_time BIGINT,
    disappeared_time BIGINT,
    occurrence_ms INTEGER,
    update_ms INTEGER,
    disappeared_ms INTEGER,
    alarm_class VARCHAR(255),
    alarm_type VARCHAR(255),
    alarm_status VARCHAR(100),
    sort INTEGER,
    severity VARCHAR(100),
    tpid_type INTEGER,
    tpid_length INTEGER,
    tpid TEXT,
    protect_group_work_status INTEGER,
    protect_type INTEGER,
    reason INTEGER,
    return_mode VARCHAR(255),
    protect_tpid_type INTEGER,
    protect_tpid_length INTEGER,
    protect_tpid TEXT,
    source_tpid_type INTEGER,
    source_tpid_length INTEGER,
    source_tpid TEXT,
    switch_tpid_type INTEGER,
    previous_tpid_length INTEGER,
    current_tpid_length INTEGER,
    previous_tpid TEXT,
    current_tpid TEXT,
    perf_alarm_period VARCHAR(255),
    perf_alarm_type VARCHAR(255),
    perf_alarm_value TEXT,
    description TEXT,
    caption TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 创建通知上报表
CREATE TABLE IF NOT EXISTS telemetry.notification_report (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id VARCHAR(255) NOT NULL,
    flow_id BIGINT NOT NULL,
    code BIGINT NOT NULL,
    occur_time BIGINT,
    occur_ms INTEGER,
    classification VARCHAR(255),
    sort INTEGER,
    severity VARCHAR(100),
    tpid_type INTEGER,
    tpid_length INTEGER,
    tpid TEXT,
    description TEXT,
    caption TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_alarm_report_timestamp ON telemetry.alarm_report (timestamp);
CREATE INDEX IF NOT EXISTS idx_alarm_report_system_id ON telemetry.alarm_report (system_id);
CREATE INDEX IF NOT EXISTS idx_alarm_report_flow_id ON telemetry.alarm_report (flow_id);
CREATE INDEX IF NOT EXISTS idx_alarm_report_code ON telemetry.alarm_report (code);
CREATE INDEX IF NOT EXISTS idx_alarm_report_severity ON telemetry.alarm_report (severity);

CREATE INDEX IF NOT EXISTS idx_notification_report_timestamp ON telemetry.notification_report (timestamp);
CREATE INDEX IF NOT EXISTS idx_notification_report_system_id ON telemetry.notification_report (system_id);
CREATE INDEX IF NOT EXISTS idx_notification_report_flow_id ON telemetry.notification_report (flow_id);
CREATE INDEX IF NOT EXISTS idx_notification_report_code ON telemetry.notification_report (code);
CREATE INDEX IF NOT EXISTS idx_notification_report_severity ON telemetry.notification_report (severity);

-- 如果使用TimescaleDB，创建超表
-- SELECT create_hypertable('telemetry.alarm_report', 'timestamp', if_not_exists => TRUE);
-- SELECT create_hypertable('telemetry.notification_report', 'timestamp', if_not_exists => TRUE);

-- 添加注释
COMMENT ON TABLE telemetry.alarm_report IS '告警上报数据表';
COMMENT ON TABLE telemetry.notification_report IS '通知上报数据表';

COMMENT ON COLUMN telemetry.alarm_report.timestamp IS '告警时间戳';
COMMENT ON COLUMN telemetry.alarm_report.system_id IS '系统标识';
COMMENT ON COLUMN telemetry.alarm_report.flow_id IS '流水号';
COMMENT ON COLUMN telemetry.alarm_report.alarm_id IS '告警ID';
COMMENT ON COLUMN telemetry.alarm_report.alarm_type IS '告警类型';
COMMENT ON COLUMN telemetry.alarm_report.severity IS '告警严重程度';
COMMENT ON COLUMN telemetry.alarm_report.description IS '告警描述';

COMMENT ON COLUMN telemetry.notification_report.timestamp IS '通知时间戳';
COMMENT ON COLUMN telemetry.notification_report.system_id IS '系统标识';
COMMENT ON COLUMN telemetry.notification_report.flow_id IS '流水号';
COMMENT ON COLUMN telemetry.notification_report.notification_id IS '通知ID';
COMMENT ON COLUMN telemetry.notification_report.notification_type IS '通知类型';
COMMENT ON COLUMN telemetry.notification_report.severity IS '通知严重程度';
COMMENT ON COLUMN telemetry.notification_report.description IS '通知描述';