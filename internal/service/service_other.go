//go:build !windows

package service

import (
	"context"
)

// IsService на Linux/Mac всегда возвращает false.
// Systemd управляет процессом извне через SIGTERM, специальный код не нужен.
func IsService() bool {
	return false
}

// Run — заглушка для не-Windows платформ.
// На Linux/Mac этот метод не вызывается, т.к. IsService() всегда false.
func Run(name string, start func(ctx context.Context)) error {
	return nil
}
