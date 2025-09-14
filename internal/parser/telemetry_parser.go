package parser

import (
	"fmt"
	"strings"
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
		return fmt.Sprintf("UNKNOWN_%d", value)
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
		return fmt.Sprintf("ADMIN_STATUS_UNKNOWN_%d", value)
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
		return fmt.Sprintf("OPER_STATUS_UNKNOWN_%d", value)
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
		return fmt.Sprintf("IPV4OPERSTATUS_STATUS_UNKNOWN_%d", value)
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
		return fmt.Sprintf("IPV6OPERSTATUS_STATUS_UNKNOWN_%d", value)
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
		return fmt.Sprintf("PHY_STATUS_UNKNOWN_%d", value)
	}
}

// ParseResult 解析结果
type ParseResult struct {
	SystemID             string
	SensorPath           string
	Timestamp            time.Time
	PlatformMetrics      []models.PlatformMetric
	InterfaceMetrics     []models.InterfaceMetric
	SubinterfaceMetrics  []models.SubinterfaceMetric
}

// TelemetryParser telemetry数据解析器
type TelemetryParser struct {
	logger *logrus.Logger
}

// NewTelemetryParser 创建新的解析器
func NewTelemetryParser(logger *logrus.Logger) *TelemetryParser {
	return &TelemetryParser{
		logger: logger,
	}
}

