package config

import (
	"fmt"
	"log"
	"strconv"

	"gopkg.in/ini.v1"
)

// Proxy содержит параметры конфигурации одного SOCKS5 прокси-сервера.
type Proxy struct {
	Name       string
	ListenPort int
	Interface  string
	IPv6Enable bool
	DNSBasic   string
	DNSCrypto  string
}

// Global содержит глобальные настройки приложения (логирование и т.д.).
type Global struct {
	LogDir        string
	LogMaxBackups int
	VerboseLog    bool
}

// Load читает и парсит конфиг
func Load(path string) (Global, []Proxy, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return Global{}, nil, fmt.Errorf("ошибка загрузки %s: %v", path, err)
	}

	global := Global{
		LogDir:        cfg.Section("SocksToEth").Key("log_dir").MustString("Logs"),
		LogMaxBackups: cfg.Section("SocksToEth").Key("log_max_backups").MustInt(30),
		VerboseLog:    cfg.Section("SocksToEth").Key("verbose_log").MustBool(false),
	}

	var proxies []Proxy
	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == "DEFAULT" || name == "SocksToEth" {
			continue
		}

		portStr := section.Key("listen_port").String()
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			log.Printf("ERROR: [%s] Невалидный порт '%s', секция пропущена", name, portStr)
			continue
		}

		proxies = append(proxies, Proxy{
			Name:       name,
			ListenPort: port,
			Interface:  section.Key("interface").String(),
			IPv6Enable: section.Key("ipv6_enable").MustBool(false),
			DNSBasic:   section.Key("dns_basic").String(),
			DNSCrypto:  section.Key("dns_crypto").String(),
		})
	}

	return global, proxies, nil
}
