package softphone

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/ringcentral/ringcentral-go"
)

func TestAuthorize(t *testing.T) {
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
		createSipRegistrationResponse,
	}
	softphone.Register()

	rc.Revoke()
}

func TestSipMessage(t *testing.T) {
	sipMessage := SipMessage{
		Subject: "SIP/2.0 100 Trying",
		Headers: map[string]string{
			"CSeq":    "8082 REGISTER",
			"Call-ID": "21ee3d44-98d6-4bde-b541-fdc4dce63b13",
		},
		Body: "",
	}
	log.Println(sipMessage.ToString())
}
