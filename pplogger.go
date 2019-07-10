package pplogger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

type Config struct {
	StdoutWriter bool   // 是否打印到控制台
	FileWriter   bool   // 是否写到文件中
	LogPath      string // 日志文件路径
	Filename     string // 日志文件名称
	LogLevel     string // 日志输出等级
	MaxSize      int    // 单个文件最大限制，单位 M
	MaxBackups   int    // 最多保留备份数
	MaxAge       int    // 最多保留天数
	Compress     bool   // 是否压缩
}

const (
	DebugLevel  = "Debug"
	InfoLevel   = "Info"
	WarnLevel   = "Warn"
	ErrorLevel  = "Error"
	DPanicLevel = "DPanic"
	PanicLevel  = "Panic"
	FatalLevel  = "Fatal"
)

func NewEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func getFileLogger(config Config) lumberjack.Logger {

	return lumberjack.Logger{
		Filename:   filepath.Join(config.LogPath, config.Filename),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
}

func getLogLevel(str string) zapcore.Level {
	var level zapcore.Level
	switch str {
	case DebugLevel:
		level = zap.DebugLevel
	case InfoLevel:
		level = zap.InfoLevel
	case WarnLevel:
		level = zap.WarnLevel
	case ErrorLevel:
		level = zap.ErrorLevel
	case DPanicLevel:
		level = zap.DPanicLevel
	case PanicLevel:
		level = zap.PanicLevel
	case FatalLevel:
		level = zap.FatalLevel
	default:
		level = zap.InfoLevel
	}

	return level
}

func NewPPLogger(config Config) (*zap.Logger, *zap.SugaredLogger) {

	// 设置默认值

	if config.MaxSize == 0 {
		config.MaxSize = 500
	}

	if config.MaxBackups == 0 {
		config.MaxBackups = 3
	}

	if config.MaxAge == 0 {
		config.MaxAge = 30
	}

	logPath := config.LogPath

	if logPath == "" || logPath == "./" {
		logPath = "./logs"
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		absRegexp, _ := regexp.Compile(`^(/|([a-zA-Z]:\\)).*`)
		if !absRegexp.MatchString(logPath) {
			_, currentFilePath, _, _ := runtime.Caller(1)
			workPath := filepath.Join(filepath.Dir(currentFilePath), "../")
			logPath = filepath.Join(workPath, logPath)
		}
	}

	if err := os.MkdirAll(logPath, os.ModePerm); err != nil {
		log.Fatal("foundation logger: ", err)
	}

	config.LogPath = logPath

	var multiWriters zapcore.WriteSyncer

	if config.StdoutWriter && config.FileWriter {
		fileLogger := getFileLogger(config)
		multiWriters = zapcore.NewMultiWriteSyncer(zapcore.AddSync(&fileLogger), zapcore.AddSync(os.Stdout))
	} else if config.FileWriter {
		fileLogger := getFileLogger(config)
		multiWriters = zapcore.NewMultiWriteSyncer(zapcore.AddSync(&fileLogger))
	} else if config.StdoutWriter {
		multiWriters = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))
	} else {
		log.Fatal("Logfile and Stdout must have one set to true")
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(NewEncoderConfig()),
		multiWriters,
		getLogLevel(config.LogLevel),
	)

	opts := []zap.Option{zap.AddCaller()}
	opts = append(opts, zap.AddStacktrace(zap.ErrorLevel))
	opts = append(opts, zap.AddCallerSkip(0))
	logger := zap.New(core, opts...)
	sugar := logger.Sugar()

	return logger, sugar
}

func NewPPLoggerLite(fileName string, logLevel string) (*zap.Logger, *zap.SugaredLogger) {
	if fileName == "" {
		fileName = "./logs/pplogger.log"
	}
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     30, // days
	})
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(NewEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout),
			w),
		getLogLevel(logLevel),
	)

	logger := zap.New(core, zap.AddCaller())
	sugar := logger.Sugar()
	return logger, sugar
}