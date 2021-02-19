package softphone

import (
	"crypto/md5"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/ringcentral/ringcentral-go"
)

// GenerateResponse generate response field in the authorization header
func GenerateResponse(username string, password string, realm string, method string, uri string, nonce string) string {
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%x", ha1, nonce, ha2)))
	return fmt.Sprintf("%x", response)
}

// GenerateAuthorization generate the authorization header
func GenerateAuthorization(sipInfo ringcentral.SIPInfoResponse, method string, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, sipInfo.Domain,
		GenerateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+sipInfo.Domain, nonce),
	)
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
