package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"socketserver/connections"
	"socketserver/connections/longPollingFactory"
	"socketserver/mock"

	"testing"
	"time"
)

var keyMistake, _ = hex.DecodeString("421A69BC2B99BEB4BF13BE39D0344C9E31B853E646812F123DFE909F3D63")

func TestEncryptAES(t *testing.T) {
	plaintext := []byte("Привет Катя!")
	ciphertext, err := connections.EncryptAES(plaintext)

	assert.NoError(t, err, "Encryption should not return an error")
	assert.NotEmpty(t, ciphertext, "Ciphertext should not be empty")
}

func TestDecryptAES(t *testing.T) {
	plaintext := []byte("Привет Катя!")
	ciphertext, err := connections.EncryptAES(plaintext)
	assert.NoError(t, err, "Encryption should not return an error")
	decryptedPlaintext, err := connections.DecryptAES(ciphertext)
	assert.NoError(t, err, "Decryption should not return an error")
	assert.Equal(t, plaintext, decryptedPlaintext, "Decrypted text should match plaintext")
}

//позитивный сценарий теста проверяем, что 2 пользователя регистрируются, отправляют по 2 сообщения и
//оба получают 4 сообщения: 2 своих и 2 чужого пользователя
func TestFirstCaseHandleWsMonitor(t *testing.T) {
	//запускаем гоурутину регистрации
	go connections.RegisterAndUnregisterUsers()
	var sendMessageFirstUser []connections.Message
	var sendMessageSecondUser []connections.Message
	messagesFirstUser := []connections.Message{
		{
			User:     "User1",
			Reciever: "User2",
			Body:     "Test message №1 First User",
		},
		{
			User:     "User1",
			Reciever: "User2",
			Body:     "Test message №2 First User",
		},
	}
	messagesSecondUser := []connections.Message{
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №1 Second User",
		},
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №2 Second User",
		},
	}
	//отправляем имена пользователей для дальнейше регистрации
	mockConnFirstUser := &mock.MockConnection{UserName: "User1"}
	mockConnSecondUser := &mock.MockConnection{UserName: "User2"}

	//запускаем гоурутину HandleWsMonitor для каждого пользователя
	go connections.HandleConnection(mockConnFirstUser)
	go connections.HandleConnection(mockConnSecondUser)

	time.Sleep(1 * time.Second)
	//проверяем, что все пользователи были зарегистрированы
	registersUsers := make(map[string]*connections.Client)
	registersUsers["User1"] = &connections.Client{Websocket: mockConnFirstUser, IsConnection: true}
	registersUsers["User2"] = &connections.Client{Websocket: mockConnSecondUser, IsConnection: true}
	assert.Equal(t, registersUsers, connections.Clients)
	//маршалим, кодируем сообщения пользователей и отправляем в канал
	getEncryptBodiesUser(t, messagesFirstUser, mockConnFirstUser)
	getEncryptBodiesUser(t, messagesSecondUser, mockConnSecondUser)
	time.Sleep(5 * time.Second)
	//проверка сообщений пришедшие первому юзеру
	returnMessageFirstUser := mockConnFirstUser.WriteMessageUsers
	sendMessageFirstUser = getMessagesAfterReturn(t, returnMessageFirstUser, sendMessageFirstUser)
	time.Sleep(1 * time.Second)
	for _, messageSecondUser := range messagesSecondUser {
		messagesFirstUser = append(messagesFirstUser, messageSecondUser)
	}
	assert.Equal(t, messagesFirstUser, sendMessageFirstUser)
	//проверка сообщений пришедшие второму юзеру
	returnMessageSecondUser := mockConnSecondUser.WriteMessageUsers
	sendMessageSecondUser = getMessagesAfterReturn(t, returnMessageSecondUser, sendMessageSecondUser)
	time.Sleep(1 * time.Second)
	assert.Equal(t, messagesFirstUser, sendMessageSecondUser)
}

//функция декодирует сообщения пользователей и осуществляет анмаршалинг в структуру Message
func getMessagesAfterReturn(t *testing.T, returnMessages [][]byte, sendMessage []connections.Message) []connections.Message {
	for _, returnMessage := range returnMessages {
		decryptMessage, err := connections.DecryptAES(returnMessage)
		assert.NoError(t, err)
		var responseFirstUser connections.Message
		err = json.Unmarshal(decryptMessage, &responseFirstUser)
		assert.NoError(t, err)
		sendMessage = append(sendMessage, responseFirstUser)
	}
	return sendMessage
}

