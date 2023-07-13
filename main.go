package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type JoinRequest struct {
	Name string
	Room string
}

var upgrader = websocket.Upgrader{} // use default options

func serveWebsocket(w http.ResponseWriter, r *http.Request) {
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

	room := getRoom(in_room)
	room.AddUser(in_user, c)

	// Handle messages from user
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		log.Printf("recv: %s", message)
		msg := strings.Split(string(message), " ")
		if len(msg) == 0 {
			continue
		}
		switch msg[0] {
		case "start", "reset":
			if len(msg) == 1 {
				continue
			}
			room.StartTimer(msg[1])
		case "stop":
			if len(msg) == 1 {
				continue
			}
			room.StopTimer(msg[1])
		case "new":
			if len(msg) == 1 {
				continue
			}
			room.CreateTimer(msg[1])
		}

		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

	// Remove user
	room.RemoveUser(in_user)
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
	http.HandleFunc("/ws", serveWebsocket)
	http.HandleFunc("/room", serveRoom)
	http.HandleFunc("/", serveHome)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))))
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal(err)
	}
}
