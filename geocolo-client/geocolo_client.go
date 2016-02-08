/*-
 * Copyright (c) 2012-2016 Tonnerre Lombard <tonnerre@ancient-solutions.com>,
 *                         Ancient Solutions. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 * 1. Redistributions  of source code must retain  the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions  in   binary  form  must   reproduce  the  above
 *    copyright  notice, this  list  of conditions  and the  following
 *    disclaimer in the  documentation and/or other materials provided
 *    with the distribution.
 *
 * THIS  SOFTWARE IS  PROVIDED BY  ANCIENT SOLUTIONS  AND CONTRIBUTORS
 * ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO,  THE IMPLIED WARRANTIES OF  MERCHANTABILITY AND FITNESS
 * FOR A  PARTICULAR PURPOSE  ARE DISCLAIMED.  IN  NO EVENT  SHALL THE
 * FOUNDATION  OR CONTRIBUTORS  BE  LIABLE FOR  ANY DIRECT,  INDIRECT,
 * INCIDENTAL,   SPECIAL,    EXEMPLARY,   OR   CONSEQUENTIAL   DAMAGES
 * (INCLUDING, BUT NOT LIMITED  TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE,  DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
 * STRICT  LIABILITY,  OR  TORT  (INCLUDING NEGLIGENCE  OR  OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED
 * OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package main

import (
	"flag"
	"fmt"
	"github.com/tonnerre/geocolo"
	"github.com/tonnerre/go-urlconnection"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	var endpoint, uri, origin, candidates string
	var cert, key, ca string
	var maxdistance float64
	var client geocolo.GeoProximityServiceClient
	var mode string
	var conn *grpc.ClientConn
	var ctx context.Context = context.Background()
	var timeout time.Duration
	var detailed bool
	var err error

	flag.StringVar(&endpoint, "endpoint", "",
		"The service URL to connect to")
	flag.StringVar(&uri, "etcd-uri", os.Getenv("ETCD_URI"),
		"etcd URI to connect to")
	flag.StringVar(&origin, "origin", "",
		"Country which we're looking for close countries for")
	flag.StringVar(&candidates, "candidates", "",
		"Comma separated list of countries to consider")
	flag.StringVar(&mode, "mode", "country",
		"Method to contact (country or ip)")
	flag.Float64Var(&maxdistance, "max-distance", 0,
		"Maximum distance from the closest IP to consider")
	flag.BoolVar(&detailed, "detailed", false,
		"Whether to give a detailed response")

	flag.StringVar(&cert, "cert", "",
		"Certificate for connecting (if empty, don't use encryption)")
	flag.StringVar(&key, "key", "",
		"Private key for connecting")
	flag.StringVar(&ca, "ca-cert", "",
		"CA certificate for verifying etcd and geocolo")
	flag.DurationVar(&timeout, "timeout", 0*time.Nanosecond,
		"Maximum time to wait for a response from the server")
	flag.Parse()

	if uri != "" {
		if err = urlconnection.SetupEtcd([]string{uri},
			cert, key, ca); err != nil {
			log.Fatal("Error initializing etcd connection to ",
				uri, ": ", err.Error())
		}
	}

	if timeout.Nanoseconds() > 0 {
		ctx, _ = context.WithTimeout(ctx, timeout)
		conn, err = grpc.Dial(endpoint,
			grpc.WithDialer(urlconnection.ConnectTimeout),
			grpc.WithTimeout(timeout))
	} else {
		conn, err = grpc.Dial(endpoint,
			grpc.WithDialer(urlconnection.ConnectTimeout))
	}
	if err != nil {
		log.Fatal("Error connecting to ", endpoint, ": ", err.Error())
	}
	client = geocolo.NewGeoProximityServiceClient(conn)

	if mode == "country" {
		var req geocolo.GeoProximityRequest
		var res *geocolo.GeoProximityResponse

		if len(candidates) > 0 {
			req.Candidates = strings.Split(candidates, ",")
		}

		req.Origin = &origin
		req.DetailedResponse = &detailed

		res, err = client.GetProximity(ctx, &req)
		if err != nil {
			log.Fatal("Error sending proximity request: ",
				err.Error())
		}

		if res.Closest == nil {
			log.Fatal("Failed to fetch closest country")
		} else {
			fmt.Printf("Closest country: %s\n", *res.Closest)
		}

		for _, detail := range res.FullMap {
			if detail == nil {
				log.Print("Error: detail is nil?")
			} else if detail.Country == nil {
				log.Print("Error: country is nil?")
				if detail.Distance != nil {
					log.Printf("(distance was %f)",
						*detail.Distance)
				}
			} else if detail.Distance == nil {
				log.Print("Error: distance is nil?")
				if detail.Country != nil {
					log.Printf("(country was %s)", *detail.Country)
				}
			} else {
				fmt.Printf("Country %s: distance %f\n", *detail.Country,
					*detail.Distance)
			}
		}
	} else if mode == "ip" {
		var req geocolo.GeoProximityByIPRequest
		var res *geocolo.GeoProximityByIPResponse

		req.Candidates = strings.Split(candidates, ",")
		req.DetailedResponse = &detailed
		req.Origin = &origin
		req.MaxDistance = &maxdistance

		res, err = client.GetProximityByIP(ctx, &req)
		if err != nil {
			log.Fatal("Error sending proximity request: ",
				err.Error())
		}

		for _, addr := range res.Closest {
			fmt.Printf("Close IP: %s\n", addr)
		}

		for _, detail := range res.FullMap {
			fmt.Printf("IP: %s, distance: %f\n", *detail.Ip,
				*detail.Distance)
		}
	}
}
