package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config 应用配置 - 扩展版本
type Config struct {
	Database       DatabaseConfig       `yaml:"database"`
	Server         ServerConfig         `yaml:"server"`
	Buffer         BufferConfig         `yaml:"buffer"`
	DatabaseWriter DatabaseWriterConfig `yaml:"database_writer"`
	Memory         MemoryConfig         `yaml:"memory"`
	Logging        LoggingConfig        `yaml:"logging"`
	Monitoring     MonitoringConfig     `yaml:"monitoring"`
	Performance    PerformanceConfig    `yaml:"performance"`
	Compression    CompressionConfig    `yaml:"compression"`
	Recovery       RecoveryConfig       `yaml:"recovery"`
	Debug          DebugConfig          `yaml:"debug"`
}

// DatabaseConfig 数据库配置 - 扩展版本
type DatabaseConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	User              string        `yaml:"user"`
	Password          string        `yaml:"password"`
	Database          string        `yaml:"dbname"`
	Schema            string        `yaml:"schema"`
	SSLMode           string        `yaml:"sslmode"`
	MaxOpenConns      int           `yaml:"max_open_conns"`
	MaxIdleConns      int           `yaml:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime   time.Duration `yaml:"conn_max_idle_time"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port                  int           `yaml:"port"`
	MaxRecvMsgSize        int           `yaml:"max_recv_msg_size"`
	MaxSendMsgSize        int           `yaml:"max_send_msg_size"`
	MaxConcurrentStreams  uint32        `yaml:"max_concurrent_streams"`
	KeepaliveTime         time.Duration `yaml:"keepalive_time"`
	KeepaliveTimeout      time.Duration `yaml:"keepalive_timeout"`
}

// BufferConfig 缓冲区配置 - 扩展版本
type BufferConfig struct {
	MaxSize                    int `yaml:"max_size"`
	FlushInterval              time.Duration `yaml:"flush_interval"`
	BatchSize                  int `yaml:"batch_size"`
	FlushThreshold             int `yaml:"flush_threshold"`
	PlatformBufferSize         int `yaml:"platform_buffer_size"`
	InterfaceBufferSize        int `yaml:"interface_buffer_size"`
	SubinterfaceBufferSize     int `yaml:"subinterface_buffer_size"`
}

// DatabaseWriterConfig 数据库写入配置
type DatabaseWriterConfig struct {
	BatchTimeout              time.Duration `yaml:"batch_timeout"`
	RetryAttempts             int           `yaml:"retry_attempts"`
	RetryDelay                time.Duration `yaml:"retry_delay"`
	ParallelWriters           int           `yaml:"parallel_writers"`
	MaxBatchSize              int           `yaml:"max_batch_size"`
	EnableParallelTableWrites bool          `yaml:"enable_parallel_table_writes"`
	PlatformWriterCount       int           `yaml:"platform_writer_count"`
	InterfaceWriterCount      int           `yaml:"interface_writer_count"`
	SubinterfaceWriterCount   int           `yaml:"subinterface_writer_count"`
}

// MemoryConfig 内存管理配置
type MemoryConfig struct {
	MaxMemoryUsage   string `yaml:"max_memory_usage"`
	GCTargetPercent  int    `yaml:"gc_target_percent"`
	BufferPoolSize   int    `yaml:"buffer_pool_size"`
}

// LoggingConfig 日志配置 - 扩展版本
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
	Compress   bool   `yaml:"compress"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled            bool                    `yaml:"enabled"`
	MetricsInterval    time.Duration           `yaml:"metrics_interval"`
	HealthCheckPort    int                     `yaml:"health_check_port"`
	PrometheusEnabled  bool                    `yaml:"prometheus_enabled"`
	PrometheusPort     int                     `yaml:"prometheus_port"`
	AlertThresholds    AlertThresholdsConfig   `yaml:"alert_thresholds"`
}

// AlertThresholdsConfig 告警阈值配置
type AlertThresholdsConfig struct {
	BufferUsagePercent          int `yaml:"buffer_usage_percent"`
	DBConnectionUsagePercent    int `yaml:"db_connection_usage_percent"`
	MemoryUsagePercent          int `yaml:"memory_usage_percent"`
	ErrorRatePerMinute          int `yaml:"error_rate_per_minute"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	GCPercent        int    `yaml:"gc_percent"`
	MaxProcs         int    `yaml:"max_procs"`
	EnablePprof      bool   `yaml:"enable_pprof"`
	PprofPort        int    `yaml:"pprof_port"`
	TCPKeepalive     bool   `yaml:"tcp_keepalive"`
	TCPNoDelay       bool   `yaml:"tcp_no_delay"`
	ReadBufferSize   int    `yaml:"read_buffer_size"`
	WriteBufferSize  int    `yaml:"write_buffer_size"`
}

