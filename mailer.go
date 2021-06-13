// Copyright (c) 2021 Ronny Bangsund
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package main

import (
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Message contains an e-mail message and recipients.
type Message struct {
	// Recipients to mail.
	Recipients []string
	// Subject of the message.
	Subject string
	// Body contains the logs.
	Body string
}

var errMalformedMailhost = errors.New("malformed mailhost string")

// GetMailConfig from user:oass@host:ip string.
func (srv *Server) ConfigureMail(s string) error {
	// Usernames may be e-mail addresses, so split on the first colon to get the username.
	u := strings.SplitN(s, ":", 2)
	if len(u) != 2 {
		return errMalformedMailhost
	}

	user := u[0]
	// Now split on the first @. This makes it an illegal password character, but that's not my problem.
	p := strings.Split(u[1], "@")
	if len(p) != 2 {
		return errMalformedMailhost
	}

	pass := p[0]
	// The rest of the string should now be host:port
	host, port, err := net.SplitHostPort(p[1])
	if err != nil {
		return err
	}

	srv.auth = smtp.PlainAuth("", user, pass, host)
	srv.mailserver = net.JoinHostPort(host, port)
	srv.sender = fmt.Sprintf("noreply@%s", host)
	return nil
}

func (srv *Server) sendMail(msg Message) {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(srv.sender)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(msg.Subject)
	b.WriteString("\r\n\r\n")
	b.WriteString(msg.Body)
	b.WriteString("\r\n")

	err := smtp.SendMail(srv.mailserver, srv.auth, srv.sender, msg.Recipients, []byte(b.String()))
	if err != nil {
		srv.E("Error sending mail: %s", err.Error())
	}
}
