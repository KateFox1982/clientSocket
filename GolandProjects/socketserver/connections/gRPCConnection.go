package connections

import (
	"awesomeProject5/pb/message"
	"awesomeProject5/pb/service"
	"context"
	"encoding/json"
	"log"
	"socketserver/connections/gRPCFactory"
)

type Server struct {
	service.UnimplementedEchoServiceServer
}

func (s *Server) Echo(ctx context.Context, in *message.StringMessage) (*message.StringMessage, error) {
	log.Println("Вошла в Эхо")
	body := in.GetBody()
	userName := in.GetUser()
	receiver := in.GetReceiver()
	message1 := &Message{User: userName, Body: body, Reciever: receiver}
	log.Println("пришедшее по grps сообщение", message1)
	messageJSON, err := json.Marshal(message1)
	if err != nil {
		log.Println("JSON marshaling error:", err)
		return nil, err
	}

	encryptMessage, err := EncryptAES(messageJSON)
	if err != nil {
		log.Println("Error encrypting message:", err)
		return nil, err
	}
	rpc := gRPCFactory.NewGRPC(userName, &ctx)
	go HandleConnection(rpc)
	rpc.BodiesMessage <- encryptMessage
	messageByte := <-rpc.WriteMessageUsers

	decryptedMessage, err := DecryptAES(messageByte)
	if err != nil {
		log.Println("Error decrypting message:", err)
	}
	if err := json.Unmarshal(decryptedMessage, &message1); err != nil {
		log.Println("JSON unmarshaling error:", err)
	}
	// Разбор JSON строки в структуру
	//var message Message

	return &message.StringMessage{User: message1.User, Body: message1.Body, Receiver: message1.Reciever}, nil
}
