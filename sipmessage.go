package softphone

import (
	"fmt"
	"regexp"
	"strings"
)

type SipMessage struct {
	Method string
	Address string
	Subject string
	Headers map[string]string
	Body string
}

func (sm *SipMessage) addAuthorization(softphone Softphone, nonce string) *SipMessage {
	sm.Headers["Authorization"] = generateAuthorization(softphone.SipInfo, "REGISTER", nonce)
	return sm
}

func (sm *SipMessage) newViaBranch() *SipMessage {
	if val, ok := sm.Headers["Via"]; ok {
		sm.Headers["Via"] = regexp.MustCompile(";branch=z9hG4bK.+?$").ReplaceAllString(val, ";branch="+branch())
	}
	return sm
}

func (sm *SipMessage) addCseq(softphone *Softphone) *SipMessage {
	sm.Headers["CSeq"] = fmt.Sprintf("%d %s", softphone.cseq, sm.Method)
	softphone.cseq += 1
	return sm
}

func (sm *SipMessage) addCallId(softphone Softphone) *SipMessage {
	sm.Headers["Call-ID"] = softphone.callId
	return sm
}

func (sm *SipMessage) addUserAgent() *SipMessage {
	sm.Headers["User-Agent"] = "ringcentral-softphone-go"
	return sm
}

func (sm SipMessage) ToString() string {
	arr := []string { fmt.Sprintf("%s sip:%s SIP/2.0", sm.Method, sm.Address) }
	for k, v := range sm.Headers {
		arr = append(arr, fmt.Sprintf("%s: %s", k, v))
	}
	arr = append(arr, fmt.Sprintf("Content-Length: %d", len(sm.Body)))
	arr = append(arr, "", sm.Body)
	return strings.Join(arr, "\r\n")
}

func (sm SipMessage) FromString(s string) SipMessage {
	parts := strings.Split(s, "\r\n\r\n")
	sm.Body = strings.Join(parts[1:], "\r\n\r\n")
	parts = strings.Split(parts[0], "\r\n")
	sm.Subject = parts[0]
	sm.Headers = make(map[string]string)
	for _, line := range parts[1:] {
		tokens := strings.Split(line, ": ")
		sm.Headers[tokens[0]] = tokens[1]
	}
	return sm
}