package network

import (
	"fmt"
	"net"
)

// CheckInterfaceExists проверяет, существует ли сетевой адаптер по имени или привязан ли IP-адрес к одному из них
func CheckInterfaceExists(input string) error {
	if ip := net.ParseIP(input); ip != nil {
		ifaces, err := net.Interfaces()
		if err != nil {
			return fmt.Errorf("ошибка получения списка интерфейсов: %v", err)
		}
		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var boundIP net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					boundIP = v.IP
				case *net.IPAddr:
					boundIP = v.IP
				}
				if boundIP != nil && boundIP.Equal(ip) {
					return nil
				}
			}
		}
		return fmt.Errorf("IP-адрес '%s' не привязан ни к одному адаптеру", input)
	}

	_, err := net.InterfaceByName(input)
	if err != nil {
		return fmt.Errorf("сетевой адаптер '%s' не найден", input)
	}
	return nil
}

// GetIPFromInterface возвращает IP-адрес по имени интерфейса или проверяет, является ли строка IP
func GetIPFromInterface(input string) (string, error) {
	if net.ParseIP(input) != nil {
		return input, nil
	}

	iface, err := net.InterfaceByName(input)
	if err != nil {
		return "", fmt.Errorf("интерфейс '%s' не найден", input)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("IPv4 не найден на '%s'", input)
}
