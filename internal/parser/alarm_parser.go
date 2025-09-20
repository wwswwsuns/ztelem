package parser

import (
	"fmt"
	"time"

	"github.com/wwswwsuns/ztelem/internal/models"
	zteTelemetry "github.com/wwswwsuns/ztelem/proto/zte_telemetry"
	zxr10Alarm "github.com/wwswwsuns/ztelem/proto/zxr10_alarm"
	"google.golang.org/protobuf/proto"
)

// parseAlarmData 解析告警数据 (data_type = ALARM)
func (p *TelemetryParser) parseAlarmData(msg *zteTelemetry.Telemetry) ([]models.AlarmReportMetric, []models.NotificationReportMetric, error) {
	p.logger.Debugf("🚨 开始解析告警数据: system_id=%s, sensor_path=%s, data_gpb_count=%d", 
		msg.SystemId, msg.SensorPath, len(msg.DataGpb))

	var alarmMetrics []models.AlarmReportMetric
	var notificationMetrics []models.NotificationReportMetric

	// 处理GPB编码的数据
	for i, dataGpb := range msg.DataGpb {
		p.logger.Debugf("处理第 %d 个GPB数据包，大小: %d bytes", i+1, len(dataGpb.GetContent()))
		
		// 根据sensor_path或proto_path判断具体的告警类型
		switch {
		case msg.SensorPath == "alm:current-alarm-report" || msg.ProtoPath == "zxr10.alarm.AlarmReport":
			// 当前告警上报
			alarms, err := p.parseCurrentAlarmReportFromGpb(dataGpb, msg.SystemId, msg.MsgTimestamp)
			if err != nil {
				p.logger.WithError(err).Warnf("解析当前告警上报失败")
				continue
			}
			alarmMetrics = append(alarmMetrics, alarms...)
			p.logger.Debugf("✅ 解析到 %d 条当前告警", len(alarms))

		case msg.SensorPath == "alm:notification-report" || msg.ProtoPath == "zxr10.alarm.NotificationReport":
			// 通知上报
			notifications, err := p.parseNotificationReportFromGpb(dataGpb, msg.SystemId, msg.MsgTimestamp)
			if err != nil {
				p.logger.WithError(err).Warnf("解析通知上报失败")
				continue
			}
			notificationMetrics = append(notificationMetrics, notifications...)
			p.logger.Debugf("✅ 解析到 %d 条通知", len(notifications))

		default:
			// 尝试通用告警解析
			p.logger.Debugf("尝试通用告警解析: sensor_path=%s, proto_path=%s", msg.SensorPath, msg.ProtoPath)
			
			// 先尝试解析为告警上报
			if alarms, err := p.parseCurrentAlarmReportFromGpb(dataGpb, msg.SystemId, msg.MsgTimestamp); err == nil && len(alarms) > 0 {
				alarmMetrics = append(alarmMetrics, alarms...)
				p.logger.Debugf("✅ 通用解析到 %d 条告警", len(alarms))
			} else if notifications, err := p.parseNotificationReportFromGpb(dataGpb, msg.SystemId, msg.MsgTimestamp); err == nil && len(notifications) > 0 {
				// 再尝试解析为通知上报
				notificationMetrics = append(notificationMetrics, notifications...)
				p.logger.Debugf("✅ 通用解析到 %d 条通知", len(notifications))
			} else {
				p.logger.Warnf("无法解析告警数据: sensor_path=%s, proto_path=%s", msg.SensorPath, msg.ProtoPath)
			}
		}
	}

	p.logger.Infof("🎯 告警数据解析完成: 告警=%d条, 通知=%d条", len(alarmMetrics), len(notificationMetrics))
	return alarmMetrics, notificationMetrics, nil
}

