package softphone

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media/oggwriter"
	"github.com/ringcentral/ringcentral-go"
	"github.com/ringcentral/ringcentral-go/definitions"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Softphone struct {
	Rc         ringcentral.RestClient
	SipInfo    definitions.SIPInfoResponse
	wsConn     *websocket.Conn
	fakeDomain string
	fakeEmail  string
	fromTag    string
	toTag      string
	callId     string
	cseq       int
	messages   chan string
}

type TrackReader struct {
	track *webrtc.Track
}

func (trackReader TrackReader) Read(p []byte) (n int, err error) {
	rtpPacket, err := trackReader.track.ReadRTP()
	if err != nil {
		return 0, err
	}
	return copy(p, rtpPacket.Payload), nil
}

func (TrackReader) Close() error { return nil }

func (softphone Softphone) request(sipMessage SipMessage, expectedResp string) string {
	println(sipMessage.ToString())
	softphone.wsConn.WriteMessage(1, []byte(sipMessage.ToString()))
	if expectedResp != "" {
		for {
			message := <-softphone.messages
			if (strings.Contains(message, expectedResp)) {
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
		message := <-softphone.messages
		if (strings.HasPrefix(message, "INVITE sip:")) {
			inviteMessage := SipMessage{}.FromString(message)

			dict := map[string]string{"Contact": fmt.Sprintf(`<sip:%s;transport=ws>`, softphone.fakeDomain)}
			responseMsg := inviteMessage.Response(softphone, 180, dict, "")
			println(responseMsg)
			softphone.wsConn.WriteMessage(1, []byte(responseMsg))

			var msg Msg
			xml.Unmarshal([]byte(inviteMessage.Headers["P-rc"]), &msg)
			sipMessage := SipMessage{}
			sipMessage.Method = "MESSAGE"
			sipMessage.Address = msg.Hdr.From
			sipMessage.Headers = make(map[string]string)
			sipMessage.Headers["Via"] = fmt.Sprintf("SIP/2.0/WSS %s;branch=%s", softphone.fakeDomain, branch())
			sipMessage.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", softphone.SipInfo.Username, softphone.SipInfo.Domain, softphone.fromTag)
			sipMessage.Headers["To"] = fmt.Sprintf("<sip:%s>", msg.Hdr.From)
			sipMessage.Headers["Content-Type"] = "x-rc/agent"
			sipMessage.addCseq(&softphone).addCallId(softphone).addUserAgent()
			sipMessage.Body = fmt.Sprintf(`<Msg><Hdr SID="%s" Req="%s" From="%s" To="%s" Cmd="17"/><Bdy Cln="%s"/></Msg>`, msg.Hdr.SID, msg.Hdr.Req, msg.Hdr.To, msg.Hdr.From, softphone.SipInfo.AuthorizationId)
			softphone.request(sipMessage, "SIP/2.0 200 OK")

			var re = regexp.MustCompile(`\r\na=rtpmap:111 OPUS/48000/2\r\n`)
			// to workaround a pion/webrtc bug: https://github.com/pion/webrtc/issues/879
			sdp := re.ReplaceAllString(inviteMessage.Body, "\r\na=rtpmap:111 OPUS/48000/2\r\na=mid:0\r\n")

			offer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  sdp,
			}

			mediaEngine := webrtc.MediaEngine{}
			mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
			err := mediaEngine.PopulateFromSDP(offer)
			if err != nil {
				panic(err)
			}

			//audioCodec := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeAudio)[0]

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

			if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
				panic(err)
			}

			//oggFile, err := oggwriter.New("output.ogg", 48000, 2)
			//if err != nil {
			//	panic(err)
			//}
			peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
				fmt.Printf("OnTrack\n")
				// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
				go func() {
					ticker := time.NewTicker(time.Second * 3)
					for range ticker.C {
						errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
						if errSend != nil {
							fmt.Println(errSend)
						}
					}
				}()

				codec := track.Codec()
				if codec.Name == webrtc.Opus {
					fmt.Println("Got Opus track, saving to disk as output.opus (48 kHz, 2 channels)")
					//saveToDisk(oggFile, track)
					//trackReader := TrackReader{}
					//trackReader.track = track
					//streamer, format, err := vorbis.Decode(trackReader)
					pr, pw := io.Pipe()
					go (func() {  // play the audio
						streamer, format, err := vorbis.Decode(ioutil.NopCloser(pr))
						if err != nil {
							log.Fatal(err)
						}
						speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
						speaker.Play(streamer)
					})()
					oggWritter, err := oggwriter.NewWith(pw, 48000, 2)
					if err != nil {
						log.Fatal(err)
					}
					saveToDisk(oggWritter, track)

				}
			})

			peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				fmt.Printf("OnICEConnectionStateChange %s \n", connectionState.String())
			})
			peerConnection.OnSignalingStateChange(func(state webrtc.SignalingState) {
				fmt.Printf("OnSignalingStateChange %s\n", state.String())
			})
			peerConnection.OnDataChannel(func(channel *webrtc.DataChannel) {
				fmt.Printf("OnDataChannel\n")
			})
			peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
				if candidate == nil {
					fmt.Printf("OnICECandidate nil\n")
				} else {
					fmt.Printf("OnICECandidate %s\n", candidate.String())
				}
			})
			peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
				fmt.Printf("OnConnectionStateChange %s\n", state.String())
			})
			peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
				fmt.Printf("OnICEGatheringStateChange %s\n", state.String())
			})

			// Set the remote SessionDescription
			err = peerConnection.SetRemoteDescription(offer)
			if err != nil {
				panic(err)
			}

			//// Create Track that we send audio back to browser on
			//audioTrack, err := peerConnection.NewTrack(audioCodec.PayloadType, rand.Uint32(), "audio", "pion")
			//if err != nil {
			//	panic(err)
			//}
			//
			//// Add this newly created track to the PeerConnection
			//if _, err = peerConnection.AddTrack(audioTrack); err != nil {
			//	panic(err)
			//}

			// Create an answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				panic(err)
			}
			err = peerConnection.SetLocalDescription(answer)
			if err != nil {
				panic(err)
			}

			dict = map[string]string{
				"Contact":      fmt.Sprintf("<sip:%s;transport=ws>", softphone.fakeEmail),
				"Content-Type": "application/sdp",
			}
			responseMsg = inviteMessage.Response(softphone, 200, dict, answer.SDP)
			println(responseMsg)
			softphone.wsConn.WriteMessage(1, []byte(responseMsg))

			// Block forever
			select {}
		}
	}
}
