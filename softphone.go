package softphone

import (
	"encoding/json"
	"github.com/ringcentral/ringcentral"
	"github.com/ringcentral/ringcentral/definitions"
	"strings"
)

type Softphone struct {
	Rc ringcentral.RestClient
	SipInfo definitions.SIPInfoResponse
}

func (softphone *Softphone) Register() {
	bytes := softphone.Rc.Post("/restapi/v1.0/client-info/sip-provision", strings.NewReader(`{"sipInfo":[{"transport":"WSS"}]}`))
	var createSipRegistrationResponse definitions.CreateSipRegistrationResponse
	json.Unmarshal(bytes, &createSipRegistrationResponse)
	softphone.SipInfo = createSipRegistrationResponse.SipInfo[0]
}
