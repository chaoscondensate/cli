package netpolicy

import (
	"net"
	"testing"
)

func TestPublicIPPolicy(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"8.8.8.8", true}, {"2606:4700:4700::1111", true},
		{"127.0.0.1", false}, {"::1", false}, {"10.0.0.1", false}, {"172.16.0.1", false}, {"192.168.0.1", false},
		{"::ffff:127.0.0.1", false}, {"::ffff:100.64.0.1", false}, {"100.64.0.1", false},
		{"198.18.0.1", false}, {"192.0.2.1", false}, {"198.51.100.1", false}, {"203.0.113.1", false},
		{"0.0.0.0", false}, {"169.254.1.1", false}, {"224.0.0.1", false}, {"240.0.0.1", false},
		{"fc00::1", false}, {"fe80::1", false}, {"ff02::1", false}, {"2001:db8::1", false},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if got := PublicIP(net.ParseIP(test.value)); got != test.want {
				t.Fatalf("PublicIP(%s)=%v, want %v", test.value, got, test.want)
			}
		})
	}
}
