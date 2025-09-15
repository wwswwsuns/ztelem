package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// PrometheusServer Prometheus指标服务器
type PrometheusServer struct {
	server *http.Server
	logger *logrus.Logger
	
	// 自定义指标
	dbWriteDuration     *prometheus.HistogramVec
	dbRecordsWritten    *prometheus.CounterVec
	dbWriteErrors       *prometheus.CounterVec
	dbBatchSize         *prometheus.HistogramVec
	dbPoolConnections   *prometheus.GaugeVec
	dbPoolWaitDuration  prometheus.Histogram
	recordsProcessed    *prometheus.CounterVec
	dbQueueDepth        *prometheus.GaugeVec
	bufferSize          *prometheus.GaugeVec
	bufferFlushes       *prometheus.CounterVec
	systemMemory        *prometheus.GaugeVec
	systemGoroutines    prometheus.Gauge
}

// NewPrometheusServer 创建Prometheus指标服务器
func NewPrometheusServer(port int, logger *logrus.Logger) *PrometheusServer {
	// 创建自定义指标
	dbWriteDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telemetry_db_write_duration_seconds",
			Help:    "数据库写入操作耗时分布",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"table", "operation"},
	)

	dbRecordsWritten := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_db_records_written_total",
			Help: "数据库写入记录总数",
		},
		[]string{"table", "status"},
	)

	dbWriteErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_db_write_errors_total",
			Help: "数据库写入错误总数",
		},
		[]string{"table", "error_type"},
	)

	dbBatchSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telemetry_db_batch_size",
			Help:    "数据库批次大小分布",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"table"},
	)

	dbPoolConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_db_pool_connections",
			Help: "数据库连接池连接数",
		},
		[]string{"state"}, // open, in_use, idle
	)

	dbPoolWaitDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "telemetry_db_pool_wait_duration_seconds",
			Help:    "数据库连接池等待时间",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
	)

	recordsProcessed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_records_processed_total",
			Help: "处理的遥测记录总数",
		},
		[]string{"type", "status"}, // type: platform/interface/subinterface, status: success/error
	)

	dbQueueDepth := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_db_queue_depth",
			Help: "数据库写入队列深度",
		},
		[]string{"table"},
	)

	bufferSize := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_buffer_size",
			Help: "缓冲区当前大小",
		},
		[]string{"type"}, // platform, interface, subinterface
	)

	bufferFlushes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_buffer_flushes_total",
			Help: "缓冲区刷新次数",
		},
		[]string{"type", "reason"}, // reason: threshold, timeout, manual
	)

	systemMemory := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_system_memory_bytes",
			Help: "系统内存使用情况",
		},
		[]string{"type"}, // alloc, sys, heap_alloc, heap_sys
	)

	systemGoroutines := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "telemetry_system_goroutines",
			Help: "当前Goroutine数量",
		},
	)

	// 注册所有指标
	prometheus.MustRegister(
		dbWriteDuration,
		dbRecordsWritten,
		dbWriteErrors,
		dbBatchSize,
		dbPoolConnections,
		dbPoolWaitDuration,
		recordsProcessed,
		dbQueueDepth,
		bufferSize,
		bufferFlushes,
		systemMemory,
		systemGoroutines,
	)

	mux := http.NewServeMux()
	
	// 注册Prometheus指标端点
	mux.Handle("/metrics", promhttp.Handler())
	
	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// 指标说明端点
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Telemetry Database Metrics</title>
</head>
<body>
    <h1>遥测数据库性能指标</h1>
    <h2>可用端点:</h2>
    <ul>
        <li><a href="/metrics">/metrics</a> - Prometheus指标</li>
        <li><a href="/health">/health</a> - 健康检查</li>
    </ul>
    
    <h2>关键指标说明:</h2>
    <ul>
        <li><strong>telemetry_db_write_duration_seconds</strong> - 数据库写入延迟分布</li>
        <li><strong>telemetry_db_records_written_total</strong> - 数据库写入记录总数</li>
        <li><strong>telemetry_db_write_errors_total</strong> - 数据库写入错误总数</li>
        <li><strong>telemetry_db_batch_size</strong> - 数据库批次大小分布</li>
        <li><strong>telemetry_db_pool_connections</strong> - 数据库连接池连接数</li>
        <li><strong>telemetry_db_pool_wait_duration_seconds</strong> - 数据库连接池等待时间</li>
        <li><strong>telemetry_records_processed_total</strong> - 处理的遥测记录总数</li>
        <li><strong>telemetry_db_queue_depth</strong> - 数据库写入队列深度</li>
    </ul>
    
    <h2>性能监控查询示例:</h2>
    <pre>
# 写入吞吐量 (records/second)
rate(telemetry_db_records_written_total[5m])

# 平均写入延迟
rate(telemetry_db_write_duration_seconds_sum[5m]) / rate(telemetry_db_write_duration_seconds_count[5m])

# 错误率
rate(telemetry_db_write_errors_total[5m]) / rate(telemetry_db_records_written_total[5m])

