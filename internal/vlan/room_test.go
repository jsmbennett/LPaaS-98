package vlan

import (
	"testing"
)

type mockTransport struct {
	data [][]byte
}

func (m *mockTransport) SendControl(msg interface{}) error {
	return nil
}

func (m *mockTransport) SendData(data []byte) error {
	m.data = append(m.data, data)
	return nil
}

func (m *mockTransport) Recv() (Message, error) {
	return Message{}, nil
}

func (m *mockTransport) Close() error {
	return nil
}

func TestRoomAddPeer(t *testing.T) {
	room := NewRoom("test", "openarena", "host-client", 4)

	trans := &mockTransport{}
	peer, err := room.AddPeer("Alice", trans)
	if err != nil {
		t.Fatalf("AddPeer failed: %v", err)
	}

	if peer.ID != 1 || peer.Nickname != "Alice" || !peer.IsHost {
		t.Fatalf("peer data mismatch")
	}

	if room.HostPeerID != 1 {
		t.Fatalf("host not set correctly")
	}
}

func TestRoomFull(t *testing.T) {
	room := NewRoom("test", "openarena", "host-client", 1)

	trans1 := &mockTransport{}
	_, err := room.AddPeer("Alice", trans1)
	if err != nil {
		t.Fatalf("first AddPeer failed: %v", err)
	}

	trans2 := &mockTransport{}
	_, err = room.AddPeer("Bob", trans2)
	if err == nil {
		t.Fatal("expected AddPeer to fail when full")
	}
}

func TestForwardBroadcast(t *testing.T) {
	room := NewRoom("test", "openarena", "broadcast", 4)

	trans1 := &mockTransport{}
	peer1, _ := room.AddPeer("Alice", trans1)

	trans2 := &mockTransport{}
	_, _ = room.AddPeer("Bob", trans2)

	data := []byte("game packet")
	if err := room.ForwardData(peer1.ID, data); err != nil {
		t.Fatalf("ForwardData failed: %v", err)
	}

	if len(trans2.data) != 1 || string(trans2.data[0]) != "game packet" {
		t.Fatalf("data not forwarded correctly")
	}

	if len(trans1.data) != 0 {
		t.Fatal("sender should not receive their own packet")
	}
}

func TestForwardHostClient(t *testing.T) {
	room := NewRoom("test", "openarena", "host-client", 4)

	trans1 := &mockTransport{}
	_, _ = room.AddPeer("Host", trans1)

	trans2 := &mockTransport{}
	client, _ := room.AddPeer("Client", trans2)

	packet := append([]byte{0x00}, []byte("data")...)
	if err := room.ForwardData(client.ID, packet); err != nil {
		t.Fatalf("ForwardData failed: %v", err)
	}

	if len(trans1.data) != 1 {
		t.Fatal("host should receive data")
	}
}
