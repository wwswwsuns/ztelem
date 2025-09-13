package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// AlarmStatus 告警状态枚举
type AlarmStatus int32

const (
	AlarmStatusInvalid AlarmStatus = 0
	AlarmStatusNormal  AlarmStatus = 1
	AlarmStatusAlarm   AlarmStatus = 2
)

func (a AlarmStatus) String() string {
	switch a {
	case AlarmStatusNormal:
		return "NORMAL"
	case AlarmStatusAlarm:
		return "ALARM"
	default:
		return "INVALID"
	}
}

func (a AlarmStatus) Value() (driver.Value, error) {
	return int64(a), nil
}

// AdminStatus 管理状态枚举
type AdminStatus int32

const (
	AdminStatusInvalid AdminStatus = 0
	AdminStatusUp      AdminStatus = 1
	AdminStatusDown    AdminStatus = 2
	AdminStatusTesting AdminStatus = 3
)

func (a AdminStatus) String() string {
	switch a {
	case AdminStatusUp:
		return "UP"
	case AdminStatusDown:
		return "DOWN"
	case AdminStatusTesting:
		return "TESTING"
	default:
		return "INVALID"
	}
}

func (a AdminStatus) Value() (driver.Value, error) {
	return int64(a), nil
}

// OperStatus 操作状态枚举
type OperStatus int32

const (
	OperStatusInvalid         OperStatus = 0
	OperStatusUp              OperStatus = 1
	OperStatusDown            OperStatus = 2
	OperStatusTesting         OperStatus = 3
	OperStatusUnknown         OperStatus = 4
	OperStatusDormant         OperStatus = 5
	OperStatusNotPresent      OperStatus = 6
	OperStatusLowerLayerDown  OperStatus = 7
)

func (o OperStatus) String() string {
	switch o {
	case OperStatusUp:
		return "UP"
	case OperStatusDown:
		return "DOWN"
	case OperStatusTesting:
		return "TESTING"
	case OperStatusUnknown:
		return "UNKNOWN"
	case OperStatusDormant:
		return "DORMANT"
	case OperStatusNotPresent:
		return "NOT_PRESENT"
	case OperStatusLowerLayerDown:
		return "LOWER_LAYER_DOWN"
	default:
		return "INVALID"
	}
}

func (o OperStatus) Value() (driver.Value, error) {
	return int64(o), nil
}

// PhyStatus 物理状态枚举
type PhyStatus int32

const (
	PhyStatusInvalid PhyStatus = 0
	PhyStatusUp      PhyStatus = 1
	PhyStatusDown    PhyStatus = 2
)

func (p PhyStatus) String() string {
	switch p {
	case PhyStatusUp:
		return "UP"
	case PhyStatusDown:
		return "DOWN"
	default:
		return "INVALID"
	}
}

func (p PhyStatus) Value() (driver.Value, error) {
	return int64(p), nil
}

// IPv4OperStatus IPv4操作状态枚举
type IPv4OperStatus int32

const (
	IPv4OperStatusInvalid IPv4OperStatus = 0
	IPv4OperStatusUp      IPv4OperStatus = 1
	IPv4OperStatusDown    IPv4OperStatus = 2
)

func (i IPv4OperStatus) String() string {
	switch i {
	case IPv4OperStatusUp:
		return "UP"
	case IPv4OperStatusDown:
		return "DOWN"
	default:
		return "INVALID"
	}
}

func (i IPv4OperStatus) Value() (driver.Value, error) {
	return int64(i), nil
}

// IPv6OperStatus IPv6操作状态枚举
type IPv6OperStatus int32

const (
	IPv6OperStatusInvalid IPv6OperStatus = 0
	IPv6OperStatusUp      IPv6OperStatus = 1
	IPv6OperStatusDown    IPv6OperStatus = 2
)

func (i IPv6OperStatus) String() string {
	switch i {
	case IPv6OperStatusUp:
		return "UP"
	case IPv6OperStatusDown:
		return "DOWN"
	default:
		return "INVALID"
	}
}

