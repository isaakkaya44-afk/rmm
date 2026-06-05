package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: websocket-test <token> <url>")
	}

	token := os.Args[1]
	url := os.Args[2]

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Fatalf("connection failed: %v", err)
	}
	defer conn.Close()

	log.Printf("connected to %s", url)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}
		log.Printf("received: %s", string(message))
	}
}
