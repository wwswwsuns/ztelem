package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/wwswwsuns/ztelem/internal/models"
)

// Database 数据库连接结构
type Database struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger
}

// NewDatabase 创建新的数据库连接
func NewDatabase(host string, port int, user, password, dbname string, logger *logrus.Logger) (*Database, error) {
	// 构建pgx连接字符串
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable search_path=telemetry,public",
		host, port, user, password, dbname)

	// 创建连接池配置
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析数据库配置失败: %v", err)
	}

	// 设置连接池参数
	poolConfig.MaxConns = 200
	poolConfig.MinConns = 25
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("创建数据库连接池失败: %v", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	logger.Infof("pgx数据库连接池初始化成功 max_conns=%d max_idle_time=%v max_lifetime=%v min_conns=%d",
		poolConfig.MaxConns, poolConfig.MaxConnIdleTime, poolConfig.MaxConnLifetime, poolConfig.MinConns)

	return &Database{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close 关闭数据库连接
func (db *Database) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}



// BatchInsertInterfaceMetricsWithContext 批量插入接口指标（带Context）
func (db *Database) BatchInsertInterfaceMetricsWithContext(ctx context.Context, metrics []models.InterfaceMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// 获取连接
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}
	defer conn.Release()

	// 准备数据 - 使用实际的字段名
	var rows [][]interface{}
	for _, metric := range metrics {
		row := []interface{}{
			metric.Timestamp,
			metric.SystemID,
			metric.InterfaceName,
			metric.InOctets,
			metric.OutOctets,
			metric.InUnicastPkts,  // 正确的字段名
			metric.OutUnicastPkts, // 正确的字段名
			metric.InDiscards,
			metric.OutDiscards,
			metric.InErrors,
			metric.OutErrors,
			metric.InUnknownProtos,
			metric.InMulticastPkts,
			metric.OutMulticastPkts,
			metric.InBroadcastPkts,
			metric.OutBroadcastPkts,
			metric.AdminStatusStr,
			metric.OperStatusStr,
			metric.LastChange,
			metric.Ifindex,
			metric.Type,
			metric.PhyStatusStr,
			metric.IPv4OperStatusStr,
			metric.Logical,
			metric.ZteifType,
			metric.ZteifIfindex,
			metric.ZteifAdminStatusStr,
			metric.ZteifOperStatusStr,
			metric.ZteifPhyStatusStr,
			metric.ZteifIPv4OperStatusStr,
			metric.ZteifIPv6OperStatusStr,
			metric.InFcsErrors,
			metric.CarrierTransitions,
			metric.LastClear,
			metric.InPkts,
			metric.OutPkts,
			safeNumericString(metric.InputUtilization),
			safeNumericString(metric.OutputUtilization),
			metric.InTrafficRate,
			metric.InPacketRate,
			metric.OutTrafficRate,
			metric.OutPacketRate,
			metric.InV4Octets,
			metric.OutV4Octets,
			metric.InV4Pkts,
			metric.OutV4Pkts,
			metric.InV6Octets,
			metric.OutV6Octets,
			metric.InV6Pkts,
			metric.OutV6Pkts,
			metric.InV4TrafficRate,
			metric.InV4PacketRate,
			metric.OutV4TrafficRate,
			metric.OutV4PacketRate,
			metric.InV6TrafficRate,
			metric.InV6PacketRate,
			metric.OutV6TrafficRate,
			metric.OutV6PacketRate,
			safeNumericString(metric.InputV4Utilization),
			safeNumericString(metric.OutputV4Utilization),
			safeNumericString(metric.InputV6Utilization),
			safeNumericString(metric.OutputV6Utilization),
			metric.InBierOctets,
			metric.InBierPkts,
			metric.OutBierOctets,
			metric.OutBierPkts,
		}
		rows = append(rows, row)
	}

	// 执行COPY FROM STDIN - 使用完整的字段列表
	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{"interface_metrics"}, 
		[]string{
			"timestamp", "system_id", "interface_name", "in_octets", "out_octets", "in_unicast_pkts", "out_unicast_pkts",
			"in_discards", "out_discards", "in_errors", "out_errors", "in_unknown_protos", "in_multicast_pkts",
			"out_multicast_pkts", "in_broadcast_pkts", "out_broadcast_pkts", "admin_status", "oper_status",
			"last_change", "ifindex", "type", "phy_status", "ipv4_oper_status", "logical",
			"zteif_type", "zteif_ifindex", "zteif_admin_status", "zteif_oper_status", "zteif_phy_status",
			"zteif_ipv4_oper_status", "zteif_ipv6_oper_status", "in_fcs_errors", "carrier_transitions",
			"last_clear", "in_pkts", "out_pkts", "input_utilization", "output_utilization",
			"in_traffic_rate", "in_packet_rate", "out_traffic_rate", "out_packet_rate",
			"in_v4_octets", "out_v4_octets", "in_v4_pkts", "out_v4_pkts",
			"in_v6_octets", "out_v6_octets", "in_v6_pkts", "out_v6_pkts",
			"in_v4_traffic_rate", "in_v4_packet_rate", "out_v4_traffic_rate", "out_v4_packet_rate",
			"in_v6_traffic_rate", "in_v6_packet_rate", "out_v6_traffic_rate", "out_v6_packet_rate",
			"input_v4_utilization", "output_v4_utilization", "input_v6_utilization", "output_v6_utilization",
			"in_bier_octets", "in_bier_pkts", "out_bier_octets", "out_bier_pkts",
		},
		pgx.CopyFromRows(rows))

	if err != nil {
		return fmt.Errorf("COPY FROM STDIN 插入接口指标失败: %v", err)
	}

	db.logger.Debugf("成功批量插入接口指标 %d 条", len(metrics))
	return nil
}

