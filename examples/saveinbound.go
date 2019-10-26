package main

import (
	"github.com/pion/webrtc/v2"
	"github.com/zaf/g711"
	"log"
	"os"
	"os/user"

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
	softphone.Register()

	softphone.OnTrack = func(track *webrtc.Track) {
		fileName := "temp.raw"
		os.Remove(fileName)
		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		for {
			rtp, err := track.ReadRTP()
			if err != nil {
				log.Fatal(err)
			}
			// g711.DecodeUlaw(...): change u-law PCM to LPCM
			f.Write(g711.DecodeUlaw(rtp.Payload))
		}
	}

	softphone.WaitForIncomingCall()
}

// to play the saved audio:  play -b 16 -e signed -c 1 -r 8000 temp.raw
