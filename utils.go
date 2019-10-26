package softphone

import (
	"crypto/md5"
	"fmt"
	"github.com/google/uuid"
	"github.com/ringcentral/ringcentral-go/definitions"
	log "github.com/sirupsen/logrus"
	"os"
)

func generateResponse(username, password, realm, method, uri, nonce string) string {
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%x", ha1, nonce, ha2)))
	return fmt.Sprintf("%x", response)
}

func generateAuthorization(sipInfo definitions.SIPInfoResponse, method, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, sipInfo.Domain,
		generateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+sipInfo.Domain, nonce),
	)
}

func generateProxyAuthorization(sipInfo definitions.SIPInfoResponse, method, targetUser, nonce string) string {
	return fmt.Sprintf(
		`Digest algorithm=MD5, username="%s", realm="%s", nonce="%s", uri="sip:%s@%s", response="%s"`,
		sipInfo.AuthorizationId, sipInfo.Domain, nonce, targetUser, sipInfo.Domain,
		generateResponse(sipInfo.AuthorizationId, sipInfo.Password, sipInfo.Domain, method, "sip:"+targetUser+"@"+sipInfo.Domain, nonce),
	)
}

func branch() string {
	return "z9hG4bK" + uuid.New().String()
}

func configureLog() {
	logLevel := os.Getenv("RINGCENTRAL_SOFTPHONE_DEBUG")
	if logLevel == "all" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.FatalLevel)
	}
}