// BatchInsertSubinterfaceMetricsWithContext 批量插入子接口指标（带Context）
func (db *Database) BatchInsertSubinterfaceMetricsWithContext(ctx context.Context, metrics []models.SubinterfaceMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// 获取连接
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}
	defer conn.Release()

	// 准备数据
	var rows [][]interface{}
	for _, metric := range metrics {
		row := []interface{}{
			metric.Timestamp,
			metric.SystemID,
			metric.InterfaceName,
			metric.SubinterfaceName, // 使用正确的字段名
			metric.Ifindex,
			metric.AdminStatusStr,
			metric.OperStatusStr,
			metric.LastChange,
			metric.Logical,
			metric.IPv4OperStatusStr,
			metric.ZteifIfindex,
			metric.ZteifAdminStatusStr,
			metric.ZteifOperStatusStr,
			metric.ZteifPhyStatusStr,
			metric.ZteifIPv4OperStatusStr,
			metric.ZteifIPv6OperStatusStr,
			metric.InOctets,
			metric.InUnicastPkts,
			metric.InBroadcastPkts,
			metric.InMulticastPkts,
			metric.InDiscards,
			metric.InErrors,
			metric.InUnknownProtos,
			metric.InFcsErrors,
			metric.OutOctets,
			metric.OutUnicastPkts,
			metric.OutBroadcastPkts,
			metric.OutMulticastPkts,
			metric.OutDiscards,
			metric.OutErrors,
			metric.CarrierTransitions,
			metric.LastClear,
			metric.InPkts,
			metric.OutPkts,
			safeNumericString(metric.InputUtilization),
			safeNumericString(metric.OutputUtilization),
			metric.InTrafficRate,
			metric.InPacketRate,
			metric.OutTrafficRate,
			metric.OutPacketRate,
			metric.InV4Octets,
			metric.OutV4Octets,
			metric.InV4Pkts,
			metric.OutV4Pkts,
			metric.InV6Octets,
			metric.OutV6Octets,
			metric.InV6Pkts,
			metric.OutV6Pkts,
			metric.InV4TrafficRate,
			metric.InV4PacketRate,
			metric.OutV4TrafficRate,
			metric.OutV4PacketRate,
			metric.InV6TrafficRate,
			metric.InV6PacketRate,
			metric.OutV6TrafficRate,
			metric.OutV6PacketRate,
			safeNumericString(metric.InputV4Utilization),
			safeNumericString(metric.OutputV4Utilization),
			safeNumericString(metric.InputV6Utilization),
			safeNumericString(metric.OutputV6Utilization),
			metric.InBierOctets,
			metric.InBierPkts,
			metric.OutBierOctets,
			metric.OutBierPkts,
		}
		rows = append(rows, row)
	}

	// 执行COPY FROM STDIN
	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{"subinterface_metrics"}, 
		[]string{
			"timestamp", "system_id", "interface_name", "subinterface_index", "ifindex", "admin_status", "oper_status",
			"last_change", "logical", "ipv4_oper_status", "zteif_ifindex", "zteif_admin_status", "zteif_oper_status",
			"zteif_phy_status", "zteif_ipv4_oper_status", "zteif_ipv6_oper_status", "in_octets", "in_unicast_pkts",
			"in_broadcast_pkts", "in_multicast_pkts", "in_discards", "in_errors", "in_unknown_protos", "in_fcs_errors",
			"out_octets", "out_unicast_pkts", "out_broadcast_pkts", "out_multicast_pkts", "out_discards", "out_errors",
			"carrier_transitions", "last_clear", "in_pkts", "out_pkts", "input_utilization", "output_utilization",
			"in_traffic_rate", "in_packet_rate", "out_traffic_rate", "out_packet_rate",
			"in_v4_octets", "out_v4_octets", "in_v4_pkts", "out_v4_pkts",
			"in_v6_octets", "out_v6_octets", "in_v6_pkts", "out_v6_pkts",
			"in_v4_traffic_rate", "in_v4_packet_rate", "out_v4_traffic_rate", "out_v4_packet_rate",
			"in_v6_traffic_rate", "in_v6_packet_rate", "out_v6_traffic_rate", "out_v6_packet_rate",
			"input_v4_utilization", "output_v4_utilization", "input_v6_utilization", "output_v6_utilization",
			"in_bier_octets", "in_bier_pkts", "out_bier_octets", "out_bier_pkts",
		},
		pgx.CopyFromRows(rows))

	if err != nil {
		return fmt.Errorf("COPY FROM STDIN 插入子接口指标失败: %v", err)
	}

	db.logger.Debugf("成功批量插入子接口指标 %d 条", len(metrics))
	return nil
}

