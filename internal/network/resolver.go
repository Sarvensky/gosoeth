package network

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/http2"
)

// bootstrapEntry хранит закэшированный IP DNS-сервера и время истечения кэша.
type bootstrapEntry struct {
	ip        string
	expiresAt time.Time
}

// interfaceCache хранит закэшированный IP сетевого адаптера и время истечения кэша.
type interfaceCache struct {
	ip        string
	expiresAt time.Time
}

// CustomResolver реализует интерфейс socks5.NameResolver.
// Резолвит доменные имена через DoH, DoT или UDP DNS, привязываясь к конкретному сетевому адаптеру.
type CustomResolver struct {
	InterfaceName string
	IPv6Disable   bool
	DNSBasic      string
	DNSCrypto     string

	mu             sync.RWMutex
	bootstrapIPs   map[string]bootstrapEntry
	ifaceCache     interfaceCache
	httpClient     *http.Client
}

// GetCachedInterfaceIP возвращает IP адаптера с кэшированием на 10 секунд.
// Использует double-check locking для потокобезопасности.
func (r *CustomResolver) GetCachedInterfaceIP() (string, error) {
	r.mu.RLock()
	if time.Now().Before(r.ifaceCache.expiresAt) && r.ifaceCache.ip != "" {
		ip := r.ifaceCache.ip
		r.mu.RUnlock()
		return ip, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Повторная проверка после захвата Lock
	if time.Now().Before(r.ifaceCache.expiresAt) && r.ifaceCache.ip != "" {
		return r.ifaceCache.ip, nil
	}

	ip, err := GetIPFromInterface(r.InterfaceName)
	if err != nil {
		return "", err
	}

	r.ifaceCache = interfaceCache{
		ip:        ip,
		expiresAt: time.Now().Add(10 * time.Second),
	}
	return ip, nil
}

// Resolve резолвит доменное имя в IP-адрес.
// Порядок попыток: DoH/DoT -> UDP DNS -> системный резолвер.
func (r *CustomResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	if ip := net.ParseIP(name); ip != nil {
		return ctx, ip, nil
	}

	outIP, err := r.GetCachedInterfaceIP()
	if err != nil {
		return ctx, nil, err
	}

	var resolvedIP net.IP

	if r.DNSCrypto != "" {
		if strings.HasPrefix(r.DNSCrypto, "https://") {
			log.Printf("[DNS] DoH запрос: %s", name)
			resolvedIP, err = r.resolveDoH(name, outIP)
		} else if strings.HasPrefix(r.DNSCrypto, "tls://") || strings.Contains(r.DNSCrypto, ":853") {
			log.Printf("[DNS] DoT запрос: %s", name)
			resolvedIP, err = r.resolveDoT(name, outIP)
		}
		
		if err == nil && resolvedIP != nil {
			return ctx, resolvedIP, nil
		}
	}

	if r.DNSBasic != "" {
		log.Printf("[DNS] UDP запрос (%s): %s", r.DNSBasic, name)
		resolvedIP, err = r.resolveUDP(name, r.DNSBasic, outIP)
		if err == nil && resolvedIP != nil {
			return ctx, resolvedIP, nil
		}
	}

	resolvedIP, err = r.resolveSystem(ctx, name, outIP)
	return ctx, resolvedIP, err
}

// getBootstrapIP резолвит IP адрес DNS-сервера через публичные UDP DNS (1.1.1.1, 8.8.8.8, 77.88.8.8).
// Результат кэшируется на 1 час для предотвращения рекурсии DoH/DoT -> DNS -> DoH/DoT.
func (r *CustomResolver) getBootstrapIP(host string, localIP string) (string, error) {
	r.mu.RLock()
	entry, ok := r.bootstrapIPs[host]
	if ok && time.Now().Before(entry.expiresAt) {
		ip := entry.ip
		r.mu.RUnlock()
		return ip, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bootstrapIPs == nil {
		r.bootstrapIPs = make(map[string]bootstrapEntry)
	}

	log.Printf("[DNS] Bootstrap резолв для %s...", host)
	resolvers := []string{"1.1.1.1:53", "8.8.8.8:53", "77.88.8.8:53"}
	for _, res := range resolvers {
		ip, err := r.resolveUDP(host, res, localIP)
		if err == nil {
			r.bootstrapIPs[host] = bootstrapEntry{
				ip:        ip.String(),
				expiresAt: time.Now().Add(1 * time.Hour), // Кэшируем на час
			}
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("bootstrap fail")
}

// resolveDoH выполняет DNS-запрос через DNS-over-HTTPS (RFC 8484).
// HTTP-клиент создаётся один раз и переиспользуется для всех последующих запросов.
func (r *CustomResolver) resolveDoH(name string, localIP string) (net.IP, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), dns.TypeA)
	pack, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("ошибка упаковки DNS-запроса: %v", err)
	}
	dnsParam := base64.RawURLEncoding.EncodeToString(pack)
	
	u := strings.TrimPrefix(r.DNSCrypto, "https://")
	idx := strings.Index(u, "/")
	if idx == -1 { idx = len(u) }
	hostName := u[:idx]

	serverIP, err := r.getBootstrapIP(hostName, localIP)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if r.httpClient == nil {
		dialer := &net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(localIP)}, Timeout: 2 * time.Second}
		t := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if strings.HasPrefix(addr, hostName) {
					addr = serverIP + ":443"
				}
				return dialer.DialContext(ctx, network, addr)
			},
			TLSClientConfig: &tls.Config{
				ServerName: hostName,
				NextProtos: []string{"h2", "http/1.1"},
			},
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		}
		_ = http2.ConfigureTransport(t)
		r.httpClient = &http.Client{Transport: t, Timeout: 5 * time.Second}
	}
	r.mu.Unlock()

	finalURL := r.DNSCrypto
	if strings.Contains(finalURL, "?") {
		finalURL += "&dns=" + dnsParam
	} else {
		finalURL += "?dns=" + dnsParam
	}

	req, err := http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP-запроса: %v", err)
	}
	req.Header.Set("Accept", "application/dns-message")
	req.Header.Set("User-Agent", "gosoeth/1.0")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа DoH: %v", err)
	}
	msgResp := new(dns.Msg)
	if err := msgResp.Unpack(body); err != nil {
		return nil, err
	}

	for _, answer := range msgResp.Answer {
		if a, ok := answer.(*dns.A); ok {
			return a.A, nil
		}
	}
	return nil, fmt.Errorf("IP не найден")
}

