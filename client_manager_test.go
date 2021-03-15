package netcode

import (
	"testing"

	"inet.af/netaddr"
)

func TestNewClientManager(t *testing.T) {
	timeout := float64(4)
	maxClients := 2
	cm := NewClientManager(timeout, maxClients)

	if cm.FindFreeClientIndex() == -1 {
		t.Fatalf("free client index should not return -1 when empty")
	}

	addr := &netaddr.IPPort{}
	if cm.FindClientIndexByAddress(addr) != -1 {
		t.Fatalf("client index by empty address should return -1")
	}

	if cm.FindClientIndexById(0) != -1 {
		t.Fatalf("should not have any clients")
	}

}

func TestAddEncryptionMapping(t *testing.T) {
	timeout := float64(4)
	maxClients := 2
	servers := make([]netaddr.IPPort, 1)

	ip, _ := netaddr.ParseIP("::1")
	servers[0] = netaddr.IPPort{IP: ip, Port: 40000}

	addr := &netaddr.IPPort{IP: ip, Port: 62424}
	addr2 := &netaddr.IPPort{IP: ip, Port: 62425}
	overAddrs := make([]*netaddr.IPPort, (maxClients)*8)
	for i := 0; i < len(overAddrs); i++ {
		overAddrs[i] = &netaddr.IPPort{IP: ip, Port: uint16(6000 + i)}
	}

	connectToken := testGenerateConnectToken(servers, TEST_PRIVATE_KEY, t)

	cm := NewClientManager(timeout, maxClients)

	serverTime := float64(1.0)
	expireTime := float64(1.1)
	if !cm.AddEncryptionMapping(connectToken.PrivateData, addr, serverTime, expireTime) {
		t.Fatalf("error adding encryption mapping\n")
	}

	// add it again
	if !cm.AddEncryptionMapping(connectToken.PrivateData, addr, serverTime, expireTime) {
		t.Fatalf("error re-adding encryption mapping\n")
	}

	if !cm.AddEncryptionMapping(connectToken.PrivateData, addr2, serverTime, expireTime) {
		t.Fatalf("error adding 2nd encryption mapping\n")
	}

	failed := false
	for i := 0; i < len(overAddrs); i++ {
		if cm.AddEncryptionMapping(connectToken.PrivateData, overAddrs[i], serverTime, expireTime) {
			failed = true
		}
	}

	if !failed {
		t.Fatalf("error we added more encryption mappings than should have been allowed\n")
	}
}

func TestAddEncryptionMappingTimeout(t *testing.T) {
	timeout := float64(4)
	maxClients := 2
	servers := make([]netaddr.IPPort, 1)
	ip, _ := netaddr.ParseIP(("::1"))
	servers[0] = netaddr.IPPort{IP: ip, Port: 40000}

	addr := &netaddr.IPPort{IP: ip, Port: 62424}
	connectToken := testGenerateConnectToken(servers, TEST_PRIVATE_KEY, t)

	cm := NewClientManager(timeout, maxClients)

	serverTime := float64(1.0)
	expireTime := float64(1.1)

	if !cm.AddEncryptionMapping(connectToken.PrivateData, addr, serverTime, expireTime) {
		t.Fatalf("error adding encryption mapping\n")
	}

	idx := cm.FindEncryptionEntryIndex(addr, serverTime)
	if idx == -1 {
		t.Fatalf("error getting encryption entry index\n")
	}

	if !cm.SetEncryptionEntryExpiration(idx, float64(0.1)) {
		t.Fatalf("error setting entry expiration\n")
	}

	// remove the client.
	cm.CheckTimeouts(serverTime)

	idx = cm.FindEncryptionEntryIndex(addr, serverTime)
	if idx != -1 {
		t.Fatalf("error got encryption entry index when it should have been removed\n")
	}
}

func TestDisconnectClient(t *testing.T) {
	timeout := float64(4)
	maxClients := 2
	servers := make([]netaddr.IPPort, 1)
	ip, _ := netaddr.ParseIP("::1")
	servers[0] = netaddr.IPPort{IP: ip, Port: 40000}

	addr := &netaddr.IPPort{IP: ip, Port: 62424}

	connectToken := testGenerateConnectToken(servers, TEST_PRIVATE_KEY, t)

	cm := NewClientManager(timeout, maxClients)

	serverTime := float64(1.0)
	expireTime := float64(1.1)
	if !cm.AddEncryptionMapping(connectToken.PrivateData, addr, serverTime, expireTime) {
		t.Fatalf("error adding encryption mapping\n")
	}

	token := NewChallengeToken(TEST_CLIENT_ID)
	client := cm.ConnectClient(addr, token)
	clientIndex := cm.FindClientIndexById(TEST_CLIENT_ID)
	if clientIndex == -1 {
		t.Fatalf("error finding client index")
	}

	if cm.ConnectedClientCount() != 1 {
		t.Fatalf("error client connected count should be 1")
	}

	cm.DisconnectClient(clientIndex, false, serverTime)
	if client.connected {
		t.Fatalf("error client should be disconnected")
	}

}
