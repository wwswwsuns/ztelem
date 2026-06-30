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
	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/models"
)

// Database 数据库连接结构
type Database struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger
}

// NewDatabase 创建新的数据库连接
func NewDatabase(host string, port int, user, password, dbname string, logger *logrus.Logger) (*Database, error) {
	return NewDatabaseWithConfig(config.DatabaseConfig{
		Host: host, Port: port, User: user, Password: password, Database: dbname,
		MaxOpenConns: 200, MaxIdleConns: 25,
		ConnMaxLifetime: time.Hour, ConnMaxIdleTime: 30 * time.Minute,
	}, logger)
}

// NewDatabaseWithConfig 从配置创建数据库连接
func NewDatabaseWithConfig(cfg config.DatabaseConfig, logger *logrus.Logger) (*Database, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable search_path=telemetry,public",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析数据库配置失败: %v", err)
	}

	if cfg.MaxOpenConns > 0 {
		poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	} else {
		poolConfig.MaxConns = 200
	}
	if cfg.MaxIdleConns > 0 {
		poolConfig.MinConns = int32(cfg.MaxIdleConns)
	} else {
		poolConfig.MinConns = 25
	}
	if cfg.ConnMaxLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	} else {
		poolConfig.MaxConnLifetime = time.Hour
	}
	if cfg.ConnMaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime
	} else {
		poolConfig.MaxConnIdleTime = 30 * time.Minute
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("创建数据库连接池失败: %v", err)
	}

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
		c := safeCommon(&metric)
		cpu := safeCPU(&metric)
		mem := safeMem(&metric)
		tmp := safeTemp(&metric)
		fan := safeFan(&metric)
		pwr := safePower(&metric)
		opt := safeOptical(&metric)
		row := []interface{}{
			metric.Timestamp,
			metric.SystemID,
			metric.ComponentName,
			safeString(c.OperStatus),
			safeString(c.Uptime),
			safeUint32(c.UsedPower),
			safeUint32(c.AllocatedPower),
			safeString(c.CurrentVoltage),
			safeString(c.CurrentCurrent),
			safeString(c.TotalCapacity),
			safeString(c.UsedCapacity),
			safeString(c.Type),
			safeString(c.RedundancyType),
			safeString(c.Modules),
			safeString(c.TotalInputPower),
			safeUint32(fan.FanSpeed),
			safeString(fan.FanState),
			safeString(fan.FanPhyStatus),
			safeString(fan.FanWorkMode),
			safeString(fan.FanCurrentPower),
			safeString(fan.FanCurrentVoltage),
			safeString(fan.FanCurrentCurrent),
			safeString(fan.FanSpeedPercent),
			safeUint64(mem.MemAvailable),
			safeUint64(mem.MemUtilized),
			safeUint64(mem.MemFree),
			safeFloat64(mem.MemUsage),
			safeString(mem.MemAlarmStatus),
			safeFloat64(mem.StorageAvailability),
			safeFloat64(tmp.TempInstant),
			safeFloat64(tmp.TempAvg),
			safeFloat64(tmp.TempMin),
			safeFloat64(tmp.TempMax),
			safeUint64(tmp.TempInterval),
			safeTime(tmp.TempMinTime),
			safeTime(tmp.TempMaxTime),
			safeBool(tmp.AlarmStatus),
			safeFloat64(tmp.TempAlarmThreshold),
			safeString(tmp.TempAlarmSeverity),
			safeFloat64(tmp.TempMinorThreshold),
			safeFloat64(tmp.TempMajorThreshold),
			safeFloat64(tmp.TempFatalThreshold),
			safeString(tmp.TempInstantString),
			safeString(tmp.TempStatus),
			safeString(tmp.TempDescription),
			safeBool(pwr.PowerEnable),
			safeFloat64(pwr.PowerCapacity),
			safeFloat64(pwr.PowerInputCurrent),
			safeFloat64(pwr.PowerInputVoltage),
			safeFloat64(pwr.PowerOutputCurrent),
			safeFloat64(pwr.PowerOutputVoltage),
			safeFloat64(pwr.PowerOutputPower),
			safeString(pwr.PowerWorkState),
			safeString(pwr.PowerName),
			safeString(pwr.PowerPhyState),
			safeString(pwr.PowerState),
			safeString(pwr.PowerComState),
			safeString(pwr.PowerTemperature),
			safeString(pwr.PowerAvailable),
			safeString(pwr.PowerCapacityString),
			safeString(pwr.PowerInputPower),
			safeFloat64(pwr.PowerInput2Current),
			safeFloat64(pwr.PowerInput2Voltage),
			safeFloat64(pwr.PowerOutput2Current),
			safeFloat64(pwr.PowerOutput2Voltage),
			safeString(pwr.LinecardPowerAdminState),
			safeFloat64(cpu.CPUInstant),
			safeFloat64(cpu.CPUAvg),
			safeFloat64(cpu.CPUMin),
			safeFloat64(cpu.CPUMax),
			safeUint64(cpu.CPUInterval),
			safeTime(cpu.CPUMinTime),
			safeTime(cpu.CPUMaxTime),
			safeString(cpu.CPUAlarmStatus),
			safeFloat64(opt.OpticalInPower),
			safeFloat64(opt.OpticalOutPower),
			safeFloat64(opt.OpticalBiasCurrent),
			safeFloat64(opt.OpticalTemperature),
			safeFloat64(opt.OpticalVoltageVol33),
			safeFloat64(opt.OpticalVoltageVol5),
			safeString(opt.OpticalAlarmLosStatus),
			safeUint32(opt.OpticalAlarmLosInfoEventID),
			safeUint32(opt.OpticalAlarmLosInfoEventInterval),
			safeFloat64(opt.OpticalAlarmLosInfoInPower),
			safeFloat64(opt.OpticalAlarmLosInfoOutPower),
			safeString(opt.OpticalOnlineStatus),
			safeFloat64(opt.OpticalRxThresholdHighAlarm),
			safeFloat64(opt.OpticalRxThresholdPreHighAlarm),
			safeFloat64(opt.OpticalRxThresholdLowAlarm),
			safeFloat64(opt.OpticalRxThresholdPreLowAlarm),
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

// safeDeref 安全解引用指针，nil 返回零值
func safeDeref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// safeCommon 安全获取 CommonState
func safeCommon(m *models.PlatformMetric) *models.CommonState {
	if m.CommonState == nil {
		return &models.CommonState{}
	}
	return m.CommonState
}

// safeCPU 安全获取 CPUData
func safeCPU(m *models.PlatformMetric) *models.CPUData {
	if m.CPUData == nil {
		return &models.CPUData{}
	}
	return m.CPUData
}

// safeMem 安全获取 MemData
func safeMem(m *models.PlatformMetric) *models.MemData {
	if m.MemData == nil {
		return &models.MemData{}
	}
	return m.MemData
}

// safeTemp 安全获取 TempData
func safeTemp(m *models.PlatformMetric) *models.TempData {
	if m.TempData == nil {
		return &models.TempData{}
	}
	return m.TempData
}

// safeFan 安全获取 FanData
func safeFan(m *models.PlatformMetric) *models.FanData {
	if m.FanData == nil {
		return &models.FanData{}
	}
	return m.FanData
}

// safePower 安全获取 PowerData
func safePower(m *models.PlatformMetric) *models.PowerData {
	if m.PowerData == nil {
		return &models.PowerData{}
	}
	return m.PowerData
}

// safeOptical 安全获取 OpticalData
func safeOptical(m *models.PlatformMetric) *models.OpticalData {
	if m.OpticalData == nil {
		return &models.OpticalData{}
	}
	return m.OpticalData
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