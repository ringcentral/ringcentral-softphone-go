package softphone

import (
	"crypto/tls"
	"log"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/ringcentral/ringcentral-go"
)

// Softphone softphone
type Softphone struct {
	createSipRegistrationResponse ringcentral.CreateSipRegistrationResponse
}

// Register register the softphone
func (softphone Softphone) Register() {
	sipInfo := softphone.createSipRegistrationResponse.SipInfo[0]
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
		}
	}()
}
