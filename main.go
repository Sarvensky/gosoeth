package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Sarvensky/gosoeth/internal/config"
	"github.com/Sarvensky/gosoeth/internal/logger"
	"github.com/Sarvensky/gosoeth/internal/proxy"
)

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

	if len(cfgProxies) == 0 {
		logger.Error("В конфиге не описано ни одной секции прокси")
		os.Exit(1)
	}

	// 3. Контекст для корректного завершения
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 4. Запуск прокси-серверов
	for _, p := range cfgProxies {
		wg.Add(1)
		go func(pc config.Proxy) {
			defer wg.Done()
			proxy.Run(ctx, pc, output)
		}(p)
	}

	// 5. Ожидание системного сигнала прерывания
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Завершение работы по сигналу...")
	cancel()
	wg.Wait()
	logger.Info("Программа успешно остановлена.")
}
