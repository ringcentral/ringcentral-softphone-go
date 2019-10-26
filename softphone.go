package softphone

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v2"
	"github.com/ringcentral/ringcentral-go"
	"github.com/ringcentral/ringcentral-go/definitions"
)

type Softphone struct {
	Device   definitions.SipRegistrationDeviceInfo
	OnTrack  func(track *webrtc.Track)
	OnInvite func(inviteMessage SipMessage)

	rc               ringcentral.RestClient
	sipInfo          definitions.SIPInfoResponse
	wsConn           *websocket.Conn
	fakeDomain       string
	fakeEmail        string
	fromTag          string
	toTag            string
	callId           string
	cseq             int
	responses        chan string
	messageListeners []func(message string)
}

func NewSoftPhone(rc ringcentral.RestClient) *Softphone {
	softphone := Softphone{}
	softphone.rc = rc

	softphone.fakeDomain = uuid.New().String() + ".invalid"
	softphone.fakeEmail = uuid.New().String() + "@" + softphone.fakeDomain
	softphone.fromTag = uuid.New().String()
	softphone.toTag = uuid.New().String()
	softphone.callId = uuid.New().String()
	softphone.cseq = rand.Intn(10000) + 1

	softphone.messageListeners = []func(message string){}
	softphone.OnInvite = func(inviteMessage SipMessage) {}
	softphone.OnTrack = func(track *webrtc.Track) {}

	return &softphone
}

func (softphone Softphone) request(sipMessage SipMessage, expectedResp string) string {
	println(sipMessage.ToString())
	softphone.wsConn.WriteMessage(1, []byte(sipMessage.ToString()))
	if expectedResp != "" {
		for {
			response := <-softphone.responses
			if strings.Contains(response, expectedResp) {
				return response
			}
		}
	}
	return ""
}

func (softphone Softphone) WaitForIncomingCall() {
	for {
		message := <-softphone.responses
		if strings.HasPrefix(message, "INVITE sip:") {
			inviteMessage := SipMessage{}.FromString(message)

			dict := map[string]string{"Contact": fmt.Sprintf(`<sip:%s;transport=ws>`, softphone.fakeDomain)}
			responseMsg := inviteMessage.Response(softphone, 180, dict, "")
			println(responseMsg)
			softphone.wsConn.WriteMessage(1, []byte(responseMsg))

			var msg Msg
			xml.Unmarshal([]byte(inviteMessage.headers["P-rc"]), &msg)
			sipMessage := SipMessage{}
			sipMessage.method = "MESSAGE"
			sipMessage.address = msg.Hdr.From
			sipMessage.headers = make(map[string]string)
			sipMessage.headers["Via"] = fmt.Sprintf("SIP/2.0/WSS %s;branch=%s", softphone.fakeDomain, branch())
			sipMessage.headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", softphone.sipInfo.Username, softphone.sipInfo.Domain, softphone.fromTag)
			sipMessage.headers["To"] = fmt.Sprintf("<sip:%s>", msg.Hdr.From)
			sipMessage.headers["Content-Type"] = "x-rc/agent"
			sipMessage.addCseq(&softphone).addCallId(softphone).addUserAgent()
			sipMessage.body = fmt.Sprintf(`<Msg><Hdr SID="%s" Req="%s" From="%s" To="%s" Cmd="17"/><Bdy Cln="%s"/></Msg>`, msg.Hdr.SID, msg.Hdr.Req, msg.Hdr.To, msg.Hdr.From, softphone.sipInfo.AuthorizationId)
			softphone.request(sipMessage, "SIP/2.0 200 OK")

			softphone.OnInvite(inviteMessage)
		}
	}
}
