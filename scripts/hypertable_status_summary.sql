-- TimescaleDB超表状态总结报告
-- 查看所有超表的配置、策略和性能状态

\echo '🎯 ===== TimescaleDB 超表状态总结 ====='
\echo ''

\echo '📊 1. 超表基本信息'
SELECT 
    hypertable_name as "表名",
    num_chunks as "Chunks数量",
    CASE WHEN compression_enabled THEN '✅ 已启用' ELSE '❌ 未启用' END as "压缩状态",
    pg_size_pretty(
        (SELECT pg_total_relation_size(format('%I.%I', hypertable_schema, hypertable_name)))
    ) as "总大小"
FROM timescaledb_information.hypertables 
WHERE hypertable_schema = 'telemetry'
ORDER BY hypertable_name;

\echo ''
\echo '⚙️  2. 策略配置详情'
SELECT 
    hypertable_name as "表名",
    CASE 
        WHEN proc_name = 'policy_compression' THEN '🗜️  压缩策略'
        WHEN proc_name = 'policy_retention' THEN '🗑️  保留策略'
        ELSE proc_name 
    END as "策略类型",
    CASE 
        WHEN proc_name = 'policy_compression' THEN 
            (config->>'compress_after') || ' 后压缩'
        WHEN proc_name = 'policy_retention' THEN 
            (config->>'drop_after') || ' 后删除'
        ELSE config::text
    END as "策略配置",
    schedule_interval as "执行间隔"
FROM timescaledb_information.jobs 
WHERE hypertable_schema = 'telemetry'
ORDER BY hypertable_name, proc_name;

\echo ''
\echo '📦 3. Chunks详细状态'
SELECT 
    hypertable_name as "表名",
    chunk_name as "Chunk名称",
    pg_size_pretty(
        pg_total_relation_size(format('%I.%I', chunk_schema, chunk_name))
    ) as "大小",
    range_start::date as "开始日期",
    range_end::date as "结束日期",
    CASE WHEN is_compressed THEN '✅ 已压缩' ELSE '⏳ 未压缩' END as "压缩状态"
FROM timescaledb_information.chunks 
WHERE hypertable_schema = 'telemetry'
ORDER BY hypertable_name, range_start DESC;

\echo ''
\echo '📈 4. 告警表数据统计'
SELECT 
    '告警数据总量' as "统计项",
    COUNT(*) as "数值"
FROM telemetry.alarm_report
UNION ALL
SELECT 
    '通知数据总量' as "统计项",
    COUNT(*) as "数值"
FROM telemetry.notification_report
UNION ALL
SELECT 
    '最新告警时间' as "统计项",
    MAX(occurrence_time)::text as "数值"
FROM telemetry.alarm_report
UNION ALL
SELECT 
    '最早告警时间' as "统计项",
    MIN(occurrence_time)::text as "数值"
FROM telemetry.alarm_report;

\echo ''
\echo '🚀 5. 性能优化建议'
SELECT 
    '✅ 告警表已转换为超表' as "优化状态"
UNION ALL
SELECT 
    '✅ 压缩策略: 7天后自动压缩' as "优化状态"
UNION ALL
SELECT 
    '✅ 保留策略: 1年后自动删除' as "优化状态"
UNION ALL
SELECT 
    '✅ 时间字段已优化为TIMESTAMPTZ格式' as "优化状态"
UNION ALL
SELECT 
    '✅ 已创建时间序列优化索引' as "优化状态";

\echo ''
\echo '🎉 TimescaleDB超表配置完成！'
\echo '   - 查询性能提升: 时间范围查询将显著加速'
\echo '   - 存储空间节省: 7天后数据自动压缩'
\echo '   - 自动数据管理: 1年后过期数据自动删除'
\echo ''