-- 迁移告警表时间字段从BIGINT到TIMESTAMPTZ
-- 执行前请备份数据！

-- 开始事务
BEGIN;

-- 1. 为告警上报表添加新的时间字段
ALTER TABLE telemetry.alarm_report 
ADD COLUMN occurrence_time_new TIMESTAMPTZ,
ADD COLUMN update_time_new TIMESTAMPTZ,
ADD COLUMN disappeared_time_new TIMESTAMPTZ;

-- 2. 将现有的BIGINT时间戳转换为TIMESTAMPTZ（假设原始数据是Unix时间戳）
UPDATE telemetry.alarm_report 
SET 
    occurrence_time_new = CASE 
        WHEN occurrence_time IS NOT NULL AND occurrence_time > 0 
        THEN to_timestamp(occurrence_time) 
        ELSE NULL 
    END,
    update_time_new = CASE 
        WHEN update_time IS NOT NULL AND update_time > 0 
        THEN to_timestamp(update_time) 
        ELSE NULL 
    END,
    disappeared_time_new = CASE 
        WHEN disappeared_time IS NOT NULL AND disappeared_time > 0 
        THEN to_timestamp(disappeared_time) 
        ELSE NULL 
    END;

-- 3. 删除旧字段
ALTER TABLE telemetry.alarm_report 
DROP COLUMN occurrence_time,
DROP COLUMN update_time,
DROP COLUMN disappeared_time;

-- 4. 重命名新字段
ALTER TABLE telemetry.alarm_report 
RENAME COLUMN occurrence_time_new TO occurrence_time;
ALTER TABLE telemetry.alarm_report 
RENAME COLUMN update_time_new TO update_time;
ALTER TABLE telemetry.alarm_report 
RENAME COLUMN disappeared_time_new TO disappeared_time;

-- 5. 为通知上报表添加新的时间字段
ALTER TABLE telemetry.notification_report 
ADD COLUMN occur_time_new TIMESTAMPTZ;

-- 6. 将现有的BIGINT时间戳转换为TIMESTAMPTZ
UPDATE telemetry.notification_report 
SET occur_time_new = CASE 
    WHEN occur_time IS NOT NULL AND occur_time > 0 
    THEN to_timestamp(occur_time) 
    ELSE NULL 
END;

-- 7. 删除旧字段并重命名新字段
ALTER TABLE telemetry.notification_report 
DROP COLUMN occur_time;
ALTER TABLE telemetry.notification_report 
RENAME COLUMN occur_time_new TO occur_time;

-- 8. 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_alarm_report_occurrence_time ON telemetry.alarm_report (occurrence_time);
CREATE INDEX IF NOT EXISTS idx_alarm_report_update_time ON telemetry.alarm_report (update_time);
CREATE INDEX IF NOT EXISTS idx_notification_report_occur_time ON telemetry.notification_report (occur_time);

-- 9. 更新表注释
COMMENT ON COLUMN telemetry.alarm_report.occurrence_time IS '告警产生时间（UTC时间）';
COMMENT ON COLUMN telemetry.alarm_report.update_time IS '告警更新时间（UTC时间）';
COMMENT ON COLUMN telemetry.alarm_report.disappeared_time IS '告警消失时间（UTC时间）';
COMMENT ON COLUMN telemetry.notification_report.occur_time IS '通知产生时间（UTC时间）';

-- 提交事务
COMMIT;

-- 验证迁移结果
SELECT 
    'alarm_report' as table_name,
    COUNT(*) as total_records,
    COUNT(occurrence_time) as occurrence_time_count,
    COUNT(update_time) as update_time_count,
    COUNT(disappeared_time) as disappeared_time_count
FROM telemetry.alarm_report
UNION ALL
SELECT 
    'notification_report' as table_name,
    COUNT(*) as total_records,
    COUNT(occur_time) as occur_time_count,
    0 as update_time_count,
    0 as disappeared_time_count
FROM telemetry.notification_report;

-- 显示字段类型
SELECT 
    table_name,
    column_name,
    data_type,
    is_nullable
FROM information_schema.columns 
WHERE table_schema = 'telemetry' 
    AND table_name IN ('alarm_report', 'notification_report')
    AND column_name IN ('occurrence_time', 'update_time', 'disappeared_time', 'occur_time')
ORDER BY table_name, column_name;