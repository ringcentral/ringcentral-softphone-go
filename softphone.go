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
	fakeDomain := fmt.Sprintf("%s.invalid", uuid.New().String())
	fakeEmail := fmt.Sprintf("%s@%s", uuid.New().String(), fakeDomain)
	registerMessage := SipMessage{
		Subject: fmt.Sprintf("REGISTER sip:%s SIP/2.0", sipInfo.Domain),
		Headers: map[string]string{
			"Call-ID": uuid.New().String(),
			"Contact": fmt.Sprintf("<sip:%s;transport=ws>;expires=600", fakeEmail),
			"Via":     fmt.Sprintf("SIP/2.0/WSS %s;branch=z9hG4bK%s", fakeDomain, uuid.New().String()),
			"From":    fmt.Sprintf("<sip:%s@%s>;tag=%s", sipInfo.Username, sipInfo.Domain, uuid.New().String()),
			"To":      fmt.Sprintf("<sip:%s@%s>", sipInfo.Username, sipInfo.Domain),
			"CSeq":    "8082 REGISTER",
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

			registerMessage.Headers["Authorization"] = GenerateAuthorization(sipInfo, "REGISTER", nonce)
			registerMessage.IncreaseSeq()
			registerMessage.Headers["Via"] = fmt.Sprintf("SIP/2.0/TCP %s;branch=z9hG4bK%s", fakeDomain, uuid.New().String())
			softphone.Send(registerMessage, nil)

			return true
		}
		return false
	})

	softphone.addMessageListener(func(strMessage string) {
		if strings.Contains(strMessage, "INVITE sip:") {
			log.Println("received invite message")
			// todo: handle invite
		}
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
