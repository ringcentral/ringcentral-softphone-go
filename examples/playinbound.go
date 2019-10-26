package main

import (
	"log"
	"os"
	"os/user"

	"github.com/hajimehoshi/oto"
	"github.com/pion/webrtc/v2"
	"github.com/zaf/g711"

	"github.com/joho/godotenv"
	"github.com/ringcentral/ringcentral-go"
	sp "github.com/ringcentral/ringcentral-softphone-go"
)

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	err = godotenv.Overload(usr.HomeDir + "/.env.prod")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	rc := ringcentral.RestClient{
		ClientID:     os.Getenv("RINGCENTRAL_CLIENT_ID"),
		ClientSecret: os.Getenv("RINGCENTRAL_CLIENT_SECRET"),
		Server:       os.Getenv("RINGCENTRAL_SERVER_URL"),
	}
	rc.Authorize(
		os.Getenv("RINGCENTRAL_USERNAME"),
		os.Getenv("RINGCENTRAL_EXTENSION"),
		os.Getenv("RINGCENTRAL_PASSWORD"),
	)
	softphone := sp.NewSoftPhone(rc)

	softphone.OnInvite = func(inviteMessage sp.SipMessage) {
		softphone.Answer(inviteMessage)
	}

	softphone.OnTrack = func(track *webrtc.Track) {
		player, err := oto.NewPlayer(8000, 1, 2, 1)
		if err != nil {
			log.Fatal(err)
		}
		for {
			rtp, err := track.ReadRTP()
			if err != nil {
				log.Fatal(err)
			}
			// g711.DecodeUlaw(...): change u-law PCM to LPCM
			player.Write(g711.DecodeUlaw(rtp.Payload))
		}
	}

	softphone.Register()
	softphone.WaitForIncomingCall()

	// Block forever
	select {}
}
