package softphone

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc"
	"github.com/ringcentral/ringcentral-go"
	"github.com/ringcentral/ringcentral-go/definitions"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
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
	println(sipMessage.ToString())
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
	sipMessage.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", softphone.SipInfo.Username, softphone.SipInfo.Domain, softphone.fromTag)
	sipMessage.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", softphone.SipInfo.Username, softphone.SipInfo.Domain)
	sipMessage.addCseq(softphone).addCallId(*softphone).addUserAgent()
	message := softphone.request(sipMessage, "Www-Authenticate: Digest")

	authenticateHeader := SipMessage{}.FromString(message).Headers["Www-Authenticate"]
	regex := regexp.MustCompile(`, nonce="(.+?)"`)
	nonce := regex.FindStringSubmatch(authenticateHeader)[1]
	sipMessage.addAuthorization(*softphone, nonce).addCseq(softphone).newViaBranch()
	message = softphone.request(sipMessage, "SIP/2.0 200 OK")
}

func (softphone Softphone) WaitForIncomingCall() {
	for {
		message := <- softphone.messages
		if(strings.HasPrefix(message, "INVITE sip:")) {
			inviteMessage := SipMessage{}.FromString(message)

			var re = regexp.MustCompile(`\r\nm=audio (.+?)\r\n`)
			sdp := re.ReplaceAllString(inviteMessage.Body, "\r\nm=audio $1\r\na=mid:1\r\n")
			println(sdp)

			offer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP: sdp,
			}

			mediaEngine := webrtc.MediaEngine{}
			mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
			err := mediaEngine.PopulateFromSDP(offer)
			if err != nil {
				panic(err)
			}

			audioCodec := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeAudio)[0]

			api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))

			config := webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{"stun:74.125.194.127:19302"},
					},
				},
			}

			peerConnection, err := api.NewPeerConnection(config)
			if err != nil {
				panic(err)
			}
			// Set the remote SessionDescription
			err = peerConnection.SetRemoteDescription(offer)
			if err != nil {
				panic(err)
			}

			// Create Track that we send video back to browser on
			audioTrack, err := peerConnection.NewTrack(audioCodec.PayloadType, rand.Uint32(), "audio", "pion")
			if err != nil {
				panic(err)
			}

			// Add this newly created track to the PeerConnection
			if _, err = peerConnection.AddTrack(audioTrack); err != nil {
				panic(err)
			}

			// Create an answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				panic(err)
			}

			// Sets the LocalDescription, and starts our UDP listeners
			err = peerConnection.SetLocalDescription(answer)
			if err != nil {
				panic(err)
			}

			dict := make(map[string]string)
			dict["Contact"] = fmt.Sprintf("<sip:%s;transport=ws>", softphone.fakeEmail)
			dict["Content-Type"] = "application/sdp"
			responseMsg := inviteMessage.Response(softphone, 200, dict, answer.SDP)
			println(responseMsg)
			softphone.wsConn.WriteMessage(1, []byte(responseMsg))

			// Block forever
			select {}
		}
	}
}
