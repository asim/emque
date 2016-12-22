package client

import (
	"testing"
)

func TestIPResolver(t *testing.T) {
	ip := new(IPResolver)
	ips, err := ip.Resolve("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) == 0 {
		t.Fatal("expected ip got", ips)
	}
	if ips[0] != "127.0.0.1" {
		t.Fatal("expected 127.0.0.1 got", ips[0])
	}
}

func TestDNSResolver(t *testing.T) {
	dns := new(DNSResolver)

	ips, err := dns.Resolve("localhost")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) == 0 {
		t.Fatal("expected ip got", ips)
	}
}
