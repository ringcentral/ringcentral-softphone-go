package softphone

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ringcentral/ringcentral-go"
)

// SipMessage SIP message
type SipMessage struct {
	Subject string
	Headers map[string]string
	Body    string
}

// ToString from SipMessage to string message
func (sipMessage SipMessage) ToString() (message string) {
	list := []string{}
	list = append(list, sipMessage.Subject)
	for key, value := range sipMessage.Headers {
		list = append(list, fmt.Sprintf("%s: %s", key, value))
	}
	list = append(list, "")
	list = append(list, sipMessage.Body)
	return strings.Join(list, "\r\n")
}

// FromStringToSipMessage from string message to SipMessage
func FromStringToSipMessage(message string) (sipMessage SipMessage) {
	paragraphs := strings.Split(message, "\r\n\r\n")
	body := strings.Join(paragraphs[1:], "\r\n\r\n")
	paragraphs = strings.Split(paragraphs[0], "\r\n")
	subject := paragraphs[0]
	headers := make(map[string]string)
	for _, line := range paragraphs[1:] {
		tokens := strings.Split(line, ": ")
		headers[tokens[0]] = tokens[1]
	}
	return SipMessage{
		Subject: subject,
		Headers: headers,
		Body:    body,
	}
}

// Softphone softphone
type Softphone struct {
	CreateSipRegistrationResponse ringcentral.CreateSipRegistrationResponse
	messageListeners              map[string]func(string)
	Conn                          *websocket.Conn
}

// Register register the softphone
func (softphone *Softphone) Register() {
	softphone.messageListeners = make(map[string]func(string))
	sipInfo := softphone.CreateSipRegistrationResponse.SipInfo[0]
	url := url.URL{Scheme: strings.ToLower(sipInfo.Transport), Host: sipInfo.OutboundProxy, Path: ""}
	dialer := websocket.DefaultDialer
	dialer.Subprotocols = []string{"sip"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			_, bytes, err := conn.ReadMessage()
			if err != nil {
				log.Fatal(err)
			}
			message := string(bytes)
			log.Println("↓↓↓\n", message)
			for _, messageListener := range softphone.messageListeners {
				go messageListener(message)
			}
		}
	}()

}

// Send send message via WebSocket
func (softphone *Softphone) Send(sipMessage SipMessage, responseHandler func(string) bool) {
	stringMessage := sipMessage.ToString()
	log.Println("↑↑↑\n", stringMessage)
	if responseHandler != nil {
		var key string
		key = softphone.addMessageListener(func(message string) {
			done := responseHandler(message)
			if done {
				softphone.removeMessageListener(key)
			}
		})
	}
	err := softphone.Conn.WriteMessage(1, []byte(sipMessage.ToString()))
	if err != nil {
		log.Fatal(err)
	}
}

func (softphone *Softphone) addMessageListener(messageListener func(string)) string {
	key := uuid.New().String()
	softphone.messageListeners[key] = messageListener
	return key
}
func (softphone *Softphone) removeMessageListener(key string) {
	delete(softphone.messageListeners, key)
}
