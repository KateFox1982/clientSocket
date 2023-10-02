package connections

import (
	"example.com/awesomeProject5/pb/service"
	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"log"
)

func SendMessage(cc *grpc.ClientConn, user, recipient *string) {
	newConnection := service.NewEchoServiceClient(cc)
	var message *Message
	err := scanMessageFromTerminal(message, user, recipient)
	if err != nil {
		log.Println("Error scanMessageFromTerminal", err)
		return
	}
	ctx := context.Background()
	newConnection.Echo(ctx, message)
}

func GetMessage(cc *grpc.ClientConn) {
	newConnection := service.NewEchoServiceClient(cc)
	for {
		var responseJSON string
		err := newConnection.(conn, &responseJSON)
		if err != nil {
			log.Println("Receive response:", err)
			return
		}
}
