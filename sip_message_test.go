package softphone

import (
	"fmt"
	"reflect"
	"testing"
)

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

	if sipMessage.Subject != sipMessage2.Subject {
		t.Error("SipMessage Subject was changed during transformation")
	}
	if sipMessage.Body != sipMessage2.Body {
		t.Error("SipMessage Body was changed during transformation")
	}
	sipMessage.Headers["Content-Length"] = fmt.Sprintf("%d", len(sipMessage.Body))
	if !reflect.DeepEqual(sipMessage.Headers, sipMessage2.Headers) {
		t.Error("SipMessage Headers was changed during transformation")
	}
}
