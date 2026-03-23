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
	// FileLogger пишет только в файл лога (для системных событий: старт, стоп, ошибки).
	FileLogger *log.Logger
	// Verbose определяет, дублировать ли весь вывод (включая DNS) в файл.
	Verbose    bool
)

// Setup настраивает логирование: создаёт папку, открывает файл лога и запускает очистку старых логов.
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

// cleanOldLogs удаляет файлы логов старше maxDays дней по дате модификации.
func cleanOldLogs(dir string, maxDays int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()).Hours() > float64(24*maxDays) {
			os.Remove(filepath.Join(dir, f.Name()))
		}
	}
}

// Info пишет и в консоль, и в файл (всегда)
func Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
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
