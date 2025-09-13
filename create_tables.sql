-- Telemetry数据库表结构创建脚本

-- 切换到telemetry schema
SET search_path TO telemetry;

-- 创建平台指标表
CREATE TABLE IF NOT EXISTS platform_metrics (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    system_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    
    -- 组件通用数据字段
    oper_status TEXT,
    uptime TEXT,
    used_power INTEGER,
    allocated_power INTEGER,
    current_voltage TEXT,
    current_current TEXT,
    total_capacity TEXT,
    used_capacity TEXT,
    type TEXT,
    redundancy_type TEXT,
    modules TEXT,
    total_input_power TEXT,
    
    -- 风扇数据字段
    fan_speed INTEGER,
    fan_state TEXT,
    fan_phy_status TEXT,
    fan_work_mode TEXT,
    fan_current_power TEXT,
    fan_current_voltage TEXT,
    fan_current_current TEXT,
    fan_speed_percent TEXT,
    
    -- 内存数据字段
    mem_available BIGINT,
    mem_utilized BIGINT,
    mem_free BIGINT,
    mem_usage NUMERIC(5,2),
    mem_alarm_status TEXT,
    
    -- 存储数据字段
    storage_availability NUMERIC(5,2),
    
    -- 温度数据字段
    temp_instant DOUBLE PRECISION,
    temp_avg DOUBLE PRECISION,
    temp_min DOUBLE PRECISION,
    temp_max DOUBLE PRECISION,
    temp_interval BIGINT,
    temp_min_time TIMESTAMPTZ,
    temp_max_time TIMESTAMPTZ,
    alarm_status BOOLEAN,
    temp_alarm_threshold DOUBLE PRECISION,
    temp_alarm_severity TEXT,
    temp_minor_threshold DOUBLE PRECISION,
    temp_major_threshold DOUBLE PRECISION,
    temp_fatal_threshold DOUBLE PRECISION,
    temp_instant_string TEXT,
    temp_status TEXT,
    temp_description TEXT,
    
    -- 电源数据字段
    power_enable BOOLEAN,
    power_capacity DOUBLE PRECISION,
    power_input_current DOUBLE PRECISION,
    power_input_voltage DOUBLE PRECISION,
    power_output_current DOUBLE PRECISION,
    power_output_voltage DOUBLE PRECISION,
    power_output_power DOUBLE PRECISION,
    power_work_state TEXT,
    power_name TEXT,
    power_phy_state TEXT,
    power_state TEXT,
    power_com_state TEXT,
    power_temperature TEXT,
    power_available TEXT,
    power_capacity_string TEXT,
    power_input_power TEXT,
    power_input2_current DOUBLE PRECISION,
    power_input2_voltage DOUBLE PRECISION,
    power_output2_current DOUBLE PRECISION,
    power_output2_voltage DOUBLE PRECISION,
    
    -- 线卡数据字段
    linecard_power_admin_state TEXT,
    
    -- CPU数据字段
    cpu_instant NUMERIC(5,2),
    cpu_avg NUMERIC(5,2),
    cpu_min NUMERIC(5,2),
    cpu_max NUMERIC(5,2),
    cpu_interval BIGINT,
    cpu_min_time TIMESTAMPTZ,
    cpu_max_time TIMESTAMPTZ,
    cpu_alarm_status TEXT,
    
    -- 光模块数据字段
    optical_in_power DOUBLE PRECISION,
    optical_out_power DOUBLE PRECISION,
    optical_bias_current DOUBLE PRECISION,
    optical_temperature DOUBLE PRECISION,
    optical_voltage_vol33 DOUBLE PRECISION,
    optical_voltage_vol5 DOUBLE PRECISION,
    optical_alarm_los_status INTEGER,
    optical_alarm_los_info_event_id INTEGER,
    optical_alarm_los_info_event_interval INTEGER,
    optical_alarm_los_info_in_power DOUBLE PRECISION,
    optical_alarm_los_info_out_power DOUBLE PRECISION,
    optical_online_status TEXT,
    optical_rx_threshold_high_alarm DOUBLE PRECISION,
    optical_rx_threshold_pre_high_alarm DOUBLE PRECISION,
    optical_rx_threshold_low_alarm DOUBLE PRECISION,
    optical_rx_threshold_pre_low_alarm DOUBLE PRECISION
);

