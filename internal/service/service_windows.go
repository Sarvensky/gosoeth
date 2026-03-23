//go:build windows

package service

import (
	"context"

	"golang.org/x/sys/windows/svc"
)

// IsService определяет, запущен ли процесс как служба Windows (через SCM).
func IsService() bool {
	is, _ := svc.IsWindowsService()
	return is
}

// gosoethService реализует интерфейс svc.Handler для обработки команд SCM.
type gosoethService struct {
	start func(ctx context.Context)
}

// Execute обрабатывает жизненный цикл службы: старт, обработка команд SCM, остановка.
func (s *gosoethService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск основной логики в отдельной горутине
	done := make(chan struct{})
	go func() {
		s.start(ctx)
		close(done)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				<-done
				return false, 0
			}
		case <-done:
			return false, 0
		}
	}
}

// Run запускает приложение как службу Windows с указанным именем.
// start — функция с основной логикой, которая должна завершиться при отмене ctx.
func Run(name string, start func(ctx context.Context)) error {
	return svc.Run(name, &gosoethService{start: start})
}
