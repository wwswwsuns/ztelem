package parser

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wwswwsuns/ztelem/internal/models"
	interfaceProto "github.com/wwswwsuns/ztelem/proto/zxr10_interfaces"
	platformProto "github.com/wwswwsuns/ztelem/proto/openconfig_platform"
	zteTelemetry "github.com/wwswwsuns/ztelem/proto/zte_telemetry"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// 枚举转换函数

// convertAlarmStatus 转换AlarmStatus枚举值
func convertAlarmStatus(value int32) string {
	switch value {
	case 0:
		return "INVALID"
	case 1:
		return "NORMAL"
	case 2:
		return "ALARM"
	default:
		return "UNKNOWN_" + strconv.Itoa(int(value))
	}
}

// convertAdminStatus 转换AdminStatus枚举值
func convertAdminStatus(value int32) string {
	switch value {
	case 0:
		return "ADMIN_STATUS_INVALID"
	case 1:
		return "ADMIN_STATUS_UP"
	case 2:
		return "ADMIN_STATUS_DOWN"
	case 3:
		return "ADMIN_STATUS_TESTING"
	default:
		return "ADMIN_STATUS_UNKNOWN_" + strconv.Itoa(int(value))
	}
}

// convertOperStatus 转换OperStatus枚举值
func convertOperStatus(value int32) string {
	switch value {
	case 0:
		return "OPER_STATUS_INVALID"
	case 1:
		return "OPER_STATUS_UP"
	case 2:
		return "OPER_STATUS_DOWN"
	case 3:
		return "OPER_STATUS_TESTING"
	case 4:
		return "OPER_STATUS_UNKNOWN"
	case 5:
		return "OPER_STATUS_DORMANT"
	case 6:
		return "OPER_STATUS_NOT_PRESENT"
	case 7:
		return "OPER_STATUS_LOWER_LAYER_DOWN"
	default:
		return "OPER_STATUS_UNKNOWN_" + strconv.Itoa(int(value))
	}
}

// convertIPv4OperStatus 转换IPv4OperStatus枚举值
func convertIPv4OperStatus(value int32) string {
	switch value {
	case 0:
		return "IPV4OPERSTATUS_STATUS_INVALID"
	case 1:
		return "IPV4OPERSTATUS_STATUS_UP"
	case 2:
		return "IPV4OPERSTATUS_STATUS_DOWN"
	default:
		return "IPV4OPERSTATUS_STATUS_UNKNOWN_" + strconv.Itoa(int(value))
	}
}

// convertIPv6OperStatus 转换IPv6OperStatus枚举值
func convertIPv6OperStatus(value int32) string {
	switch value {
	case 0:
		return "IPV6OPERSTATUS_STATUS_INVALID"
	case 1:
		return "IPV6OPERSTATUS_STATUS_UP"
	case 2:
		return "IPV6OPERSTATUS_STATUS_DOWN"
	default:
		return "IPV6OPERSTATUS_STATUS_UNKNOWN_" + strconv.Itoa(int(value))
	}
}

// convertPhyStatus 转换PhyStatus枚举值
func convertPhyStatus(value int32) string {
	switch value {
	case 0:
		return "PHY_STATUS_INVALID"
	case 1:
		return "PHY_STATUS_UP"
	case 2:
		return "PHY_STATUS_DOWN"
	default:
		return "PHY_STATUS_UNKNOWN_" + strconv.Itoa(int(value))
	}
}

// ParseResult 解析结果
type ParseResult struct {
	SystemID                 string
	SensorPath               string
	Timestamp                time.Time
	PlatformMetrics          []models.PlatformMetric
	InterfaceMetrics         []models.InterfaceMetric
	SubinterfaceMetrics      []models.SubinterfaceMetric
	AlarmReportMetrics       []models.AlarmReportMetric
	NotificationReportMetrics []models.NotificationReportMetric
}

// TelemetryParser telemetry数据解析器
type TelemetryParser struct {
	logger          *logrus.Logger
	telemetryPool   sync.Pool
	componentPool   sync.Pool
	interfacePool   sync.Pool
}

// NewTelemetryParser 创建新的解析器
func NewTelemetryParser(logger *logrus.Logger) *TelemetryParser {
	return &TelemetryParser{
		logger: logger,
		telemetryPool: sync.Pool{
			New: func() interface{} { return new(zteTelemetry.Telemetry) },
		},
		componentPool: sync.Pool{
			New: func() interface{} { return new(platformProto.ComponentInfo) },
		},
		interfacePool: sync.Pool{
			New: func() interface{} { return new(interfaceProto.InterfaceInfo) },
		},
	}
}

