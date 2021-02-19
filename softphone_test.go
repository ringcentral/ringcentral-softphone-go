package softphone

import (
	"encoding/json"
	"log"
	"os"
	"os/user"
	"strings"
	"testing"

	"github.com/joho/godotenv"
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
		log.Println("OnInvite handler")
	}

	rc.Revoke()

	select {} //block forever
}
