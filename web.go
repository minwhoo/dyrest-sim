package main

import (
	//	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	//	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func readWebsocketConnection(sm *simulationManager, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		log.Println("WEB: Connection closed!")
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println("message receieved: ", string(message))
		switch string(message) {
		case "initialize":
			sm.initializeNodes()
		case "start":
			sm.start()
		}
	}
}

func writeWebsocketConnection(sm *simulationManager, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		log.Println("WEB: Connection closed!")
	}()

	for {
		data := <-sm.supervisor.lg.c
		if len(data) <= upgrader.WriteBufferSize {
			err := conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				break
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWebsocket(sm *simulationManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("WEB: Connection created!")
	go readWebsocketConnection(sm, conn)
	go writeWebsocketConnection(sm, conn)
}

func serveWeb(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

func startWebInterface(sm *simulationManager) {
	//flag.Parse()
	http.HandleFunc("/", serveWeb)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWebsocket(sm, w, r)
	})
	log.Println("WEB: Starting web server...")
	http.ListenAndServe(":8080", nil)
}
