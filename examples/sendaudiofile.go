package main

import (
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

	// todo: read audio file to remote

	softphone.OpenToInvite()

	// Block forever
	select {}
}
