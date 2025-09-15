-- 初始化数据库脚本
-- 创建TimescaleDB扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 创建telemetry schema
CREATE SCHEMA IF NOT EXISTS telemetry;

-- 设置搜索路径
ALTER DATABASE telemetrydb SET search_path TO telemetry,public;

-- 切换到telemetry schema
SET search_path TO telemetry;

-- 创建平台指标表
CREATE TABLE IF NOT EXISTS platform_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    component_name TEXT,
    oper_status TEXT,
    admin_status TEXT,
    alarm_status TEXT,
    temperature NUMERIC,
    cpu_usage NUMERIC,
    memory_usage NUMERIC,
    power_consumption NUMERIC
);

-- 创建接口指标表
CREATE TABLE IF NOT EXISTS interface_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    admin_status TEXT,
    oper_status TEXT,
    in_octets BIGINT,
    out_octets BIGINT,
    in_pkts BIGINT,
    out_pkts BIGINT,
    in_errors BIGINT,
    out_errors BIGINT,
    in_discards BIGINT,
    out_discards BIGINT,
    speed BIGINT,
    mtu INTEGER,
    duplex TEXT,
    description TEXT
);

-- 创建子接口指标表
CREATE TABLE IF NOT EXISTS subinterface_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    subinterface_name TEXT NOT NULL,
    admin_status TEXT,
    oper_status TEXT,
    in_octets BIGINT,
    out_octets BIGINT,
    in_pkts BIGINT,
    out_pkts BIGINT,
    vlan_id INTEGER,
    description TEXT
);

-- 转换为TimescaleDB超表（如果不存在）
DO $$
BEGIN
    -- 检查是否已经是超表
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_name = 'platform_metrics' AND hypertable_schema = 'telemetry'
    ) THEN
        PERFORM create_hypertable('platform_metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_name = 'interface_metrics' AND hypertable_schema = 'telemetry'
    ) THEN
        PERFORM create_hypertable('interface_metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_name = 'subinterface_metrics' AND hypertable_schema = 'telemetry'
    ) THEN
        PERFORM create_hypertable('subinterface_metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');
    END IF;
END
$$;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_platform_system_time ON platform_metrics(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_platform_timestamp ON platform_metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_platform_component ON platform_metrics(component_name);

CREATE INDEX IF NOT EXISTS idx_interface_system_time ON interface_metrics(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_interface_timestamp ON interface_metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_interface_name ON interface_metrics(interface_name);

CREATE INDEX IF NOT EXISTS idx_subinterface_system_time ON subinterface_metrics(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_subinterface_timestamp ON subinterface_metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_subinterface_name ON subinterface_metrics(subinterface_name);

-- 设置数据保留策略（保留90天）
DO $$
BEGIN
    -- 检查是否已经设置了保留策略
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.jobs 
        WHERE proc_name = 'policy_retention' 
        AND hypertable_name = 'platform_metrics'
    ) THEN
        PERFORM add_retention_policy('platform_metrics', INTERVAL '90 days');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.jobs 
        WHERE proc_name = 'policy_retention' 
        AND hypertable_name = 'interface_metrics'
    ) THEN
        PERFORM add_retention_policy('interface_metrics', INTERVAL '90 days');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.jobs 
        WHERE proc_name = 'policy_retention' 
        AND hypertable_name = 'subinterface_metrics'
    ) THEN
        PERFORM add_retention_policy('subinterface_metrics', INTERVAL '90 days');
    END IF;
END
$$;

-- 启用压缩（7天后压缩数据）
DO $$
BEGIN
    -- 设置压缩策略
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.compression_settings 
        WHERE hypertable_name = 'platform_metrics'
    ) THEN
        ALTER TABLE platform_metrics SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'system_id'
        );
        PERFORM add_compression_policy('platform_metrics', INTERVAL '7 days');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.compression_settings 
        WHERE hypertable_name = 'interface_metrics'
    ) THEN
        ALTER TABLE interface_metrics SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'system_id'
        );
        PERFORM add_compression_policy('interface_metrics', INTERVAL '7 days');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.compression_settings 
        WHERE hypertable_name = 'subinterface_metrics'
    ) THEN
        ALTER TABLE subinterface_metrics SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'system_id'
        );
        PERFORM add_compression_policy('subinterface_metrics', INTERVAL '7 days');
    END IF;
END
$$;

-- 创建一些有用的视图
CREATE OR REPLACE VIEW latest_platform_status AS
SELECT DISTINCT ON (system_id, component_name)
    system_id,
    component_name,
    oper_status,
    admin_status,
    alarm_status,
    temperature,
    cpu_usage,
    memory_usage,
    power_consumption,
    timestamp
FROM platform_metrics
ORDER BY system_id, component_name, timestamp DESC;

CREATE OR REPLACE VIEW latest_interface_status AS
SELECT DISTINCT ON (system_id, interface_name)
    system_id,
    interface_name,
    admin_status,
    oper_status,
    speed,
    mtu,
    duplex,
    description,
    timestamp
FROM interface_metrics
ORDER BY system_id, interface_name, timestamp DESC;

-- 创建统计函数
CREATE OR REPLACE FUNCTION get_data_summary(start_time TIMESTAMPTZ DEFAULT NOW() - INTERVAL '24 hours')
RETURNS TABLE(
    table_name TEXT,
    record_count BIGINT,
    latest_timestamp TIMESTAMPTZ,
    unique_systems BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'platform_metrics'::TEXT,
        COUNT(*)::BIGINT,
        MAX(timestamp),
        COUNT(DISTINCT system_id)::BIGINT
    FROM platform_metrics 
    WHERE timestamp >= start_time
    
    UNION ALL
    
    SELECT 
        'interface_metrics'::TEXT,
        COUNT(*)::BIGINT,
        MAX(timestamp),
        COUNT(DISTINCT system_id)::BIGINT
    FROM interface_metrics 
    WHERE timestamp >= start_time
    
    UNION ALL
    
    SELECT 
        'subinterface_metrics'::TEXT,
        COUNT(*)::BIGINT,
        MAX(timestamp),
        COUNT(DISTINCT system_id)::BIGINT
    FROM subinterface_metrics 
    WHERE timestamp >= start_time;
END;
$$ LANGUAGE plpgsql;

-- 授权给telemetry_app用户
GRANT ALL PRIVILEGES ON SCHEMA telemetry TO telemetry_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA telemetry TO telemetry_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA telemetry TO telemetry_app;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA telemetry TO telemetry_app;

-- 设置默认权限
ALTER DEFAULT PRIVILEGES IN SCHEMA telemetry GRANT ALL ON TABLES TO telemetry_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA telemetry GRANT ALL ON SEQUENCES TO telemetry_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA telemetry GRANT ALL ON FUNCTIONS TO telemetry_app;

-- 输出初始化完成信息
DO $$
BEGIN
    RAISE NOTICE 'Database initialization completed successfully!';
    RAISE NOTICE 'Created schema: telemetry';
    RAISE NOTICE 'Created tables: platform_metrics, interface_metrics, subinterface_metrics';
    RAISE NOTICE 'Configured TimescaleDB hypertables with 1-day chunks';
    RAISE NOTICE 'Set retention policy: 90 days';
    RAISE NOTICE 'Set compression policy: 7 days';
    RAISE NOTICE 'Created indexes for optimal query performance';
    RAISE NOTICE 'Created views: latest_platform_status, latest_interface_status';
    RAISE NOTICE 'Created function: get_data_summary()';
END
$$;