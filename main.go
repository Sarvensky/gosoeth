package main

import (
	"context"
	_ "embed"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Sarvensky/gosoeth/internal/config"
	"github.com/Sarvensky/gosoeth/internal/logger"
	"github.com/Sarvensky/gosoeth/internal/proxy"
	"github.com/Sarvensky/gosoeth/internal/service"
)

//go:embed version.txt
var appVersionRaw string

// AppVersion содержит версию приложения
var AppVersion = strings.TrimSpace(appVersionRaw)

func main() {
	// 1. Загрузка конфигурации
	cfgGlobal, cfgProxies, err := config.Load("config.ini")
	if err != nil {
		log.Fatalf("Критическая ошибка: %v", err)
	}

	// 2. Настройка логирования
	output, err := logger.Setup(cfgGlobal.LogDir, cfgGlobal.VerboseLog, cfgGlobal.LogMaxBackups)
	if err != nil {
		log.Fatalf("Ошибка настройки логов: %v", err)
	}

	logger.Info("=== Запуск gosoeth v%s ===", AppVersion)

	if len(cfgProxies) == 0 {
		logger.Error("В конфиге не описано ни одной секции прокси")
		os.Exit(1)
	}

	// 3. Выбор режима запуска: служба Windows или интерактивный
	if service.IsService() {
		logger.Info("Запуск в режиме службы Windows...")
		err := service.Run("gosoeth", func(ctx context.Context) {
			startProxies(ctx, cfgProxies, output)
		})
		if err != nil {
			logger.Error("Ошибка работы службы: %v", err)
		}
	} else {
		runInteractive(cfgProxies, output)
	}
}

// startProxies запускает все прокси-серверы и блокируется до отмены контекста.
func startProxies(ctx context.Context, proxies []config.Proxy, output io.Writer) {
	var wg sync.WaitGroup
	for _, p := range proxies {
		wg.Add(1)
		go func(pc config.Proxy) {
			defer wg.Done()
			proxy.Run(ctx, pc, output)
		}(p)
	}
	wg.Wait()
}

// runInteractive запускает программу в интерактивном режиме с ожиданием сигнала остановки.
func runInteractive(proxies []config.Proxy, output io.Writer) {
	ctx, cancel := context.WithCancel(context.Background())

	// Запуск прокси в фоне, канал done сигнализирует о завершении всех серверов
	done := make(chan struct{})
	go func() {
		startProxies(ctx, proxies, output)
		close(done)
	}()

	// Ожидание системного сигнала прерывания
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Завершение работы по сигналу...")
	cancel()
	<-done
	logger.Info("Программа успешно остановлена.")
}
