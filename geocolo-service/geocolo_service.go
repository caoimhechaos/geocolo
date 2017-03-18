/*-
 * Copyright (c) 2012-2016 Caoimhe Chaos <caoimhechaos@protonmail.com>,
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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"log"
	"net"

	"github.com/caoimhechaos/geocolo"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

func main() {
	var service *geocolo.GeoProximityService
	var server *grpc.Server
	var config *geocolo.GeoProximityServiceConfig
	var configpath string
	var listen_net, listen_ip string
	var listener net.Listener
	var bdata []byte
	var err error

	flag.StringVar(&configpath, "config", "",
		"Path to the geocolo service configuration")
	flag.StringVar(&listen_net, "listen-proto", "tcp",
		"Protocol type to listen on (e.g. tcp)")
	flag.StringVar(&listen_ip, "listen-addr", "[::]:1234",
		"IP address to listen on")
	flag.Parse()

	config = new(geocolo.GeoProximityServiceConfig)
	bdata, err = ioutil.ReadFile(configpath)
	if err != nil {
		log.Fatal("Error reading ", configpath, ": ", err)
	}

	err = proto.UnmarshalText(string(bdata), config)
	if err != nil {
		var err2 error = proto.Unmarshal(bdata, config)
		if err2 != nil {
			log.Print("Error parsing ", configpath, " as text: ",
				err)
			log.Fatal("Error parsing ", configpath, ": ", err2)
		}
	}

	service, err = geocolo.NewGeoProximityService(config)
	if err != nil {
		log.Fatal("Error creating GeoProximityService: ", err)
	}

	if config.ServiceCertificate != nil && config.ServiceKey != nil {
		var cert tls.Certificate
		var tls_config *tls.Config
		var root *x509.CertPool = x509.NewCertPool()
		var cacert *x509.Certificate
		var cablock *pem.Block
		var cadata []byte

		cert, err = tls.LoadX509KeyPair(config.GetServiceCertificate(),
			config.GetServiceKey())
		if err != nil {
			log.Fatal("Error loading X.509 key pair from ",
				config.GetServiceCertificate(), " and ",
				config.GetServiceKey(), ": ", err)
		}

		cadata, err = ioutil.ReadFile(config.GetCaCertificate())
		if err != nil {
			log.Fatal("Error reading CA certificate from ",
				config.GetCaCertificate(), ": ", err)
		}

		cablock, _ = pem.Decode(cadata)
		cacert, err = x509.ParseCertificate(cablock.Bytes)
		if err != nil {
			log.Fatal("Error parsing X.509 certificate ",
				config.GetCaCertificate(), ": ", err)
		}
		root.AddCert(cacert)

		tls_config = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
			RootCAs:      root,
		}
		listener, err = tls.Listen(listen_net, listen_ip, tls_config)
		if err != nil {
			log.Fatal("Error listening on ", listen_ip, ": ", err)
		}
	} else {
		listener, err = net.Listen(listen_net, listen_ip)
		if err != nil {
			log.Fatal("Error listening on ", listen_ip, ": ", err)
		}
	}

	server = grpc.NewServer()
	geocolo.RegisterGeoProximityServiceServer(server, service)
	server.Serve(listener)
}
