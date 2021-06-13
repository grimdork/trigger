// Copyright (c) 2021 Ronny Bangsund
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/fsnotify/fsnotify"
)

// TriggerHook is called on each event in a directory.
type TriggerHook func(fsnotify.Event)

// Trigger holds a watcher, path and its callback.
type Trigger struct {
	// Name for reference.
	Name string
	// Subject for the mail message.
	Subject string
	// Path to watch.
	Path string
	// Modes to react to.
	Modes []string
	// Watchers is a list of e-mail addresses.
	Watchers []string

	sync.WaitGroup
	w         *fsnotify.Watcher
	op        fsnotify.Op
	quit      chan interface{}
	log       []string
	loglock   sync.Mutex
	mailqueue chan Message
}

// LoadTrigger from file.
func (srv *Server) LoadTrigger(fn string) error {
	tr := Trigger{
		quit:      make(chan interface{}),
		mailqueue: srv.mailqueue,
	}
	f, err := os.Open(fn)
	if err != nil {
		return err
	}

	defer f.Close()
	dec := gojay.NewDecoder(f)
	err = dec.DecodeObject(&tr)
	if err != nil {
		return err
	}

	tr.w, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	tr.w.Add(tr.Path)
	srv.triggers[tr.Name] = &tr
	for _, x := range tr.Modes {
		switch x {
		case "create":
			tr.op |= fsnotify.Create
		case "chmod":
			tr.op |= fsnotify.Chmod
		case "remove":
			tr.op |= fsnotify.Remove
		case "write":
			tr.op |= fsnotify.Write
		}
	}
	return nil
}

// UnmarshalJSONObject decodes JSON to Trigger.
func (tr *Trigger) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "name":
		return dec.String(&tr.Name)
	case "subject":
		return dec.String(&tr.Subject)
	case "paths":
		return dec.String(&tr.Path)
	case "modes":
		return dec.SliceString(&tr.Modes)
	case "watchers":
		return dec.SliceString(&tr.Watchers)
	}
	return nil
}

// NKeys in a trigger path.
func (tg *Trigger) NKeys() int {
	return 5
}

// Start watching a directory.
func (tr *Trigger) Start() {
	tr.Add(1)
	go func() {
		t := time.NewTicker(time.Minute * 5)
		defer tr.w.Close()
		for {
			select {
			case event, ok := <-tr.w.Events:
				if ok {
					go tr.trigger(event)
				}

			case <-t.C:
				tr.SendLogs()

			case <-tr.quit:
				t.Stop()
				tr.Done()
				return

			default:
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
}

// Quit shuts down the watcher.
func (tr *Trigger) Quit() {
	tr.quit <- true
	tr.Wait()
	tr.SendLogs()
}

// SendLogs creates a message with all events so far and mails it to the watchers.
func (tr *Trigger) SendLogs() {
	if len(tr.log) == 0 {
		return
	}

	tr.loglock.Lock()
	b := strings.Builder{}
	for _, s := range tr.log {
		b.WriteString(s)
	}
	tr.log = []string{}
	tr.loglock.Unlock()
	msg := Message{
		Recipients: tr.Watchers,
		Subject:    tr.Subject,
		Body:       b.String(),
	}
	tr.mailqueue <- msg
}

func (tr *Trigger) trigger(event fsnotify.Event) {
	if event.Op&tr.op == 0 {
		return
	}

	fn := filepath.Join(tr.Path, event.Name)
	tr.loglock.Lock()
	s := fmt.Sprintf("%s: %s %s\n", NowString(), event.Op.String(), fn)
	tr.log = append(tr.log, s)
	tr.loglock.Unlock()
}

const timeFmt = "%s %s %02d %02d:%02d:%02d.%06d %d"

// NowString returns a very detailed time string.
func NowString() string {
	t := time.Now()
	return fmt.Sprintf(timeFmt, t.Weekday().String()[0:3], t.Month().String()[0:3], t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Year())
}
