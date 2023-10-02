package gRPCFactory

import (
	"context"
	"log"
	"time"
)

type GRPC struct {
	CloseUser          func() error
	WriteMessageUsers  chan []byte
	LastTimeConnection time.Time
	EndTimeConnection  time.Time
	UserName           string
	BodiesMessage      chan []byte
	Ctx                *context.Context
	FlagForSend        chan struct{}
	//Mu                 sync.Mutex
}

func NewGRPC(userName string, ctx *context.Context) *GRPC {
	return &GRPC{
		UserName:          userName,
		Ctx:               ctx,
		BodiesMessage:     make(chan []byte),
		WriteMessageUsers: make(chan []byte, 100),
	}
}
func (g *GRPC) ReadMessage() (int, []byte, error) {
	log.Println("Вошла в ReadMessage")
	message := <-g.BodiesMessage
	return 1, message, nil

}

func (g *GRPC) WriteMessage(messageType int, data []byte) error {
	g.WriteMessageUsers <- data

	return nil
}

func (g *GRPC) Close() error {
	if g.CloseUser != nil {
		return g.CloseUser()
	}
	return nil
}

func (g *GRPC) Query(key string, defaultValue ...string) string {
	return g.UserName
}
