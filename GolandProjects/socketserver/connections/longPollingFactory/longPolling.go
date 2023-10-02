package longPollingFactory

import (
	"github.com/gofiber/fiber/v2"
	"time"
)

type LongPolling struct {
	CloseUser          func() error
	WriteMessageUsers  chan []byte
	LastTimeConnection time.Time
	EndTimeConnection  time.Time
	UserName           string
	BodiesMessage      chan []byte
	Ctx                *fiber.Ctx
	FlagForSend        chan struct{}
	//Mu                 sync.Mutex
}

func NewLongPolling(userName string, ctx *fiber.Ctx) *LongPolling {
	return &LongPolling{
		UserName:          userName,
		Ctx:               ctx,
		BodiesMessage:     make(chan []byte),
		WriteMessageUsers: make(chan []byte, 100),
		FlagForSend:       make(chan struct{}),
	}
}

func (l *LongPolling) ReadMessage() (int, []byte, error) {
	message := <-l.BodiesMessage
	return 1, message, nil

}

func (l *LongPolling) WriteMessage(messageType int, data []byte) error {
	l.WriteMessageUsers <- data
	l.FlagForSend <- struct{}{}
	return nil
}

func (l *LongPolling) Close() error {
	if l.CloseUser != nil {
		return l.CloseUser()
	}
	return nil
}

func (l *LongPolling) Query(key string, defaultValue ...string) string {
	return l.UserName
}
