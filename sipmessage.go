package softphone

import (
	"fmt"
	"strings"
)

type SipMessage struct {
	Subject string
	Headers map[string]string
	Body string
}


func (sm SipMessage) ToString() string {
	arr := []string { sm.Subject }
	for k, v := range sm.Headers {
		arr = append(arr, fmt.Sprintf("%s: %s", k, v))
	}
	arr = append(arr, fmt.Sprintf("Content-Length: %d", len(sm.Body)))
	arr = append(arr, "", sm.Body)
	return strings.Join(arr, "\r\n")
}