// Copyright (c) 2021 Ronny Bangsund
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package main

import (
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Urethramancer/signor/env"
	"github.com/Urethramancer/signor/log"
)

// Derver holds the basic runtime configuration and subtasks.
type Server struct {
	sync.WaitGroup
	log.LogShortcuts
	triggerpath string // Path to trigger configurations
	triggers    map[string]*Trigger
	msgsrv      string // Message server to shout through
	msgtoken    string // Token for message server access
	auth        smtp.Auth
	mailserver  string
	sender      string
	mailqueue   chan Message
	mailquit    chan interface{}
}

// NewServer creates a server with an embedded sweb server.
func NewServer() (*Server, error) {
	srv := &Server{
		triggers:  make(map[string]*Trigger),
		msgsrv:    os.Getenv("MESSAGE_SERVER"),
		msgtoken:  os.Getenv("MESSAGE_TOKEN"),
		mailqueue: make(chan Message, 10),
		mailquit:  make(chan interface{}),
	}

	var err error
	srv.triggerpath, err = filepath.Abs(strings.TrimSpace(env.Get("TRIGGERS_PATH", "triggers")))
	if err != nil {
		return nil, err
	}

	srv.Logger = log.Default
	srv.L = log.Default.TMsg
	srv.E = log.Default.TErr

	err = srv.ConfigureMail(os.Getenv("MAILHOST"))
	if err != nil {
		srv.E("Error: %s", err.Error())
	}

	dir, err := os.ReadDir(srv.triggerpath)
	if err != nil {
		return nil, err
	}

	for _, d := range dir {
		fn := filepath.Join(srv.triggerpath, d.Name())
		err := srv.LoadTrigger(fn)
		if err != nil {
			srv.E("Error loading trigger: %s", err.Error())
		}
	}

	return srv, nil
}

// Start the mailer background task.
func (srv *Server) Start() {
	srv.mailqueue = make(chan Message)
	for _, tr := range srv.triggers {
		tr.mailqueue = srv.mailqueue
		tr.Start()
	}

	srv.Add(1)
	go func() {
		for {
			select {
			case msg := <-srv.mailqueue:
				srv.sendMail(msg)

			case <-srv.mailquit:
				close(srv.mailqueue)
				srv.Done()
				return

			default:
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	srv.L("Started Trigger.")
}

// Stop the server.
func (srv *Server) Stop() {
	for _, tr := range srv.triggers {
		tr.Quit()
	}
	srv.mailquit <- true
	srv.Wait()
	srv.L("Stopped Trigger.")
}
