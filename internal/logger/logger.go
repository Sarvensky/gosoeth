package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	FileLogger *log.Logger
	Verbose    bool
)

// Setup настроит логирование
func Setup(logDir string, verbose bool, maxBackups int) (io.Writer, error) {
	Verbose = verbose
	
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать папку логов: %v", err)
	}

	fileName := time.Now().Format("2006-01-02") + ".log"
	filePath := filepath.Join(logDir, fileName)

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл лога: %v", err)
	}

	// FileLogger будет писать ТОЛЬКО в файл (для системных событий)
	FileLogger = log.New(file, "", log.LstdFlags)

	if Verbose {
		// Если включен verbose, всё (включая DNS) дублируем в файл
		mw := io.MultiWriter(os.Stdout, file)
		log.SetOutput(mw)
	} else {
		// Если нет - DNS только в консоль
		log.SetOutput(os.Stdout)
	}

	go cleanOldLogs(logDir, maxBackups)

	return os.Stdout, nil
}

func cleanOldLogs(dir string, maxDays int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, f := range files {
		if f.IsDir() { continue }
		info, err := f.Info()
		if err != nil { continue }
		if now.Sub(info.ModTime()).Hours() > float64(24*maxDays) {
			os.Remove(filepath.Join(dir, f.Name()))
		}
	}
}

// Info пишет и в консоль, и в файл (всегда)
func Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	// Если verbose выключен, мы должны вручную писать и в консоль, и в файл
	// Если включен, log.Print и так сделает обе записи (но тогда будет дубль в файле)
	// Решение: пишем в консоль через log.Print, а в файл через FileLogger
	// Но если Verbose=true, log.SetOutput уже настроен на MultiWriter
	log.Print("INFO: " + msg)
	if !Verbose && FileLogger != nil {
		FileLogger.Print("INFO: " + msg)
	}
}

// Error пишет и в консоль, и в файл (всегда)
func Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Print("ERROR: " + msg)
	if !Verbose && FileLogger != nil {
		FileLogger.Print("ERROR: " + msg)
	}
}
