package netcode

import (
	"fmt"
	"testing"
	"time"

	"inet.af/netaddr"
)

const testClientCommsEnabled = false // this is for testing servers independently

func TestClientInit(t *testing.T) {
	ip, _ := netaddr.ParseIP("127.0.0.1")
	server := netaddr.IPPort{IP: ip, Port: 40000}
	servers := make([]netaddr.IPPort, 1)
	servers[0] = server

	connectToken := testGenerateConnectToken(servers, TEST_PRIVATE_KEY, t)

	c := NewClient(connectToken)
	if err := c.Connect(); err != nil {
		t.Fatalf("error connecting: %s\n", err)
	}

	if c.conn.RemoteAddr().String() != "127.0.0.1:40000" {
		t.Fatalf("remote address was incorrect.")
	}

}

func TestClientCommunications(t *testing.T) {
	if !testClientCommsEnabled {
		return
	}
	ip, _ := netaddr.ParseIP("::1")
	server := netaddr.IPPort{IP: ip, Port: 40000}
	servers := make([]netaddr.IPPort, 1)
	servers[0] = server

	connectToken := testGenerateConnectToken(servers, TEST_PRIVATE_KEY, t)

	clientTime := float64(0)
	delta := float64(1.0 / 60.0)
	deltaTime := time.Duration(delta * float64(time.Second))

	c := NewClient(connectToken)

	if err := c.Connect(); err != nil {
		t.Fatalf("error connecting: %s\n", err)
	}

	packetData := make([]byte, MAX_PAYLOAD_BYTES)
	for i := 0; i < MAX_PAYLOAD_BYTES; i += 1 {
		packetData[i] = byte(i)
	}
	count := 0

	// fake game loop
	for {
		if count == 20 {
			c.Close()
			t.Fatalf("never recv'd a payload packet")
			return
		}
		c.Update(clientTime)
		if c.GetState() == StateConnected {
			c.SendData(packetData)
		}

		for {
			if payload, seq := c.RecvData(); payload == nil {
				break
			} else {
				fmt.Printf("seq: %d recv'd payload: of %d bytes\n", seq, len(payload))
				return
			}
		}
		time.Sleep(deltaTime)
		clientTime += deltaTime.Seconds()
		count++
	}
}