// GetStats 获取数据库连接统计信息
func (db *Database) GetStats() sql.DBStats {
	// pgx连接池没有直接的DBStats，返回一个模拟的统计信息
	return sql.DBStats{
		MaxOpenConnections: int(db.pool.Config().MaxConns),
		OpenConnections:    int(db.pool.Stat().TotalConns()),
		InUse:              int(db.pool.Stat().AcquiredConns()),
		Idle:               int(db.pool.Stat().IdleConns()),
	}
}

// 适配器方法 - 为了兼容buffer包的DatabaseInterface接口
// 这些方法不包含context参数，内部使用context.Background()

func (db *Database) BatchInsertPlatformMetrics(metrics []models.PlatformMetric) error {
	return db.BatchInsertPlatformMetricsWithContext(context.Background(), metrics)
}

func (db *Database) BatchInsertInterfaceMetrics(metrics []models.InterfaceMetric) error {
	return db.BatchInsertInterfaceMetricsWithContext(context.Background(), metrics)
}

func (db *Database) BatchInsertSubinterfaceMetrics(metrics []models.SubinterfaceMetric) error {
	return db.BatchInsertSubinterfaceMetricsWithContext(context.Background(), metrics)
}

func (db *Database) BatchInsertAlarmReportMetrics(metrics []models.AlarmReportMetric) error {
	return db.BatchInsertAlarmReportMetricsWithContext(context.Background(), metrics)
}

func (db *Database) BatchInsertNotificationReportMetrics(metrics []models.NotificationReportMetric) error {
	return db.BatchInsertNotificationReportMetricsWithContext(context.Background(), metrics)
}

