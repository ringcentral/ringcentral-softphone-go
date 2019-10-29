package softphone

import (
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"regexp"
	"time"
)

func (softphone *Softphone) Answer(inviteMessage SipMessage) {
	var re = regexp.MustCompile(`\r\na=rtpmap:111 OPUS/48000/2\r\n`)
	// to workaround a pion/webrtc bug: https://github.com/pion/webrtc/issues/879
	sdp := re.ReplaceAllString(inviteMessage.body, "\r\na=rtpmap:111 OPUS/48000/2\r\na=mid:0\r\n")

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	mediaEngine := webrtc.MediaEngine{}
	err := mediaEngine.PopulateFromSDP(offer)
	if err != nil {
		panic(err)
	}
	mediaEngine.RegisterCodec(webrtc.NewRTPPCMUCodec(webrtc.DefaultPayloadTypePCMU, 8000))
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

	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		log.Debug("OnTrack")
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
				if errSend != nil {
					log.Debug(errSend)
				}
			}
		}()

		codec := track.Codec()
		if codec.Name == webrtc.PCMU {
			log.Debug("Got PCMU track")
			softphone.OnTrack(track)
		}
	})

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create Track that we send audio back to browser on
	audioTrack, err := peerConnection.NewTrack(audioCodec.PayloadType, rand.Uint32(), "audio", "pion")
	if err != nil {
		panic(err)
	}
	// Add this newly created track to the PeerConnection
	if _, err = peerConnection.AddTrack(audioTrack); err != nil {
		panic(err)
	}
	softphone.AudioTrack = audioTrack
	//audioTrack.WriteSample()

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	dict := map[string]string{
		"Contact":      fmt.Sprintf("<sip:%s;transport=ws>", softphone.fakeEmail),
		"Content-Type": "application/sdp",
	}
	responseMsg := inviteMessage.Response(*softphone, 200, dict, answer.SDP)
	softphone.response(responseMsg)
}
