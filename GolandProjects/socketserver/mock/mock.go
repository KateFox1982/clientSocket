package mock

import (
	"time"
)

type MockConnection struct {
	CloseUser         func() error
	WriteMessageUsers [][]byte
	UserName          string
	BodiesMessage     chan []byte
	CodeError         chan error
}

func (m *MockConnection) ReadMessage() (int, []byte, error) {

	for {
		select {
		case message := <-m.BodiesMessage:
			return 0, message, nil
		default:
			time.Sleep(500 * time.Millisecond)
		case err := <-m.CodeError:
			return 0, nil, err
		}
	}

}
func (m *MockConnection) HandleConnection() {

}

func (m *MockConnection) WriteMessage(messageType int, data []byte) error {
	m.WriteMessageUsers = append(m.WriteMessageUsers, data)
	return nil
}

func (m *MockConnection) Close() error {
	if m.CloseUser != nil {
		return m.CloseUser()
	}
	return nil

}

func (m *MockConnection) Query(key string, defaultValue ...string) string {

	return m.UserName
}