func (i IPv6OperStatus) Value() (driver.Value, error) {
	return int64(i), nil
}

// PlatformMetric 平台指标数据结构
type PlatformMetric struct {
	Timestamp     time.Time `json:"timestamp" db:"time"`
	SystemID      string    `json:"system_id" db:"system_id"`
	ComponentName string    `json:"component_name" db:"component_name"`

	// 组件通用数据
	OperStatus       *string `json:"oper_status,omitempty"`
	Uptime           *string `json:"uptime,omitempty"`           // 转换为dd:hh:mm:ss格式
	UsedPower        *uint32 `json:"used_power,omitempty"`
	AllocatedPower   *uint32 `json:"allocated_power,omitempty"`
	CurrentVoltage   *string `json:"current_voltage,omitempty"`
	CurrentCurrent   *string `json:"current_current,omitempty"`
	TotalCapacity    *string `json:"total_capacity,omitempty"`
	UsedCapacity     *string `json:"used_capacity,omitempty"`
	Type             *string `json:"type,omitempty"`
	RedundancyType   *string `json:"redundancy_type,omitempty"`
	Modules          *string `json:"modules,omitempty"`
	TotalInputPower  *string `json:"total_input_power,omitempty"`

	// 风扇数据
	FanSpeed            *uint32 `json:"fan_speed,omitempty"`
	FanState            *string `json:"fan_state,omitempty"`
	FanPhyStatus        *string `json:"fan_phy_status,omitempty"`
	FanWorkMode         *string `json:"fan_work_mode,omitempty"`
	FanCurrentPower     *string `json:"fan_current_power,omitempty"`
	FanCurrentVoltage   *string `json:"fan_current_voltage,omitempty"`
	FanCurrentCurrent   *string `json:"fan_current_current,omitempty"`
	FanSpeedPercent     *string `json:"fan_speed_percent,omitempty"`

	// 内存数据
	MemAvailable    *uint64  `json:"mem_available,omitempty"`    // MB
	MemUtilized     *uint64  `json:"mem_utilized,omitempty"`     // MB
	MemFree         *uint64  `json:"mem_free,omitempty"`         // MB
	MemUsage        *float64 `json:"mem_usage,omitempty"`        // %
	MemAlarmStatus  *string  `json:"mem_alarm_status,omitempty"`

	// 存储数据
	StorageAvailability *float64 `json:"storage_availability,omitempty"` // %

	// 温度数据
	TempInstant          *float64   `json:"temp_instant,omitempty"`
	TempAvg              *float64   `json:"temp_avg,omitempty"`
	TempMin              *float64   `json:"temp_min,omitempty"`
	TempMax              *float64   `json:"temp_max,omitempty"`
	TempInterval         *uint64    `json:"temp_interval,omitempty"`         // 秒
	TempMinTime          *time.Time `json:"temp_min_time,omitempty"`
	TempMaxTime          *time.Time `json:"temp_max_time,omitempty"`
	AlarmStatus          *bool      `json:"alarm_status,omitempty"`
	TempAlarmThreshold   *float64   `json:"temp_alarm_threshold,omitempty"`
	TempAlarmSeverity    *string    `json:"temp_alarm_severity,omitempty"`
	TempMinorThreshold   *float64   `json:"temp_minor_threshold,omitempty"`
	TempMajorThreshold   *float64   `json:"temp_major_threshold,omitempty"`
	TempFatalThreshold   *float64   `json:"temp_fatal_threshold,omitempty"`
	TempInstantString    *string    `json:"temp_instant_string,omitempty"`
	TempStatus           *string    `json:"temp_status,omitempty"`
	TempDescription      *string    `json:"temp_description,omitempty"`

	// 电源数据
	PowerEnable          *bool    `json:"power_enable,omitempty"`
	PowerCapacity        *float64 `json:"power_capacity,omitempty"`
	PowerInputCurrent    *float64 `json:"power_input_current,omitempty"`
	PowerInputVoltage    *float64 `json:"power_input_voltage,omitempty"`
	PowerOutputCurrent   *float64 `json:"power_output_current,omitempty"`
	PowerOutputVoltage   *float64 `json:"power_output_voltage,omitempty"`
	PowerOutputPower     *float64 `json:"power_output_power,omitempty"`
	PowerWorkState       *string  `json:"power_work_state,omitempty"`
	PowerName            *string  `json:"power_name,omitempty"`
	PowerPhyState        *string  `json:"power_phy_state,omitempty"`
	PowerState           *string  `json:"power_state,omitempty"`
	PowerComState        *string  `json:"power_com_state,omitempty"`
	PowerTemperature     *string  `json:"power_temperature,omitempty"`
	PowerAvailable       *string  `json:"power_available,omitempty"`
	PowerCapacityString  *string  `json:"power_capacity_string,omitempty"`
	PowerInputPower      *string  `json:"power_input_power,omitempty"`
	PowerInput2Current   *float64 `json:"power_input2_current,omitempty"`
	PowerInput2Voltage   *float64 `json:"power_input2_voltage,omitempty"`
	PowerOutput2Current  *float64 `json:"power_output2_current,omitempty"`
	PowerOutput2Voltage  *float64 `json:"power_output2_voltage,omitempty"`

	// 线卡数据
	LinecardPowerAdminState *string `json:"linecard_power_admin_state,omitempty"`

	// CPU数据
	CPUInstant     *float64   `json:"cpu_instant,omitempty"`     // %
	CPUAvg         *float64   `json:"cpu_avg,omitempty"`         // %
	CPUMin         *float64   `json:"cpu_min,omitempty"`         // %
	CPUMax         *float64   `json:"cpu_max,omitempty"`         // %
	CPUInterval    *uint64    `json:"cpu_interval,omitempty"`    // 秒
	CPUMinTime     *time.Time `json:"cpu_min_time,omitempty"`
	CPUMaxTime     *time.Time `json:"cpu_max_time,omitempty"`
	CPUAlarmStatus *string    `json:"cpu_alarm_status,omitempty"`

	// 光模块数据
	OpticalInPower                    *float64 `json:"optical_in_power,omitempty"`
	OpticalOutPower                   *float64 `json:"optical_out_power,omitempty"`
	OpticalBiasCurrent                *float64 `json:"optical_bias_current,omitempty"`
	OpticalTemperature                *float64 `json:"optical_temperature,omitempty"`
	OpticalVoltageVol33               *float64 `json:"optical_voltage_vol33,omitempty"`
	OpticalVoltageVol5                *float64 `json:"optical_voltage_vol5,omitempty"`
	OpticalAlarmLosStatus             *string  `json:"optical_alarm_los_status,omitempty"`
	OpticalAlarmLosInfoEventID        *uint32  `json:"optical_alarm_los_info_event_id,omitempty"`
	OpticalAlarmLosInfoEventInterval  *uint32  `json:"optical_alarm_los_info_event_interval,omitempty"`
	OpticalAlarmLosInfoInPower        *float64 `json:"optical_alarm_los_info_in_power,omitempty"`
	OpticalAlarmLosInfoOutPower       *float64 `json:"optical_alarm_los_info_out_power,omitempty"`
	OpticalOnlineStatus               *string  `json:"optical_online_status,omitempty"`
	OpticalRxThresholdHighAlarm       *float64 `json:"optical_rx_threshold_high_alarm,omitempty"`
	OpticalRxThresholdPreHighAlarm    *float64 `json:"optical_rx_threshold_pre_high_alarm,omitempty"`
	OpticalRxThresholdLowAlarm        *float64 `json:"optical_rx_threshold_low_alarm,omitempty"`
	OpticalRxThresholdPreLowAlarm     *float64 `json:"optical_rx_threshold_pre_low_alarm,omitempty"`
}

