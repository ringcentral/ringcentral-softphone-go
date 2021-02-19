package softphone

import (
	"fmt"
	"strings"
)

// SipMessage SIP message
type SipMessage struct {
	Subject string
	Headers map[string]string
	Body    string
}

// ToString from SipMessage to string message
func (sipMessage SipMessage) ToString() (message string) {
	list := []string{}
	list = append(list, sipMessage.Subject)
	for key, value := range sipMessage.Headers {
		list = append(list, fmt.Sprintf("%s: %s", key, value))
	}
	list = append(list, "")
	list = append(list, sipMessage.Body)
	return strings.Join(list, "\r\n")
}

// FromStringToSipMessage from string message to SipMessage
func FromStringToSipMessage(message string) (sipMessage SipMessage) {
	paragraphs := strings.Split(message, "\r\n\r\n")
	body := strings.Join(paragraphs[1:], "\r\n\r\n")
	paragraphs = strings.Split(paragraphs[0], "\r\n")
	subject := paragraphs[0]
	headers := make(map[string]string)
	for _, line := range paragraphs[1:] {
		tokens := strings.Split(line, ": ")
		headers[tokens[0]] = tokens[1]
	}
	return SipMessage{
		Subject: subject,
		Headers: headers,
		Body:    body,
	}
}