// ParseTelemetryData 解析telemetry数据
func (p *TelemetryParser) ParseTelemetryData(data []byte) (*ParseResult, error) {
	// 从 pool 获取 telemetry 消息对象
	telemetryMsg := p.telemetryPool.Get().(*zteTelemetry.Telemetry)
	if err := proto.Unmarshal(data, telemetryMsg); err != nil {
		p.telemetryPool.Put(telemetryMsg)
		return nil, fmt.Errorf("解析telemetry消息失败: %v", err)
	}
	defer func() {
		proto.Reset(telemetryMsg)
		p.telemetryPool.Put(telemetryMsg)
	}()

	p.logger.Debugf("解析到telemetry消息: system_id=%s, sensor_path=%s, data_type=%s", 
		telemetryMsg.SystemId, telemetryMsg.SensorPath, telemetryMsg.DataType.String())
	
	// 特别记录所有接收到的sensor_path和data_type
	if p.logger.Level <= logrus.DebugLevel {
		p.logger.Debugf("📡 接收到sensor_path: %s, data_type: %s (来自设备: %s)", 
			telemetryMsg.SensorPath, telemetryMsg.DataType.String(), telemetryMsg.SystemId)
	}

	result := &ParseResult{
		SystemID:    telemetryMsg.SystemId,
		SensorPath:  telemetryMsg.SensorPath,
		Timestamp:   time.UnixMilli(int64(telemetryMsg.MsgTimestamp)),
	}

	// 首先检查data_type，告警数据优先处理
	if telemetryMsg.DataType == zteTelemetry.TelemetryDataType_ALARM {
		p.logger.Debugf("🚨 检测到告警数据类型: device_id=%s, sensor_path=%s, data_size=%d", 
			telemetryMsg.SystemId, telemetryMsg.SensorPath, len(telemetryMsg.DataGpb))
		
		// 告警数据处理
		alarmMetrics, notificationMetrics, err := p.parseAlarmData(telemetryMsg)
		if err != nil {
			p.logger.WithError(err).Error("解析告警数据失败")
			return nil, err
		}
		
		result.AlarmReportMetrics = alarmMetrics
		result.NotificationReportMetrics = notificationMetrics
		
		p.logger.Debugf("✅ 成功解析告警数据: alarm_reports=%d, notifications=%d", 
			len(alarmMetrics), len(notificationMetrics))
		
		return result, nil
	}

	// sensor_path 路由表 — 按前缀长度降序排列，长前缀优先匹配
	type routeEntry struct {
		prefix  string
		handler func(*zteTelemetry.Telemetry) (interface{}, error)
		exact   bool
	}

	routes := []routeEntry{
		// 精确匹配
		{"oc-platform:components/component", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentsData(m) }, true},
		// 平台组件 — 长前缀优先
		{"oc-platform:components/component/oc-transceiver:transceiver/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentTransceiverState(m) }, false},
		{"oc-platform:components/component/cpu/oc-cpu:utilization/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentCPUState(m) }, false},
		{"oc-platform:components/component/oc-linecard:linecard/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentLinecardState(m) }, false},
		{"oc-platform:components/component/power-supply/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentPowerSupplyState(m) }, false},
		{"oc-platform:components/component/fan/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentFanState(m) }, false},
		{"oc-platform:components/component/state/memory", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentMemoryState(m) }, false},
		{"oc-platform:components/component/state/storage", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentStorageState(m) }, false},
		{"oc-platform:components/component/state/temperature", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentTemperatureState(m) }, false},
		{"oc-platform:components/component/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseComponentState(m) }, false},
		// 子接口
		{"oc-if:interfaces/interface/subinterfaces/subinterface/state/counters", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseSubinterfaceCounters(m) }, false},
		{"oc-if:interfaces/interface/subinterfaces/subinterface/zte-if:state-period", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseSubinterfaceZteState(m) }, false},
		{"oc-if:interfaces/interface/subinterfaces/subinterface/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseSubinterfaceState(m) }, false},
		// 接口
		{"oc-if:interfaces/interface/state/counters", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseInterfaceCounters(m) }, false},
		{"oc-if:interfaces/interface/zte-if:state-period", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseInterfaceZteState(m) }, false},
		{"oc-if:interfaces/interface/state", func(m *zteTelemetry.Telemetry) (interface{}, error) { return p.parseInterfaceState(m) }, false},
	}

	sensorPath := telemetryMsg.SensorPath
	for _, route := range routes {
		if route.exact {
			if sensorPath != route.prefix {
				continue
			}
		} else if !strings.HasPrefix(sensorPath, route.prefix) {
			continue
		}

		val, err := route.handler(telemetryMsg)
		if err != nil {
			return nil, err
		}
		switch v := val.(type) {
		case []models.PlatformMetric:
			result.PlatformMetrics = v
		case []models.InterfaceMetric:
			result.InterfaceMetrics = v
		case []models.SubinterfaceMetric:
			result.SubinterfaceMetrics = v
		}
		return result, nil
	}

	p.logger.Warnf("未知的sensor_path: %s", telemetryMsg.SensorPath)

	return result, nil
}

// parseComponentState 解析组件通用状态数据
func (p *TelemetryParser) parseComponentState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric
	
	// 遍历所有DataGpb条目，每个可能包含一个或多个组件的信息
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 解析CommonState数据
		if commonState := componentInfo.GetCommonState(); commonState != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				CommonState:   &models.CommonState{},
			}

			metric.OperStatus = stringPtr(commonState.GetOperStatus())
			metric.Uptime = stringPtr(formatUptime(commonState.GetUptime()))
			metric.UsedPower = uint32Ptr(commonState.GetUsedPower())
			metric.AllocatedPower = uint32Ptr(commonState.GetAllocatedPower())
			metric.CurrentVoltage = stringPtr(commonState.GetCurrentVoltage())
			metric.CurrentCurrent = stringPtr(commonState.GetCurrentCurrent())
			metric.TotalCapacity = stringPtr(commonState.GetTotalCapacity())
			metric.UsedCapacity = stringPtr(commonState.GetUsedCapacity())
			metric.Type = stringPtr(commonState.GetType())
			metric.RedundancyType = stringPtr(commonState.GetRedundancyType())
			metric.Modules = stringPtr(commonState.GetModules())
			metric.TotalInputPower = stringPtr(commonState.GetTotalInputPower())
			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentsData 解析组件综合数据 (处理oc-platform:components/component路径)
