package resolver

import (
	"net"
)

// IP resolver returns the IP specified
type IP struct{}

// DNS resolver returns IPs based on a DNS name
type DNS struct{}

func (ip *IP) Resolve(name string) ([]string, error) {
	return []string{name}, nil
}

func (dns *DNS) Resolve(name string) ([]string, error) {
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
