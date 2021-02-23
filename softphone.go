package softphone

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/ringcentral/ringcentral-go"
)

// Softphone softphone
type Softphone struct {
	CreateSipRegistrationResponse ringcentral.CreateSipRegistrationResponse
	MessageListeners              map[string]func(string)
	Conn                          *websocket.Conn
	OnInvite                      func(inviteMessage SipMessage)
	OnTrack                       func(track *webrtc.TrackRemote)
}

var fakeDomain = fmt.Sprintf("%s.invalid", uuid.New().String())
var fakeEmail = fmt.Sprintf("%s@%s", uuid.New().String(), fakeDomain)

// Register register the softphone
func (softphone *Softphone) Register() {
	softphone.MessageListeners = make(map[string]func(string))
	sipInfo := softphone.CreateSipRegistrationResponse.SipInfo[0]
	url := url.URL{Scheme: strings.ToLower(sipInfo.Transport), Host: sipInfo.OutboundProxy, Path: ""}
	dialer := websocket.DefaultDialer
	dialer.Subprotocols = []string{"sip"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		panic(err)
	}
	softphone.Conn = conn
	go func() {
		for {
			_, bytes, err := softphone.Conn.ReadMessage()
			if err != nil {
				panic(err)
			}
			message := string(bytes)
			log.Println("↓↓↓\n", message)
			for _, messageListener := range softphone.MessageListeners {
				go messageListener(message)
			}
		}
	}()

	registerMessage := SipMessage{
		Subject: fmt.Sprintf("REGISTER sip:%s SIP/2.0", sipInfo.Domain),
		Headers: map[string]string{
			"Call-ID": uuid.New().String(),
			"Contact": fmt.Sprintf("<sip:%s;transport=ws>;expires=600", fakeEmail),
			"Via":     fmt.Sprintf("SIP/2.0/WSS %s;branch=z9hG4bK%s", fakeDomain, uuid.New().String()),
			"From":    fmt.Sprintf("<sip:%s@%s>;tag=%s", sipInfo.Username, sipInfo.Domain, uuid.New().String()),
			"To":      fmt.Sprintf("<sip:%s@%s>", sipInfo.Username, sipInfo.Domain),
			"CSeq":    "8082 REGISTER",
		},
		Body: "",
	}
	softphone.Send(registerMessage, func(strMessage string) bool {
		if strings.Contains(strMessage, "SIP/2.0 401 Unauthorized") {
			unAuthMessage := FromStringToSipMessage(strMessage)
			authHeader := unAuthMessage.Headers["WWW-Authenticate"]
			regex := regexp.MustCompile(", nonce=\"(.+?)\"")
			match := regex.FindStringSubmatch(authHeader)
			nonce := match[1]

			registerMessage.Headers["Authorization"] = GenerateAuthorization(sipInfo, "REGISTER", nonce)
			registerMessage.IncreaseSeq()
			registerMessage.Headers["Via"] = fmt.Sprintf("SIP/2.0/TCP %s;branch=z9hG4bK%s", fakeDomain, uuid.New().String())
			softphone.Send(registerMessage, nil)

			return true
		}
		return false
	})

	softphone.addMessageListener(func(strMessage string) {
		if strings.Contains(strMessage, "INVITE sip:") {
			softphone.OnInvite(FromStringToSipMessage(strMessage))
		}
	})
}

// Answer answer an incoming call
func (softphone *Softphone) Answer(inviteMessage SipMessage) {
	var re = regexp.MustCompile(`\r\na=rtpmap:111 OPUS/48000/2\r\n`)
	// to workaround a pion/webrtc bug: https://github.com/pion/webrtc/issues/879
	sdp := re.ReplaceAllString(inviteMessage.Body, "\r\na=rtpmap:111 OPUS/48000/2\r\na=mid:0\r\n")
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}
	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/pcmu", ClockRate: 48000, Channels: 2, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))
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
	peerConnection.CreateDataChannel("audio", nil)

	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}
	<-gatherComplete

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("OnTrack")
		codec := track.Codec()
		log.Println(codec)
		if softphone.OnTrack != nil {
			softphone.OnTrack(track)
		}
	})

	responseMessage := SipMessage{
		Subject: "SIP/2.0 200 OK",
		Headers: map[string]string{
			"Contact":      fmt.Sprintf("<sip:%s;transport=ws>", fakeEmail),
			"Content-Type": "application/sdp",
			"Via":          inviteMessage.Headers["Via"],
			"From":         inviteMessage.Headers["From"],
			"CSeq":         inviteMessage.Headers["CSeq"],
			"Call-Id":      inviteMessage.Headers["Call-Id"],
			"To":           fmt.Sprintf("%s;tag=%s", inviteMessage.Headers["To"], uuid.New().String()),
		},
		Body: peerConnection.LocalDescription().SDP,
	}

	softphone.Send(responseMessage, nil)
}
