package softphone

import "testing"

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
