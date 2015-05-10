/*-
 * Copyright (c) 2012 Caoimhe Chaos <caoimhechaos@protonmail.com>,
 *                    Ancient Solutions. All rights reserved.
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
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"

	"code.google.com/p/goprotobuf/proto"
	"github.com/caoimhechaos/geocolo"
)

func main() {
	var service *geocolo.GeoProximityService
	var exporter *exportedservice.ServiceExporter
	var config *geocolo.GeoProximityServiceConfig
	var configpath string
	var listen_net, listen_ip string
	var listener net.Listener
	var configfile *os.File
	var bdata []byte
	var err error

	flag.StringVar(&configpath, "config", "",
		"Path to the geocolo service configuration")
	flag.StringVar(&listen_net, "listen-proto", "tcp",
		"Protocol type to listen on (e.g. tcp)")
	flag.StringVar(&listen_ip, "listen-addr", "[::]",
		"IP address to listen on")
	flag.Parse()

	config = new(geocolo.GeoProximityServiceConfig)
	configfile, err = os.Open(configpath)
	if err != nil {
		log.Fatal("Error opening ", configpath, ": ", err)
	}

	bdata, err = ioutil.ReadAll(configfile)
	if err != nil {
		configfile.Close()
		log.Fatal("Error reading ", configpath, ": ", err)
	}

	err = proto.Unmarshal(bdata, config)
	if err != nil {
		configfile.Close()
		log.Fatal("Error parsing ", configpath, ": ", err)
	}

	err = configfile.Close()
	if err != nil {
		log.Print("Error closing ", configpath, ": ", err)
	}

	service, err = geocolo.NewGeoProximityService(config)
	if err != nil {
		log.Fatal("Error creating GeoProximityService: ", err)
	}

	rpc.Register(service)
	rpc.HandleHTTP()

	if config.ServiceCertificate != nil && config.ServiceKey != nil {
		exporter, err = exportedservice.NewTLSExporter(
			config.EtcdUrl, *config.ServiceCertificate,
			*config.ServiceKey, *config.CaCertificate)
		if err != nil {
			log.Fatal("Error opening port exporter: ", err)
		}
	} else {
		exporter, err = exportedservice.NewExporter(config.EtcdUrl)
		if err != nil {
			log.Fatal("Error opening port exporter: ", err)
		}
	}

	listener, err = exporter.NewExportedPort(listen_net, listen_ip,
		*config.ExportedServiceName)
	if err != nil {
		log.Fatal("Error opening exported port: ", err)
	}

	rpc.Accept(listener)
}
