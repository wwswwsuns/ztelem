package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"telemetry-collector/internal/config"
	"telemetry-collector/internal/models"
	_ "github.com/lib/pq"
)

// ExtendedDB 扩展的数据库连接管理器
type ExtendedDB struct {
	conn   *sql.DB
	config config.DatabaseConfig
}

// NewExtendedConnection 创建扩展的数据库连接
func NewExtendedConnection(config config.DatabaseConfig) (*ExtendedDB, error) {
	// 在DSN中直接设置search_path，确保所有连接都使用正确的schema
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode)
	
	// 如果配置了schema，在DSN中添加search_path参数
	if config.Schema != "" {
		dsn += fmt.Sprintf(" search_path=%s", config.Schema)
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 设置连接池参数
	conn.SetMaxOpenConns(config.MaxOpenConns)
	conn.SetMaxIdleConns(config.MaxIdleConns)
	conn.SetConnMaxLifetime(config.ConnMaxLifetime)
	if config.ConnMaxIdleTime > 0 {
		conn.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	// 设置schema
	if config.Schema != "" {
		_, err = conn.Exec(fmt.Sprintf("SET search_path TO %s", config.Schema))
		if err != nil {
			return nil, fmt.Errorf("设置schema失败: %v", err)
		}
	}

	log.Printf("数据库连接池配置: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v", 
		config.MaxOpenConns, config.MaxIdleConns, config.ConnMaxLifetime)

	return &ExtendedDB{conn: conn, config: config}, nil
}

// Close 关闭数据库连接
func (db *ExtendedDB) Close() error {
	return db.conn.Close()
}

// GetStats 获取连接池统计信息
func (db *ExtendedDB) GetStats() sql.DBStats {
	return db.conn.Stats()
}

// BatchInsertPlatformMetrics 批量插入平台指标数据 - 优化版本
func (db *ExtendedDB) BatchInsertPlatformMetrics(data []models.PlatformMetric) error {
	if len(data) == 0 {
		return nil
	}

	// 使用事务提高性能
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 构建批量插入SQL
	sql := `INSERT INTO platform_metrics (
		time, system_id, component_name, oper_status, uptime, used_power, allocated_power,
		current_voltage, current_current, total_capacity, used_capacity, type, redundancy_type,
		modules, total_input_power, fan_speed, fan_state, fan_phy_status, fan_work_mode,
		fan_current_power, fan_current_voltage, fan_current_current, fan_speed_percent,
		mem_available, mem_utilized, mem_free, mem_usage, mem_alarm_status,
		storage_availability, temp_instant, temp_avg, temp_min, temp_max, temp_interval,
		temp_min_time, temp_max_time, alarm_status, temp_alarm_threshold, temp_alarm_severity,
		temp_minor_threshold, temp_major_threshold, temp_fatal_threshold, temp_instant_string,
		temp_status, temp_description, power_enable, power_capacity, power_input_current,
		power_input_voltage, power_output_current, power_output_voltage, power_output_power,
		power_work_state, power_name, power_phy_state, power_state, power_com_state,
		power_temperature, power_available, power_capacity_string, power_input_power,
		power_input2_current, power_input2_voltage, power_output2_current, power_output2_voltage,
		linecard_power_admin_state, cpu_instant, cpu_avg, cpu_min, cpu_max, cpu_interval,
		cpu_min_time, cpu_max_time, cpu_alarm_status, optical_in_power, optical_out_power,
		optical_bias_current, optical_temperature, optical_voltage_vol33, optical_voltage_vol5,
		optical_alarm_los_status, optical_alarm_los_info_event_id, optical_alarm_los_info_event_interval,
		optical_alarm_los_info_in_power, optical_alarm_los_info_out_power, optical_online_status,
		optical_rx_threshold_high_alarm, optical_rx_threshold_pre_high_alarm, optical_rx_threshold_low_alarm,
		optical_rx_threshold_pre_low_alarm
	) VALUES `

	var valueStrings []string
	var valueArgs []interface{}

	paramIndex := 1
	for _, metric := range data {
		// 生成参数占位符 (实际字段数量为90个)
		var params []string
		for j := 0; j < 90; j++ {
			params = append(params, fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		}
		valueStrings = append(valueStrings, "("+strings.Join(params, ",")+")")
		
		// 转换模型为数据库参数
		var memAlarmStatus interface{}
		if metric.MemAlarmStatus != nil {
			switch *metric.MemAlarmStatus {
			case "INVALID":
				memAlarmStatus = 0
			case "NORMAL":
				memAlarmStatus = 1
			case "ALARM":
				memAlarmStatus = 2
			default:
				memAlarmStatus = 0
			}
		} else {
			memAlarmStatus = nil
		}
		
		var cpuAlarmStatus interface{}
		if metric.CPUAlarmStatus != nil {
			switch *metric.CPUAlarmStatus {
			case "INVALID":
				cpuAlarmStatus = 0
			case "NORMAL":
				cpuAlarmStatus = 1
			case "ALARM":
				cpuAlarmStatus = 2
			default:
				cpuAlarmStatus = 0
			}
		} else {
			cpuAlarmStatus = nil
		}
		
		var opticalAlarmLosStatus interface{}
		if metric.OpticalAlarmLosStatus != nil {
			opticalAlarmLosStatus = *metric.OpticalAlarmLosStatus
		} else {
			opticalAlarmLosStatus = nil
		}
		
		valueArgs = append(valueArgs,
			metric.Timestamp, metric.SystemID, metric.ComponentName,
			metric.OperStatus, metric.Uptime, metric.UsedPower, metric.AllocatedPower,
			metric.CurrentVoltage, metric.CurrentCurrent, metric.TotalCapacity, metric.UsedCapacity,
			metric.Type, metric.RedundancyType, metric.Modules, metric.TotalInputPower,
			metric.FanSpeed, metric.FanState, metric.FanPhyStatus, metric.FanWorkMode,
			metric.FanCurrentPower, metric.FanCurrentVoltage, metric.FanCurrentCurrent, metric.FanSpeedPercent,
			metric.MemAvailable, metric.MemUtilized, metric.MemFree, metric.MemUsage, memAlarmStatus,
			metric.StorageAvailability, metric.TempInstant, metric.TempAvg, metric.TempMin, metric.TempMax,
			metric.TempInterval, metric.TempMinTime, metric.TempMaxTime, metric.AlarmStatus,
			metric.TempAlarmThreshold, metric.TempAlarmSeverity, metric.TempMinorThreshold, metric.TempMajorThreshold,
			metric.TempFatalThreshold, metric.TempInstantString, metric.TempStatus, metric.TempDescription,
			metric.PowerEnable, metric.PowerCapacity, metric.PowerInputCurrent, metric.PowerInputVoltage,
			metric.PowerOutputCurrent, metric.PowerOutputVoltage, metric.PowerOutputPower, metric.PowerWorkState,
			metric.PowerName, metric.PowerPhyState, metric.PowerState, metric.PowerComState,
			metric.PowerTemperature, metric.PowerAvailable, metric.PowerCapacityString, metric.PowerInputPower,
			metric.PowerInput2Current, metric.PowerInput2Voltage, metric.PowerOutput2Current, metric.PowerOutput2Voltage,
			metric.LinecardPowerAdminState, metric.CPUInstant, metric.CPUAvg, metric.CPUMin, metric.CPUMax,
			metric.CPUInterval, metric.CPUMinTime, metric.CPUMaxTime, cpuAlarmStatus,
			metric.OpticalInPower, metric.OpticalOutPower, metric.OpticalBiasCurrent, metric.OpticalTemperature,
			metric.OpticalVoltageVol33, metric.OpticalVoltageVol5, opticalAlarmLosStatus,
			metric.OpticalAlarmLosInfoEventID, metric.OpticalAlarmLosInfoEventInterval,
			metric.OpticalAlarmLosInfoInPower, metric.OpticalAlarmLosInfoOutPower, metric.OpticalOnlineStatus,
			metric.OpticalRxThresholdHighAlarm, metric.OpticalRxThresholdPreHighAlarm,
			metric.OpticalRxThresholdLowAlarm, metric.OpticalRxThresholdPreLowAlarm,
		)
	}

	sql += strings.Join(valueStrings, ",")

	_, err = tx.Exec(sql, valueArgs...)
	if err != nil {
		return fmt.Errorf("批量插入平台指标数据失败: %v", err)
	}

	return tx.Commit()
}

// BatchInsertInterfaceMetrics 批量插入接口指标数据 - 优化版本
func (db *ExtendedDB) BatchInsertInterfaceMetrics(data []models.InterfaceMetric) error {
	if len(data) == 0 {
		return nil
	}

	// 使用事务提高性能
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 构建批量插入SQL
	sql := `INSERT INTO interface_metrics (
		time, system_id, interface_name, ifindex, admin_status, oper_status, last_change, logical, type, phy_status, ipv4_oper_status,
		zteif_type, zteif_ifindex, zteif_admin_status, zteif_oper_status, zteif_phy_status, zteif_ipv4_oper_status, zteif_ipv6_oper_status,
		in_octets, in_unicast_pkts, in_broadcast_pkts, in_multicast_pkts, in_discards, in_errors, in_unknown_protos, in_fcs_errors,
		out_octets, out_unicast_pkts, out_broadcast_pkts, out_multicast_pkts, out_discards, out_errors, carrier_transitions, last_clear,
		in_pkts, out_pkts, input_utilization, output_utilization, in_traffic_rate, in_packet_rate, out_traffic_rate, out_packet_rate,
		in_v4_octets, out_v4_octets, in_v4_pkts, out_v4_pkts, in_v6_octets, out_v6_octets, in_v6_pkts, out_v6_pkts,
		in_v4_traffic_rate, in_v4_packet_rate, out_v4_traffic_rate, out_v4_packet_rate, in_v6_traffic_rate, in_v6_packet_rate, out_v6_traffic_rate, out_v6_packet_rate,
		input_v4_utilization, output_v4_utilization, input_v6_utilization, output_v6_utilization, in_bier_octets, in_bier_pkts, out_bier_octets, out_bier_pkts
	) VALUES `

	var valueStrings []string
	var valueArgs []interface{}

	paramIndex := 1
	for _, metric := range data {
		// 生成参数占位符 (接口指标有66个字段)
		var params []string
		for j := 0; j < 66; j++ {
			params = append(params, fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		}
		valueStrings = append(valueStrings, "("+strings.Join(params, ",")+")")
		
		// 转换模型为数据库参数
		args := []interface{}{
			metric.Timestamp, metric.SystemID, metric.InterfaceName,
			metric.Ifindex, 
			func() interface{} {
				if metric.AdminStatusStr != nil {
					return *metric.AdminStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.OperStatusStr != nil {
					return *metric.OperStatusStr
				}
				return nil
			}(),
			metric.LastChange, metric.Logical,
			metric.Type, 
			func() interface{} {
				if metric.PhyStatusStr != nil {
					return *metric.PhyStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.IPv4OperStatusStr != nil {
					return *metric.IPv4OperStatusStr
				}
				return nil
			}(),
			metric.ZteifType, metric.ZteifIfindex, 
			func() interface{} {
				if metric.ZteifAdminStatusStr != nil {
					return *metric.ZteifAdminStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifOperStatusStr != nil {
					return *metric.ZteifOperStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifPhyStatusStr != nil {
					return *metric.ZteifPhyStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifIPv4OperStatusStr != nil {
					return *metric.ZteifIPv4OperStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifIPv6OperStatusStr != nil {
					return *metric.ZteifIPv6OperStatusStr
				}
				return nil
			}(),
			metric.InOctets, metric.InUnicastPkts, metric.InBroadcastPkts, metric.InMulticastPkts,
			metric.InDiscards, metric.InErrors, metric.InUnknownProtos, metric.InFcsErrors,
			metric.OutOctets, metric.OutUnicastPkts, metric.OutBroadcastPkts, metric.OutMulticastPkts,
			metric.OutDiscards, metric.OutErrors, metric.CarrierTransitions, metric.LastClear,
			metric.InPkts, metric.OutPkts, metric.InputUtilization, metric.OutputUtilization,
			metric.InTrafficRate, metric.InPacketRate, metric.OutTrafficRate, metric.OutPacketRate,
			metric.InV4Octets, metric.OutV4Octets, metric.InV4Pkts, metric.OutV4Pkts,
			metric.InV6Octets, metric.OutV6Octets, metric.InV6Pkts, metric.OutV6Pkts,
			metric.InV4TrafficRate, metric.InV4PacketRate, metric.OutV4TrafficRate, metric.OutV4PacketRate,
			metric.InV6TrafficRate, metric.InV6PacketRate, metric.OutV6TrafficRate, metric.OutV6PacketRate,
			metric.InputV4Utilization, metric.OutputV4Utilization, metric.InputV6Utilization, metric.OutputV6Utilization,
			metric.InBierOctets, metric.InBierPkts, metric.OutBierOctets, metric.OutBierPkts,
		}
		valueArgs = append(valueArgs, args...)
	}

	sql += strings.Join(valueStrings, ",")

	_, err = tx.Exec(sql, valueArgs...)
	if err != nil {
		return fmt.Errorf("批量插入接口指标数据失败: %v", err)
	}

	return tx.Commit()
}

// BatchInsertSubinterfaceMetrics 批量插入子接口指标数据 - 优化版本
func (db *ExtendedDB) BatchInsertSubinterfaceMetrics(data []models.SubinterfaceMetric) error {
	if len(data) == 0 {
		return nil
	}

	// 使用事务提高性能
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 构建批量插入SQL
	sql := `INSERT INTO subinterface_metrics (
		time, system_id, interface_name, subinterface_index, ifindex, admin_status, oper_status, last_change, logical, ipv4_oper_status,
		zteif_ifindex, zteif_admin_status, zteif_oper_status, zteif_phy_status, zteif_ipv4_oper_status, zteif_ipv6_oper_status,
		in_octets, in_unicast_pkts, in_broadcast_pkts, in_multicast_pkts, in_discards, in_errors, in_unknown_protos, in_fcs_errors,
		out_octets, out_unicast_pkts, out_broadcast_pkts, out_multicast_pkts, out_discards, out_errors, carrier_transitions, last_clear,
		in_pkts, out_pkts, input_utilization, output_utilization, in_traffic_rate, in_packet_rate, out_traffic_rate, out_packet_rate,
		in_v4_octets, out_v4_octets, in_v4_pkts, out_v4_pkts, in_v6_octets, out_v6_octets, in_v6_pkts, out_v6_pkts,
		in_v4_traffic_rate, in_v4_packet_rate, out_v4_traffic_rate, out_v4_packet_rate, in_v6_traffic_rate, in_v6_packet_rate, out_v6_traffic_rate, out_v6_packet_rate,
		input_v4_utilization, output_v4_utilization, input_v6_utilization, output_v6_utilization, in_bier_octets, in_bier_pkts, out_bier_octets, out_bier_pkts
	) VALUES `

	var valueStrings []string
	var valueArgs []interface{}

	paramIndex := 1
	for _, metric := range data {
		// 生成参数占位符 (子接口指标有64个字段)
		var params []string
		for j := 0; j < 64; j++ {
			params = append(params, fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		}
		valueStrings = append(valueStrings, "("+strings.Join(params, ",")+")")
		
		// 转换模型为数据库参数
		args := []interface{}{
			metric.Timestamp, metric.SystemID, metric.InterfaceName, metric.SubinterfaceName,
			metric.Ifindex, 
			func() interface{} {
				if metric.AdminStatusStr != nil {
					return *metric.AdminStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.OperStatusStr != nil {
					return *metric.OperStatusStr
				}
				return nil
			}(),
			metric.LastChange, metric.Logical, 
			func() interface{} {
				if metric.IPv4OperStatusStr != nil {
					return *metric.IPv4OperStatusStr
				}
				return nil
			}(),
			metric.ZteifIfindex, 
			func() interface{} {
				if metric.ZteifAdminStatusStr != nil {
					return *metric.ZteifAdminStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifOperStatusStr != nil {
					return *metric.ZteifOperStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifPhyStatusStr != nil {
					return *metric.ZteifPhyStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifIPv4OperStatusStr != nil {
					return *metric.ZteifIPv4OperStatusStr
				}
				return nil
			}(),
			func() interface{} {
				if metric.ZteifIPv6OperStatusStr != nil {
					return *metric.ZteifIPv6OperStatusStr
				}
				return nil
			}(),
			metric.InOctets, metric.InUnicastPkts, metric.InBroadcastPkts, metric.InMulticastPkts,
			metric.InDiscards, metric.InErrors, metric.InUnknownProtos, metric.InFcsErrors,
			metric.OutOctets, metric.OutUnicastPkts, metric.OutBroadcastPkts, metric.OutMulticastPkts,
			metric.OutDiscards, metric.OutErrors, metric.CarrierTransitions, metric.LastClear,
			metric.InPkts, metric.OutPkts, metric.InputUtilization, metric.OutputUtilization,
			metric.InTrafficRate, metric.InPacketRate, metric.OutTrafficRate, metric.OutPacketRate,
			metric.InV4Octets, metric.OutV4Octets, metric.InV4Pkts, metric.OutV4Pkts,
			metric.InV6Octets, metric.OutV6Octets, metric.InV6Pkts, metric.OutV6Pkts,
			metric.InV4TrafficRate, metric.InV4PacketRate, metric.OutV4TrafficRate, metric.OutV4PacketRate,
			metric.InV6TrafficRate, metric.InV6PacketRate, metric.OutV6TrafficRate, metric.OutV6PacketRate,
			metric.InputV4Utilization, metric.OutputV4Utilization, metric.InputV6Utilization, metric.OutputV6Utilization,
			metric.InBierOctets, metric.InBierPkts, metric.OutBierOctets, metric.OutBierPkts,
		}
		valueArgs = append(valueArgs, args...)
	}

	sql += strings.Join(valueStrings, ",")

	_, err = tx.Exec(sql, valueArgs...)
	if err != nil {
		return fmt.Errorf("批量插入子接口指标数据失败: %v", err)
	}

	return tx.Commit()
}

// TestConnection 测试数据库连接
func (db *ExtendedDB) TestConnection() error {
	var result int
	err := db.conn.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("数据库连接测试失败: %v", err)
	}
	log.Printf("数据库连接测试成功")
	return nil
}

// ExecuteWithTimeout 带超时的SQL执行
func (db *ExtendedDB) ExecuteWithTimeout(query string, timeout time.Duration, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	_, err := db.conn.ExecContext(ctx, query, args...)
	return err
}