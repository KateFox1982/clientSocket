package connections

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"os"
	"time"
)

type Message struct {
	User     string `json:"user"`
	Reciever string `json:"reciever"`
	Body     string `json:"body"`
}

var key, _ = hex.DecodeString("421A69BC2B99BEB97AA4BF13BE39D0344C9E31B853E646812F123DFE909F3D63")

//прием сообщений по вебсокет соединению
func HandleWebSocketMessages(conn *websocket.Conn) {
	for {
		var responseJSON string
		err := websocket.Message.Receive(conn, &responseJSON)
		if err != nil {
			log.Println("Receive response:", err)
			return
		}
		if responseJSON == "" {
			fmt.Println("Error marshaling structure")
		}
		dencryptMessage, err := DecryptAES([]byte(responseJSON))
		if err != nil {
			log.Println("Error dencrypt", err)
		}

		// Разбор JSON строки в структуру
		var response Message
		if err := json.Unmarshal(dencryptMessage, &response); err != nil {
			log.Println("JSON unmarshaling or encrypt or encrypt error:", err)
			continue
		}
		// Сообщение выводящееся в терминале
		fmt.Printf("%s, %s : %s\n", response.User, time.Now().Format("15:04:01 02.01.2006"), response.Body)
	}
}
func scanMessageFromTerminal(message *Message, user, recipient *string) error {

	var err error
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		input := scanner.Text()
		if input != "" {
			message.Reciever = *recipient
			message.Body = input
			message.User = *user
			// Преобразуем сообщение в JSON

			return nil
		}
		err = errors.New("Структура пустая")

	}
	return err
}

//отправка сообщения по вебсоккет соединению
func SendMessages(conn *websocket.Conn, user *string, recipient *string) {
	var message *Message
	//Бесконечный цикл  чтение с терминала
	err := scanMessageFromTerminal(message, user, recipient)
	if err != nil {
		log.Println("Error scanMessageFromTerminal", err)
		return
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Println("JSON marshaling error:", err)
		return
	}
	encryptMessage, err := EncryptAES(messageJSON)
	if err != nil {
		log.Println("Error encrypt", err)
		return
	}
	// Отправка сообщения
	err = websocket.Message.Send(conn, string(encryptMessage))
	if err != nil {
		log.Println("Send message:", err)
		return
	}

	// Закрываем соединение
	conn.Close()
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
		fmt.Println("Ошибка NewCipher", err)
		return nil, err
	}
	// Create a new GCM cipher with the given key
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println("Ошибка NewGCM", err)
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