//функция маршалит, кодирует сообщения пользователей и отправляем в канал, после чего канал закрывается
func getEncryptBodiesUser(t *testing.T, messages []connections.Message, mockConnUser *mock.MockConnection) {
	//создаем канал
	mockConnUser.BodiesMessage = make(chan []byte)
	for _, message := range messages {
		messageJSON, err := json.Marshal(message)
		assert.NoError(t, err)
		encryptMessage, err := connections.EncryptAES(messageJSON)
		assert.NoError(t, err)
		// Отправьте данные в канал.
		mockConnUser.BodiesMessage <- encryptMessage
	}
	//закрываем канал после записи
	close(mockConnUser.BodiesMessage)
}

//негативный сценарий теста проверяем, что 1 пользователь регистрируюется и отправляет  2 сообщения
//незарегистрованному пользователю,  2 других должны вернуться к зарегистрированному пользователю с пометкой
//"Сообщение пользователю было не доставлено, пользователь не зарегистрирован"
func TestSecondCaseHandleWsMonitor(t *testing.T) {
	//очищаем мапу пользователей после предыдущего теста
	for username := range connections.Clients {
		delete(connections.Clients, username)
	}
	//запускаем гоурутину регистрации
	go connections.RegisterAndUnregisterUsers()
	var sendMessageFirstUser []connections.Message
	messagesFirstUser := []connections.Message{
		{
			User:     "User1",
			Reciever: "User3",
			Body:     "Test message №1 First User",
		},
		{
			User:     "User1",
			Reciever: "User3",
			Body:     "Test message №2 First User",
		},
	}

	//отправляем имена пользователей для дальнейше регистрации
	mockConnFirstUser := &mock.MockConnection{UserName: "User1"}

	//запускаем гоурутину HandleWsMonitor для каждого пользователя
	go connections.HandleConnection(mockConnFirstUser)

	time.Sleep(1 * time.Second)
	//проверяем, что все пользователи были зарегистрированы
	registersUsers := make(map[string]*connections.Client)
	registersUsers["User1"] = &connections.Client{Websocket: mockConnFirstUser, IsConnection: true}

	assert.Equal(t, registersUsers, connections.Clients)
	//маршалим, кодируем сообщения пользователей и отправляем в канал
	getEncryptBodiesUser(t, messagesFirstUser, mockConnFirstUser)

	time.Sleep(5 * time.Second)
	updatedMessagesFirstUser := make([]connections.Message, len(messagesFirstUser))
	copy(updatedMessagesFirstUser, messagesFirstUser)

	for i := range updatedMessagesFirstUser {
		updatedMessagesFirstUser[i].User = fmt.Sprintf("Сообщение пользователю было не доставлено, пользователь не зарегистрирован %s", updatedMessagesFirstUser[i].User)
	}
	// Проверка сообщений, пришедших первому пользователю
	returnMessageFirstUser := mockConnFirstUser.WriteMessageUsers
	sendMessageFirstUser = getMessagesAfterReturn(t, returnMessageFirstUser, sendMessageFirstUser)
	time.Sleep(1 * time.Second)
	assert.Equal(t, updatedMessagesFirstUser, sendMessageFirstUser)
}

