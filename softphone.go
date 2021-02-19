package softphone

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"regexp"
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
	MessageListeners              map[string]func(string)
	Conn                          *websocket.Conn
}

// Register register the softphone
func (softphone *Softphone) Register() {
	softphone.MessageListeners = make(map[string]func(string))
	sipInfo := softphone.CreateSipRegistrationResponse.SipInfo[0]
	url := url.URL{Scheme: strings.ToLower(sipInfo.Transport), Host: sipInfo.OutboundProxy, Path: ""}
	dialer := websocket.DefaultDialer
	dialer.Subprotocols = []string{"sip"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	softphone.Conn = conn
	go func() {
		for {
			_, bytes, err := softphone.Conn.ReadMessage()
			if err != nil {
				log.Fatal(err)
			}
			message := string(bytes)
			log.Println("↓↓↓\n", message)
			for _, messageListener := range softphone.MessageListeners {
				go messageListener(message)
			}
		}
	}()
	userAgent := "github.com/ringcentral/ringcentral-softphone-go"
	fakeDomain := fmt.Sprintf("%s.invalid", uuid.New().String())
	fakeEmail := fmt.Sprintf("%s@%s", uuid.New().String(), fakeDomain)
	registerMessage := SipMessage{
		Subject: fmt.Sprintf("REGISTER sip:%s SIP/2.0", sipInfo.Domain),
		Headers: map[string]string{
			"Call-ID":        uuid.New().String(),
			"User-Agent":     userAgent,
			"Contact":        fmt.Sprintf("<sip:%s;transport=ws>;expires=600", fakeEmail),
			"Via":            fmt.Sprintf("SIP/2.0/WSS %s;branch=z9hG4bK%s", fakeDomain, uuid.New().String()),
			"From":           fmt.Sprintf("<sip:%s@%s>;tag=%s", sipInfo.Username, sipInfo.Domain, uuid.New().String()),
			"To":             fmt.Sprintf("<sip:%s@%s>", sipInfo.Username, sipInfo.Domain),
			"CSeq":           "8082 REGISTER",
			"Content-Length": "0",
			"Max-Forwards":   "70",
		},
		Body: "",
	}
	softphone.Send(registerMessage, func(strMessage string) bool {
		if strings.Contains(strMessage, "SIP/2.0 401 Unauthorized") {
			unAuthMessage := FromStringToSipMessage(strMessage)
			authHeader := unAuthMessage.Headers["WWW-Authenticate"]
			regex := regexp.MustCompile(", nonce=\"(.+?)\"")
			match := regex.FindStringSubmatch(authHeader)
			nonce := match[1]
			log.Println(nonce)
			return true
		}
		return false
	})
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
	err := softphone.Conn.WriteMessage(1, []byte(stringMessage))
	if err != nil {
		log.Fatal(err)
	}
}

func (softphone *Softphone) addMessageListener(messageListener func(string)) string {
	key := uuid.New().String()
	softphone.MessageListeners[key] = messageListener
	return key
}
func (softphone *Softphone) removeMessageListener(key string) {
	delete(softphone.MessageListeners, key)
}
