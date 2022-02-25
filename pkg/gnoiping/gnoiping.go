//
// Copyright (c) 2016-2017, Arista Networks, Inc. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//   * Redistributions of source code must retain the above copyright notice,
//   this list of conditions and the following disclaimer.
//
//   * Redistributions in binary form must reproduce the above copyright
//   notice, this list of conditions and the following disclaimer in the
//   documentation and/or other materials provided with the distribution.
//
//   * Neither the name of Arista Networks nor the names of its
//   contributors may be used to endorse or promote products derived from
//   this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL ARISTA NETWORKS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR
// BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE
// OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN
// IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
package gnoiping

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/golang/glog"
	system "github.com/openconfig/gnoi/system"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type NewGnoiPing struct {
	Username    string
	Password    string
	Target      string
	Destination map[string]string
	Interval    int
	Source      string
	Time        int64
	TTl         int64
	Name        string
}

//Make the grpc connection
func (c *NewGnoiPing) ConnectGnoi() (int, string, string, error) {
	conn, err := grpc.Dial(c.Target, grpc.WithInsecure())
	if err != nil {
		log.Exitf("Failed to %s Error: %v", c.Target, err)
	}
	defer conn.Close()
	// Create the new grpc service connection
	Sys := system.NewSystemClient(conn)
	// pass in context blank information with the timeout.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// cancel when the function is over.
	defer cancel()
	// Since Metadata needs a map to pass into the header of gRPC request create a map for it.
	metamap := make(map[string]string)
	// Set the username and password
	metamap["username"] = c.Username
	metamap["password"] = c.Password
	// Set the metadata needed in the metadata package
	md := metadata.New(metamap)
	// set the ctx to use the metadata in every update.

	var CurrentDestination string

	ctx = metadata.NewOutgoingContext(ctx, md)
	for name, dest := range c.Destination {
		response, err := Sys.Ping(ctx, &system.PingRequest{Destination: dest})
		if err != nil {
			log.Errorf("Error trying to ping: %v", err)
			continue
		}
		PingResponse, err := response.Recv()
		if err != nil && err != io.EOF {
			log.Errorf("Ping failed.")
			continue
		}
		c.Time = PingResponse.Time
		c.Name = name
		CurrentDestination = dest
		fmt.Println(name, "\t", PingResponse)
		time.Sleep(time.Duration(c.Interval))
	}
	return int(c.Time), c.Name, CurrentDestination, nil
}