// 这个函数处理包含多个组件信息的数据，包括CPU、内存等
func (p *TelemetryParser) parseComponentsData(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}

	var metrics []models.PlatformMetric
	
	// 遍历所有DataGpb条目，每个可能包含一个或多个组件的信息
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 创建单个组件的完整指标记录
		metric := models.PlatformMetric{
			Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
			SystemID:      msg.SystemId,
			ComponentName: componentInfo.GetName(),
			CommonState:   &models.CommonState{},
		}

		// 解析通用状态信息
		if commonState := componentInfo.GetCommonState(); commonState != nil {
			metric.OperStatus = stringPtr(commonState.GetOperStatus())
			metric.Uptime = stringPtr(formatUptime(commonState.GetUptime()))
			metric.UsedPower = uint32Ptr(commonState.GetUsedPower())
			metric.AllocatedPower = uint32Ptr(commonState.GetAllocatedPower())
			metric.CurrentVoltage = stringPtr(commonState.GetCurrentVoltage())
			metric.CurrentCurrent = stringPtr(commonState.GetCurrentCurrent())
			metric.TotalCapacity = stringPtr(commonState.GetTotalCapacity())
			metric.UsedCapacity = stringPtr(commonState.GetUsedCapacity())
			metric.Type = stringPtr(commonState.GetType())
			metric.RedundancyType = stringPtr(commonState.GetRedundancyType())
			metric.Modules = stringPtr(commonState.GetModules())
			metric.TotalInputPower = stringPtr(commonState.GetTotalInputPower())
		}

		// 解析CPU信息
		if cpuInfo := componentInfo.GetCpuInfo(); cpuInfo != nil {
			metric.CPUData = &models.CPUData{}
			metric.CPUInstant = float64Ptr(float64(cpuInfo.GetInstant()))
			metric.CPUAvg = float64Ptr(float64(cpuInfo.GetAvg()))
			metric.CPUMin = float64Ptr(float64(cpuInfo.GetMin()))
			metric.CPUMax = float64Ptr(float64(cpuInfo.GetMax()))
			metric.CPUInterval = uint64Ptr(nanosToSeconds(cpuInfo.GetInterval()))
			metric.CPUMinTime = timePtr(nanosToTimestamp(cpuInfo.GetMinTime()))
			metric.CPUMaxTime = timePtr(nanosToTimestamp(cpuInfo.GetMaxTime()))
			
			if cpuInfo.GetAlarmStatus() != 0 {
				alarmStatusStr := convertAlarmStatus(int32(cpuInfo.GetAlarmStatus()))
				metric.CPUAlarmStatus = &alarmStatusStr
			}
		}

		// 解析内存信息
		if memInfo := componentInfo.GetMemInfo(); memInfo != nil {
			metric.MemData = &models.MemData{}
			metric.MemAvailable = uint64Ptr(uint64(bytesToMB(memInfo.GetAvailable())))
			metric.MemUtilized = uint64Ptr(uint64(bytesToMB(memInfo.GetUtilized())))
			metric.MemFree = uint64Ptr(uint64(bytesToMB(memInfo.GetFree())))
			metric.MemUsage = float64Ptr(float64(memInfo.GetUsage()))
			
			// 使用枚举转换函数
			alarmStatusStr := convertAlarmStatus(int32(memInfo.GetAlarmStatus()))
			metric.MemAlarmStatus = &alarmStatusStr
		}

		// 解析温度信息
		if tempInfo := componentInfo.GetTempInfo(); tempInfo != nil {
			metric.TempData = &models.TempData{}
			metric.TempInstant = float64Ptr(float64(tempInfo.GetInstant()))
			metric.TempAvg = float64Ptr(float64(tempInfo.GetAvg()))
			metric.TempMin = float64Ptr(float64(tempInfo.GetMin()))
			metric.TempMax = float64Ptr(float64(tempInfo.GetMax()))
			metric.TempInterval = uint64Ptr(nanosToSeconds(tempInfo.GetInterval()))
			metric.TempMinTime = timePtr(nanosToTimestamp(tempInfo.GetMinTime()))
			metric.TempMaxTime = timePtr(nanosToTimestamp(tempInfo.GetMaxTime()))
			metric.AlarmStatus = boolPtr(tempInfo.GetAlarmStatus())
			metric.TempAlarmThreshold = float64Ptr(float64(tempInfo.GetAlarmThreshold()))
			metric.TempAlarmSeverity = stringPtr(tempInfo.GetAlarmSeverity())
			metric.TempMinorThreshold = float64Ptr(float64(tempInfo.GetMinorThreshold()))
			metric.TempMajorThreshold = float64Ptr(float64(tempInfo.GetMajorThreshold()))
			metric.TempFatalThreshold = float64Ptr(float64(tempInfo.GetFatalThreshold()))
			metric.TempInstantString = stringPtr(tempInfo.GetInstantString())
			metric.TempStatus = stringPtr(tempInfo.GetStatus())
			metric.TempDescription = stringPtr(tempInfo.GetDescription())
		}

		// 解析风扇信息
		if fanInfo := componentInfo.GetFanInfo(); fanInfo != nil {
			metric.FanData = &models.FanData{}
			metric.FanSpeed = uint32Ptr(fanInfo.GetSpeed())
			metric.FanState = stringPtr(fanInfo.GetState())
			metric.FanPhyStatus = stringPtr(fanInfo.GetPhyStatus())
			metric.FanWorkMode = stringPtr(fanInfo.GetWorkMode())
			metric.FanCurrentPower = stringPtr(fanInfo.GetCurrentPower())
			metric.FanCurrentVoltage = stringPtr(fanInfo.GetCurrentVoltage())
			metric.FanCurrentCurrent = stringPtr(fanInfo.GetCurrentCurrent())
			metric.FanSpeedPercent = stringPtr(fanInfo.GetSpeedPercent())
		}

		// 解析电源信息
		if powerInfo := componentInfo.GetPowerInfo(); powerInfo != nil {
			metric.PowerData = &models.PowerData{}
			metric.PowerEnable = boolPtr(powerInfo.GetEnable())
			metric.PowerCapacity = float64Ptr(float64(powerInfo.GetCapacity()))
			metric.PowerInputCurrent = float64Ptr(float64(powerInfo.GetInputCurrent()))
			metric.PowerInputVoltage = float64Ptr(float64(powerInfo.GetInputVoltage()))
			metric.PowerOutputCurrent = float64Ptr(float64(powerInfo.GetOutputCurrent()))
			metric.PowerOutputVoltage = float64Ptr(float64(powerInfo.GetOutputVoltage()))
			metric.PowerOutputPower = float64Ptr(float64(powerInfo.GetOutputPower()))
			metric.PowerWorkState = stringPtr(powerInfo.GetWorkState())
			metric.PowerPhyState = stringPtr(powerInfo.GetPhyState())
			metric.PowerComState = stringPtr(powerInfo.GetComState())
			metric.PowerTemperature = stringPtr(powerInfo.GetTemperature())
			metric.PowerAvailable = stringPtr(powerInfo.GetAvailable())
			metric.PowerCapacityString = stringPtr(powerInfo.GetCapacityString())
			metric.PowerInputPower = stringPtr(powerInfo.GetInputPower())
			metric.PowerInput2Current = float64Ptr(float64(powerInfo.GetInput2Current()))
			metric.PowerInput2Voltage = float64Ptr(float64(powerInfo.GetInput2Voltage()))
			metric.PowerOutput2Current = float64Ptr(float64(powerInfo.GetOutput2Current()))
			metric.PowerOutput2Voltage = float64Ptr(float64(powerInfo.GetOutput2Voltage()))
		}

		// 解析存储信息
		if storageInfo := componentInfo.GetStorageInfo(); storageInfo != nil {
			if metric.MemData == nil {
				metric.MemData = &models.MemData{}
			}
			metric.MemData.StorageAvailability = float64Ptr(float64(storageInfo.GetAvailability()))
		}

		// 解析光模块信息
		if opticalInfo := componentInfo.GetOpticalInfo(); opticalInfo != nil {
			metric.OpticalData = &models.OpticalData{}
			if inPower := opticalInfo.GetInPower(); inPower != nil {
				metric.OpticalInPower = opticalPowerPtr(float64(inPower.GetInstant()), true)
			} else {
				metric.OpticalInPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
			}
			if outPower := opticalInfo.GetOutPower(); outPower != nil {
				metric.OpticalOutPower = opticalPowerPtr(float64(outPower.GetInstant()), true)
			} else {
				metric.OpticalOutPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
			}
			if biasCurrent := opticalInfo.GetBiasCurrent(); biasCurrent != nil {
				metric.OpticalBiasCurrent = float64Ptr(float64(biasCurrent.GetInstant()))
			}
			if temperature := opticalInfo.GetTemperature(); temperature != nil {
				metric.OpticalTemperature = float64Ptr(float64(temperature.GetInstant()))
			}
			if voltage := opticalInfo.GetVoltage(); voltage != nil {
				metric.OpticalVoltageVol33 = float64Ptr(float64(voltage.GetVol33()))
				metric.OpticalVoltageVol5 = float64Ptr(float64(voltage.GetVol5()))
			}
			
			if opticalAlarm := opticalInfo.GetAlarm(); opticalAlarm != nil {
				if opticalAlarm.GetLosStatus() != 0 {
					alarmStatusStr := convertAlarmStatus(int32(opticalAlarm.GetLosStatus()))
					metric.OpticalAlarmLosStatus = &alarmStatusStr
				}
				
				if losInfo := opticalAlarm.GetLosInfo(); losInfo != nil {
					metric.OpticalAlarmLosInfoEventID = uint32Ptr(losInfo.GetEventId())
					metric.OpticalAlarmLosInfoEventInterval = uint32Ptr(losInfo.GetEventInterval())
					
					if len(losInfo.GetOptInPower()) > 0 {
						metric.OpticalAlarmLosInfoInPower = opticalPowerPtr(float64(losInfo.GetOptInPower()[0].GetInstant()), true)
					} else {
						metric.OpticalAlarmLosInfoInPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
					}
					if len(losInfo.GetOptOutPower()) > 0 {
						metric.OpticalAlarmLosInfoOutPower = opticalPowerPtr(float64(losInfo.GetOptOutPower()[0].GetInstant()), true)
					} else {
						metric.OpticalAlarmLosInfoOutPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
					}
				}
			}
			
			if onlineStatus := opticalInfo.GetOnlineStatus(); onlineStatus != nil {
				metric.OpticalOnlineStatus = stringPtr(onlineStatus.GetOnlineStatus())
			}
			
			if rxThreshold := opticalInfo.GetRxThreshold(); rxThreshold != nil {
				metric.OpticalRxThresholdHighAlarm = float64Ptr(float64(rxThreshold.GetHighAlarm()))
				metric.OpticalRxThresholdPreHighAlarm = float64Ptr(float64(rxThreshold.GetPreHighAlarm()))
				metric.OpticalRxThresholdLowAlarm = float64Ptr(float64(rxThreshold.GetLowAlarm()))
				metric.OpticalRxThresholdPreLowAlarm = float64Ptr(float64(rxThreshold.GetPreLowAlarm()))
			}
		}

		// 解析线卡信息
		if linecardInfo := componentInfo.GetPowerAdminState(); linecardInfo != nil {
			if metric.PowerData == nil {
				metric.PowerData = &models.PowerData{}
			}
			metric.PowerData.LinecardPowerAdminState = stringPtr(linecardInfo.GetPowerAdminState())
		}

		// 将完整的组件指标添加到结果中
		metrics = append(metrics, metric)

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentFanState 解析组件风扇状态数据
func (p *TelemetryParser) parseComponentFanState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目，每个可能包含一个或多个组件的信息
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 解析FanInfo数据
		if fanInfo := componentInfo.GetFanInfo(); fanInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				FanData:       &models.FanData{},
			}

			metric.FanSpeed = uint32Ptr(fanInfo.GetSpeed())
			metric.FanState = stringPtr(fanInfo.GetState())
			metric.FanPhyStatus = stringPtr(fanInfo.GetPhyStatus())
			metric.FanWorkMode = stringPtr(fanInfo.GetWorkMode())
			metric.FanCurrentPower = stringPtr(fanInfo.GetCurrentPower())
			metric.FanCurrentVoltage = stringPtr(fanInfo.GetCurrentVoltage())
			metric.FanCurrentCurrent = stringPtr(fanInfo.GetCurrentCurrent())
			metric.FanSpeedPercent = stringPtr(fanInfo.GetSpeedPercent())

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentMemoryState 解析组件内存状态数据
func (p *TelemetryParser) parseComponentMemoryState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目，每个可能包含一个或多个组件的信息
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 解析MemoryInfo数据
		if memInfo := componentInfo.GetMemInfo(); memInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				MemData:       &models.MemData{},
			}

			metric.MemAvailable = uint64Ptr(uint64(bytesToMB(memInfo.GetAvailable())))
			metric.MemUtilized = uint64Ptr(uint64(bytesToMB(memInfo.GetUtilized())))
			metric.MemFree = uint64Ptr(uint64(bytesToMB(memInfo.GetFree())))
			metric.MemUsage = float64Ptr(float64(memInfo.GetUsage()))
			
			if memInfo.GetAlarmStatus() != 0 {
				alarmStatusStr := convertAlarmStatus(int32(memInfo.GetAlarmStatus()))
				metric.MemAlarmStatus = &alarmStatusStr
			}

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentStorageState 解析组件存储状态数据
func (p *TelemetryParser) parseComponentStorageState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析StorageInfo数据
		if storageInfo := componentInfo.GetStorageInfo(); storageInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				MemData:       &models.MemData{},
			}

			metric.MemData.StorageAvailability = float64Ptr(float64(storageInfo.GetAvailability()))

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentTemperatureState 解析组件温度状态数据
func (p *TelemetryParser) parseComponentTemperatureState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析Temperature数据
		if tempInfo := componentInfo.GetTempInfo(); tempInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				TempData:      &models.TempData{},
			}

			metric.TempInstant = float64Ptr(float64(tempInfo.GetInstant()))
			metric.TempAvg = float64Ptr(float64(tempInfo.GetAvg()))
			metric.TempMin = float64Ptr(float64(tempInfo.GetMin()))
			metric.TempMax = float64Ptr(float64(tempInfo.GetMax()))
			metric.TempInterval = uint64Ptr(tempInfo.GetInterval() / 1e9) // 纳秒转秒
			metric.TempMinTime = timePtr(nanosToTimestamp(tempInfo.GetMinTime()))
			metric.TempMaxTime = timePtr(nanosToTimestamp(tempInfo.GetMaxTime()))
			metric.AlarmStatus = boolPtr(tempInfo.GetAlarmStatus())
			metric.TempAlarmThreshold = float64Ptr(float64(tempInfo.GetAlarmThreshold()))
			metric.TempAlarmSeverity = stringPtr(tempInfo.GetAlarmSeverity())
			metric.TempMinorThreshold = float64Ptr(float64(tempInfo.GetMinorThreshold()))
			metric.TempMajorThreshold = float64Ptr(float64(tempInfo.GetMajorThreshold()))
			metric.TempFatalThreshold = float64Ptr(float64(tempInfo.GetFatalThreshold()))
			metric.TempInstantString = stringPtr(tempInfo.GetInstantString())
			metric.TempStatus = stringPtr(tempInfo.GetStatus())
			metric.TempDescription = stringPtr(tempInfo.GetDescription())

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentPowerSupplyState 解析组件电源状态数据
func (p *TelemetryParser) parseComponentPowerSupplyState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析PowerInfo数据
		if powerInfo := componentInfo.GetPowerInfo(); powerInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				PowerData:     &models.PowerData{},
			}

			metric.PowerEnable = boolPtr(powerInfo.GetEnable())
			metric.PowerCapacity = float64Ptr(float64(powerInfo.GetCapacity()))
			metric.PowerInputCurrent = float64Ptr(float64(powerInfo.GetInputCurrent()))
			metric.PowerInputVoltage = float64Ptr(float64(powerInfo.GetInputVoltage()))
			metric.PowerOutputCurrent = float64Ptr(float64(powerInfo.GetOutputCurrent()))
			metric.PowerOutputVoltage = float64Ptr(float64(powerInfo.GetOutputVoltage()))
			metric.PowerOutputPower = float64Ptr(float64(powerInfo.GetOutputPower()))
			metric.PowerWorkState = stringPtr(powerInfo.GetWorkState())
			metric.PowerName = stringPtr(powerInfo.GetPowerName())
			metric.PowerPhyState = stringPtr(powerInfo.GetPhyState())
			metric.PowerState = stringPtr(powerInfo.GetPowerState())
			metric.PowerComState = stringPtr(powerInfo.GetComState())
			metric.PowerTemperature = stringPtr(powerInfo.GetTemperature())
			metric.PowerAvailable = stringPtr(powerInfo.GetAvailable())
			metric.PowerCapacityString = stringPtr(powerInfo.GetCapacityString())
			metric.PowerInputPower = stringPtr(powerInfo.GetInputPower())
			metric.PowerInput2Current = float64Ptr(float64(powerInfo.GetInput2Current()))
			metric.PowerInput2Voltage = float64Ptr(float64(powerInfo.GetInput2Voltage()))
			metric.PowerOutput2Current = float64Ptr(float64(powerInfo.GetOutput2Current()))
			metric.PowerOutput2Voltage = float64Ptr(float64(powerInfo.GetOutput2Voltage()))

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentLinecardState 解析组件线卡状态数据
func (p *TelemetryParser) parseComponentLinecardState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析LinecardInfo数据
		if linecardInfo := componentInfo.GetPowerAdminState(); linecardInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				PowerData:     &models.PowerData{},
			}

			metric.LinecardPowerAdminState = stringPtr(linecardInfo.GetPowerAdminState())

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentCPUState 解析组件CPU状态数据
func (p *TelemetryParser) parseComponentCPUState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		if cpuInfo := componentInfo.GetCpuInfo(); cpuInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				CPUData:       &models.CPUData{},
			}

			metric.CPUInstant = float64Ptr(float64(cpuInfo.GetInstant()))
			metric.CPUAvg = float64Ptr(float64(cpuInfo.GetAvg()))
			metric.CPUMin = float64Ptr(float64(cpuInfo.GetMin()))
			metric.CPUMax = float64Ptr(float64(cpuInfo.GetMax()))
			metric.CPUInterval = uint64Ptr(cpuInfo.GetInterval() / 1e9) // 纳秒转秒
			metric.CPUMinTime = timePtr(nanosToTimestamp(cpuInfo.GetMinTime()))
			metric.CPUMaxTime = timePtr(nanosToTimestamp(cpuInfo.GetMaxTime()))
			
			// 使用枚举转换函数
			alarmStatusStr := convertAlarmStatus(int32(cpuInfo.GetAlarmStatus()))
			metric.CPUAlarmStatus = &alarmStatusStr

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseComponentTransceiverState 解析组件光模块状态数据
func (p *TelemetryParser) parseComponentTransceiverState(msg *zteTelemetry.Telemetry) ([]models.PlatformMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.PlatformMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
			p.componentPool.Put(componentInfo)
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析OpticalInfo数据
		if opticalInfo := componentInfo.GetOpticalInfo(); opticalInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
				OpticalData:   &models.OpticalData{},
			}

			if inPower := opticalInfo.GetInPower(); inPower != nil {
				metric.OpticalInPower = opticalPowerPtr(float64(inPower.GetInstant()), true)
			} else {
				metric.OpticalInPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
			}
			if outPower := opticalInfo.GetOutPower(); outPower != nil {
				metric.OpticalOutPower = opticalPowerPtr(float64(outPower.GetInstant()), true)
			} else {
				metric.OpticalOutPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
			}
			if biasCurrent := opticalInfo.GetBiasCurrent(); biasCurrent != nil {
				metric.OpticalBiasCurrent = float64Ptr(float64(biasCurrent.GetInstant()))
			}
			if temperature := opticalInfo.GetTemperature(); temperature != nil {
				metric.OpticalTemperature = float64Ptr(float64(temperature.GetInstant()))
			}
			if voltage := opticalInfo.GetVoltage(); voltage != nil {
				metric.OpticalVoltageVol33 = float64Ptr(float64(voltage.GetVol33()))
				metric.OpticalVoltageVol5 = float64Ptr(float64(voltage.GetVol5()))
			}

			// 解析告警数据
			if alarm := opticalInfo.GetAlarm(); alarm != nil {
				if alarm.GetLosStatus() != 0 {
					alarmStatusStr := convertAlarmStatus(int32(alarm.GetLosStatus()))
					metric.OpticalAlarmLosStatus = &alarmStatusStr
				}
				
				if alarmInfo := alarm.GetLosInfo(); alarmInfo != nil {
					metric.OpticalAlarmLosInfoEventID = uint32Ptr(alarmInfo.GetEventId())
					metric.OpticalAlarmLosInfoEventInterval = uint32Ptr(alarmInfo.GetEventInterval())
					
					if inPowers := alarmInfo.GetOptInPower(); len(inPowers) > 0 {
						metric.OpticalAlarmLosInfoInPower = opticalPowerPtr(float64(inPowers[0].GetInstant()), true)
					} else {
						metric.OpticalAlarmLosInfoInPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
					}
					if outPowers := alarmInfo.GetOptOutPower(); len(outPowers) > 0 {
						metric.OpticalAlarmLosInfoOutPower = opticalPowerPtr(float64(outPowers[0].GetInstant()), true)
					} else {
						metric.OpticalAlarmLosInfoOutPower = opticalPowerPtr(0, false) // 无光功率数据，设为-60
					}
				}
			}

			if onlineStatus := opticalInfo.GetOnlineStatus(); onlineStatus != nil {
				metric.OpticalOnlineStatus = stringPtr(onlineStatus.GetOnlineStatus())
			}

			if rxThreshold := opticalInfo.GetRxThreshold(); rxThreshold != nil {
				metric.OpticalRxThresholdHighAlarm = float64Ptr(float64(rxThreshold.GetHighAlarm()))
				metric.OpticalRxThresholdPreHighAlarm = float64Ptr(float64(rxThreshold.GetPreHighAlarm()))
				metric.OpticalRxThresholdLowAlarm = float64Ptr(float64(rxThreshold.GetLowAlarm()))
				metric.OpticalRxThresholdPreLowAlarm = float64Ptr(float64(rxThreshold.GetPreLowAlarm()))
			}

			metrics = append(metrics, metric)
		}

		proto.Reset(componentInfo)
		p.componentPool.Put(componentInfo)
	}

	return metrics, nil
}