// 重命名原有方法为带Context的版本 - 修复版本，正确处理nil指针
func (db *Database) BatchInsertPlatformMetricsWithContext(ctx context.Context, metrics []models.PlatformMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// 获取连接
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}
	defer conn.Release()

	// 准备数据 - 正确处理nil指针
	var rows [][]interface{}
	for _, metric := range metrics {
		row := []interface{}{
			metric.Timestamp,
			metric.SystemID,
			metric.ComponentName,
			safeString(metric.OperStatus),
			safeString(metric.Uptime),
			safeUint32(metric.UsedPower),
			safeUint32(metric.AllocatedPower),
			safeString(metric.CurrentVoltage),
			safeString(metric.CurrentCurrent),
			safeString(metric.TotalCapacity),
			safeString(metric.UsedCapacity),
			safeString(metric.Type),
			safeString(metric.RedundancyType),
			safeString(metric.Modules),
			safeString(metric.TotalInputPower),
			safeUint32(metric.FanSpeed),
			safeString(metric.FanState),
			safeString(metric.FanPhyStatus),
			safeString(metric.FanWorkMode),
			safeString(metric.FanCurrentPower),
			safeString(metric.FanCurrentVoltage),
			safeString(metric.FanCurrentCurrent),
			safeString(metric.FanSpeedPercent),
			safeUint64(metric.MemAvailable),
			safeUint64(metric.MemUtilized),
			safeUint64(metric.MemFree),
			safeFloat64(metric.MemUsage),
			safeString(metric.MemAlarmStatus),
			safeFloat64(metric.StorageAvailability),
			safeFloat64(metric.TempInstant),
			safeFloat64(metric.TempAvg),
			safeFloat64(metric.TempMin),
			safeFloat64(metric.TempMax),
			safeUint64(metric.TempInterval),
			safeTime(metric.TempMinTime),
			safeTime(metric.TempMaxTime),
			safeBool(metric.AlarmStatus),
			safeFloat64(metric.TempAlarmThreshold),
			safeString(metric.TempAlarmSeverity),
			safeFloat64(metric.TempMinorThreshold),
			safeFloat64(metric.TempMajorThreshold),
			safeFloat64(metric.TempFatalThreshold),
			safeString(metric.TempInstantString),
			safeString(metric.TempStatus),
			safeString(metric.TempDescription),
			safeBool(metric.PowerEnable),
			safeFloat64(metric.PowerCapacity),
			safeFloat64(metric.PowerInputCurrent),
			safeFloat64(metric.PowerInputVoltage),
			safeFloat64(metric.PowerOutputCurrent),
			safeFloat64(metric.PowerOutputVoltage),
			safeFloat64(metric.PowerOutputPower),
			safeString(metric.PowerWorkState),
			safeString(metric.PowerName),
			safeString(metric.PowerPhyState),
			safeString(metric.PowerState),
			safeString(metric.PowerComState),
			safeString(metric.PowerTemperature),
			safeString(metric.PowerAvailable),
			safeString(metric.PowerCapacityString),
			safeString(metric.PowerInputPower),
			safeFloat64(metric.PowerInput2Current),
			safeFloat64(metric.PowerInput2Voltage),
			safeFloat64(metric.PowerOutput2Current),
			safeFloat64(metric.PowerOutput2Voltage),
			safeString(metric.LinecardPowerAdminState),
			safeFloat64(metric.CPUInstant),
			safeFloat64(metric.CPUAvg),
			safeFloat64(metric.CPUMin),
			safeFloat64(metric.CPUMax),
			safeUint64(metric.CPUInterval),
			safeTime(metric.CPUMinTime),
			safeTime(metric.CPUMaxTime),
			safeString(metric.CPUAlarmStatus),
			safeFloat64(metric.OpticalInPower),
			safeFloat64(metric.OpticalOutPower),
			safeFloat64(metric.OpticalBiasCurrent),
			safeFloat64(metric.OpticalTemperature),
			safeFloat64(metric.OpticalVoltageVol33),
			safeFloat64(metric.OpticalVoltageVol5),
			safeString(metric.OpticalAlarmLosStatus),
			safeUint32(metric.OpticalAlarmLosInfoEventID),
			safeUint32(metric.OpticalAlarmLosInfoEventInterval),
			safeFloat64(metric.OpticalAlarmLosInfoInPower),
			safeFloat64(metric.OpticalAlarmLosInfoOutPower),
			safeString(metric.OpticalOnlineStatus),
			safeFloat64(metric.OpticalRxThresholdHighAlarm),
			safeFloat64(metric.OpticalRxThresholdPreHighAlarm),
			safeFloat64(metric.OpticalRxThresholdLowAlarm),
			safeFloat64(metric.OpticalRxThresholdPreLowAlarm),
		}
		rows = append(rows, row)
	}

	// 执行COPY FROM STDIN
	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{"platform_metrics"}, 
		[]string{
			"timestamp", "system_id", "component_name", "oper_status", "uptime", "used_power", "allocated_power",
			"current_voltage", "current_current", "total_capacity", "used_capacity", "type", "redundancy_type",
			"modules", "total_input_power", "fan_speed", "fan_state", "fan_phy_status", "fan_work_mode",
			"fan_current_power", "fan_current_voltage", "fan_current_current", "fan_speed_percent",
			"mem_available", "mem_utilized", "mem_free", "mem_usage", "mem_alarm_status", "storage_availability",
			"temp_instant", "temp_avg", "temp_min", "temp_max", "temp_interval", "temp_min_time", "temp_max_time",
			"alarm_status", "temp_alarm_threshold", "temp_alarm_severity", "temp_minor_threshold",
			"temp_major_threshold", "temp_fatal_threshold", "temp_instant_string", "temp_status",
			"temp_description", "power_enable", "power_capacity", "power_input_current", "power_input_voltage",
			"power_output_current", "power_output_voltage", "power_output_power", "power_work_state",
			"power_name", "power_phy_state", "power_state", "power_com_state", "power_temperature",
			"power_available", "power_capacity_string", "power_input_power", "power_input2_current",
			"power_input2_voltage", "power_output2_current", "power_output2_voltage",
			"linecard_power_admin_state", "cpu_instant", "cpu_avg", "cpu_min", "cpu_max", "cpu_interval",
			"cpu_min_time", "cpu_max_time", "cpu_alarm_status", "optical_in_power", "optical_out_power",
			"optical_bias_current", "optical_temperature", "optical_voltage_vol33", "optical_voltage_vol5",
			"optical_alarm_los_status", "optical_alarm_los_info_event_id", "optical_alarm_los_info_event_interval",
			"optical_alarm_los_info_in_power", "optical_alarm_los_info_out_power", "optical_online_status",
			"optical_rx_threshold_high_alarm", "optical_rx_threshold_pre_high_alarm",
			"optical_rx_threshold_low_alarm", "optical_rx_threshold_pre_low_alarm",
		},
		pgx.CopyFromRows(rows))

	if err != nil {
		return fmt.Errorf("COPY FROM STDIN 插入平台指标失败: %v", err)
	}

	db.logger.Debugf("成功批量插入平台指标 %d 条", len(metrics))
	return nil
}

