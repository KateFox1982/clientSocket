package connections

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

var (
	register   = make(chan *Client)
	unregister = make(chan *Client)
	Clients    = make(map[string]*Client)
	Mutex      = sync.RWMutex{}
)
var key, _ = hex.DecodeString("421A69BC2B99BEB97AA4BF13BE39D0344C9E31B853E646812F123DFE909F3D63")

type Client struct {
	Websocket    Connection
	IsConnection bool
}

type Message struct {
	User     string `json:"user"`
	Reciever string `json:"reciever"`
	Body     string `json:"body"`
}

func RegisterAndUnregisterUsers() {
	for {
		select {
		case c := <-register:
			// Добавляем клиента в мапу при регистрации.
			username := getUsername(c.Websocket)
			getClientRegistration(c, username)
		case c := <-unregister:
			// Удаляем клиента из мапы при разрегистрации.
			username := getUsername(c.Websocket)
			getClientUnregister(c, username)
		}

	}
}
func getClientRegistration(c *Client, username string) {
	Mutex.Lock()
	defer Mutex.Unlock()
	Clients[username] = c

	log.Printf("Registered user: %s\n", username)

}
func getClientUnregister(c *Client, username string) {
	Mutex.Lock()
	defer Mutex.Unlock()

	if !c.IsConnection {
		delete(Clients, username)

		log.Printf("Unregistered user: %s\n", username)
	}
	log.Printf("Unregistered user: %s\n", username)
}
func getUsername(c Connection) string {
	// Извлекаем имя пользователя из URL запроса.
	return c.Query("user")
}
func newMessage() Message {
	return Message{}
}

type Connection interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
	Query(key string, defaultValue ...string) string
}

func HandleConnection(c Connection) {
	fmt.Println("Вошла в HandleConnection")
	var messageError string
	// Регистрируем клиента. При регистрации помечаем как неподключенного
	client := &Client{Websocket: c, IsConnection: true}
	register <- client
	defer func() {
		// Разрегистрируем клиента при завершении работы
		client.IsConnection = false
		unregister <- client
		c.Close()
	}()
	//бесконечный цикл чтения сообщений
	for {
		//
		messageType, message, err := c.ReadMessage()
		if err != nil {

			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Соединение было закрыто нормально, завершаем цикл.
				client.IsConnection = true
				log.Printf("Connection closed by client: %s", err)
				break
			} else {
				// Другая ошибка чтения, которую можно обработать
				log.Println("read error:", err)

			}
		}

		if message != nil {

			messageNew := newMessage()

			client.IsConnection = true
			decryptMessage, err := DecryptAES(message)
			if err != nil {
				log.Println("Error decrypt", err)
				messageError = ""
			}
			if err := json.Unmarshal(decryptMessage, &messageNew); err != nil {
				log.Println("JSON unmarshaling error:", err)
				messageError = ""
				continue
			}

			//исправить название
			messageNew.handleMessage(c, messageType, messageError)
			if !client.IsConnection {
				log.Printf("Client is no longer connected, exiting loop.")
				break
			}
		}
	}
}

//функция отправки сообщений получателю и отправителю
func (m *Message) handleMessage(c Connection, messageType int, message string) {

	if m.Body != "" {
		// Отправляем сообщение получателю.
		message, err := m.marshalStruct()
		if err != nil {
			message = ""
		}

		//отправляем структуру Message на шифрование
		encryptMessage, err := EncryptAES([]byte(message))
		if err != nil {
			log.Println("Error encrypt", err)
			message = ""
		}
		receiverConn := findClient(m.Reciever)
		if receiverConn != nil {
			if err := receiverConn.Websocket.WriteMessage(messageType, encryptMessage); err != nil {
				log.Printf("Error sending message to %s: %s", m.Reciever, err)
			}
		} else {
			m.User = fmt.Sprintf("Сообщение пользователю было не доставлено, пользователь не зарегистрирован %s", m.User)
			fmt.Println("Cooбщение на отправку", m)
			message, err = m.marshalStruct()
			if err != nil {
				message = ""
			}
			encryptMessage, err = EncryptAES([]byte(message))
			if err != nil {
				log.Println("Error encrypt", err)
				message = ""
			}
		}
		// Отправляем копию сообщения отправителю.
		if err := c.WriteMessage(messageType, encryptMessage); err != nil {
			log.Println("Error sending message to sender:", err)
		}
	}
}

//Маршалим структуру Message
func (m *Message) marshalStruct() (string, error) {
	messageJSON, err := json.Marshal(m)
	if err != nil {
		log.Println("JSON marshaling error:", err)
		return "", err
	}
	messageString := string(messageJSON)
	return messageString, err
}

//Шифрование структуры Message
func EncryptAES(plaintext []byte) ([]byte, error) {
	// Create a new Cipher Block from the key
	block, err := aes.NewCipher([]byte(key))
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

//рассшифровка структуры Message
func DecryptAES(ciphertext []byte) ([]byte, error) {
	// Create a new Cipher Block from the key
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	// Create a new GCM cipher with the given key
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// Get the nonce size
	nonceSize := gcm.NonceSize()

	// Extract the nonce from the ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the ciphertext using the nonce
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