// InterfaceMetric 接口指标数据结构
type InterfaceMetric struct {
	Timestamp     time.Time `json:"timestamp" db:"time"`
	SystemID      string    `json:"system_id" db:"system_id"`
	InterfaceName string    `json:"interface_name" db:"interface_name"`

	// 接口状态数据
	Ifindex          *uint32    `json:"ifindex,omitempty"`
	AdminStatusStr  *string  `json:"admin_status,omitempty"`
	OperStatusStr   *string   `json:"oper_status,omitempty"`
	LastChange       *time.Time `json:"last_change,omitempty"`
	Logical          *bool      `json:"logical,omitempty"`
	Type             *uint32    `json:"type,omitempty"`
	PhyStatusStr    *string    `json:"phy_status,omitempty"`
	IPv4OperStatusStr *string `json:"ipv4_oper_status,omitempty"`

	// ZTE接口数据
	ZteifType            *uint32 `json:"zteif_type,omitempty"`
	ZteifIfindex         *uint32 `json:"zteif_ifindex,omitempty"`
	ZteifAdminStatusStr  *string  `json:"zteif_admin_status,omitempty"`
	ZteifOperStatusStr   *string   `json:"zteif_oper_status,omitempty"`
	ZteifPhyStatusStr    *string    `json:"zteif_phy_status,omitempty"`
	ZteifIPv4OperStatusStr *string `json:"zteif_ipv4_oper_status,omitempty"`
	ZteifIPv6OperStatusStr *string `json:"zteif_ipv6_oper_status,omitempty"`

	// 计数器数据
	InOctets             *uint64    `json:"in_octets,omitempty"`
	InUnicastPkts        *uint64    `json:"in_unicast_pkts,omitempty"`
	InBroadcastPkts      *uint64    `json:"in_broadcast_pkts,omitempty"`
	InMulticastPkts      *uint64    `json:"in_multicast_pkts,omitempty"`
	InDiscards           *uint64    `json:"in_discards,omitempty"`
	InErrors             *uint64    `json:"in_errors,omitempty"`
	InUnknownProtos      *uint64    `json:"in_unknown_protos,omitempty"`
	InFcsErrors          *uint64    `json:"in_fcs_errors,omitempty"`
	OutOctets            *uint64    `json:"out_octets,omitempty"`
	OutUnicastPkts       *uint64    `json:"out_unicast_pkts,omitempty"`
	OutBroadcastPkts     *uint64    `json:"out_broadcast_pkts,omitempty"`
	OutMulticastPkts     *uint64    `json:"out_multicast_pkts,omitempty"`
	OutDiscards          *uint64    `json:"out_discards,omitempty"`
	OutErrors            *uint64    `json:"out_errors,omitempty"`
	CarrierTransitions   *uint64    `json:"carrier_transitions,omitempty"`
	LastClear            *time.Time `json:"last_clear,omitempty"`
	InPkts               *uint64    `json:"in_pkts,omitempty"`
	OutPkts              *uint64    `json:"out_pkts,omitempty"`
	InputUtilization     *float64 `json:"input_utilization,omitempty"`     // %
	OutputUtilization    *float64 `json:"output_utilization,omitempty"`    // %
	InTrafficRate        *string `json:"in_traffic_rate,omitempty"`       // Mbps
	InPacketRate         *string `json:"in_packet_rate,omitempty"`        // Kfps
	OutTrafficRate       *string `json:"out_traffic_rate,omitempty"`      // Mbps
	OutPacketRate        *string `json:"out_packet_rate,omitempty"`       // Kfps
	InV4Octets           *uint64 `json:"in_v4_octets,omitempty"`
	OutV4Octets          *uint64 `json:"out_v4_octets,omitempty"`
	InV4Pkts             *uint64 `json:"in_v4_pkts,omitempty"`
	OutV4Pkts            *uint64 `json:"out_v4_pkts,omitempty"`
	InV6Octets           *uint64 `json:"in_v6_octets,omitempty"`
	OutV6Octets          *uint64 `json:"out_v6_octets,omitempty"`
	InV6Pkts             *uint64 `json:"in_v6_pkts,omitempty"`
	OutV6Pkts            *uint64 `json:"out_v6_pkts,omitempty"`
	InV4TrafficRate      *string `json:"in_v4_traffic_rate,omitempty"`    // Mbps
	InV4PacketRate       *string `json:"in_v4_packet_rate,omitempty"`     // Kfps
	OutV4TrafficRate     *string `json:"out_v4_traffic_rate,omitempty"`   // Mbps
	OutV4PacketRate      *string `json:"out_v4_packet_rate,omitempty"`    // Kfps
	InV6TrafficRate      *string `json:"in_v6_traffic_rate,omitempty"`    // Mbps
	InV6PacketRate       *string `json:"in_v6_packet_rate,omitempty"`     // Kfps
	OutV6TrafficRate     *string `json:"out_v6_traffic_rate,omitempty"`   // Mbps
	OutV6PacketRate      *string `json:"out_v6_packet_rate,omitempty"`    // Kfps
	InputV4Utilization   *float64 `json:"input_v4_utilization,omitempty"`  // %
	OutputV4Utilization  *float64 `json:"output_v4_utilization,omitempty"` // %
	InputV6Utilization   *float64 `json:"input_v6_utilization,omitempty"`  // %
	OutputV6Utilization  *float64 `json:"output_v6_utilization,omitempty"` // %
	InBierOctets         *uint64 `json:"in_bier_octets,omitempty"`
	InBierPkts           *uint64 `json:"in_bier_pkts,omitempty"`
	OutBierOctets        *uint64 `json:"out_bier_octets,omitempty"`
	OutBierPkts          *uint64 `json:"out_bier_pkts,omitempty"`
}