// parseCurrentAlarmReportFromGpb 从GPB数据解析当前告警上报
func (p *TelemetryParser) parseCurrentAlarmReportFromGpb(dataGpb *zteTelemetry.NotificationGpb, systemId string, msgTimestamp uint64) ([]models.AlarmReportMetric, error) {
	p.logger.Debugf("🔍 开始解析告警GPB数据，大小: %d bytes", len(dataGpb.GetContent()))
	
	var metrics []models.AlarmReportMetric
	timestamp := time.UnixMilli(int64(msgTimestamp))
	
	// 首先尝试解析为AlarmInfo容器
	var alarmInfo zxr10Alarm.AlarmInfo
	if err := proto.Unmarshal(dataGpb.GetContent(), &alarmInfo); err == nil {
		p.logger.Debugf("🔍 Proto解析成功(AlarmInfo)，alarm_report数量=%d, notification_report数量=%d", 
			len(alarmInfo.GetAlarmReport()), len(alarmInfo.GetNotificationReport()))
		
		// 处理当前告警上报
		for _, alarm := range alarmInfo.GetAlarmReport() {
			metric := models.AlarmReportMetric{
				Timestamp:         timestamp,
				SystemID:          systemId,
				FlowID:            alarm.GetFlowId(),
				AlarmTimestamp:    alarm.GetTimestamp(),
				Code:              alarm.GetCode(),
				OccurrenceTime:    alarm.GetOccurrenceTime(),
				UpdateTime:        alarm.GetUpdateTime(),
				DisappearedTime:   alarm.GetDisappearedTime(),
				OccurrenceMs:      alarm.GetOccurrenceMs(),
				UpdateMs:          alarm.GetUpdateMs(),
				DisappearedMs:     alarm.GetDisappearedMs(),
				AlarmClass:        stringPtr(alarm.GetAlarmClass()),
				AlarmType:         stringPtr(alarm.GetAlarmType()),
				AlarmStatus:       stringPtr(alarm.GetAlarmStatus()),
				Sort:              uint32Ptr(alarm.GetSort()),
				Severity:          stringPtr(alarm.GetSeverity()),
				TpidType:          uint32Ptr(alarm.GetTpidType()),
				TpidLength:        uint32Ptr(alarm.GetTpidLength()),
				Tpid:              bytesToBase64Ptr(alarm.GetTpid()),
				Description:       stringPtr(alarm.GetDescription()),
				Caption:           stringPtr(alarm.GetCaption()),
			}
			metrics = append(metrics, metric)
			
			p.logger.Debugf("✅ 解析到告警: FlowID=%d, 类型=%s, 严重性=%s", 
				metric.FlowID, safeString(metric.AlarmType), safeString(metric.Severity))
		}
		
		if len(metrics) > 0 {
			p.logger.Debugf("🎯 告警解析完成，共解析 %d 条告警", len(metrics))
			return metrics, nil
		}
	}
	
	// 如果AlarmInfo解析失败或为空，尝试直接解析为CurrentAlarm
	p.logger.Debugf("🔍 尝试直接解析为CurrentAlarm")
	var currentAlarm zxr10Alarm.CurrentAlarm
	if err := proto.Unmarshal(dataGpb.GetContent(), &currentAlarm); err != nil {
		// 输出原始数据用于调试
		content := dataGpb.GetContent()
		hexData := ""
		if len(content) > 0 {
			maxLen := 100
			if len(content) < maxLen {
				maxLen = len(content)
			}
			hexData = fmt.Sprintf("%x", content[:maxLen])
		}
		p.logger.WithError(err).Errorf("🔍 直接解析CurrentAlarm也失败，原始数据(前100字节): %s", hexData)
		return nil, err
	}
	
	p.logger.Debugf("🔍 Proto解析成功(CurrentAlarm)，FlowID=%d", currentAlarm.GetFlowId())
	
	// 处理单个告警
	metric := models.AlarmReportMetric{
		Timestamp:         timestamp,
		SystemID:          systemId,
		FlowID:            currentAlarm.GetFlowId(),
		AlarmTimestamp:    currentAlarm.GetTimestamp(),
		Code:              currentAlarm.GetCode(),
		OccurrenceTime:    currentAlarm.GetOccurrenceTime(),
		UpdateTime:        currentAlarm.GetUpdateTime(),
		DisappearedTime:   currentAlarm.GetDisappearedTime(),
		OccurrenceMs:      currentAlarm.GetOccurrenceMs(),
		UpdateMs:          currentAlarm.GetUpdateMs(),
		DisappearedMs:     currentAlarm.GetDisappearedMs(),
		AlarmClass:        stringPtr(currentAlarm.GetAlarmClass()),
		AlarmType:         stringPtr(currentAlarm.GetAlarmType()),
		AlarmStatus:       stringPtr(currentAlarm.GetAlarmStatus()),
		Sort:              uint32Ptr(currentAlarm.GetSort()),
		Severity:          stringPtr(currentAlarm.GetSeverity()),
		TpidType:          uint32Ptr(currentAlarm.GetTpidType()),
		TpidLength:        uint32Ptr(currentAlarm.GetTpidLength()),
		Tpid:              bytesToBase64Ptr(currentAlarm.GetTpid()),
		Description:       stringPtr(currentAlarm.GetDescription()),
		Caption:           stringPtr(currentAlarm.GetCaption()),
	}
	metrics = append(metrics, metric)
	
	p.logger.Debugf("✅ 解析到告警: FlowID=%d, 类型=%s, 严重性=%s", 
		metric.FlowID, safeString(metric.AlarmType), safeString(metric.Severity))
	
	p.logger.Debugf("🎯 告警解析完成，共解析 %d 条告警", len(metrics))
	return metrics, nil
}

