package softphone

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ringcentral/ringcentral"
	"github.com/ringcentral/ringcentral/definitions"
	"log"
	"math/rand"
	"net/url"
	"strings"
	"time"
)

type Softphone struct {
	Rc ringcentral.RestClient
	SipInfo definitions.SIPInfoResponse
	wsConn *websocket.Conn
	fakeDomain string
	fakeEmail string
	fromTag string
	toTag string
	callId string
	cseq int
	messages chan string
}

func (softphone Softphone) request(sipMessage SipMessage, expectedResp string) string {
	softphone.wsConn.WriteMessage(1, []byte(sipMessage.ToString()))
	if expectedResp != "" {
		for {
			message := <- softphone.messages
			if(strings.Contains(message, expectedResp)) {
				return message
			}
		}
	}
	return ""
}

func branch() string {
	return "z9hG4bK" + uuid.New().String()
}

func (softphone *Softphone) Register() {
	softphone.fakeDomain = uuid.New().String() + ".invalid"
	softphone.fakeEmail = uuid.New().String() + "@" + softphone.fakeDomain
	softphone.fromTag = uuid.New().String()
	softphone.toTag = uuid.New().String()
	softphone.callId = uuid.New().String()
	softphone.cseq = rand.Intn(10000) + 1

	bytes := softphone.Rc.Post("/restapi/v1.0/client-info/sip-provision", strings.NewReader(`{"sipInfo":[{"transport":"WSS"}]}`))
	var createSipRegistrationResponse definitions.CreateSipRegistrationResponse
	json.Unmarshal(bytes, &createSipRegistrationResponse)
	softphone.SipInfo = createSipRegistrationResponse.SipInfo[0]
	bytes2, _ := json.Marshal(softphone.SipInfo)
	println(string(bytes2))
	u := url.URL{Scheme: strings.ToLower(softphone.SipInfo.Transport), Host: softphone.SipInfo.OutboundProxy, Path: ""}
	dialer := websocket.DefaultDialer
	dialer.Subprotocols = []string{"sip"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	softphone.wsConn = conn
	softphone.messages = make(chan string)
	go func() {
		for {
			_, bytes, err := softphone.wsConn.ReadMessage()
			if err != nil {
				log.Fatal(err)
			}
			message := string(bytes)
			println(message)
			softphone.messages <- message
		}
	}()

	sipMessage := SipMessage{}
	sipMessage.Method = "REGISTER"
	sipMessage.Address = softphone.SipInfo.Domain
	sipMessage.Headers = make(map[string]string)
	sipMessage.Headers["Contact"] = fmt.Sprintf("<sip:%s;transport=ws>;expires=600", softphone.fakeEmail)
	sipMessage.Headers["Via"] = fmt.Sprintf("SIP/2.0/WSS %s;branch=%s", softphone.fakeDomain, branch())
	sipMessage.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", softphone.SipInfo.Username, softphone.fakeDomain, softphone.fromTag)
	sipMessage.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", softphone.SipInfo.Username, softphone.fakeDomain)
	sipMessage.addContentLength().addCseq(softphone).addCallId(*softphone).addUserAgent()
	message := softphone.request(sipMessage, "Www-Authenticate: Digest")
	println(message)

	time.Sleep(time.Second * 3)
}
