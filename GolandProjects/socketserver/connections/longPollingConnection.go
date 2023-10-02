package connections

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"log"
	"socketserver/connections/longPollingFactory"
	"time"
)

// получение сообщения от пользователей
func HandlePutMessage(c *fiber.Ctx) error {
	fmt.Println()
	userName := c.Params("user")
	fmt.Println([]byte(userName))
	fmt.Println("Имя пользователя", userName)
	bodyBytes := c.Body()
	lp := getLongPollingConnection(userName)

	fmt.Println("Мапа клиентов", Clients)

	if lp != nil {
		//lp.Mu.Lock()
		lp.BodiesMessage <- bodyBytes
		lp.LastTimeConnection = time.Now()
		//lp.Mu.Unlock()
	} else {
		fmt.Println("такой пользователь не существует")
		errors.New("Пользователь не существует")
		c.Status(fiber.StatusBadRequest).SendString("Пользователь не существует")
		return nil
	}
	return c.SendStatus(fiber.StatusOK)
}

//отправка сообщений пользователю
func HandleGetMessages(c *fiber.Ctx) error {
	fmt.Println("Вошла в HandleGetMessages ")
	userName := c.Params("user")
	fmt.Println([]byte(userName))
	fmt.Println("Имя пользователя", userName)
	lp := getLongPollingConnection(userName)
	if lp == nil {

		lp := longPollingFactory.NewLongPolling(userName, c)
		go HandleConnection(lp)
		//lp.Mu.Lock()
		lp.LastTimeConnection = time.Now()
		err := sendMessage(c, lp)
		//lp.Mu.Unlock()
		if err != nil {
			log.Println("Error in send message", err)
		}

	} else {
		//lp.Mu.Lock()
		lp.LastTimeConnection = time.Now()
		lp.Ctx = c
		//lp.Mu.Unlock()

		err := sendMessage(c, lp)
		if err != nil {
			log.Println("Error in send message", err)
		}

	}
	return nil
}
func findClient(userName string) *Client {
	Mutex.RLock()
	defer Mutex.RUnlock()
	if client, ok := Clients[userName]; ok {

		return client
	}

	return nil
}

//проверка является ли данное соединение long polling и отправка структуры LongPolling
func getLongPollingConnection(userName string) *longPollingFactory.LongPolling {
	client := findClient(userName)
	if client != nil {
		fmt.Println("Нашла пользователя")
		//проверка является ли соединенние LongPolling
		longPolling, isLongPolling := client.Websocket.(*longPollingFactory.LongPolling)
		if isLongPolling {

			return longPolling
		}
	}

	return nil
}

func sendMessage(c *fiber.Ctx, longPolling *longPollingFactory.LongPolling) error {
	//бесконечный цикл
	for {
		var messages []byte
		select {
		case <-longPolling.FlagForSend:
			//	цикл добавлени сообщений из буферизированного не закрытого канала в слайс байт используя разделитель
		ggg:
			for {
				select {
				case message := <-longPolling.WriteMessageUsers:
					messages = append(messages, message...)
					messages = append(messages, []byte{234, 102, 87}...)

				default:
					break ggg
				}
			}
			//время отправки сообщения
			longPolling.EndTimeConnection = time.Now()
			//отправления слайса сообщений
			err := c.Send(messages)
			if err != nil {
				longPolling.FlagForSend <- struct{}{}
				fmt.Println("Error sending message")
				return err
			}
			return nil
		//	отпрака через 60 секунд если канал пустой
		case <-time.After(60 * time.Second):
			fmt.Println("Прошло 60 секунд без новых сообщений, завершаем цикл")
			err := c.Send([]byte("Нет сообщений"))
			if err != nil {
				fmt.Println("Error sending message")
			}
			return nil
		}
	}

	return nil
}

//проверяю старые лонг пуллер соединения и удаляю те по которым не было движения больше 10 секунд
func CheckConnectionLongPollingAndDelete() {
	for {
		longPollingUser, longPolling := checkClient()
		if longPolling != nil {
			if len(longPolling.WriteMessageUsers) < 1 {
				if time.Since(longPolling.LastTimeConnection) >= 70*time.Second &&
					time.Since(longPolling.EndTimeConnection) >= 10*time.Second {
					getClientLongPollingUnregister(longPollingUser)
				}
			}
		}
		time.Sleep(8 * time.Second)
	}
}

//проверка, является ли данное соединение long polling
func checkClient() (string, *longPollingFactory.LongPolling) {
	Mutex.RLock()
	defer Mutex.RUnlock()
	for longPollingUser, value := range Clients {
		longPolling, isLongPolling := value.Websocket.(*longPollingFactory.LongPolling)
		if isLongPolling {

			return longPollingUser, longPolling
		}
	}
	return "", nil
}

//удаление мапы клиентов
func getClientLongPollingUnregister(username string) {
	Mutex.Lock()
	defer Mutex.Unlock()
	delete(Clients, username)
	fmt.Printf("Unregistered user: %s\n", username)
}
