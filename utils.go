package softphone

import (
	"crypto/md5"
	"fmt"

	"github.com/ringcentral/ringcentral-go"
)

// GenerateResponse generate response field in the authorization header
func GenerateResponse(username string, password string, realm string, method string, uri string, nonce string) string {
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%x", ha1, nonce, ha2)))
	return fmt.Sprintf("%x", response)
}

// GenerateAuthorization generate the authorization header
func GenerateAuthorization(sipInfo ringcentral.SIPInfoResponse, method string, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, sipInfo.Domain,
		GenerateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+sipInfo.Domain, nonce),
	)
}