// SubinterfaceMetric 子接口指标数据结构
type SubinterfaceMetric struct {
	// 基本信息
	Timestamp        time.Time `json:"timestamp" db:"time"`
	SystemID         string    `json:"system_id" db:"system_id"`
	InterfaceName    string    `json:"interface_name" db:"interface_name"`
	SubinterfaceName string    `json:"subinterface_name" db:"subinterface_index"`
	
	// 子接口状态字段
	Ifindex           *uint32    `json:"ifindex,omitempty" db:"ifindex"`
	AdminStatusStr    *string    `json:"admin_status,omitempty" db:"admin_status"`
	OperStatusStr     *string    `json:"oper_status,omitempty" db:"oper_status"`
	LastChange        *time.Time `json:"last_change,omitempty" db:"last_change"`
	Logical           *bool      `json:"logical,omitempty" db:"logical"`
	IPv4OperStatusStr *string    `json:"ipv4_oper_status,omitempty" db:"ipv4_oper_status"`
	
	// ZTE子接口扩展字段
	ZteifIfindex           *uint32 `json:"zteif_ifindex,omitempty" db:"zteif_ifindex"`
	ZteifAdminStatusStr    *string `json:"zteif_admin_status,omitempty" db:"zteif_admin_status"`
	ZteifOperStatusStr     *string `json:"zteif_oper_status,omitempty" db:"zteif_oper_status"`
	ZteifPhyStatusStr      *string `json:"zteif_phy_status,omitempty" db:"zteif_phy_status"`
	ZteifIPv4OperStatusStr *string `json:"zteif_ipv4_oper_status,omitempty" db:"zteif_ipv4_oper_status"`
	ZteifIPv6OperStatusStr *string `json:"zteif_ipv6_oper_status,omitempty" db:"zteif_ipv6_oper_status"`
	
	// 子接口计数器字段（与接口计数器相同）
	InOctets              *uint64    `json:"in_octets,omitempty" db:"in_octets"`
	InUnicastPkts         *uint64    `json:"in_unicast_pkts,omitempty" db:"in_unicast_pkts"`
	InBroadcastPkts       *uint64    `json:"in_broadcast_pkts,omitempty" db:"in_broadcast_pkts"`
	InMulticastPkts       *uint64    `json:"in_multicast_pkts,omitempty" db:"in_multicast_pkts"`
	InDiscards            *uint64    `json:"in_discards,omitempty" db:"in_discards"`
	InErrors              *uint64    `json:"in_errors,omitempty" db:"in_errors"`
	InUnknownProtos       *uint64    `json:"in_unknown_protos,omitempty" db:"in_unknown_protos"`
	InFcsErrors           *uint64    `json:"in_fcs_errors,omitempty" db:"in_fcs_errors"`
	OutOctets             *uint64    `json:"out_octets,omitempty" db:"out_octets"`
	OutUnicastPkts        *uint64    `json:"out_unicast_pkts,omitempty" db:"out_unicast_pkts"`
	OutBroadcastPkts      *uint64    `json:"out_broadcast_pkts,omitempty" db:"out_broadcast_pkts"`
	OutMulticastPkts      *uint64    `json:"out_multicast_pkts,omitempty" db:"out_multicast_pkts"`
	OutDiscards           *uint64    `json:"out_discards,omitempty" db:"out_discards"`
	OutErrors             *uint64    `json:"out_errors,omitempty" db:"out_errors"`
	CarrierTransitions    *uint64    `json:"carrier_transitions,omitempty" db:"carrier_transitions"`
	LastClear             *time.Time `json:"last_clear,omitempty" db:"last_clear"`
	InPkts                *uint64    `json:"in_pkts,omitempty" db:"in_pkts"`
	OutPkts               *uint64    `json:"out_pkts,omitempty" db:"out_pkts"`
	InputUtilization      *float64    `json:"input_utilization,omitempty" db:"input_utilization"`      // %
	OutputUtilization     *float64    `json:"output_utilization,omitempty" db:"output_utilization"`    // %
	InTrafficRate         *string    `json:"in_traffic_rate,omitempty" db:"in_traffic_rate"`          // Mbps
	InPacketRate          *string    `json:"in_packet_rate,omitempty" db:"in_packet_rate"`            // Kfps
	OutTrafficRate        *string    `json:"out_traffic_rate,omitempty" db:"out_traffic_rate"`        // Mbps
	OutPacketRate         *string    `json:"out_packet_rate,omitempty" db:"out_packet_rate"`          // Kfps
	InV4Octets            *uint64    `json:"in_v4_octets,omitempty" db:"in_v4_octets"`
	OutV4Octets           *uint64    `json:"out_v4_octets,omitempty" db:"out_v4_octets"`
	InV4Pkts              *uint64    `json:"in_v4_pkts,omitempty" db:"in_v4_pkts"`
	OutV4Pkts             *uint64    `json:"out_v4_pkts,omitempty" db:"out_v4_pkts"`
	InV6Octets            *uint64    `json:"in_v6_octets,omitempty" db:"in_v6_octets"`
	OutV6Octets           *uint64    `json:"out_v6_octets,omitempty" db:"out_v6_octets"`
	InV6Pkts              *uint64    `json:"in_v6_pkts,omitempty" db:"in_v6_pkts"`
	OutV6Pkts             *uint64    `json:"out_v6_pkts,omitempty" db:"out_v6_pkts"`
	InV4TrafficRate       *string    `json:"in_v4_traffic_rate,omitempty" db:"in_v4_traffic_rate"`    // Mbps
	InV4PacketRate        *string    `json:"in_v4_packet_rate,omitempty" db:"in_v4_packet_rate"`      // Kfps
	OutV4TrafficRate      *string    `json:"out_v4_traffic_rate,omitempty" db:"out_v4_traffic_rate"`  // Mbps
	OutV4PacketRate       *string    `json:"out_v4_packet_rate,omitempty" db:"out_v4_packet_rate"`    // Kfps
	InV6TrafficRate       *string    `json:"in_v6_traffic_rate,omitempty" db:"in_v6_traffic_rate"`    // Mbps
	InV6PacketRate        *string    `json:"in_v6_packet_rate,omitempty" db:"in_v6_packet_rate"`      // Kfps
	OutV6TrafficRate      *string    `json:"out_v6_traffic_rate,omitempty" db:"out_v6_traffic_rate"`  // Mbps
	OutV6PacketRate       *string    `json:"out_v6_packet_rate,omitempty" db:"out_v6_packet_rate"`    // Kfps
	InputV4Utilization    *float64    `json:"input_v4_utilization,omitempty" db:"input_v4_utilization"`  // %
	OutputV4Utilization   *float64    `json:"output_v4_utilization,omitempty" db:"output_v4_utilization"` // %
	InputV6Utilization    *float64    `json:"input_v6_utilization,omitempty" db:"input_v6_utilization"`  // %
	OutputV6Utilization   *float64    `json:"output_v6_utilization,omitempty" db:"output_v6_utilization"` // %
	InBierOctets          *uint64    `json:"in_bier_octets,omitempty" db:"in_bier_octets"`
	InBierPkts            *uint64    `json:"in_bier_pkts,omitempty" db:"in_bier_pkts"`
	OutBierOctets         *uint64    `json:"out_bier_octets,omitempty" db:"out_bier_octets"`
	OutBierPkts           *uint64    `json:"out_bier_pkts,omitempty" db:"out_bier_pkts"`
}

