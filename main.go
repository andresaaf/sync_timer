package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
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

	// Parse room
	spl := strings.Split(r.URL.Path, "/")
	if len(spl) < 3 || spl[2] == "" {
		http.Error(w, "Empty room", http.StatusBadRequest)
		return
	}
	room, found := getRoom(spl[2])
	if !found {
		http.Error(w, "No room", http.StatusBadRequest)
		return
	}

	// Wait for name and room
	in_user := ""
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		s := strings.Split(string(message), " ")
		if s[0] == "name" && len(s) > 1 {
			in_user = strings.Join(s[1:], " ")
			break
		}
	}
	if in_user == "" {
		c.WriteMessage(websocket.TextMessage, []byte("error in"))
		return
	}

	// Join
	room.AddUser(in_user, c)
	defer room.RemoveUser(in_user)

	// Handle messages from user
	for {
		_, message, err := c.ReadMessage()
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
			dur, err := strconv.Atoi(msg[1])
			if err != nil {
				return
			}
			room.CreateTimer(msg[2], uint32(dur))
			break
		case "del":
			if len(msg) == 1 {
				continue
			}
			room.RemoveTimer(msg[1])
			break
		case "set":
			if len(msg) < 2 {
				continue
			}
			dur, err := strconv.Atoi(msg[1])
			if err != nil {
				return
			}
			room.SetTime(msg[2], uint32(dur))
			break
		}
	}
}

func serveRoom(w http.ResponseWriter, r *http.Request) {
	// Parse room
	spl := strings.Split(r.URL.Path, "/")
	room_name := ""
	if len(spl) > 2 {
		room_name = spl[2]
	}
	room, found := getRoom(room_name)
	if !found {
		if room_name == "" {
			room = CreateRoom()
			http.Redirect(w, r, room.Id, http.StatusTemporaryRedirect)
			return
		} else {
			http.Error(w, "Room does not exist", http.StatusBadRequest)
			return
		}
	}

	if r.Method == http.MethodGet {
		// Ask for name
		tmpl, err := template.ParseFiles("join.html")
		if err != nil {
			panic(err)
		}
		err = tmpl.Execute(w, room)
		if err != nil {
			panic(err)
		}
		return
	}

	// Parse name
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form error", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	match, err := regexp.MatchString("^[A-Za-z0-9]{1,12}$", name)
	if name == "" || err != nil || !match {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}

	// Show room
	tmpl, err := template.ParseFiles("room.html")
	if err != nil {
		panic(err)
	}
	request := JoinRequest{Name: name, Room: room.Id}
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
	initRooms()

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/ws/", serveWebsocket)
	http.HandleFunc("/room/", serveRoom)
	http.HandleFunc("/", serveHome)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))))
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal(err)
	}
}
