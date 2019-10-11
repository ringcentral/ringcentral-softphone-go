package softphone

import (
	"fmt"
	"strconv"
	"strings"
)

type SipMessage struct {
	Method string
	Address string
	Headers map[string]string
	Body string
}

func (sm *SipMessage) addContentLength() *SipMessage {
	sm.Headers["Content-Length"] = strconv.Itoa(len(sm.Body))
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