// parseInterfaceState 解析接口状态数据
func (p *TelemetryParser) parseInterfaceState(msg *zteTelemetry.Telemetry) ([]models.InterfaceMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.InterfaceMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		interfaceInfo := p.interfacePool.Get().(*interfaceProto.InterfaceInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), interfaceInfo); err != nil {
			p.interfacePool.Put(interfaceInfo)
			return nil, fmt.Errorf("解析接口信息失败: %v", err)
		}

		// 解析InterfaceState数据 (GetState返回数组，取第一个)
		if states := interfaceInfo.GetState(); len(states) > 0 {
			state := states[0]
			metric := models.InterfaceMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				InterfaceName: interfaceInfo.GetName(),
			}

			metric.Ifindex = uint32Ptr(state.GetIfindex())
			
			// 将枚举值转换为字符串
			adminStatusStr := convertAdminStatus(int32(state.GetAdminStatus()))
			metric.AdminStatusStr = &adminStatusStr
			
			operStatusStr := convertOperStatus(int32(state.GetOperStatus()))
			metric.OperStatusStr = &operStatusStr
			
			metric.LastChange = timePtr(nanosToTimestamp(state.GetLastChange()))
			metric.Logical = boolPtr(state.GetLogical())
			metric.Type = uint32Ptr(state.GetType())
			
			phyStatusStr := convertPhyStatus(int32(state.GetPhyStatus()))
			metric.PhyStatusStr = &phyStatusStr
			
			ipv4OperStatusStr := convertIPv4OperStatus(int32(state.GetIpv4OperStatus()))
			metric.IPv4OperStatusStr = &ipv4OperStatusStr

			metrics = append(metrics, metric)
		}

		proto.Reset(interfaceInfo)
		p.interfacePool.Put(interfaceInfo)
	}

	return metrics, nil
}