// ParseTelemetryData 解析telemetry数据
func (p *TelemetryParser) ParseTelemetryData(data []byte) (*ParseResult, error) {
	// 解析ZTE Telemetry消息
	var telemetryMsg zteTelemetry.Telemetry
	if err := proto.Unmarshal(data, &telemetryMsg); err != nil {
		return nil, fmt.Errorf("解析telemetry消息失败: %v", err)
	}

	p.logger.Debugf("解析到telemetry消息: system_id=%s, sensor_path=%s", 
		telemetryMsg.SystemId, telemetryMsg.SensorPath)

	result := &ParseResult{
		SystemID:    telemetryMsg.SystemId,
		SensorPath:  telemetryMsg.SensorPath,
		Timestamp:   time.UnixMilli(int64(telemetryMsg.MsgTimestamp)),
	}

	// 根据sensor_path路由到不同的解析函数
	switch {
	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/state/memory"):
		// 组件内存数据 (优先匹配更具体的路径)
		metrics, err := p.parseComponentMemoryState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/state/storage"):
		// 组件存储数据
		metrics, err := p.parseComponentStorageState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/state/temperature"):
		// 组件温度数据
		metrics, err := p.parseComponentTemperatureState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/state"):
		// 组件通用数据 (匹配 oc-platform:components/component/state )
		metrics, err := p.parseComponentState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case telemetryMsg.SensorPath == "oc-platform:components/component":
		// 组件综合数据 (包含CPU、内存等多种数据)
		metrics, err := p.parseComponentsData(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/fan/state"):
		// 组件风扇数据
		metrics, err := p.parseComponentFanState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics



	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/power-supply/state"):
		// 组件电源数据
		metrics, err := p.parseComponentPowerSupplyState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/oc-linecard:linecard/state"):
		// 组件线卡数据
		metrics, err := p.parseComponentLinecardState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/cpu/oc-cpu:utilization/state"):
		// 组件CPU数据
		metrics, err := p.parseComponentCPUState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-platform:components/component/oc-transceiver:transceiver/state"):
		// 组件光模块数据
		metrics, err := p.parseComponentTransceiverState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.PlatformMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/state") &&
		 !strings.Contains(strings.TrimPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/state"), "/"):
		// 接口状态数据 (精确匹配)
		metrics, err := p.parseInterfaceState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.InterfaceMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/zte-if:state-period"):
		// 接口ZTE扩展数据
		metrics, err := p.parseInterfaceZteState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.InterfaceMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/state/counters"):
		// 接口计数器数据
		metrics, err := p.parseInterfaceCounters(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.InterfaceMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/subinterfaces/subinterface/state") &&
		 !strings.Contains(strings.TrimPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/subinterfaces/subinterface/state"), "/"):
		// 子接口状态数据 (精确匹配)
		metrics, err := p.parseSubinterfaceState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.SubinterfaceMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/subinterfaces/subinterface/zte-if:state-period"):
		// 子接口ZTE扩展数据
		metrics, err := p.parseSubinterfaceZteState(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.SubinterfaceMetrics = metrics

	case strings.HasPrefix(telemetryMsg.SensorPath, "oc-if:interfaces/interface/subinterfaces/subinterface/state/counters"):
		// 子接口计数器数据
		metrics, err := p.parseSubinterfaceCounters(&telemetryMsg)
		if err != nil {
			return nil, err
		}
		result.SubinterfaceMetrics = metrics

	default:
		p.logger.Warnf("未知的sensor_path: %s", telemetryMsg.SensorPath)
		return result, nil
	}

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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 解析CommonState数据
		if commonState := componentInfo.GetCommonState(); commonState != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 创建单个组件的完整指标记录
		metric := models.PlatformMetric{
			Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
			SystemID:      msg.SystemId,
			ComponentName: componentInfo.GetName(),
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
			metric.StorageAvailability = float64Ptr(float64(storageInfo.GetAvailability()))
		}

		// 解析光模块信息
		if opticalInfo := componentInfo.GetOpticalInfo(); opticalInfo != nil {
			if inPower := opticalInfo.GetInPower(); inPower != nil {
				metric.OpticalInPower = float64Ptr(float64(inPower.GetInstant()))
			}
			if outPower := opticalInfo.GetOutPower(); outPower != nil {
				metric.OpticalOutPower = float64Ptr(float64(outPower.GetInstant()))
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
						metric.OpticalAlarmLosInfoInPower = float64Ptr(float64(losInfo.GetOptInPower()[0].GetInstant()))
					}
					if len(losInfo.GetOptOutPower()) > 0 {
						metric.OpticalAlarmLosInfoOutPower = float64Ptr(float64(losInfo.GetOptOutPower()[0].GetInstant()))
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
			metric.LinecardPowerAdminState = stringPtr(linecardInfo.GetPowerAdminState())
		}

		// 将完整的组件指标添加到结果中
		metrics = append(metrics, metric)
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 解析FanInfo数据
		if fanInfo := componentInfo.GetFanInfo(); fanInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			p.logger.Warnf("解析组件信息失败: %v", err)
			continue
		}

		// 解析MemoryInfo数据
		if memInfo := componentInfo.GetMemInfo(); memInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析StorageInfo数据
		if storageInfo := componentInfo.GetStorageInfo(); storageInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
			}

			metric.StorageAvailability = float64Ptr(float64(storageInfo.GetAvailability()))

			metrics = append(metrics, metric)
		}
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析Temperature数据
		if tempInfo := componentInfo.GetTempInfo(); tempInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析PowerInfo数据
		if powerInfo := componentInfo.GetPowerInfo(); powerInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析LinecardInfo数据
		if linecardInfo := componentInfo.GetPowerAdminState(); linecardInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
			}

			metric.LinecardPowerAdminState = stringPtr(linecardInfo.GetPowerAdminState())

			metrics = append(metrics, metric)
		}
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		if cpuInfo := componentInfo.GetCpuInfo(); cpuInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
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
		var componentInfo platformProto.ComponentInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &componentInfo); err != nil {
			return nil, fmt.Errorf("解析组件信息失败: %v", err)
		}

		// 解析OpticalInfo数据
		if opticalInfo := componentInfo.GetOpticalInfo(); opticalInfo != nil {
			metric := models.PlatformMetric{
				Timestamp:     time.UnixMilli(int64(msg.MsgTimestamp)),
				SystemID:      msg.SystemId,
				ComponentName: componentInfo.GetName(),
			}

			if inPower := opticalInfo.GetInPower(); inPower != nil {
				metric.OpticalInPower = float64Ptr(float64(inPower.GetInstant()))
			}
			if outPower := opticalInfo.GetOutPower(); outPower != nil {
				metric.OpticalOutPower = float64Ptr(float64(outPower.GetInstant()))
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
						metric.OpticalAlarmLosInfoInPower = float64Ptr(float64(inPowers[0].GetInstant()))
					}
					if outPowers := alarmInfo.GetOptOutPower(); len(outPowers) > 0 {
						metric.OpticalAlarmLosInfoOutPower = float64Ptr(float64(outPowers[0].GetInstant()))
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
		var interfaceInfo interfaceProto.InterfaceInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &interfaceInfo); err != nil {
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
		var interfaceInfo interfaceProto.InterfaceInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &interfaceInfo); err != nil {
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
		var interfaceInfo interfaceProto.InterfaceInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &interfaceInfo); err != nil {
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
		var interfaceInfo interfaceProto.InterfaceInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &interfaceInfo); err != nil {
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
		var interfaceInfo interfaceProto.InterfaceInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &interfaceInfo); err != nil {
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
		var interfaceInfo interfaceProto.InterfaceInfo
		if err := proto.Unmarshal(dataGpb.GetContent(), &interfaceInfo); err != nil {
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
	}

	return metrics, nil
}