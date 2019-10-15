package softphone

import (
	"crypto/md5"
	"fmt"
	"github.com/ringcentral/ringcentral-go/definitions"
)

func GenerateResponse(username, password, realm, method, uri, nonce string) string {
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	println(fmt.Sprintf("%x", ha1))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	println(fmt.Sprintf("%x", ha2))
	response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%x", ha1, nonce, ha2)))
	return fmt.Sprintf("%x", response)
}

func generateAuthorization(sipInfo definitions.SIPInfoResponse, method, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, sipInfo.Domain,
		GenerateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+sipInfo.Domain, nonce),
	)
}

func generateProxyAuthorization(sipInfo definitions.SIPInfoResponse, method, targetUser, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s@%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, targetUser, sipInfo.Domain,
		GenerateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+targetUser+"@"+sipInfo.Domain, nonce),
	)
}
