package main

import (
	"testing"
)

type MockConnection struct {
	written string
	calls   int
}

func (conn *MockConnection) WriteMessage(msg int, data []byte) error {
	conn.written += string(data)
	conn.calls += 1
	return nil
}

func TestEmptyRoom(t *testing.T) {
	initRooms()

	if len(rooms) > 0 {
		t.Error("Should have no rooms")
	}
}

func TestCreatingRoom(t *testing.T) {

	room := CreateRoom()

	r, found := getRoom(room.Id)
	if !found || r.Id != room.Id {
		t.Error("Room not found or wrong ID")
	}
}

func TestAddUser(t *testing.T) {

	m := &MockConnection{}

	room := CreateRoom()
	room.AddUser("user", m)

	if len(room.Users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(room.Users))
	}
}

func TestAddUsers(t *testing.T) {
	rooms = make(map[string]Room)

	m1 := &MockConnection{}
	m2 := &MockConnection{}

	room := CreateRoom()
	room.AddUser("user1", m1)
	room.AddUser("user2", m2)

	if len(room.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(room.Users))
	}
}