// resolveDoT выполняет DNS-запрос через DNS-over-TLS (RFC 7858).
// ВАЖНО: в отличие от DoH, каждый вызов создаёт новое TLS-соединение,
// что создаёт бо́льшую нагрузку. При высоком трафике рекомендуется использовать DoH.
func (r *CustomResolver) resolveDoT(name, localIP string) (net.IP, error) {
	server := strings.TrimPrefix(r.DNSCrypto, "tls://")
	port := "853"
	if strings.Contains(server, ":") {
		parts := strings.Split(server, ":")
		server = parts[0]
		port = parts[1]
	}

	serverIP, err := r.getBootstrapIP(server, localIP)
	if err != nil {
		return nil, err
	}

	c := new(dns.Client)
	c.Net = "tcp-tls"
	c.Dialer = &net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(localIP)}, Timeout: 2 * time.Second}
	c.TLSConfig = &tls.Config{ServerName: server}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), dns.TypeA)
	in, _, err := c.Exchange(m, serverIP+":"+port)
	if err != nil {
		return nil, err
	}

	for _, answer := range in.Answer {
		if a, ok := answer.(*dns.A); ok {
			return a.A, nil
		}
	}
	return nil, fmt.Errorf("IP не найден")
}

// resolveUDP выполняет обычный DNS-запрос через UDP (порт 53).
func (r *CustomResolver) resolveUDP(name, server, localIP string) (net.IP, error) {
	c := new(dns.Client)
	c.Dialer = &net.Dialer{LocalAddr: &net.UDPAddr{IP: net.ParseIP(localIP)}, Timeout: 2 * time.Second}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), dns.TypeA)
	in, _, err := c.Exchange(m, server)
	if err != nil {
		return nil, err
	}
	for _, answer := range in.Answer {
		if a, ok := answer.(*dns.A); ok {
			return a.A, nil
		}
	}
	return nil, fmt.Errorf("не найден")
}

// resolveSystem использует системный DNS-резолвер как последний вариант.
// Привязывается к локальному IP адаптера для маршрутизации через нужный интерфейс.
func (r *CustomResolver) resolveSystem(ctx context.Context, name, localIP string) (net.IP, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{LocalAddr: &net.UDPAddr{IP: net.ParseIP(localIP)}, Timeout: 2 * time.Second}
			return d.DialContext(ctx, network, address)
		},
	}
	ips, err := resolver.LookupIP(ctx, "ip4", name)
	if err != nil || len(ips) == 0 {
		return nil, err
	}
	return ips[0], nil
}
