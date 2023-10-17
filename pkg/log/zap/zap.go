// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zap

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Environment variable key
const (
	LogFilename         = "LOG_FILENAME"
	LogStorageDirectory = "LOG_STORAGE_DIRECTORY"
	LogStoragePolicy    = "LOG_STORAGE_POLICY"
	LogMaxSize          = "LOG_MAX_SIZE"
	LogMaxBackups       = "LOG_MAX_BACKUPS"
	LogMaxAge           = "LOG_MAX_AGE"
	LogCompress         = "LOG_COMPRESS"
)

var (
	// Log storage directory. default is current project root directory
	// if your need set environment variable LOG_STORAGE_DIRECTORY value. eg: /var
	logStorageDirectory = isEnvDefault(LogStorageDirectory, ".").(string)
	// Current log file contains log below error
	currentFilePath = fmt.Sprintf("%s/log/current.log", logStorageDirectory)
	// Error log file contains equal to error and more
	errorFilePath = fmt.Sprintf("%s/log/error.log", logStorageDirectory)
	// Log storage policy. default append write file
	// If your need set environment LOG_STORAGE_POLICY value. eg: debug
	// Current mode has debug and default, when write file will trunc in debug mode
	logStoragePolicy = isEnvDefault(LogStoragePolicy, "default").(string)
	// Log max size. default size is 10M
	logMaxSize = isEnvDefault(LogMaxSize, 10).(int)
	// Log max backups. default number is 5
	logMaxBackups = isEnvDefault(LogMaxBackups, 5).(int)
	// Log storage max age. default age is 30 days
	logMaxAge = isEnvDefault(LogMaxAge, 30).(int)
	// Whether the log is compressed. default is not
	logCompress = isEnvDefault(LogCompress, false).(bool)
)

func init() {
	if val := os.Getenv(LogFilename); val != "" {
		currentFilePath = fmt.Sprintf("%s/log/%s.log", logStorageDirectory, val)
	}
}

func New() logr.Logger {
	return zapr.NewLogger(NewZapLogger())
}

func GetEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "name",
		CallerKey:      "line",
		MessageKey:     "msg",
		FunctionKey:    "func",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	return zapcore.NewJSONEncoder(encoderConfig)
}

func NewZapLogger() *zap.Logger {
	cores := NewCore()
	return zap.New(zapcore.NewTee(cores...), zap.AddCaller())
}

func NewCore() []zapcore.Core {
	var coreArr []zapcore.Core
	var infoDir = filepath.Dir(currentFilePath)

	err := isCreateDefault(infoDir)
	if err != nil {
		fmt.Println(err)
	}

	mode := logStoragePolicy
	switch mode {
	case "debug":
		_, _ = os.OpenFile(currentFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
		_, _ = os.OpenFile(errorFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	default:
		_, _ = os.OpenFile(currentFilePath, os.O_WRONLY|os.O_APPEND, 0644)
		_, _ = os.OpenFile(errorFilePath, os.O_WRONLY|os.O_APPEND, 0644)
	}

	highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev >= zap.ErrorLevel
	})

	lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev < zap.ErrorLevel
	})

	infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   currentFilePath,
		MaxSize:    logMaxSize,
		MaxBackups: logMaxBackups,
		MaxAge:     logMaxAge,
		Compress:   logCompress,
	})
	infoFileCore := zapcore.NewCore(GetEncoder(), zapcore.NewMultiWriteSyncer(infoFileWriteSyncer, zapcore.AddSync(os.Stdout)), lowPriority)

	errorFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   errorFilePath,
		MaxSize:    logMaxSize,
		MaxBackups: logMaxBackups,
		MaxAge:     logMaxAge,
		Compress:   logCompress,
	})
	errorFileCore := zapcore.NewCore(GetEncoder(), zapcore.NewMultiWriteSyncer(errorFileWriteSyncer, zapcore.AddSync(os.Stdout)), highPriority)

	coreArr = append(coreArr, infoFileCore)
	coreArr = append(coreArr, errorFileCore)

	return coreArr
}

func isCreateDefault(opts ...string) error {
	for _, dir := range opts {
		_, err := os.Stat(dir)
		if err != nil {
			err = os.MkdirAll(dir, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func isEnvDefault(key string, defaultValue interface{}) interface{} {
	if val := os.Getenv(key); val != "" {
		if key == LogMaxAge || key == LogMaxBackups || key == LogMaxSize {
			intValue, _ := strconv.Atoi(val)
			return intValue
		}
		if key == LogCompress {
			boolValue, _ := strconv.ParseBool(val)
			return boolValue
		}

		return val
	}

	return defaultValue
}
