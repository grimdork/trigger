// Copyright (c) 2021 Ronny Bangsund
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package main

import (
	"fmt"
	"os"

	"github.com/Urethramancer/daemon"
	"github.com/Urethramancer/signor/opt"
)

var o struct {
	opt.DefaultHelp
}

func main() {
	a := opt.Parse(&o)
	if o.Help {
		a.Usage()
		return
	}

	srv, err := NewServer()
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err.Error())
		os.Exit(2)
	}

	srv.Start()
	<-daemon.BreakChannel()
	srv.Stop()
}