// 安全类型转换函数
func safeString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func safeUint32(u *uint32) interface{} {
	if u == nil {
		return nil
	}
	return int32(*u) // PostgreSQL integer类型
}

func safeUint64(u *uint64) interface{} {
	if u == nil {
		return nil
	}
	return int64(*u) // PostgreSQL bigint类型
}

func safeFloat64(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

func safeBool(b *bool) interface{} {
	if b == nil {
		return nil
	}
	return *b
}

func safeTime(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return *t
}

// safeNumericString 安全转换字符串指针为numeric类型
func safeNumericString(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	// 尝试解析为float64
	if val, err := strconv.ParseFloat(*s, 64); err == nil {
		return val
	}
	// 如果解析失败，返回nil
	return nil
}

// BatchInsertAlarmReportMetricsWithContext 批量插入告警上报数据（带Context）
func (db *Database) BatchInsertAlarmReportMetricsWithContext(ctx context.Context, metrics []models.AlarmReportMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// 获取连接
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}
	defer conn.Release()

	// 准备数据
	var rows [][]interface{}
	for _, metric := range metrics {
		row := []interface{}{
			metric.Timestamp, // 使用消息时间戳
			metric.SystemID,
			metric.FlowID,
			metric.Code,
			metric.OccurrenceTime,
			metric.UpdateTime,
			metric.DisappearedTime,
			metric.OccurrenceMs,
			metric.UpdateMs,
			metric.DisappearedMs,
			safeString(metric.AlarmClass),
			safeString(metric.AlarmType),
			safeString(metric.AlarmStatus),
			safeUint32(metric.Sort),
			safeString(metric.Severity),
			safeUint32(metric.TpidType),
			safeUint32(metric.TpidLength),
			safeString(metric.Tpid),
			safeUint32(metric.ProtectGroupWorkStatus),
			safeUint32(metric.ProtectType),
			safeUint32(metric.Reason),
			safeString(metric.ReturnMode),
			safeUint32(metric.ProtectTpidType),
			safeUint32(metric.ProtectTpidLength),
			safeString(metric.ProtectTpid),
			safeUint32(metric.SourceTpidType),
			safeUint32(metric.SourceTpidLength),
			safeString(metric.SourceTpid),
			safeUint32(metric.SwitchTpidType),
			safeUint32(metric.PreviousTpidLength),
			safeUint32(metric.CurrentTpidLength),
			safeString(metric.PreviousTpid),
			safeString(metric.CurrentTpid),
			safeString(metric.PerfAlarmPeriod),
			safeString(metric.PerfAlarmType),
			safeString(metric.PerfAlarmValue),
			safeString(metric.Description),
			safeString(metric.Caption),
		}
		rows = append(rows, row)
	}

	// 执行COPY FROM STDIN
	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{"telemetry", "alarm_report"}, 
		[]string{
			"timestamp", "system_id", "flow_id", "code", "occurrence_time", "update_time", "disappeared_time",
			"occurrence_ms", "update_ms", "disappeared_ms", "alarm_class", "alarm_type", "alarm_status",
			"sort", "severity", "tpid_type", "tpid_length", "tpid", "protect_group_work_status",
			"protect_type", "reason", "return_mode", "protect_tpid_type", "protect_tpid_length",
			"protect_tpid", "source_tpid_type", "source_tpid_length", "source_tpid", "switch_tpid_type",
			"previous_tpid_length", "current_tpid_length", "previous_tpid", "current_tpid",
			"perf_alarm_period", "perf_alarm_type", "perf_alarm_value", "description", "caption",
		},
		pgx.CopyFromRows(rows))

	if err != nil {
		return fmt.Errorf("COPY FROM STDIN 插入告警上报失败: %v", err)
	}

	db.logger.Debugf("成功批量插入告警上报 %d 条", len(metrics))
	return nil
}