-- 创建接口指标表
CREATE TABLE IF NOT EXISTS interface_metrics (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    system_id TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    
    -- 接口状态字段
    ifindex INTEGER,
    admin_status TEXT,
    oper_status TEXT,
    last_change TIMESTAMPTZ,
    logical BOOLEAN,
    type INTEGER,
    phy_status TEXT,
    ipv4_oper_status TEXT,
    
    -- ZTE接口扩展字段
    zteif_type INTEGER,
    zteif_ifindex INTEGER,
    zteif_admin_status TEXT,
    zteif_oper_status TEXT,
    zteif_phy_status TEXT,
    zteif_ipv4_oper_status TEXT,
    zteif_ipv6_oper_status TEXT,
    
    -- 接口计数器字段
    in_octets BIGINT,
    in_unicast_pkts BIGINT,
    in_broadcast_pkts BIGINT,
    in_multicast_pkts BIGINT,
    in_discards BIGINT,
    in_errors BIGINT,
    in_unknown_protos BIGINT,
    in_fcs_errors BIGINT,
    out_octets BIGINT,
    out_unicast_pkts BIGINT,
    out_broadcast_pkts BIGINT,
    out_multicast_pkts BIGINT,
    out_discards BIGINT,
    out_errors BIGINT,
    carrier_transitions BIGINT,
    last_clear TIMESTAMPTZ,
    in_pkts BIGINT,
    out_pkts BIGINT,
    input_utilization NUMERIC(5,2),
    output_utilization NUMERIC(5,2),
    in_traffic_rate TEXT,
    in_packet_rate TEXT,
    out_traffic_rate TEXT,
    out_packet_rate TEXT,
    in_v4_octets BIGINT,
    out_v4_octets BIGINT,
    in_v4_pkts BIGINT,
    out_v4_pkts BIGINT,
    in_v6_octets BIGINT,
    out_v6_octets BIGINT,
    in_v6_pkts BIGINT,
    out_v6_pkts BIGINT,
    in_v4_traffic_rate TEXT,
    in_v4_packet_rate TEXT,
    out_v4_traffic_rate TEXT,
    out_v4_packet_rate TEXT,
    in_v6_traffic_rate TEXT,
    in_v6_packet_rate TEXT,
    out_v6_traffic_rate TEXT,
    out_v6_packet_rate TEXT,
    input_v4_utilization NUMERIC(5,2),
    output_v4_utilization NUMERIC(5,2),
    input_v6_utilization NUMERIC(5,2),
    output_v6_utilization NUMERIC(5,2),
    in_bier_octets BIGINT,
    in_bier_pkts BIGINT,
    out_bier_octets BIGINT,
    out_bier_pkts BIGINT
);

-- 创建子接口指标表
CREATE TABLE IF NOT EXISTS subinterface_metrics (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    system_id TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    subinterface_index TEXT NOT NULL,
    
    -- 子接口状态字段
    ifindex INTEGER,
    admin_status TEXT,
    oper_status TEXT,
    last_change TIMESTAMPTZ,
    logical BOOLEAN,
    ipv4_oper_status TEXT,
    
    -- ZTE子接口扩展字段
    zteif_ifindex INTEGER,
    zteif_admin_status TEXT,
    zteif_oper_status TEXT,
    zteif_phy_status TEXT,
    zteif_ipv4_oper_status TEXT,
    zteif_ipv6_oper_status TEXT,
    
    -- 子接口计数器字段（与接口计数器相同）
    in_octets BIGINT,
    in_unicast_pkts BIGINT,
    in_broadcast_pkts BIGINT,
    in_multicast_pkts BIGINT,
    in_discards BIGINT,
    in_errors BIGINT,
    in_unknown_protos BIGINT,
    in_fcs_errors BIGINT,
    out_octets BIGINT,
    out_unicast_pkts BIGINT,
    out_broadcast_pkts BIGINT,
    out_multicast_pkts BIGINT,
    out_discards BIGINT,
    out_errors BIGINT,
    carrier_transitions BIGINT,
    last_clear TIMESTAMPTZ,
    in_pkts BIGINT,
    out_pkts BIGINT,
    input_utilization NUMERIC(5,2),
    output_utilization NUMERIC(5,2),
    in_traffic_rate TEXT,
    in_packet_rate TEXT,
    out_traffic_rate TEXT,
    out_packet_rate TEXT,
    in_v4_octets BIGINT,
    out_v4_octets BIGINT,
    in_v4_pkts BIGINT,
    out_v4_pkts BIGINT,
    in_v6_octets BIGINT,
    out_v6_octets BIGINT,
    in_v6_pkts BIGINT,
    out_v6_pkts BIGINT,
    in_v4_traffic_rate TEXT,
    in_v4_packet_rate TEXT,
    out_v4_traffic_rate TEXT,
    out_v4_packet_rate TEXT,
    in_v6_traffic_rate TEXT,
    in_v6_packet_rate TEXT,
    out_v6_traffic_rate TEXT,
    out_v6_packet_rate TEXT,
    input_v4_utilization NUMERIC(5,2),
    output_v4_utilization NUMERIC(5,2),
    input_v6_utilization NUMERIC(5,2),
    output_v6_utilization NUMERIC(5,2),
    in_bier_octets BIGINT,
    in_bier_pkts BIGINT,
    out_bier_octets BIGINT,
    out_bier_pkts BIGINT
);

-- 创建时序表（TimescaleDB hypertables）
SELECT create_hypertable('platform_metrics', 'time', if_not_exists => TRUE);
SELECT create_hypertable('interface_metrics', 'time', if_not_exists => TRUE);
SELECT create_hypertable('subinterface_metrics', 'time', if_not_exists => TRUE);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_platform_metrics_system_component ON platform_metrics (system_id, component_name, time DESC);
CREATE INDEX IF NOT EXISTS idx_interface_metrics_system_interface ON interface_metrics (system_id, interface_name, time DESC);
CREATE INDEX IF NOT EXISTS idx_subinterface_metrics_system_interface ON subinterface_metrics (system_id, interface_name, subinterface_index, time DESC);

-- 设置数据保留策略（可选，保留30天数据）
-- SELECT add_retention_policy('platform_metrics', INTERVAL '30 days', if_not_exists => TRUE);
-- SELECT add_retention_policy('interface_metrics', INTERVAL '30 days', if_not_exists => TRUE);
-- SELECT add_retention_policy('subinterface_metrics', INTERVAL '30 days', if_not_exists => TRUE);

-- 授权给telemetry_app用户
GRANT ALL ON ALL TABLES IN SCHEMA telemetry TO telemetry_app;
GRANT ALL ON ALL SEQUENCES IN SCHEMA telemetry TO telemetry_app;
GRANT USAGE ON SCHEMA telemetry TO telemetry_app;