# 连接池使用率
telemetry_db_pool_connections{state="acquired"} / telemetry_db_pool_connections{state="total"}
    </pre>
</body>
</html>
		`))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ps := &PrometheusServer{
		server:              server,
		logger:              logger,
		dbWriteDuration:     dbWriteDuration,
		dbRecordsWritten:    dbRecordsWritten,
		dbWriteErrors:       dbWriteErrors,
		dbBatchSize:         dbBatchSize,
		dbPoolConnections:   dbPoolConnections,
		dbPoolWaitDuration:  dbPoolWaitDuration,
		recordsProcessed:    recordsProcessed,
		dbQueueDepth:        dbQueueDepth,
		bufferSize:          bufferSize,
		bufferFlushes:       bufferFlushes,
		systemMemory:        systemMemory,
		systemGoroutines:    systemGoroutines,
	}

	return ps
}

// Start 启动Prometheus指标服务器
func (ps *PrometheusServer) Start() error {
	ps.logger.WithField("addr", ps.server.Addr).Info("启动Prometheus指标服务器")
	
	go func() {
		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ps.logger.WithError(err).Error("Prometheus指标服务器启动失败")
		}
	}()
	
	return nil
}

// Stop 停止Prometheus指标服务器
func (ps *PrometheusServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	ps.logger.Info("停止Prometheus指标服务器")
	return ps.server.Shutdown(ctx)
}

// UpdateMetrics 更新指标数据
func (ps *PrometheusServer) UpdateMetrics(bufferStats interface{}, dbStats interface{}) {
	// 这里需要根据实际的统计数据结构来更新指标
	// 暂时添加一些示例数据
	
	// 更新缓冲区指标
	ps.bufferSize.WithLabelValues("platform").Set(100)
	ps.bufferSize.WithLabelValues("interface").Set(200)
	ps.bufferSize.WithLabelValues("subinterface").Set(150)
	
	// 更新数据库连接池指标
	ps.dbPoolConnections.WithLabelValues("open").Set(26)
	ps.dbPoolConnections.WithLabelValues("in_use").Set(5)
	ps.dbPoolConnections.WithLabelValues("idle").Set(21)
	
	// 更新系统指标
	ps.systemGoroutines.Set(65)
	ps.systemMemory.WithLabelValues("alloc").Set(2 * 1024 * 1024) // 2MB
	ps.systemMemory.WithLabelValues("sys").Set(12 * 1024 * 1024)  // 12MB
}

// RecordDBWrite 记录数据库写入指标
func (ps *PrometheusServer) RecordDBWrite(table string, duration time.Duration, recordCount int, success bool) {
	ps.dbWriteDuration.WithLabelValues(table, "write").Observe(duration.Seconds())
	ps.dbBatchSize.WithLabelValues(table).Observe(float64(recordCount))
	
	status := "success"
	if !success {
		status = "error"
		ps.dbWriteErrors.WithLabelValues(table, "write_failed").Inc()
	}
	ps.dbRecordsWritten.WithLabelValues(table, status).Add(float64(recordCount))
}

// RecordBufferFlush 记录缓冲区刷新
func (ps *PrometheusServer) RecordBufferFlush(bufferType, reason string) {
	ps.bufferFlushes.WithLabelValues(bufferType, reason).Inc()
}

// RecordProcessed 记录处理的记录数
func (ps *PrometheusServer) RecordProcessed(recordType string, count int, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	ps.recordsProcessed.WithLabelValues(recordType, status).Add(float64(count))
}

// UpdateBufferSize 更新缓冲区大小指标
func (ps *PrometheusServer) UpdateBufferSize(bufferType string, size float64) {
	ps.bufferSize.WithLabelValues(bufferType).Set(size)
}

// UpdateDBPoolConnections 更新数据库连接池指标
func (ps *PrometheusServer) UpdateDBPoolConnections(state string, count float64) {
	ps.dbPoolConnections.WithLabelValues(state).Set(count)
}

// UpdateSystemMetrics 更新系统指标
func (ps *PrometheusServer) UpdateSystemMetrics(goroutines, allocMem, sysMem, heapAlloc, heapSys float64) {
	ps.systemGoroutines.Set(goroutines)
	ps.systemMemory.WithLabelValues("alloc").Set(allocMem)
	ps.systemMemory.WithLabelValues("sys").Set(sysMem)
	ps.systemMemory.WithLabelValues("heap_alloc").Set(heapAlloc)
	ps.systemMemory.WithLabelValues("heap_sys").Set(heapSys)
}

// UpdateProcessedRecords 更新处理记录统计
func (ps *PrometheusServer) UpdateProcessedRecords(recordType string, count float64) {
	// 这里可以根据需要添加更多的处理记录统计
	if recordType == "total" {
		ps.recordsProcessed.WithLabelValues("all", "success").Add(count)
	} else if recordType == "errors" {
		ps.recordsProcessed.WithLabelValues("all", "error").Add(count)
	}
}