// CompressionConfig 压缩配置
type CompressionConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Algorithm string `yaml:"algorithm"`
	Level     int    `yaml:"level"`
}

// RecoveryConfig 故障恢复配置
type RecoveryConfig struct {
	EnableDataPersistence bool          `yaml:"enable_data_persistence"`
	PersistencePath       string        `yaml:"persistence_path"`
	MaxRecoveryTime       time.Duration `yaml:"max_recovery_time"`
	CheckpointInterval    time.Duration `yaml:"checkpoint_interval"`
}

// DebugConfig 调试配置
type DebugConfig struct {
	Enabled            bool          `yaml:"enabled"`
	LogRawData         bool          `yaml:"log_raw_data"`
	MetricsEnabled     bool          `yaml:"metrics_enabled"`
	ProfileEnabled     bool          `yaml:"profile_enabled"`
	SlowQueryThreshold time.Duration `yaml:"slow_query_threshold"`
}

// LoadConfig 加载配置文件 - 扩展版本
func LoadConfig(filename string) (*Config, error) {
	// 默认配置
	config := &Config{
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "telemetry_app",
			Password:        "",
			Database:        "telemetrydb",
			Schema:          "telemetry",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		},
		Server: ServerConfig{
			Port:                 50051,
			MaxRecvMsgSize:       4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:       4 * 1024 * 1024, // 4MB
			MaxConcurrentStreams: 100,
			KeepaliveTime:        30 * time.Second,
			KeepaliveTimeout:     5 * time.Second,
		},
		Buffer: BufferConfig{
			MaxSize:                1000,
			FlushInterval:          30 * time.Second,
			BatchSize:              100,
			FlushThreshold:         500,
			PlatformBufferSize:     5000,
			InterfaceBufferSize:    5000,
			SubinterfaceBufferSize: 5000,
		},
		DatabaseWriter: DatabaseWriterConfig{
			BatchTimeout:              5 * time.Second,
			RetryAttempts:             3,
			RetryDelay:                1 * time.Second,
			ParallelWriters:           1,
			MaxBatchSize:              1000,
			EnableParallelTableWrites: false,
			PlatformWriterCount:       1,
			InterfaceWriterCount:      1,
			SubinterfaceWriterCount:   1,
		},
		Memory: MemoryConfig{
			MaxMemoryUsage:  "2GB",
			GCTargetPercent: 100,
			BufferPoolSize:  100,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			FilePath:   "logs/telemetry.log",
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 3,
			Compress:   false,
		},
		Monitoring: MonitoringConfig{
			Enabled:           false,
			MetricsInterval:   30 * time.Second,
			HealthCheckPort:   8080,
			PrometheusEnabled: false,
			PrometheusPort:    9090,
			AlertThresholds: AlertThresholdsConfig{
				BufferUsagePercent:       80,
				DBConnectionUsagePercent: 85,
				MemoryUsagePercent:       90,
				ErrorRatePerMinute:       100,
			},
		},
		Performance: PerformanceConfig{
			GCPercent:       100,
			MaxProcs:        4,
			EnablePprof:     false,
			PprofPort:       6060,
			TCPKeepalive:    true,
			TCPNoDelay:      true,
			ReadBufferSize:  32768,
			WriteBufferSize: 32768,
		},
		Compression: CompressionConfig{
			Enabled:   false,
			Algorithm: "gzip",
			Level:     6,
		},
		Recovery: RecoveryConfig{
			EnableDataPersistence: false,
			PersistencePath:       "data/recovery",
			MaxRecoveryTime:       5 * time.Minute,
			CheckpointInterval:    30 * time.Second,
		},
		Debug: DebugConfig{
			Enabled:            false,
			LogRawData:         false,
			MetricsEnabled:     true,
			ProfileEnabled:     false,
			SlowQueryThreshold: 1 * time.Second,
		},
	}

	// 如果配置文件存在，则加载
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %v", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %v", err)
		}
	}

	return config, nil
}

// GetDSN 获取数据库连接字符串
func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode, d.Schema)
}