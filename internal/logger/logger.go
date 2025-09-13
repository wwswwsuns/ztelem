package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger 创建新的日志记录器
func NewLogger(debug bool) *logrus.Logger {
	log := logrus.New()

	// 设置日志级别
	if debug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	// 设置日志格式
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.WithError(err).Warn("创建日志目录失败")
	}

	// 设置日志输出到文件和控制台
	logFile := filepath.Join(logDir, "telemetry.log")
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     30, // days
		Compress:   true,
	}

	// 同时输出到文件和控制台
	log.SetOutput(fileWriter)
	if debug {
		// 调试模式下同时输出到控制台
		log.AddHook(&ConsoleHook{})
	}

	return log
}

// ConsoleHook 控制台输出钩子
type ConsoleHook struct{}

func (hook *ConsoleHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	os.Stdout.Write([]byte(line))
	return nil
}

func (hook *ConsoleHook) Levels() []logrus.Level {
	return logrus.AllLevels
}