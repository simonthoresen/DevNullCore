package invite

import (
	"encoding/base64"
	"encoding/binary"
	"net"
	"testing"
)

// encodeFixture mirrors internal/server.(*Server).inviteToken so we can
// exercise Decode against tokens shaped exactly like the server emits.
func encodeFixture(t *testing.T, sshPort uint16, lanIP, publicIP net.IP, pinggyHost string, pinggyPort uint16) string {
	t.Helper()
	hasPinggy := pinggyPort != 0 && pinggyHost != ""
	hasPublic := publicIP != nil && !publicIP.Equal(net.IPv4zero)
	needLAN := hasPublic || hasPinggy

	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf[0:2], sshPort)

	if needLAN || lanIP != nil {
		ip := lanIP
		if ip == nil {
			ip = net.IPv4zero.To4()
		}
		buf = append(buf, ip.To4()...)
	}
	if hasPublic || hasPinggy {
		ip := publicIP
		if ip == nil {
			ip = net.IPv4zero.To4()
		}
		buf = append(buf, ip.To4()...)
	}
	if hasPinggy {
		pp := make([]byte, 2)
		binary.BigEndian.PutUint16(pp, pinggyPort)
		buf = append(buf, pp...)
		buf = append(buf, []byte(pinggyHost)...)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func TestDecodePortOnly(t *testing.T) {
	tok := encodeFixture(t, 23234, nil, nil, "", 0)
	got, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	want := []Endpoint{{Host: "localhost", Port: 23234}}
	if !equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestDecodeWithLAN(t *testing.T) {
	tok := encodeFixture(t, 23234, net.IPv4(192, 168, 1, 50), nil, "", 0)
	got, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	want := []Endpoint{
		{Host: "localhost", Port: 23234},
		{Host: "192.168.1.50", Port: 23234},
	}
	if !equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestDecodeWithPublic(t *testing.T) {
	tok := encodeFixture(t, 23234, net.IPv4(192, 168, 1, 50), net.IPv4(203, 0, 113, 7), "", 0)
	got, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	want := []Endpoint{
		{Host: "localhost", Port: 23234},
		{Host: "192.168.1.50", Port: 23234},
		{Host: "203.0.113.7", Port: 23234},
	}
	if !equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestDecodeWithPinggy(t *testing.T) {
	tok := encodeFixture(t, 23234, net.IPv4(192, 168, 1, 50), net.IPv4(203, 0, 113, 7), "abc.a.pinggy.link", 51234)
	got, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	want := []Endpoint{
		{Host: "localhost", Port: 23234},
		{Host: "192.168.1.50", Port: 23234},
		{Host: "203.0.113.7", Port: 23234},
		{Host: "abc.a.pinggy.link", Port: 51234},
	}
	if !equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestDecodeSkipsZeroIP(t *testing.T) {
	// LAN absent (0.0.0.0) but Pinggy present: localhost + Pinggy only.
	tok := encodeFixture(t, 23234, net.IPv4zero, net.IPv4zero, "abc.a.pinggy.link", 51234)
	got, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	want := []Endpoint{
		{Host: "localhost", Port: 23234},
		{Host: "abc.a.pinggy.link", Port: 51234},
	}
	if !equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestDecodeRejectsEmpty(t *testing.T) {
	if _, err := Decode(""); err == nil {
		t.Fatal("expected error on empty token")
	}
}

func TestDecodeRejectsTooShort(t *testing.T) {
	short := base64.RawURLEncoding.EncodeToString([]byte{0x01})
	if _, err := Decode(short); err == nil {
		t.Fatal("expected error on 1-byte token")
	}
}

func equal(a, b []Endpoint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
