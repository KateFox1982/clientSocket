package main

import (
	"awesomeProject5/pb/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"google.golang.org/grpc"
	"log"
	"net"
	"socketserver/connections"
	"sync"
)

func main() {
	// Make it easy to find out which line prints the log.
	log.SetFlags(log.Lshortfile)
	app := fiber.New()
	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		connections.HandleConnection(conn)
	}))

	app.Get("/messages/:user", func(c *fiber.Ctx) error {
		return connections.HandleGetMessages(c)
	})

	app.Post("/message/:user", func(c *fiber.Ctx) error {
		return connections.HandlePutMessage(c)
	})

	//гоурутина, которая записывает и убирает из мапы участников чата, которые ушли или пришли
	go connections.RegisterAndUnregisterUsers()

	go connections.CheckConnectionLongPollingAndDelete()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Fatal(app.Listen(":4000"))
	}()
	grpcServer := grpc.NewServer()

	service.RegisterEchoServiceServer(grpcServer, &connections.Server{})
	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", ":7000")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("gRPC server listening on :7000")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	wg.Wait()
}
