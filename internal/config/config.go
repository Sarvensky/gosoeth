package config

import (
	"fmt"
	"strconv"

	"gopkg.in/ini.v1"
)

type Proxy struct {
	Name       string
	ListenPort int
	Interface  string
	IPv6Enable bool
	DNSBasic   string
	DNSCrypto  string
}

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
		port, _ := strconv.Atoi(portStr)
		if port == 0 {
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