// parseInterfaceZteState 解析接口ZTE扩展状态数据
func (p *TelemetryParser) parseInterfaceZteState(msg *zteTelemetry.Telemetry) ([]models.InterfaceMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.InterfaceMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		interfaceInfo := p.interfacePool.Get().(*interfaceProto.InterfaceInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), interfaceInfo); err != nil {
			p.interfacePool.Put(interfaceInfo)
			return nil, fmt.Errorf("解析接口信息失败: %v", err)
		}

		// 解析ZTE扩展状态数据
		if zteStates := interfaceInfo.GetStatePeriod(); len(zteStates) > 0 {
			zteState := zteStates[0]
			metric := models.InterfaceMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				InterfaceName: interfaceInfo.GetName(),
			}

			metric.ZteifType = uint32Ptr(zteState.GetType())
			metric.ZteifIfindex = uint32Ptr(zteState.GetIfindex())
			
			// 将枚举值转换为字符串
				zteifAdminStatusStr := convertAdminStatus(int32(zteState.GetAdminStatus()))
				metric.ZteifAdminStatusStr = &zteifAdminStatusStr
				
				zteifOperStatusStr := convertOperStatus(int32(zteState.GetOperStatus()))
				metric.ZteifOperStatusStr = &zteifOperStatusStr
				
				zteifPhyStatusStr := convertPhyStatus(int32(zteState.GetPhyStatus()))
				metric.ZteifPhyStatusStr = &zteifPhyStatusStr
				
				zteifIPv4OperStatusStr := convertIPv4OperStatus(int32(zteState.GetIpv4OperStatus()))
				metric.ZteifIPv4OperStatusStr = &zteifIPv4OperStatusStr
				
				zteifIPv6OperStatusStr := convertIPv6OperStatus(int32(zteState.GetIpv6OperStatus()))
				metric.ZteifIPv6OperStatusStr = &zteifIPv6OperStatusStr

			metrics = append(metrics, metric)
		}

		proto.Reset(interfaceInfo)
		p.interfacePool.Put(interfaceInfo)
	}

	return metrics, nil
}