//негативный сценарий теста проверяем, что 2 пользователя регистрируются, отправляют по 2 сообщения, 1 пользователь
//отправляет 2 пустых сообщения, оба пользователя получают только сообщения 2-ого пользователя
func TestThirdCaseHandleWsMonitor(t *testing.T) {
	//очищаем мапу пользователей после предыдущего теста
	for username := range connections.Clients {
		delete(connections.Clients, username)
	}
	//запускаем гоурутину регистрации
	go connections.RegisterAndUnregisterUsers()
	var sendMessageFirstUser []connections.Message
	var sendMessageSecondUser []connections.Message
	messagesFirstUser := []connections.Message{
		{
			User:     "User1",
			Reciever: "User2",
			Body:     "",
		},
		{
			User:     "User1",
			Reciever: "User2",
			Body:     "",
		},
	}
	messagesSecondUser := []connections.Message{
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №1 Second User",
		},
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №2 Second User",
		},
	}
	//отправляем имена пользователей для дальнейше регистрации
	mockConnFirstUser := &mock.MockConnection{UserName: "User1"}
	mockConnSecondUser := &mock.MockConnection{UserName: "User2"}
	//запускаем гоурутину HandleWsMonitor для каждого пользователя
	go connections.HandleConnection(mockConnFirstUser)
	go connections.HandleConnection(mockConnSecondUser)

	time.Sleep(1 * time.Second)
	//проверяем, что все пользователи были зарегистрированы
	registersUsers := make(map[string]*connections.Client)
	registersUsers["User1"] = &connections.Client{Websocket: mockConnFirstUser, IsConnection: true}
	registersUsers["User2"] = &connections.Client{Websocket: mockConnSecondUser, IsConnection: true}
	assert.Equal(t, registersUsers, connections.Clients)
	//маршалим, кодируем сообщения пользователей и отправляем в канал
	getEncryptBodiesUser(t, messagesFirstUser, mockConnFirstUser)
	getEncryptBodiesUser(t, messagesSecondUser, mockConnSecondUser)
	time.Sleep(5 * time.Second)
	//проверка сообщений пришедшие первому юзеру
	returnMessageFirstUser := mockConnFirstUser.WriteMessageUsers
	sendMessageFirstUser = getMessagesAfterReturn(t, returnMessageFirstUser, sendMessageFirstUser)
	time.Sleep(1 * time.Second)

	assert.Equal(t, messagesSecondUser, sendMessageFirstUser)
	//проверка сообщений пришедшие второму юзеру
	returnMessageSecondUser := mockConnSecondUser.WriteMessageUsers
	sendMessageSecondUser = getMessagesAfterReturn(t, returnMessageSecondUser, sendMessageSecondUser)
	time.Sleep(1 * time.Second)
	assert.Equal(t, messagesSecondUser, sendMessageSecondUser)
}

//негативный сценарий теста проверяем, что 2 пользователя регистрируются, и первый пользователь без выхода
//перерегистрируется
func TestForthCaseHandleWsMonitor(t *testing.T) {
	//очищаем мапу пользователей после предыдущего теста
	for username := range connections.Clients {
		delete(connections.Clients, username)
	}
	//запускаем гоурутину регистрации
	go connections.RegisterAndUnregisterUsers()
	//отправляем имена пользователей для дальнейше регистрации
	mockConnFirstUser := &mock.MockConnection{UserName: "User1"}
	mockConnSecondUser := &mock.MockConnection{UserName: "User2"}
	mockConnThirdUser := &mock.MockConnection{UserName: "User1"}
	//запускаем гоурутину HandleWsMonitor для каждого пользователя
	go connections.HandleConnection(mockConnFirstUser)
	go connections.HandleConnection(mockConnSecondUser)
	go connections.HandleConnection(mockConnThirdUser)
	time.Sleep(1 * time.Second)
	//проверяем, что все пользователи были зарегистрированы
	registersUsers := make(map[string]*connections.Client)
	registersUsers["User1"] = &connections.Client{Websocket: mockConnFirstUser, IsConnection: true}
	registersUsers["User2"] = &connections.Client{Websocket: mockConnSecondUser, IsConnection: true}
	assert.Equal(t, registersUsers, connections.Clients)
}

