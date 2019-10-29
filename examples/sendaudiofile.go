package main

import (
	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v2/pkg/media"
	"github.com/ringcentral/ringcentral-go"
	sp "github.com/ringcentral/ringcentral-softphone-go"
	"io"
	"log"
	"os"
	"os/user"
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

		go func() {
			file, err := os.Open("temp.raw")
			if err != nil {
				log.Fatal(err)
			}
			p := make([]byte, 8000)
			for {
				_, err := io.ReadFull(file, p)
				if err != nil {
					break
				}
				softphone.AudioTrack.WriteSample(media.Sample{p, 8000})
				file.Seek(8000, 1)
			}
		}()
	}

	softphone.OpenToInvite()

	// Block forever
	select {}
}
