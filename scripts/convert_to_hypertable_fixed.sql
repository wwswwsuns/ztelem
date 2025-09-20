-- 将告警数据表转换为TimescaleDB超表并配置压缩和保留策略（修复版本）
-- 执行前请确保已备份数据！

-- 1. 检查当前表状态
SELECT 
    schemaname, 
    tablename, 
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as table_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as data_size
FROM pg_tables 
WHERE schemaname = 'telemetry' AND tablename IN ('alarm_report', 'notification_report');

-- 2. 检查是否已经是超表
SELECT 
    hypertable_schema,
    hypertable_name,
    num_dimensions,
    num_chunks,
    compression_enabled
FROM timescaledb_information.hypertables 
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name IN ('alarm_report', 'notification_report');

-- 3. 转换告警表为超表（如果还不是）
DO $$
BEGIN
    -- 检查告警表是否已经是超表
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_schema = 'telemetry' AND hypertable_name = 'alarm_report'
    ) THEN
        -- 转换为超表，使用timestamp字段作为时间维度，chunk间隔为1天
        PERFORM create_hypertable(
            'telemetry.alarm_report', 
            'timestamp',
            chunk_time_interval => INTERVAL '1 day',
            if_not_exists => TRUE
        );
        RAISE NOTICE '告警表已转换为超表';
    ELSE
        RAISE NOTICE '告警表已经是超表';
    END IF;
END $$;

-- 4. 转换通知表为超表（如果还不是）
DO $$
BEGIN
    -- 检查通知表是否已经是超表
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_schema = 'telemetry' AND hypertable_name = 'notification_report'
    ) THEN
        -- 转换为超表，使用timestamp字段作为时间维度，chunk间隔为1天
        PERFORM create_hypertable(
            'telemetry.notification_report', 
            'timestamp',
            chunk_time_interval => INTERVAL '1 day',
            if_not_exists => TRUE
        );
        RAISE NOTICE '通知表已转换为超表';
    ELSE
        RAISE NOTICE '通知表已经是超表';
    END IF;
END $$;

-- 5. 启用告警表压缩策略（7天后压缩）
SELECT add_compression_policy(
    'telemetry.alarm_report', 
    INTERVAL '7 days',
    if_not_exists => TRUE
);

-- 6. 启用通知表压缩策略（7天后压缩）
SELECT add_compression_policy(
    'telemetry.notification_report', 
    INTERVAL '7 days',
    if_not_exists => TRUE
);

-- 7. 启用告警表数据保留策略（1年后删除）
SELECT add_retention_policy(
    'telemetry.alarm_report', 
    INTERVAL '1 year',
    if_not_exists => TRUE
);

-- 8. 启用通知表数据保留策略（1年后删除）
SELECT add_retention_policy(
    'telemetry.notification_report', 
    INTERVAL '1 year',
    if_not_exists => TRUE
);

-- 9. 手动压缩现有的旧数据（7天前的数据）
SELECT compress_chunk(chunk_schema, chunk_name)
FROM timescaledb_information.chunks
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name = 'alarm_report'
  AND range_end < NOW() - INTERVAL '7 days'
  AND NOT is_compressed;

SELECT compress_chunk(chunk_schema, chunk_name)
FROM timescaledb_information.chunks
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name = 'notification_report'
  AND range_end < NOW() - INTERVAL '7 days'
  AND NOT is_compressed;

-- 10. 创建优化的索引
-- 告警表索引
CREATE INDEX IF NOT EXISTS idx_alarm_report_time_system_id 
ON telemetry.alarm_report (timestamp DESC, system_id);

CREATE INDEX IF NOT EXISTS idx_alarm_report_occurrence_time 
ON telemetry.alarm_report (occurrence_time DESC) 
WHERE occurrence_time IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_alarm_report_severity_time 
ON telemetry.alarm_report (severity, timestamp DESC) 
WHERE severity IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_alarm_report_flow_id_time 
ON telemetry.alarm_report (flow_id, timestamp DESC);

-- 通知表索引
CREATE INDEX IF NOT EXISTS idx_notification_report_time_system_id 
ON telemetry.notification_report (timestamp DESC, system_id);

CREATE INDEX IF NOT EXISTS idx_notification_report_occur_time 
ON telemetry.notification_report (occur_time DESC) 
WHERE occur_time IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notification_report_severity_time 
ON telemetry.notification_report (severity, timestamp DESC) 
WHERE severity IS NOT NULL;

-- 11. 查看转换结果
SELECT 
    hypertable_schema,
    hypertable_name,
    num_dimensions,
    num_chunks,
    compression_enabled
FROM timescaledb_information.hypertables 
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name IN ('alarm_report', 'notification_report');

-- 12. 查看压缩策略
SELECT 
    j.hypertable_schema,
    j.hypertable_name,
    j.job_id,
    j.schedule_interval,
    j.config
FROM timescaledb_information.jobs j
WHERE j.proc_name = 'policy_compression'
  AND j.hypertable_schema = 'telemetry'
  AND j.hypertable_name IN ('alarm_report', 'notification_report');

-- 13. 查看保留策略
SELECT 
    j.hypertable_schema,
    j.hypertable_name,
    j.job_id,
    j.schedule_interval,
    j.config
FROM timescaledb_information.jobs j
WHERE j.proc_name = 'policy_retention'
  AND j.hypertable_schema = 'telemetry'
  AND j.hypertable_name IN ('alarm_report', 'notification_report');

-- 14. 查看chunks状态
SELECT 
    hypertable_name,
    chunk_name,
    range_start,
    range_end,
    is_compressed,
    pg_size_pretty(total_bytes) as chunk_size
FROM timescaledb_information.chunks c
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name IN ('alarm_report', 'notification_report')
ORDER BY hypertable_name, range_start DESC;

-- 15. 显示优化建议
SELECT 
    'TimescaleDB超表转换完成！' as status,
    '压缩策略: 7天后自动压缩' as compression_policy,
    '保留策略: 1年后自动删除' as retention_policy,
    '建议: 定期监控chunk大小和压缩率' as recommendation;