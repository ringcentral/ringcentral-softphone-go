package softphone

import (
	"crypto/tls"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/ringcentral/ringcentral"
	"github.com/ringcentral/ringcentral/definitions"
	"log"
	"net/url"
	"strings"
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
	dialer.Subprotocols = []string {"sip"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	println(c)
	defer c.Close()
}