// parseInterfaceCounters 解析接口计数器数据
func (p *TelemetryParser) parseInterfaceCounters(msg *zteTelemetry.Telemetry) ([]models.InterfaceMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.InterfaceMetric

	// 遍历所有DataGpb条目，每个可能包含一个或多个接口的信息
	for _, dataGpb := range msg.DataGpb {
		interfaceInfo := p.interfacePool.Get().(*interfaceProto.InterfaceInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), interfaceInfo); err != nil {
			p.interfacePool.Put(interfaceInfo)
			p.logger.Warnf("解析接口信息失败: %v", err)
			continue
		}

		// 解析Counters数据 (GetCounters返回数组，取第一个)
		if countersList := interfaceInfo.GetCounters(); len(countersList) > 0 {
			counters := countersList[0]
			metric := models.InterfaceMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				InterfaceName: interfaceInfo.GetName(),
			}

			metric.InOctets = uint64Ptr(counters.GetInOctets())
			metric.InUnicastPkts = uint64Ptr(counters.GetInUnicastPkts())
			metric.InBroadcastPkts = uint64Ptr(counters.GetInBroadcastPkts())
			metric.InMulticastPkts = uint64Ptr(counters.GetInMulticastPkts())
			metric.InDiscards = uint64Ptr(counters.GetInDiscards())
			metric.InErrors = uint64Ptr(counters.GetInErrors())
			metric.InUnknownProtos = uint64Ptr(counters.GetInUnknownProtos())
			metric.InFcsErrors = uint64Ptr(counters.GetInFcsErrors())
			metric.OutOctets = uint64Ptr(counters.GetOutOctets())
			metric.OutUnicastPkts = uint64Ptr(counters.GetOutUnicastPkts())
			metric.OutBroadcastPkts = uint64Ptr(counters.GetOutBroadcastPkts())
			metric.OutMulticastPkts = uint64Ptr(counters.GetOutMulticastPkts())
			metric.OutDiscards = uint64Ptr(counters.GetOutDiscards())
			metric.OutErrors = uint64Ptr(counters.GetOutErrors())
			metric.CarrierTransitions = uint64Ptr(counters.GetCarrierTransitions())
			metric.LastClear = timePtr(nanosToTimestamp(counters.GetLastClear()))
			metric.InPkts = uint64Ptr(counters.GetInPkts())
			metric.OutPkts = uint64Ptr(counters.GetOutPkts())
			metric.InputUtilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetInputUtilization())))
			metric.OutputUtilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetOutputUtilization())))
			metric.InTrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetInTrafficRate()))
			metric.InPacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetInPacketRate()))
			metric.OutTrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetOutTrafficRate()))
			metric.OutPacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetOutPacketRate()))
			metric.InV4Octets = uint64Ptr(counters.GetInV4Octets())
			metric.OutV4Octets = uint64Ptr(counters.GetOutV4Octets())
			metric.InV4Pkts = uint64Ptr(counters.GetInV4Pkts())
			metric.OutV4Pkts = uint64Ptr(counters.GetOutV4Pkts())
			metric.InV6Octets = uint64Ptr(counters.GetInV6Octets())
			metric.OutV6Octets = uint64Ptr(counters.GetOutV6Octets())
			metric.InV6Pkts = uint64Ptr(counters.GetInV6Pkts())
			metric.OutV6Pkts = uint64Ptr(counters.GetOutV6Pkts())
			metric.InV4TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetInV4TrafficRate()))
			metric.InV4PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetInV4PacketRate()))
			metric.OutV4TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetOutV4TrafficRate()))
			metric.OutV4PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetOutV4PacketRate()))
			metric.InV6TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetInV6TrafficRate()))
			metric.InV6PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetInV6PacketRate()))
			metric.OutV6TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetOutV6TrafficRate()))
			metric.OutV6PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetOutV6PacketRate()))
			metric.InputV4Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetInputV4Utilization())))
			metric.OutputV4Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetOutputV4Utilization())))
			metric.InputV6Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetInputV6Utilization())))
			metric.OutputV6Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetOutputV6Utilization())))
			metric.InBierOctets = uint64Ptr(counters.GetInBierOctets())
			metric.InBierPkts = uint64Ptr(counters.GetInBierPkts())
			metric.OutBierOctets = uint64Ptr(counters.GetOutBierOctets())
			metric.OutBierPkts = uint64Ptr(counters.GetOutBierPkts())

			metrics = append(metrics, metric)
		}

		proto.Reset(interfaceInfo)
		p.interfacePool.Put(interfaceInfo)
	}

	return metrics, nil
}

