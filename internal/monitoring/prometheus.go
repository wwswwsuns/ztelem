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
	server           *http.Server
	logger           *logrus.Logger
	dbPoolConnections *prometheus.GaugeVec
	recordsProcessed *prometheus.CounterVec
	bufferSize       *prometheus.GaugeVec
	systemMemory     *prometheus.GaugeVec
	systemGoroutines prometheus.Gauge
	grpcConnections  *prometheus.GaugeVec
	zombieRatio      prometheus.Gauge
}

// NewPrometheusServer 创建Prometheus指标服务器
func NewPrometheusServer(port int, logger *logrus.Logger) *PrometheusServer {
	dbPoolConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_db_pool_connections",
			Help: "数据库连接池连接数",
		},
		[]string{"state"},
	)

	recordsProcessed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_records_processed_total",
			Help: "处理的遥测记录总数",
		},
		[]string{"type", "status"},
	)

	bufferSize := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_buffer_size",
			Help: "缓冲区当前大小",
		},
		[]string{"type"},
	)

	systemMemory := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_system_memory_bytes",
			Help: "系统内存使用情况",
		},
		[]string{"type"},
	)

	systemGoroutines := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "telemetry_system_goroutines",
			Help: "当前Goroutine数量",
		},
	)

	grpcConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telemetry_grpc_connections",
			Help: "gRPC连接状态",
		},
		[]string{"state"},
	)

	zombieRatio := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "telemetry_zombie_ratio",
			Help: "Percent of stale connections (0-100)",
		},
	)

	prometheus.MustRegister(
		dbPoolConnections,
		recordsProcessed,
		bufferSize,
		systemMemory,
		systemGoroutines,
		grpcConnections,
		zombieRatio,
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Telemetry Metrics</title></head>
<body>
<h1>Telemetry Prometheus Metrics</h1>
<ul>
<li><a href="/metrics">/metrics</a> - Prometheus指标</li>
<li><a href="/health">/health</a> - 健康检查</li>
</ul>
<h2>关键指标:</h2>
<ul>
<li><strong>telemetry_records_processed_total</strong> - 处理记录数 (rate求速率)</li>
<li><strong>telemetry_db_pool_connections</strong> - 数据库连接池 (open/in_use/idle)</li>
<li><strong>telemetry_buffer_size</strong> - 缓冲区大小 (platform/interface/subinterface)</li>
<li><strong>telemetry_grpc_connections</strong> - gRPC连接数 (total/active/stale)</li>
<li><strong>telemetry_zombie_ratio</strong> - 僵尸连接比例</li>
<li><strong>telemetry_system_memory_bytes</strong> - 内存使用</li>
<li><strong>telemetry_system_goroutines</strong> - Goroutine数量</li>
</ul>
</body></html>`))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ps := &PrometheusServer{
		server:           server,
		logger:           logger,
		dbPoolConnections: dbPoolConnections,
		recordsProcessed: recordsProcessed,
		bufferSize:       bufferSize,
		systemMemory:     systemMemory,
		systemGoroutines: systemGoroutines,
		grpcConnections:  grpcConnections,
		zombieRatio:      zombieRatio,
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

// UpdateProcessedRecords 更新处理记录统计（增量）
func (ps *PrometheusServer) UpdateProcessedRecords(recordType string, count float64) {
	if recordType == "total" {
		ps.recordsProcessed.WithLabelValues("all", "success").Add(count)
	} else if recordType == "errors" {
		ps.recordsProcessed.WithLabelValues("all", "error").Add(count)
	}
}

// UpdateGRPCConnections 更新gRPC连接指标
func (ps *PrometheusServer) UpdateGRPCConnections(state string, count float64) {
	ps.grpcConnections.WithLabelValues(state).Set(count)
}

// UpdateZombieRatio 更新僵尸连接比例(0-100)
func (ps *PrometheusServer) UpdateZombieRatio(percent float64) {
	ps.zombieRatio.Set(percent)
}
