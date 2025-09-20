-- 分步骤将告警数据表转换为TimescaleDB超表
-- 处理现有数据和主键约束问题

-- 第一步：检查当前状态
\echo '=== 第一步：检查当前表状态 ==='
SELECT 
    schemaname, 
    tablename, 
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as table_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as data_size
FROM pg_tables 
WHERE schemaname = 'telemetry' AND tablename IN ('alarm_report', 'notification_report');

-- 第二步：检查主键约束
\echo '=== 第二步：检查主键约束 ==='
SELECT 
    tc.table_name,
    tc.constraint_name,
    tc.constraint_type,
    array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as key_columns
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu 
    ON tc.constraint_name = kcu.constraint_name 
    AND tc.table_schema = kcu.table_schema
WHERE tc.table_schema = 'telemetry' 
    AND tc.table_name IN ('alarm_report', 'notification_report')
    AND tc.constraint_type = 'PRIMARY KEY'
GROUP BY tc.table_name, tc.constraint_name, tc.constraint_type;

-- 第三步：修改通知表主键约束（添加timestamp字段）
\echo '=== 第三步：修改通知表主键约束 ==='
DO $$
BEGIN
    -- 检查通知表是否有主键约束
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_schema = 'telemetry' 
        AND table_name = 'notification_report' 
        AND constraint_type = 'PRIMARY KEY'
    ) THEN
        -- 删除现有主键约束
        ALTER TABLE telemetry.notification_report DROP CONSTRAINT notification_report_pkey;
        RAISE NOTICE '已删除通知表原有主键约束';
        
        -- 创建包含timestamp的复合主键
        ALTER TABLE telemetry.notification_report 
        ADD CONSTRAINT notification_report_pkey 
        PRIMARY KEY (id, timestamp);
        RAISE NOTICE '已创建包含timestamp的新主键约束';
    ELSE
        RAISE NOTICE '通知表没有主键约束';
    END IF;
END $$;

-- 第四步：转换告警表为超表（使用migrate_data参数）
\echo '=== 第四步：转换告警表为超表 ==='
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_schema = 'telemetry' AND hypertable_name = 'alarm_report'
    ) THEN
        PERFORM create_hypertable(
            'telemetry.alarm_report', 
            'timestamp',
            chunk_time_interval => INTERVAL '1 day',
            migrate_data => TRUE,
            if_not_exists => TRUE
        );
        RAISE NOTICE '告警表已转换为超表（包含数据迁移）';
    ELSE
        RAISE NOTICE '告警表已经是超表';
    END IF;
END $$;

-- 第五步：转换通知表为超表
\echo '=== 第五步：转换通知表为超表 ==='
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_schema = 'telemetry' AND hypertable_name = 'notification_report'
    ) THEN
        PERFORM create_hypertable(
            'telemetry.notification_report', 
            'timestamp',
            chunk_time_interval => INTERVAL '1 day',
            migrate_data => TRUE,
            if_not_exists => TRUE
        );
        RAISE NOTICE '通知表已转换为超表（包含数据迁移）';
    ELSE
        RAISE NOTICE '通知表已经是超表';
    END IF;
END $$;

-- 第六步：验证超表转换结果
\echo '=== 第六步：验证超表转换结果 ==='
SELECT 
    hypertable_schema,
    hypertable_name,
    num_dimensions,
    num_chunks,
    compression_enabled
FROM timescaledb_information.hypertables 
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name IN ('alarm_report', 'notification_report');

-- 第七步：添加压缩策略
\echo '=== 第七步：添加压缩策略 ==='
-- 告警表压缩策略
SELECT add_compression_policy(
    'telemetry.alarm_report', 
    INTERVAL '7 days',
    if_not_exists => TRUE
) as alarm_compression_job_id;

-- 通知表压缩策略
SELECT add_compression_policy(
    'telemetry.notification_report', 
    INTERVAL '7 days',
    if_not_exists => TRUE
) as notification_compression_job_id;

-- 第八步：添加保留策略
\echo '=== 第八步：添加保留策略 ==='
-- 告警表保留策略
SELECT add_retention_policy(
    'telemetry.alarm_report', 
    INTERVAL '1 year',
    if_not_exists => TRUE
) as alarm_retention_job_id;

-- 通知表保留策略
SELECT add_retention_policy(
    'telemetry.notification_report', 
    INTERVAL '1 year',
    if_not_exists => TRUE
) as notification_retention_job_id;

-- 第九步：查看策略配置
\echo '=== 第九步：查看策略配置 ==='
SELECT 
    j.hypertable_schema,
    j.hypertable_name,
    j.proc_name,
    j.job_id,
    j.schedule_interval,
    j.config
FROM timescaledb_information.jobs j
WHERE j.hypertable_schema = 'telemetry'
  AND j.hypertable_name IN ('alarm_report', 'notification_report')
  AND j.proc_name IN ('policy_compression', 'policy_retention')
ORDER BY j.hypertable_name, j.proc_name;

-- 第十步：查看chunks状态
\echo '=== 第十步：查看chunks状态 ==='
SELECT 
    hypertable_name,
    chunk_name,
    range_start,
    range_end,
    is_compressed,
    pg_size_pretty(chunk_bytes) as chunk_size
FROM timescaledb_information.chunks c
WHERE hypertable_schema = 'telemetry' 
  AND hypertable_name IN ('alarm_report', 'notification_report')
ORDER BY hypertable_name, range_start DESC;

-- 第十一步：创建优化索引
\echo '=== 第十一步：创建优化索引 ==='
-- 告警表索引
CREATE INDEX IF NOT EXISTS idx_alarm_report_time_system_id 
ON telemetry.alarm_report (timestamp DESC, system_id);

CREATE INDEX IF NOT EXISTS idx_alarm_report_severity_time 
ON telemetry.alarm_report (severity, timestamp DESC) 
WHERE severity IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_alarm_report_flow_id_time 
ON telemetry.alarm_report (flow_id, timestamp DESC);

-- 通知表索引
CREATE INDEX IF NOT EXISTS idx_notification_report_time_system_id 
ON telemetry.notification_report (timestamp DESC, system_id);

CREATE INDEX IF NOT EXISTS idx_notification_report_severity_time 
ON telemetry.notification_report (severity, timestamp DESC) 
WHERE severity IS NOT NULL;

-- 第十二步：显示最终状态
\echo '=== 第十二步：最终状态总结 ==='
SELECT 
    'TimescaleDB超表转换完成！' as status,
    '压缩策略: 7天后自动压缩' as compression_policy,
    '保留策略: 1年后自动删除' as retention_policy,
    '优化: 已创建时间序列优化索引' as optimization;

-- 显示表大小对比
SELECT 
    h.hypertable_name,
    h.num_chunks,
    pg_size_pretty(
        (SELECT SUM(pg_total_relation_size(format('%I.%I', chunk_schema, chunk_name)))
         FROM timescaledb_information.chunks 
         WHERE hypertable_schema = h.hypertable_schema 
         AND hypertable_name = h.hypertable_name)
    ) as total_size
FROM timescaledb_information.hypertables h
WHERE h.hypertable_schema = 'telemetry' 
  AND h.hypertable_name IN ('alarm_report', 'notification_report');