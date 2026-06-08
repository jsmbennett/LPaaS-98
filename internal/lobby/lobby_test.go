package lobby

import (
	"testing"
)

func TestAddAndRemoveClient(t *testing.T) {
	lb := New()

	client, err := lb.AddClient("client1", "Alice")
	if err != nil {
		t.Fatalf("AddClient failed: %v", err)
	}

	if client.ID != "client1" || client.Nickname != "Alice" {
		t.Fatalf("client data mismatch")
	}

	lb.RemoveClient("client1")
	if lb.GetClient("client1") != nil {
		t.Fatal("client should be removed")
	}
}

func TestDuplicateClient(t *testing.T) {
	lb := New()

	_, err := lb.AddClient("client1", "Alice")
	if err != nil {
		t.Fatalf("first AddClient failed: %v", err)
	}

	_, err = lb.AddClient("client1", "Bob")
	if err == nil {
		t.Fatal("duplicate AddClient should fail")
	}
}

func TestCreateRoom(t *testing.T) {
	lb := New()

	lb.AddClient("host1", "Host")
	room, err := lb.CreateRoom("openarena", "host1", 4)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	if room.GameID != "openarena" || room.HostID != "host1" {
		t.Fatalf("room data mismatch")
	}

	if len(room.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(room.Members))
	}
}

func TestJoinRoom(t *testing.T) {
	lb := New()

	lb.AddClient("host1", "Host")
	lb.AddClient("player1", "Player")

	room, err := lb.CreateRoom("openarena", "host1", 4)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	_, err = lb.JoinRoom("player1", room.ID)
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}

	if len(room.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(room.Members))
	}
}

func TestJoinRoomFull(t *testing.T) {
	lb := New()

	lb.AddClient("host1", "Host")
	lb.AddClient("player1", "Player1")
	lb.AddClient("player2", "Player2")

	room, _ := lb.CreateRoom("openarena", "host1", 2)
	lb.JoinRoom("player1", room.ID)

	_, err := lb.JoinRoom("player2", room.ID)
	if err == nil {
		t.Fatal("expected JoinRoom to fail when full")
	}
}

func TestLeaveRoom(t *testing.T) {
	lb := New()

	lb.AddClient("host1", "Host")
	lb.AddClient("player1", "Player")

	room, _ := lb.CreateRoom("openarena", "host1", 4)
	lb.JoinRoom("player1", room.ID)

	lb.LeaveRoom("player1")

	if len(room.Members) != 1 {
		t.Fatalf("expected 1 member after leave, got %d", len(room.Members))
	}
}

func TestListRooms(t *testing.T) {
	lb := New()

	lb.AddClient("host1", "Host1")
	lb.AddClient("host2", "Host2")

	lb.CreateRoom("openarena", "host1", 4)
	lb.CreateRoom("quake", "host2", 4)

	rooms := lb.ListRooms()
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms, got %d", len(rooms))
	}
}

func TestRemoveClientRemovesFromRoom(t *testing.T) {
	lb := New()

	lb.AddClient("host1", "Host")
	lb.AddClient("player1", "Player")

	room, _ := lb.CreateRoom("openarena", "host1", 4)
	lb.JoinRoom("player1", room.ID)

	lb.RemoveClient("player1")

	if len(room.Members) != 1 {
		t.Fatalf("player should be removed from room")
	}
}