// parseSubinterfaceState 解析子接口状态数据
func (p *TelemetryParser) parseSubinterfaceState(msg *zteTelemetry.Telemetry) ([]models.SubinterfaceMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.SubinterfaceMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		interfaceInfo := p.interfacePool.Get().(*interfaceProto.InterfaceInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), interfaceInfo); err != nil {
			p.interfacePool.Put(interfaceInfo)
			return nil, fmt.Errorf("解析接口信息失败: %v", err)
		}

		// 遍历子接口
		for _, subintf := range interfaceInfo.GetSubinterface() {
			// 创建基础metric
			metric := models.SubinterfaceMetric{
		Timestamp:        time.UnixMilli(int64(msg.MsgTimestamp)),
		SystemID:         msg.SystemId,
		InterfaceName:    interfaceInfo.GetName(),
		SubinterfaceName: fmt.Sprintf("%d", subintf.GetSubPort()),
	}
			
			// 解析子接口状态数据 (GetState返回数组，取第一个)
			if states := subintf.GetState(); len(states) > 0 {
				state := states[0]
				metric.Ifindex = uint32Ptr(state.GetIfindex())
				
				// 将枚举值转换为字符串
				adminStatusStr := convertAdminStatus(int32(state.GetAdminStatus()))
				metric.AdminStatusStr = &adminStatusStr
				
				operStatusStr := convertOperStatus(int32(state.GetOperStatus()))
				metric.OperStatusStr = &operStatusStr
				
				metric.LastChange = timePtr(nanosToTimestamp(state.GetLastChange()))
				metric.Logical = boolPtr(state.GetLogical())
				
				ipv4OperStatusStr := convertIPv4OperStatus(int32(state.GetIpv4OperStatus()))
				metric.IPv4OperStatusStr = &ipv4OperStatusStr
			}

			metrics = append(metrics, metric)
		}

		proto.Reset(interfaceInfo)
		p.interfacePool.Put(interfaceInfo)
	}

	return metrics, nil
}

// parseSubinterfaceZteState 解析子接口ZTE扩展状态数据
func (p *TelemetryParser) parseSubinterfaceZteState(msg *zteTelemetry.Telemetry) ([]models.SubinterfaceMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.SubinterfaceMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		interfaceInfo := p.interfacePool.Get().(*interfaceProto.InterfaceInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), interfaceInfo); err != nil {
			p.interfacePool.Put(interfaceInfo)
			return nil, fmt.Errorf("解析接口信息失败: %v", err)
		}

		// 遍历子接口
		for _, subintf := range interfaceInfo.GetSubinterface() {
			// 创建基础metric
			metric := models.SubinterfaceMetric{
		Timestamp:        time.UnixMilli(int64(msg.MsgTimestamp)),
		SystemID:         msg.SystemId,
		InterfaceName:    interfaceInfo.GetName(),
		SubinterfaceName: fmt.Sprintf("%d", subintf.GetSubPort()),
	}
			
			// 解析ZTE扩展状态数据
			if zteStates := subintf.GetSubStatePeriod(); len(zteStates) > 0 {
				zteState := zteStates[0]
				metric.ZteifIfindex = uint32Ptr(zteState.GetIfindex())
				
				// 将枚举值转换为字符串
				zteifAdminStatusStr := convertAdminStatus(int32(zteState.GetAdminStatus()))
				metric.ZteifAdminStatusStr = &zteifAdminStatusStr
				
				zteifOperStatusStr := convertOperStatus(int32(zteState.GetOperStatus()))
				metric.ZteifOperStatusStr = &zteifOperStatusStr
				
				zteifPhyStatusStr := convertPhyStatus(int32(zteState.GetPhyStatus()))
				metric.ZteifPhyStatusStr = &zteifPhyStatusStr
				
				zteifIPv4OperStatusStr := convertIPv4OperStatus(int32(zteState.GetIpv4OperStatus()))
				metric.ZteifIPv4OperStatusStr = &zteifIPv4OperStatusStr
				
				zteifIPv6OperStatusStr := convertIPv6OperStatus(int32(zteState.GetIpv6OperStatus()))
				metric.ZteifIPv6OperStatusStr = &zteifIPv6OperStatusStr
			}
		metrics = append(metrics, metric)
		}

		proto.Reset(interfaceInfo)
		p.interfacePool.Put(interfaceInfo)
	}

	return metrics, nil
}

