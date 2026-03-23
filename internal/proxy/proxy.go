package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/armon/go-socks5"
	"github.com/Sarvensky/gosoeth/internal/config"
	"github.com/Sarvensky/gosoeth/internal/logger"
	"github.com/Sarvensky/gosoeth/internal/network" // Используется для создания CustomResolver
)

// Run запускает инстанс SOCKS5 сервера для указанной конфигурации прокси.
// Сервер работает до отмены контекста или критической ошибки.
func Run(ctx context.Context, p config.Proxy, output io.Writer) {
	logger.Info("[%s] СТАРТ: Порт %d -> Адаптер %s (IPv6:%t)", p.Name, p.ListenPort, p.Interface, p.IPv6Enable)

	customResolver := &network.CustomResolver{
		InterfaceName: p.Interface,
		IPv6Disable:   !p.IPv6Enable,
		DNSBasic:      p.DNSBasic,
		DNSCrypto:     p.DNSCrypto,
	}

	conf := &socks5.Config{
		Resolver: customResolver,
		// Dial привязывает исходящие TCP-соединения к IP адаптера.
		// Использует кэшированный IP из CustomResolver, чтобы не делать
		// системный вызов на каждое подключение.
		Dial: func(ctx context.Context, netw, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Блокировка IPv6 если отключен в конфиге
			ip := net.ParseIP(host)
			if !p.IPv6Enable && ip != nil && ip.To4() == nil {
				return nil, fmt.Errorf("IPv6 заблокирован")
			}

			outIP, err := customResolver.GetCachedInterfaceIP()
			if err != nil {
				return nil, err
			}

			localAddr, err := net.ResolveTCPAddr("tcp", outIP+":0")
			if err != nil {
				return nil, err
			}

			dialer := &net.Dialer{LocalAddr: localAddr}
			return dialer.DialContext(ctx, netw, addr)
		},
		Logger: log.New(io.Discard, "", 0),
	}

	server, err := socks5.New(conf)
	if err != nil {
		logger.Error("[%s] ОШИБКА инициализации: %v", p.Name, err)
		return
	}

	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe("tcp", fmt.Sprintf("0.0.0.0:%d", p.ListenPort)); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("[%s] ОСТАНОВКА: Порт %d выключен", p.Name, p.ListenPort)
	case err := <-errChan:
		logger.Error("[%s] КРИТИЧЕСКАЯ ОШИБКА: %v", p.Name, err)
	}
}
