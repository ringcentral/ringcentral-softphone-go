package softphone

import (
	"encoding/xml"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v2"
	"github.com/ringcentral/ringcentral-go"
	"github.com/ringcentral/ringcentral-go/definitions"
	"strings"
)

type Softphone struct {
	Rc         ringcentral.RestClient
	Device     definitions.SipRegistrationDeviceInfo
	OnTrack    func(track *webrtc.Track)
	OnInvite	func(inviteMessage SipMessage)

	sipInfo    definitions.SIPInfoResponse
	wsConn     *websocket.Conn
	fakeDomain string
	fakeEmail  string
	fromTag    string
	toTag      string
	callId     string
	cseq       int

	responses  chan string
	notifications chan string
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