// parseSubinterfaceCounters 解析子接口计数器数据
func (p *TelemetryParser) parseSubinterfaceCounters(msg *zteTelemetry.Telemetry) ([]models.SubinterfaceMetric, error) {
	if len(msg.DataGpb) == 0 {
		return nil, fmt.Errorf("DataGpb为空")
	}
	
	var metrics []models.SubinterfaceMetric

	// 遍历所有DataGpb条目
	for _, dataGpb := range msg.DataGpb {
		interfaceInfo := p.interfacePool.Get().(*interfaceProto.InterfaceInfo)
		if err := proto.Unmarshal(dataGpb.GetContent(), interfaceInfo); err != nil {
			p.interfacePool.Put(interfaceInfo)
			return nil, fmt.Errorf("解析接口信息失败: %v", err)
		}

		// 遍历子接口
		for _, subintf := range interfaceInfo.GetSubinterface() {
			// 创建基础metric
			metric := models.SubinterfaceMetric{
		Timestamp:        time.UnixMilli(int64(msg.MsgTimestamp)),
		SystemID:         msg.SystemId,
		InterfaceName:    interfaceInfo.GetName(),
		SubinterfaceName: fmt.Sprintf("%d", subintf.GetSubPort()),
	}
			
			// 解析Counters数据 (GetCounters返回数组，取第一个)
			if countersList := subintf.GetCounters(); len(countersList) > 0 {
				counters := countersList[0]
				metric.InOctets = uint64Ptr(counters.GetInOctets())
				metric.InUnicastPkts = uint64Ptr(counters.GetInUnicastPkts())
				metric.InBroadcastPkts = uint64Ptr(counters.GetInBroadcastPkts())
				metric.InMulticastPkts = uint64Ptr(counters.GetInMulticastPkts())
				metric.InDiscards = uint64Ptr(counters.GetInDiscards())
				metric.InErrors = uint64Ptr(counters.GetInErrors())
				metric.InUnknownProtos = uint64Ptr(counters.GetInUnknownProtos())
				metric.InFcsErrors = uint64Ptr(counters.GetInFcsErrors())
				metric.OutOctets = uint64Ptr(counters.GetOutOctets())
				metric.OutUnicastPkts = uint64Ptr(counters.GetOutUnicastPkts())
				metric.OutBroadcastPkts = uint64Ptr(counters.GetOutBroadcastPkts())
				metric.OutMulticastPkts = uint64Ptr(counters.GetOutMulticastPkts())
				metric.OutDiscards = uint64Ptr(counters.GetOutDiscards())
				metric.OutErrors = uint64Ptr(counters.GetOutErrors())
				metric.CarrierTransitions = uint64Ptr(counters.GetCarrierTransitions())
				metric.LastClear = timePtr(nanosToTimestamp(counters.GetLastClear()))
				metric.InPkts = uint64Ptr(counters.GetInPkts())
				metric.OutPkts = uint64Ptr(counters.GetOutPkts())
				metric.InputUtilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetInputUtilization())))
				metric.OutputUtilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetOutputUtilization())))
				metric.InTrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetInTrafficRate()))
				metric.InPacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetInPacketRate()))
				metric.OutTrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetOutTrafficRate()))
				metric.OutPacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetOutPacketRate()))
				metric.InV4Octets = uint64Ptr(counters.GetInV4Octets())
				metric.OutV4Octets = uint64Ptr(counters.GetOutV4Octets())
				metric.InV4Pkts = uint64Ptr(counters.GetInV4Pkts())
				metric.OutV4Pkts = uint64Ptr(counters.GetOutV4Pkts())
				metric.InV6Octets = uint64Ptr(counters.GetInV6Octets())
				metric.OutV6Octets = uint64Ptr(counters.GetOutV6Octets())
				metric.InV6Pkts = uint64Ptr(counters.GetInV6Pkts())
				metric.OutV6Pkts = uint64Ptr(counters.GetOutV6Pkts())
				metric.InV4TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetInV4TrafficRate()))
				metric.InV4PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetInV4PacketRate()))
				metric.OutV4TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetOutV4TrafficRate()))
				metric.OutV4PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetOutV4PacketRate()))
				metric.InV6TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetInV6TrafficRate()))
				metric.InV6PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetInV6PacketRate()))
				metric.OutV6TrafficRate = stringPtr(fmt.Sprintf("%.2f Mbps", counters.GetOutV6TrafficRate()))
				metric.OutV6PacketRate = stringPtr(fmt.Sprintf("%.2f Kfps", counters.GetOutV6PacketRate()))
				metric.InputV4Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetInputV4Utilization())))
				metric.OutputV4Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetOutputV4Utilization())))
				metric.InputV6Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetInputV6Utilization())))
				metric.OutputV6Utilization = stringPtr(fmt.Sprintf("%.2f", utilizationToNumeric(counters.GetOutputV6Utilization())))
				metric.InBierOctets = uint64Ptr(counters.GetInBierOctets())
				metric.InBierPkts = uint64Ptr(counters.GetInBierPkts())
				metric.OutBierOctets = uint64Ptr(counters.GetOutBierOctets())
				metric.OutBierPkts = uint64Ptr(counters.GetOutBierPkts())
			}

			metrics = append(metrics, metric)
		}

		proto.Reset(interfaceInfo)
		p.interfacePool.Put(interfaceInfo)
	}

	return metrics, nil
}