//негативный сценарий теста проверяем, что 2 пользователя регистрируются, и первый пользователь разрегистрируется,
//первый раз второй пользователь отправляет сообщение пользователю один до его разрегистрации и он их получает
//вторая часть сообщений после разрегистрации первого пользователя должно прийти с сообщением, что пользователь,
//которому он отправил сообщение не зарегистрирован
func TestFifthCaseHandleWsMonitor(t *testing.T) {
	//очищаем мапу пользователей после предыдущего теста
	for username := range connections.Clients {
		delete(connections.Clients, username)
	}
	var sendMessageFirstUser []connections.Message
	var sendMessageSecondUserFirstPart []connections.Message
	var sendMessageSecondUserSecondPart []connections.Message
	//запускаем гоурутину регистрации
	go connections.RegisterAndUnregisterUsers()
	//отправляем имена пользователей для дальнейше регистрации
	mockConnFirstUser := &mock.MockConnection{UserName: "User1"}
	mockConnSecondUser := &mock.MockConnection{UserName: "User2"}
	messagesSecondUserFirstPart := []connections.Message{
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №1 Second User",
		},
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №2 Second User",
		},
	}
	messagesSecondUserSecondPart := []connections.Message{
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №3 Second User",
		},
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №4 Second User",
		},
	}

	//запускаем гоурутину HandleWsMonitor для каждого пользователя
	go connections.HandleConnection(mockConnFirstUser)
	go connections.HandleConnection(mockConnSecondUser)

	time.Sleep(2 * time.Second)
	//проверяем, что все пользователи были зарегистрированы
	registersUsers := make(map[string]*connections.Client)
	registersUsers["User1"] = &connections.Client{Websocket: mockConnFirstUser, IsConnection: true}
	registersUsers["User2"] = &connections.Client{Websocket: mockConnSecondUser, IsConnection: true}
	assert.Equal(t, registersUsers, connections.Clients)

	getEncryptBodiesUser(t, messagesSecondUserFirstPart, mockConnSecondUser)
	time.Sleep(2 * time.Second)
	//проверка сообщений пришедшие первому юзеру
	returnMessageFirstUser := mockConnFirstUser.WriteMessageUsers
	sendMessageFirstUser = getMessagesAfterReturn(t, returnMessageFirstUser, sendMessageFirstUser)
	time.Sleep(1 * time.Second)

	assert.Equal(t, messagesSecondUserFirstPart, sendMessageFirstUser)
	//проверка сообщений пришедшие второму юзеру
	returnMessageSecondUserFirstPart := mockConnSecondUser.WriteMessageUsers
	sendMessageSecondUserFirstPart = getMessagesAfterReturn(t, returnMessageSecondUserFirstPart, sendMessageSecondUserFirstPart)
	time.Sleep(1 * time.Second)
	assert.Equal(t, messagesSecondUserFirstPart, sendMessageSecondUserFirstPart)

	//отправляем ошибку разъединения первого пользователя
	mockConnFirstUser.CodeError = make(chan error)

	err := &websocket.CloseError{1006, "websocket: close  (abnormal closure)"}
	mockConnFirstUser.CodeError <- err
	close(mockConnFirstUser.CodeError)
	delete(registersUsers, "User1")
	time.Sleep(1 * time.Second)
	assert.Equal(t, registersUsers, connections.Clients)

	getEncryptBodiesUser(t, messagesSecondUserSecondPart, mockConnSecondUser)
	time.Sleep(5 * time.Second)
	updatedMessagesSecondUser := make([]connections.Message, len(messagesSecondUserSecondPart))
	copy(updatedMessagesSecondUser, messagesSecondUserSecondPart)

	for i := range updatedMessagesSecondUser {
		updatedMessagesSecondUser[i].User = fmt.Sprintf("Сообщение пользователю было не доставлено, пользователь не зарегистрирован %s", updatedMessagesSecondUser[i].User)
		messagesSecondUserFirstPart = append(messagesSecondUserFirstPart, updatedMessagesSecondUser[i])
	}

	//проверка сообщений пришедшие второму юзеру
	returnMessageSecondUserSecondPart := mockConnSecondUser.WriteMessageUsers
	sendMessageSecondUserSecondPart = getMessagesAfterReturn(t, returnMessageSecondUserSecondPart, sendMessageSecondUserSecondPart)
	time.Sleep(3 * time.Second)
	assert.Equal(t, messagesSecondUserFirstPart, sendMessageSecondUserSecondPart)

}