// 辅助函数：格式化uptime为dd:hh:mm:ss格式
func FormatUptime(seconds uint32) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d:%02d", days, hours, minutes, secs)
}

// 辅助函数：字节转换为MB
func BytesToMB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}

// 辅助函数：纳秒时间戳转换为time.Time
func NanosecondsToTime(ns uint64) time.Time {
	return time.Unix(0, int64(ns))
}

// 辅助函数：纳秒转换为秒
func NanosecondsToSeconds(ns uint64) uint64 {
	return ns / 1000000000
}

// 辅助函数：格式化百分比
func FormatPercentage(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}

// 辅助函数：格式化利用率（从1/10000单位转换为百分比）
func FormatUtilization(value float64) string {
	percentage := value / 10000 * 100
	return fmt.Sprintf("%.2f%%", percentage)
}

// 辅助函数：格式化流量速率
func FormatTrafficRate(value float64) string {
	return fmt.Sprintf("%.2f Mbps", value)
}

// 辅助函数：格式化包速率
func FormatPacketRate(value float64) string {
	return fmt.Sprintf("%.2f Kfps", value)
}

// GetInPkts 获取入方向包数
func (m *SubinterfaceMetric) GetInPkts() uint64 {
	if m.InPkts != nil {
		return *m.InPkts
	}
	return 0
}

// GetOutPkts 获取出方向包数
func (m *SubinterfaceMetric) GetOutPkts() uint64 {
	if m.OutPkts != nil {
		return *m.OutPkts
	}
	return 0
}

// InTrafficRateValue 计算入方向流量速率 (Mbps)
func (m *SubinterfaceMetric) InTrafficRateValue() string {
	if m.InOctets == nil || m.Timestamp.IsZero() {
		return "0"
	}
	
	// 这里应该实现实际的计算逻辑
	// 由于缺少历史数据，暂时返回固定值
	return "0"
}

// InputUtilizationValue 计算入方向利用率 (%)
func (m *SubinterfaceMetric) InputUtilizationValue() string {
	if m.InOctets == nil || m.Timestamp.IsZero() {
		return "0"
	}
	
	// 这里应该实现实际的计算逻辑
	// 由于缺少历史数据和接口带宽信息，暂时返回固定值
	return "0"
}