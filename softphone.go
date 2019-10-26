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
	messageListeners map[string]func(string)
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

	softphone.messageListeners = make(map[string]func(string))
	softphone.OnInvite = func(inviteMessage SipMessage) {}
	softphone.OnTrack = func(track *webrtc.Track) {}

	softphone.register()

	return &softphone
}

func (softphone *Softphone) addMessageListener(messageListener func(string)) string {
	key := uuid.New().String()
	softphone.messageListeners[key] = messageListener
	return key
}
func (softphone *Softphone) removeMessageListener(key string) {
	delete(softphone.messageListeners, key)
}

func (softphone *Softphone) request(sipMessage SipMessage, responseHandler func(string)bool) {
	println(sipMessage.ToString())
	if responseHandler != nil {
		var key string
		key = softphone.addMessageListener(func(message string) {
			done := responseHandler(message)
			if done {
				softphone.removeMessageListener(key)
			}
		})
	}
	softphone.wsConn.WriteMessage(1, []byte(sipMessage.ToString()))
}

func (softphone *Softphone) WaitForIncomingCall() {
	softphone.addMessageListener(func(message string) {
		if strings.HasPrefix(message, "INVITE sip:") {
			inviteMessage := SipMessage{}.FromString(message)

			dict := map[string]string{"Contact": fmt.Sprintf(`<sip:%s;transport=ws>`, softphone.fakeDomain)}
			responseMsg := inviteMessage.Response(*softphone, 180, dict, "")
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
			sipMessage.addCseq(softphone).addCallId(*softphone).addUserAgent()
			sipMessage.body = fmt.Sprintf(`<Msg><Hdr SID="%s" Req="%s" From="%s" To="%s" Cmd="17"/><Bdy Cln="%s"/></Msg>`, msg.Hdr.SID, msg.Hdr.Req, msg.Hdr.To, msg.Hdr.From, softphone.sipInfo.AuthorizationId)
			softphone.request(sipMessage, nil)

			softphone.OnInvite(inviteMessage)
		}
	})
}
