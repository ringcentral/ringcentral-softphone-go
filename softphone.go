package softphone

import (
	"encoding/xml"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"github.com/ringcentral/ringcentral-go"
	"github.com/ringcentral/ringcentral-go/definitions"
	"os"
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
	OnTrack func(track *webrtc.Track)
}

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
			//// to workaround a pion/webrtc bug: https://github.com/pion/webrtc/issues/879
			sdp := re.ReplaceAllString(inviteMessage.Body, "\r\na=rtpmap:111 OPUS/48000/2\r\na=mid:0\r\n")
			//sdp := inviteMessage.Body

			offer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  sdp,
			}

			mediaEngine := webrtc.MediaEngine{}
			//mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
			mediaEngine.RegisterCodec(webrtc.NewRTPPCMUCodec(webrtc.DefaultPayloadTypePCMU, 8000))
			//mediaEngine.RegisterCodec(webrtc.NewRTPPCMACodec(webrtc.DefaultPayloadTypePCMA, 8000))
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

				f, err := os.OpenFile("temp.raw", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					panic(err)
				}

				defer f.Close()

				codec := track.Codec()
				if codec.Name == webrtc.PCMU {
					fmt.Println("Got PCMU track")
					softphone.OnTrack(track)
				}
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
