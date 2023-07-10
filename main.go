package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

type JoinRequest struct {
	Name string
	Room string
}

type Timer struct {
	Name  string
	Time  uint
	Start uint
}

type Room struct {
	Id     string
	Users  map[string]*websocket.Conn
	Timers []Timer
}

var upgrader = websocket.Upgrader{} // use default options
var rooms = make(map[string]Room)

func serveTimer(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	defer c.Close()

	// Wait for name and room
	in_room := ""
	in_user := ""
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		s := strings.Split(string(message), " ")
		if s[0] == "room" && len(s) > 2 {
			in_room = s[1]
			in_user = strings.Join(s[2:], " ")
			break
		}
	}
	if in_room == "" || in_user == "" {
		c.WriteMessage(websocket.TextMessage, []byte("error in"))
		return
	}

	room, ok := rooms[in_room]
	if !ok {
		// Create room
		rooms[in_room] = Room{Id: in_room, Users: make(map[string]*websocket.Conn), Timers: []Timer{}}
		room = rooms[in_room]
	}

	// Send current timers
	for i := 0; i < len(room.Timers); i++ {
		c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("timeri %s %d, %d", room.Timers[i].Name, room.Timers[i].Time, room.Timers[i].Start)))
	}

	// Send users and notify users
	join_str := []byte(fmt.Sprintf("join %s", in_user))
	var sb strings.Builder
	sb.WriteString("users ")
	sb.WriteString(strconv.Itoa(len(room.Users)))
	sb.WriteString(" ")
	for name, conn := range room.Users {
		conn.WriteMessage(websocket.TextMessage, join_str)
		sb.WriteString(name)
		sb.WriteString(",")
	}
	user_str := sb.String()
	user_str = user_str[:len(user_str)-1]
	c.WriteMessage(websocket.TextMessage, []byte(user_str))

	// Append user
	room.Users[in_user] = c

	// Handle messages from user
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

	// Remove user
	delete(room.Users, in_user)

	if len(room.Users) == 0 {
		// Remove room
		delete(rooms, in_room)
	} else {
		for _, conn := range room.Users {
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("leave %s", in_user)))
		}
	}
}

func serveRoom(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/room" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form error", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	room := r.FormValue("room")
	tmpl, err := template.ParseFiles("timer.html")
	if err != nil {
		panic(err)
	}
	request := JoinRequest{Name: name, Room: room}
	err = tmpl.Execute(w, request)
	if err != nil {
		panic(err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/timer", serveTimer)
	http.HandleFunc("/room", serveRoom)
	http.HandleFunc("/", serveHome)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))))
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal(err)
	}
}
