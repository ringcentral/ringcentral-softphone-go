package softphone

import (
	"encoding/json"
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

func TestAuthorize(t *testing.T) {
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

	rc.Revoke()

	select {} //block forever
}

func TestSipMessage(t *testing.T) {
	loadDotEnv()
	sipMessage := SipMessage{
		Subject: "SIP/2.0 100 Trying",
		Headers: map[string]string{
			"CSeq":    "8082 REGISTER",
			"Call-ID": "21ee3d44-98d6-4bde-b541-fdc4dce63b13",
		},
		Body: "",
	}

	sipMessage2 := FromStringToSipMessage(sipMessage.ToString())

	if sipMessage.ToString() != sipMessage2.ToString() {
		t.Error("SipMessage was changed during transformation")
	}
}
