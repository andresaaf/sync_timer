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

	if len(rooms) > 0 {
		t.Errorf("woaw")
	}
}

func TestCreatingRoom(t *testing.T) {

	room := getRoom("test")

	if room.Id != "test" {
		t.Errorf("woaw")
	}
}

func TestAddUser(t *testing.T) {

	m := &MockConnection{}

	room := getRoom("test")
	room.AddUser("user", m)

	if len(room.Users) != 1 {
		t.Errorf("woaw")
	}
}

func TestAddUsers(t *testing.T) {
	rooms = make(map[string]Room)

	m1 := &MockConnection{}
	m2 := &MockConnection{}

	room := getRoom("test")
	room.AddUser("user1", m1)
	room.AddUser("user2", m2)

	if len(room.Users) != 2 {
		t.Error("Number of users: ", len(room.Users))
	}
}
