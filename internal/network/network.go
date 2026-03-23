package network

import (
	"fmt"
	"net"
)

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