//негативный сценарий теста проверяем, что 2 пользователя регистрируются, и первый пользователь отправляет сообщение с
//неправильной кодировкой
func TestSixthCaseHandleWsMonitor(t *testing.T) {
	//очищаем мапу пользователей после предыдущего теста
	for username := range connections.Clients {
		delete(connections.Clients, username)
	}
	var sendMessageFirstUser []connections.Message
	var sendMessageSecondUser []connections.Message

	//запускаем гоурутину регистрации
	go connections.RegisterAndUnregisterUsers()
	//отправляем имена пользователей для дальнейше регистрации
	mockConnFirstUser := &mock.MockConnection{UserName: "User1"}
	mockConnSecondUser := &mock.MockConnection{UserName: "User2"}

	messagesFirstUser := []connections.Message{
		{
			User:     "User1",
			Reciever: "User2",
			Body:     "Test message №1 First User",
		},
		{
			User:     "User1",
			Reciever: "User2",
			Body:     "Test message №2 First User",
		},
	}
	messagesSecondUser := []connections.Message{
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №1 Second User",
		},
		{
			User:     "User2",
			Reciever: "User1",
			Body:     "Test message №2 Second User",
		},
	}

	//запускаем гоурутину HandleWsMonitor для каждого пользователя
	go connections.HandleConnection(mockConnFirstUser)
	go connections.HandleConnection(mockConnSecondUser)

	time.Sleep(1 * time.Second)
	//проверяем, что все пользователи были зарегистрированы
	registersUsers := make(map[string]*connections.Client)
	registersUsers["User1"] = &connections.Client{Websocket: mockConnFirstUser, IsConnection: true}
	registersUsers["User2"] = &connections.Client{Websocket: mockConnSecondUser, IsConnection: true}
	assert.Equal(t, registersUsers, connections.Clients)
	//маршалим, кодируем сообщения пользователей и отправляем в канал
	getMistakeEncryptBodiesUser(t, messagesFirstUser, mockConnFirstUser)
	getEncryptBodiesUser(t, messagesSecondUser, mockConnSecondUser)
	time.Sleep(5 * time.Second)
	//проверка сообщений пришедшие первому юзеру
	returnMessageFirstUser := mockConnFirstUser.WriteMessageUsers
	sendMessageFirstUser = getMessagesAfterReturn(t, returnMessageFirstUser, sendMessageFirstUser)
	time.Sleep(1 * time.Second)

	assert.Equal(t, messagesSecondUser, sendMessageFirstUser)
	//проверка сообщений пришедшие второму юзеру
	returnMessageSecondUser := mockConnSecondUser.WriteMessageUsers
	sendMessageSecondUser = getMessagesAfterReturn(t, returnMessageSecondUser, sendMessageSecondUser)
	time.Sleep(1 * time.Second)
	assert.Equal(t, messagesSecondUser, sendMessageSecondUser)

}

//Шифрование структуры Message с ошибочным ключом
func mistakeEncryptAES(plaintext []byte) ([]byte, error) {
	// Create a new Cipher Block from the key
	block, err := aes.NewCipher([]byte(keyMistake))
	if err != nil {
		return nil, err
	}
	// Create a new GCM cipher with the given key
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	// Encrypt the plaintext using the nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

//функция маршалит, кодирует сообщения пользователей и отправляем в канал, после чего канал закрывается
func getMistakeEncryptBodiesUser(t *testing.T, messages []connections.Message, mockConnUser *mock.MockConnection) {
	//создаем канал
	mockConnUser.BodiesMessage = make(chan []byte)
	for _, message := range messages {
		messageJSON, err := json.Marshal(message)
		assert.NoError(t, err)
		encryptMessage, err := mistakeEncryptAES(messageJSON)
		assert.Error(t, err)
		// Отправьте данные в канал.
		mockConnUser.BodiesMessage <- encryptMessage
	}
	//закрываем канал после записи
	close(mockConnUser.BodiesMessage)
}

func TestMap(t *testing.T) {
	var (
		Clients = make(map[string]*connections.Client)
	)

	username := "Kate"
	client := &connections.Client{Websocket: longPollingFactory.NewLongPolling(username, &fiber.Ctx{}), IsConnection: true}

	Clients[username] = client

	var count int
	for {
		count++
		if count > 100 {
			break
		}
		go func() {
			client, ok := Clients[username]
			if !ok {
				t.Fatal("ftgtgt")
			}
			longPolling, isLongPolling := client.Websocket.(*longPollingFactory.LongPolling)
			if isLongPolling {

				longPolling.LastTimeConnection = time.Now()
				longPolling.Ctx = &fiber.Ctx{}
			}

			fmt.Println("Clients", Clients)
		}()
	}
}