// parseNotificationReportFromGpb 从GPB数据解析通知上报
func (p *TelemetryParser) parseNotificationReportFromGpb(dataGpb *zteTelemetry.NotificationGpb, systemId string, msgTimestamp uint64) ([]models.NotificationReportMetric, error) {
	p.logger.Debugf("🔍 开始解析通知GPB数据，大小: %d bytes", len(dataGpb.GetContent()))
	
	// 解析GPB数据为AlarmInfo
	var alarmInfo zxr10Alarm.AlarmInfo
	if err := proto.Unmarshal(dataGpb.GetContent(), &alarmInfo); err != nil {
		p.logger.WithError(err).Error("解析通知GPB数据失败")
		return nil, err
	}
	
	// 添加详细的调试信息
	p.logger.Debugf("🔍 Proto解析成功，AlarmInfo内容: alarm_report数量=%d, notification_report数量=%d", 
		len(alarmInfo.GetAlarmReport()), len(alarmInfo.GetNotificationReport()))
	
	var metrics []models.NotificationReportMetric
	timestamp := time.UnixMilli(int64(msgTimestamp))
	
	// 处理通知上报
	for _, notification := range alarmInfo.GetNotificationReport() {
		metric := models.NotificationReportMetric{
			Timestamp:             timestamp,
			SystemID:              systemId,
			FlowID:                notification.GetFlowId(),
			NotificationTimestamp: notification.GetTimestamp(),
			Code:                  notification.GetCode(),
			OccurTime:             notification.GetOccurTime(),
			OccurMs:               notification.GetOccurMs(),
			Classification:        stringPtr(notification.GetClassification()),
			Sort:                  uint32Ptr(notification.GetSort()),
			Severity:              stringPtr(notification.GetSeverity()),
		}
		metrics = append(metrics, metric)
		
		p.logger.Debugf("✅ 解析到通知: FlowID=%d, 分类=%s, 严重性=%s", 
			metric.FlowID, safeString(metric.Classification), safeString(metric.Severity))
	}
	
	p.logger.Debugf("🎯 通知解析完成，共解析 %d 条通知", len(metrics))
	return metrics, nil
}

// 辅助函数：字节数组转base64指针
func bytesToBase64Ptr(b []byte) *string {
	if len(b) == 0 {
		return nil
	}
	encoded := fmt.Sprintf("%x", b)
	return &encoded
}

// 辅助函数：安全获取字符串值
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}