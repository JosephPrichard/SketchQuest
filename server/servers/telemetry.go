/*
 * Copyright (c) Joseph Prichard 2024
 */

package servers

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type TelemetryServer struct {
	upgrade      websocket.Upgrader
	clientsCount int
	subscribers  map[chan int]struct{}
	mu           sync.Mutex // used to synchronize player store
}

func NewTelemetryServer() *TelemetryServer {
	return &TelemetryServer{
		upgrade:      CreateUpgrade(),
		clientsCount: 0,
		subscribers:  make(map[chan int]struct{}),
	}
}

func (server *TelemetryServer) AddSubscriber(subscriber chan int) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.clientsCount += 1
	server.subscribers[subscriber] = struct{}{}
	server.broadcast()
}

func (server *TelemetryServer) RemoveSubscriber(subscriber chan int) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.clientsCount -= 1
	delete(server.subscribers, subscriber)
	close(subscriber)
	server.broadcast()
}

func (server *TelemetryServer) broadcast() {
	for s := range server.subscribers {
		s <- server.clientsCount
	}
}

func (server *TelemetryServer) Subscribe(w http.ResponseWriter, r *http.Request) {
	EnableCors(&w)

	subscriber := make(chan int)

	ws, err := server.upgrade.Upgrade(w, r, nil)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to upgrade to websocket")
		return
	}

	go server.subscriberListener(ws, subscriber)
	go server.socketListener(ws, subscriber)

	server.AddSubscriber(subscriber)
}

func (server *TelemetryServer) socketListener(ws *websocket.Conn, subscriber chan int) {
	defer func() {
		// remove the subscriber when the connection ends
		server.RemoveSubscriber(subscriber)
		_ = ws.Close()
		log.Printf("Socket listener close function called")
		if panicInfo := recover(); panicInfo != nil {
			log.Println(panicInfo)
		}
	}()
	// loop until the client sends no more messages
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Client closed connection with err %s", err.Error())
			return
		}
	}
}

func (server *TelemetryServer) subscriberListener(ws *websocket.Conn, subscriber chan int) {
	defer func() {
		log.Println("Meta subscriber channel was closed")
		_ = ws.Close()
		if panicInfo := recover(); panicInfo != nil {
			log.Println(panicInfo)
		}
	}()
	for clientCount := range subscriber {
		// read values from channel and write back to socket
		log.Println("Received updated client count", clientCount)

		type MetaResp struct {
			ClientCount int `json:"clientCount"`
		}
		resp := MetaResp{ClientCount: clientCount}
		buf, err := json.Marshal(resp)
		if err != nil {
			log.Println("Failed to serialize meta resp")
			return
		}

		err = ws.WriteMessage(websocket.TextMessage, buf)
		if err != nil {
			log.Printf("Error writing message %s", err)
			return
		}
	}
}
