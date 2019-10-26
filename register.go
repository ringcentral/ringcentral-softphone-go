package softphone

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/url"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/ringcentral/ringcentral-go/definitions"
)

func (softphone *Softphone) register() {
	bytes := softphone.rc.Post("/restapi/v1.0/client-info/sip-provision", strings.NewReader(`{"sipInfo":[{"transport":"WSS"}]}`))
	var createSipRegistrationResponse definitions.CreateSipRegistrationResponse
	json.Unmarshal(bytes, &createSipRegistrationResponse)
	softphone.sipInfo = createSipRegistrationResponse.SipInfo[0]
	softphone.Device = createSipRegistrationResponse.Device
	url := url.URL{Scheme: strings.ToLower(softphone.sipInfo.Transport), Host: softphone.sipInfo.OutboundProxy, Path: ""}
	dialer := websocket.DefaultDialer
	dialer.Subprotocols = []string{"sip"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	softphone.wsConn = conn
	go func() {
		for {
			_, bytes, err := softphone.wsConn.ReadMessage()
			if err != nil {
				log.Fatal(err)
			}
			message := string(bytes)
			log.Debug("↓↓↓\n", message)
			for _, ml := range softphone.messageListeners {
				go ml(message)
			}
		}
	}()

	sipMessage := SipMessage{}
	sipMessage.method = "REGISTER"
	sipMessage.address = softphone.sipInfo.Domain
	sipMessage.headers = make(map[string]string)
	sipMessage.headers["Contact"] = fmt.Sprintf("<sip:%s;transport=ws>;expires=600", softphone.fakeEmail)
	sipMessage.headers["Via"] = fmt.Sprintf("SIP/2.0/WSS %s;branch=%s", softphone.fakeDomain, branch())
	sipMessage.headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", softphone.sipInfo.Username, softphone.sipInfo.Domain, softphone.fromTag)
	sipMessage.headers["To"] = fmt.Sprintf("<sip:%s@%s>", softphone.sipInfo.Username, softphone.sipInfo.Domain)
	sipMessage.addCseq(softphone).addCallId(*softphone).addUserAgent()
	softphone.request(sipMessage, func(message string) bool {
		if strings.Contains(message, "Www-Authenticate: Digest") {
			authenticateHeader := SipMessage{}.FromString(message).headers["Www-Authenticate"]
			regex := regexp.MustCompile(`, nonce="(.+?)"`)
			nonce := regex.FindStringSubmatch(authenticateHeader)[1]
			sipMessage.addAuthorization(*softphone, nonce).addCseq(softphone).newViaBranch()
			softphone.request(sipMessage, nil)
			return true
		}
		return false
	})
}