// BatchInsertNotificationReportMetricsWithContext 批量插入通知上报数据（带Context）
func (db *Database) BatchInsertNotificationReportMetricsWithContext(ctx context.Context, metrics []models.NotificationReportMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// 获取连接
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}
	defer conn.Release()

	// 准备数据
	var rows [][]interface{}
	for _, metric := range metrics {
		row := []interface{}{
			metric.Timestamp, // 使用消息时间戳
			metric.SystemID,
			metric.FlowID,
			metric.Code,
			metric.OccurTime,
			metric.OccurMs,
			safeString(metric.Classification),
			safeUint32(metric.Sort),
			safeString(metric.Severity),
			safeUint32(metric.TpidType),
			safeUint32(metric.TpidLength),
			safeString(metric.Tpid),
			safeString(metric.Description),
			safeString(metric.Caption),
		}
		rows = append(rows, row)
	}

	// 执行COPY FROM STDIN
	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{"telemetry", "notification_report"}, 
		[]string{
			"timestamp", "system_id", "flow_id", "code", "occur_time", "occur_ms",
			"classification", "sort", "severity", "tpid_type", "tpid_length", "tpid",
			"description", "caption",
		},
		pgx.CopyFromRows(rows))

	if err != nil {
		return fmt.Errorf("COPY FROM STDIN 插入通知上报失败: %v", err)
	}

	db.logger.Debugf("成功批量插入通知上报 %d 条", len(metrics))
	return nil
}

// 保持向后兼容的函数别名
func NewFixedConnection(host string, port int, user, password, dbname string, logger *logrus.Logger) (*Database, error) {
	return NewDatabase(host, port, user, password, dbname, logger)
}