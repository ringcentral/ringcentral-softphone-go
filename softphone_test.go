package softphone

import (
	"encoding/json"
	"log"
	"os"
	"os/user"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v3"
	"github.com/ringcentral/ringcentral-go"
)

func loadDotEnv() {
	usr, err := user.Current()
	if err == nil {
		// ignore error because we fallback to sys env vars
		godotenv.Overload(usr.HomeDir + "/.env.prod")
	}
}

func TestSoftphone(t *testing.T) {
	loadDotEnv()
	rc := ringcentral.RestClient{
		ClientID:     os.Getenv("RINGCENTRAL_CLIENT_ID"),
		ClientSecret: os.Getenv("RINGCENTRAL_CLIENT_SECRET"),
		Server:       os.Getenv("RINGCENTRAL_SERVER_URL"),
	}

	rc.Authorize(ringcentral.GetTokenRequest{
		GrantType: "password",
		Username:  os.Getenv("RINGCENTRAL_USERNAME"),
		Extension: os.Getenv("RINGCENTRAL_EXTENSION"),
		Password:  os.Getenv("RINGCENTRAL_PASSWORD"),
	})

	bytes := rc.Post("/restapi/v1.0/client-info/sip-provision", strings.NewReader(`{"sipInfo":[{"transport":"WSS"}]}`))
	var createSipRegistrationResponse ringcentral.CreateSipRegistrationResponse
	json.Unmarshal(bytes, &createSipRegistrationResponse)

	if len(createSipRegistrationResponse.SipInfo) <= 0 {
		t.Error("No SipInfo")
	}

	softphone := Softphone{
		CreateSipRegistrationResponse: createSipRegistrationResponse,
	}
	softphone.Register()

	softphone.OnInvite = func(inviteMessage SipMessage) {
		softphone.Answer(inviteMessage)
	}

	softphone.OnTrack = func(track *webrtc.TrackRemote) {
		fileName := "temp.raw"
		os.Remove(fileName)
		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		for {
			rtp, _, err := track.ReadRTP()
			if err != nil {
				log.Fatal(err)
			}
			// g711.DecodeUlaw(...): change u-law PCM to LPCM
			f.Write(rtp.Payload)
		}
	}

	rc.Revoke()

	select {} //block forever
}
