package client

import (
	"net"
)

// IPResolver returns the IP specified
type IPResolver struct{}

// DNSResolver returns IPs based on a DNS name
type DNSResolver struct{}

func (ip *IPResolver) Resolve(name string) ([]string, error) {
	return []string{name}, nil
}

func (dns *DNSResolver) Resolve(name string) ([]string, error) {
	ips, err := net.LookupIP(name)
	if err != nil {
		return nil, err
	}

	var addrs []string

	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			addrs = append(addrs, ipv4.String())
		}
	}

	return addrs, nil
}
