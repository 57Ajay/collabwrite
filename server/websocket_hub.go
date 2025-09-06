package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	// Registered clients. The map keys are the document IDs.
	rooms map[string]map[*Client]bool
	sync.RWMutex
}

type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	docID         string
	kafkaProducer *KafkaProducer
}

func newHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) serveWS(w http.ResponseWriter, r *http.Request, docID string, producer *KafkaProducer) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub: h, conn: conn, docID: docID, kafkaProducer: producer,
	}

	h.Lock()
	if _, ok := h.rooms[docID]; !ok {
		h.rooms[docID] = make(map[*Client]bool)
	}
	h.rooms[docID][client] = true
	h.Unlock()

	log.Printf("Client connected to document room: %s", docID)

	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Lock()
		delete(c.hub.rooms[c.docID], c)
		if len(c.hub.rooms[c.docID]) == 0 {
			delete(c.hub.rooms, c.docID)
		}
		c.hub.Unlock()
		log.Printf("Client disconnected from document room: %s", c.docID)
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var msg DocumentUpdateMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshalling message: %v", err)
			continue
		}

		c.kafkaProducer.ProduceMessage(context.Background(), msg)
	}
}

func (h *Hub) broadcast(docId string, message []byte) {
	h.RLock()
	defer h.RUnlock()

	if room, ok := h.rooms[docId]; ok {
		for client := range room {
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("error writing message: %v", err)
			}
		}
	}
}
