package connections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

//var FlagReceiveMessage = make(chan struct{})

func PollForLongPollingMessages(httpClient *http.Client, u url.URL, origin string, user, recipient *string) {
	fmt.Println("")
	u.Path += "/messages/" + *user
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Println("HTTP request error:", err)
	}
	req.Header.Set("Origin", origin)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("HTTP request error:", err)
	}
	go func() {
		select {
		case <-time.After(10 * time.Second):
			// Закрыть тело ответа
			resp.Body.Close()
		}
	}()

	if resp.Body != nil {
		//if resp.StatusCode != http.StatusOK {
		//	log.Println("HTTP response status:", resp.Status)
		//}
	}
	bodies, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("HTTP response error:", err)
	}
	//разделитель из сервера сообщений
	delimiter := []byte{234, 102, 87}
	//разбиваем слайс байт на слайс слайсов байт для дальнейшей обработки
	substrings := bytes.Split((bodies), delimiter)
	if string(substrings[0]) != "Нет сообщений" {
		for _, body := range substrings[:len(substrings)-1] {
			// Расшифровка сообщения
			decryptedMessage, err := DecryptAES(body)
			if err != nil {
				log.Println("Error decrypting message:", err)
			}

			// Разбор JSON строки в структуру
			var message Message
			if err := json.Unmarshal(decryptedMessage, &message); err != nil {
				log.Println("JSON unmarshaling error:", err)
			}
			// Сообщение выводящееся в терминале
			fmt.Printf("%s, %s : %s\n", message.User, time.Now().Format("15:04:01 02.01.2006"), message.Body)

		}
	} else {
		fmt.Printf(string(substrings[0]))
	}
}

func SendMessagesHTTP(httpClient *http.Client, u url.URL, user, recipient *string) {
	u.Path += "/message/" + *user
	fmt.Println("Имя пользователя", *user)

	var encryptMessage []byte
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

	encryptMessage, err = EncryptAES(messageJSON)
	if err != nil {
		log.Println("Error encrypting message:", err)
		return
	}

	resp, err := httpClient.Post(u.String(), "application/json", bytes.NewReader(encryptMessage))
	if resp.StatusCode == 400 {
		log.Println("Пользователь не существует пройдите регистрацию")
	}
	if err != nil {
		log.Println("HTTP request error:", err)
		log.Println("Пользователь не существует пройдите регистрацию")
		return
	}
	defer resp.Body.Close()
}
