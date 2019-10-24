package softphone

import (
	"crypto/md5"
	"fmt"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v2"
	"github.com/ringcentral/ringcentral-go/definitions"
)

func GenerateResponse(username, password, realm, method, uri, nonce string) string {
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%x", ha1, nonce, ha2)))
	return fmt.Sprintf("%x", response)
}

func generateAuthorization(sipInfo definitions.SIPInfoResponse, method, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, sipInfo.Domain,
		GenerateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+sipInfo.Domain, nonce),
	)
}

func generateProxyAuthorization(sipInfo definitions.SIPInfoResponse, method, targetUser, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s@%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, targetUser, sipInfo.Domain,
		GenerateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+targetUser+"@"+sipInfo.Domain, nonce),
	)
}

func branch() string {
	return "z9hG4bK" + uuid.New().String()
}

func debug(peerConnection *webrtc.PeerConnection) {
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
}