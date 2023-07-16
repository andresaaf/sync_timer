package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Connection interface {
	WriteMessage(msg int, data []byte) error
}

type Timer struct {
	Time  uint32
	Start time.Time
}

type Room struct {
	Id     string
	Users  map[string]Connection
	Timers map[string]Timer
}

var rooms map[string]Room

func initRooms() {
	rand.Seed(time.Now().UnixNano())
	rooms = make(map[string]Room)

	// TODO: Load from DB
}

func getRoom(room string) (Room, bool) {
	r, found := rooms[room]
	return r, found
}

var room_letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func CreateRoom() Room {
	// Generate name
	name := ""
	for {
		b := make([]rune, 5)
		for i := range b {
			b[i] = room_letters[rand.Intn(len(room_letters))]
		}
		name = string(b)
		if _, found := rooms[name]; !found {
			break
		}
	}

	// Create room
	new_room := Room{Id: name, Users: make(map[string]Connection), Timers: make(map[string]Timer)}
	rooms[name] = new_room
	new_room.CreateTimer("test1", 10)
	new_room.CreateTimer("test2", 62)
	return new_room
}

func (room *Room) AddUser(user string, conn Connection) {
	// Send current timers
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("sync %d", time.Now().Unix())))
	for name, timer := range room.Timers {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("timer %d %d %s", timer.Time, timer.Start.Unix(), name)))
	}

	// Send users and notify users
	join_str := []byte(fmt.Sprintf("join %s", user))
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("users %d ", len(room.Users)))
	for name, c := range room.Users {
		c.WriteMessage(websocket.TextMessage, join_str)
		sb.WriteString(name)
		sb.WriteString(",")
	}
	user_str := sb.String()
	user_str = user_str[:len(user_str)-1]
	conn.WriteMessage(websocket.TextMessage, []byte(user_str))

	// Append user
	room.Users[user] = conn
}

func (room *Room) RemoveUser(user string) {
	delete(room.Users, user)

	if len(room.Users) == 0 {
		// Remove room
		delete(rooms, user)
	} else {
		room.Broadcast(fmt.Sprintf("part %s", user))
	}
}

func (room *Room) CreateTimer(name string, dur uint32) {
	room.Timers[name] = Timer{Time: dur, Start: time.Unix(0, 0)}
	room.Broadcast(fmt.Sprintf("timer %d 0 %s", dur, name))
}

func (room *Room) RemoveTimer(name string) {
	_, ok := room.Timers[name]
	if !ok {
		return
	}
	delete(room.Timers, name)
	room.Broadcast(fmt.Sprintf("del %s", name))
}

func (room *Room) SetTime(name string, sec uint32) {
	timer, ok := room.Timers[name]
	if !ok {
		return
	}
	timer.Time = sec
	room.Broadcast(fmt.Sprintf("set %d %s", sec, name))
}

func (room *Room) StartTimer(name string) {
	timer, ok := room.Timers[name]
	if !ok {
		return
	}
	start_time := time.Now()
	timer.Start = start_time
	room.Broadcast(fmt.Sprintf("start %d %s", start_time.Unix(), name))
}

func (room *Room) StopTimer(name string) {
	timer, ok := room.Timers[name]
	if !ok {
		return
	}
	timer.Start = time.Unix(0, 0)
	room.Broadcast(fmt.Sprintf("stop %s", name))
}

func (room *Room) Broadcast(str string) {
	for _, conn := range room.Users {
		conn.WriteMessage(websocket.TextMessage, []byte(str))
	}
}
