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
}

func (softphone *Softphone) Register() {
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

	go func() {
		for {
			_, bytes, err := conn.ReadMessage()
			if err != nil {
				log.Fatal("read:", err)
				return
			}
			println("recv: %s", string(bytes))
		}
	}()

	sipMessage := SipMessage{}
	sipMessage.Subject = fmt.Sprintf("REGISTER sip:%s SIP/2.0", softphone.SipInfo.Domain)
	sipMessage.Headers = make(map[string]string)
	sipMessage.Headers["Call-ID"] = uuid.New().String()
	fakeDomain := uuid.New().String() + ".invalid"
	fakeEmail := uuid.New().String() + "@" + fakeDomain
	branch := "z9hG4bK" + uuid.New().String()
	sipMessage.Headers["Contact"] = fmt.Sprintf("<sip:%s;transport=ws>;expires=600", fakeEmail)
	sipMessage.Headers["Via"] = fmt.Sprintf("SIP/2.0/WSS %s;branch=%s", fakeDomain, branch)
	fromTag := uuid.New().String()
	sipMessage.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", softphone.SipInfo.Username, fakeDomain, fromTag)
	sipMessage.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", softphone.SipInfo.Username, fakeDomain)
	sipMessage.Headers["CSeq"] = fmt.Sprintf("%d REGISTER", rand.Intn(10000) + 1)
	println(sipMessage.ToString())
	conn.WriteMessage(1, []byte(sipMessage.ToString()))

	time.Sleep(time.Second * 6)
}
