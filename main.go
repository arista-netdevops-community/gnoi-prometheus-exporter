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
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	gnoiping "github.com/arista-netdevops-community/gnoi-prometheus-exporter/pkg/gnoiping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

//Struct for unmarshalling config.yaml
type Dests struct {
	Destinations []struct {
		Name        string `yaml:"name"`
		Destination string `yaml:"destination"`
	} `yaml:"destinations"`
}

//Struct for the return data of the gnoiping
type ReturnData struct {
	ReturnRTT          int
	Name               string
	CurrentDestination string
}

//Function to make sure that we are using necessary flags.
func checkflags(flag ...string) {
	for _, f := range flag {
		if f == "" {
			fmt.Printf("You have an empty flag please fix.")
			os.Exit(1)
		}
	}
}

//initialize the prometheus metric.
var gNOI_Ping = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "gNOI_Ping",
		Help: "Using Ping gNOI for destinations",
	},
	[]string{"name", "destination"},
)

func main() {
	// Add input parameters
	username := flag.String("username", "admin", "username for connection to gNOI")
	password := flag.String("password", "admin", "password for connection to gNOI")
	target := flag.String("target", "", "Target ip or hostname of the device running gNOI")
	yamlcfg := flag.String("config", "config.yaml", "Name of the config yaml file")
	interval := flag.Int("interval", 1, "Ping interval in time")
	path := flag.String("path", "/metrics", "Path if different than original")
	port := flag.Int("port", 9101, "Port to run prometheus on")
	flag.Parse()
	// Check for empty flags that are strings
	checkflags(*username, *password, *target, *yamlcfg)

	var c Dests

	//open up config.yaml to hold the token and other variables.
	yamlFile, err := ioutil.ReadFile(*yamlcfg)
	if err != nil {
		log.Print("Cannot find yaml file", err)
		os.Exit(1)
	}
	//unmarshall the yaml file
	yaml.Unmarshal(yamlFile, &c)
	//Initialize a map for the destination and ip
	Destmap := make(map[string]string)
	//Iterrate through the map and add values
	for _, v := range c.Destinations {
		Destmap[v.Name] = v.Destination
	}
	//Concurrently ping calling the NewGnoiPing method
	go func() {
		for {
			g := &gnoiping.NewGnoiPing{
				Username:    *username,
				Password:    *password,
				Target:      *target,
				Destination: Destmap,
				Interval:    *interval,
			}
			rtt, nameofdest, CurrentDestination, err := g.ConnectGnoi()
			if err != nil {
				log.Fatal(err)
			}
			//Structure the data for ping
			r := ReturnData{ReturnRTT: rtt, Name: nameofdest, CurrentDestination: CurrentDestination}
			//Extract the metrics and add them to Prometheus labels and the RTT as the value
			gNOI_Ping.With(prometheus.Labels{
				"name":        r.Name,
				"destination": r.CurrentDestination,
			}).Set(float64(r.ReturnRTT))
		}

	}()
	if *path == "" {
		http.Handle("/metrics", promhttp.Handler())
	} else {
		http.Handle(*path, promhttp.Handler())
	}

	if *port == 0 {
		*port = 9101
	}
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

//Intialize before hand the gNOI_Ping metric.
func init() {
	prometheus.Register(gNOI_Ping)
}
