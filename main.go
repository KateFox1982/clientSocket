package main

import (
	"clientSocket/connections"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	//var message Message
	var u url.URL
	addr := flag.String("h", "localhost:4000", "WebSocket server host:port")
	user := flag.String("u", "", "Username")
	recipient := flag.String("r", "", "Recipient")
	connectionType := flag.String("t", "", "ConnectionTipe")
	flag.Parse()
	// Проверка наличия обязательных аргументов
	if *user == "" || *recipient == "" || *connectionType == "" {
		fmt.Println("Usage: go run . -h <host:port> -u <username> -r <recipient> ")
		*connectionType = "1"
		os.Exit(1)
	}

	q := u.Query()
	if *connectionType == "1" {
		fmt.Println("Вошла в if")
		u = url.URL{
			Scheme: "ws", // Используйте "ws" для WebSocket
			Host:   *addr + ":4000",
			Path:   "/ws",
		}

		q.Set("user", *user)
	}
	if *connectionType == "2" {
		u = url.URL{
			Scheme: "http",
			Host:   *addr + ":4000",
		}
		// Добавляем параметр  REGISTER: к URL для HTTP соединения

	}
	if *connectionType == "3" {
		u = url.URL{
			Scheme: "http",
			Host:   *addr + ":7000",
		}
	}
	u.RawQuery = q.Encode()
	origin := "http://" + *addr
	fmt.Println("origin", origin)
	if *connectionType == "1" {
		// WebSocket клиент
		wsClient, err := websocket.Dial(u.String(), "", origin)
		if err != nil {
			log.Fatal("Dial:", err)
		}

		go connections.HandleWebSocketMessages(wsClient)
		connections.SendMessages(wsClient, user, recipient)
	}
	if *connectionType == "2" {
		// Лонгполлинг клиент
		httpClient := &http.Client{}
		go func() {
			for {

				connections.PollForLongPollingMessages(httpClient, u, origin, user, recipient)

			}
		}()

		//wg.Wait()
		// Отправляем сообщения по мере написания
		connections.SendMessagesHTTP(httpClient, u, user, recipient)
	}

	//проверка серсетификата сервера отсутсвует
	opts := grpc.WithInsecure()
	cc, err := grpc.Dial(origin, opts)
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	go connections.GetMessage(cc)
	connections.SendMessage(cc, user, recipient)
}
