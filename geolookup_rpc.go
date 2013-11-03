/*-
 * Copyright (c) 2012 Tonnerre Lombard <tonnerre@ancient-solutions.com>,
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

package geocolo

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	libgeo "github.com/nranchev/go-libGeoIP"
	"net"
	"strings"
)

var (
	addr192 = net.IPv4(192, 168, 0, 0)
	addr172 = net.IPv4(172, 16, 0, 0)
	addr10 = net.IPv4(10, 0, 0, 0)

	mask8 = net.IPv4Mask(255, 0, 0, 0)
	mask16 = net.IPv4Mask(255, 255, 0, 0)

	net192 = net.IPNet{
		IP: addr192,
		Mask: mask16,
	}
	net172 = net.IPNet{
		IP: addr172,
		Mask: mask16,
	}
	net10 = net.IPNet{
		IP: addr10,
		Mask: mask8,
	}
	)

type GeoProximityService struct {
	conn            *sql.DB
	gi              *libgeo.GeoIP
	rfc1918_country string
}

func isRFC1918(addr string) bool {
	var ip net.IP = net.ParseIP(addr)

	if ip.To4() == nil {
		return false
	}
	return net192.Contains(ip) || net172.Contains(ip) || net10.Contains(ip)
}

func NewGeoProximityService(
	config *GeoProximityServiceConfig) (*GeoProximityService, error) {
	var gi *libgeo.GeoIP
	var dsn string
	var rfc1918 string
	var c *sql.DB
	var err error

	dsn = fmt.Sprintf("user=%s dbname=%s host=%s port=%d",
		*config.User, *config.Dbname, *config.Host, *config.Port)

	if config.Password != nil {
		dsn += fmt.Sprintf(" password=%s", *config.Password)
	}

	if config.Sslmode != nil {
		dsn += fmt.Sprintf(" sslmode=%s", *config.Sslmode)
	}

	c, err = sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if config.GeoipPath != nil {
		gi, err = libgeo.Load(*config.GeoipPath)
		if err != nil {
			return nil, err
		}
	}

	if config.Rfc1918Country != nil {
		rfc1918 = strings.ToUpper(*config.Rfc1918Country)
	}

	if rfc1918 == "" {
		rfc1918 = "CH"
	}

	return &GeoProximityService{
		conn: c,
		gi: gi,
		rfc1918_country: rfc1918,
	}, nil
}

func (self *GeoProximityService) GetProximity(req GeoProximityRequest,
	res *GeoProximityResponse) error {
	var rows *sql.Rows
	var err error

	if req.Origin == nil {
		return errors.New("No origin specified")
	}

	if len(req.Candidates) > 0 {
		var fullsql string
		var valuecollection []string
		var value string

		for _, value = range req.Candidates {
			value = strings.ToUpper(value)
			if len(value) == 2 && value[0] >= 'A' &&
				value[0] <= 'Z' && value[1] >= 'A' &&
				value[1] <= 'Z' {
				valuecollection = append(valuecollection,
					"'" + value + "'")
			}
		}

		fullsql = strings.Join(valuecollection, ",")

		rows, err = self.conn.Query("SELECT s.iso_a2, distance(" +
			"s.the_geom, (SELECT g.the_geom FROM geoborders g " +
			"WHERE g.iso_a2 = $1 ) ) AS dist FROM geoborders s " +
			"WHERE s.iso_a2 IN ( " + fullsql + " ) ORDER BY " +
			"dist ASC", strings.ToUpper(*req.Origin))
	} else {
		rows, err = self.conn.Query("SELECT s.iso_a2, distance(" +
			"s.the_geom, (SELECT g.the_geom FROM geoborders g " +
			"WHERE g.iso_a2 = $1 ) ) AS dist FROM geoborders s " +
			"ORDER BY dist ASC", strings.ToUpper(*req.Origin))
	}
	if err != nil {
		return err
	}

	for rows.Next() {
		var detail *GeoProximityDetail = new(GeoProximityDetail)
		detail.Country = new(string)
		detail.Distance = new(float64)

		err = rows.Scan(detail.Country, detail.Distance)
		if err != nil {
			return err
		}

		// Go RPC hates 0 values :S
		if *detail.Distance == 0 {
			*detail.Distance -= 0.001
		}

		if res.Closest == nil {
			res.Closest = detail.Country
		}

		if *req.DetailedResponse {
			res.FullMap = append(res.FullMap, detail)
		}
	}

	return nil
}

// Request which list of destination IPs are closest to a given source
// IP. Takes a radius around which results are returned but we always
// return at least one.
func (self *GeoProximityService) GetProximityByIP(req GeoProximityByIPRequest,
	res *GeoProximityByIPResponse) error {
	var addrlocations = make(map[string][]*GeoProximityByIPDetail)
	var locdata []string = make([]string, 0)
	var maxdistance float64
	var loc *libgeo.Location
	var origin string

	var fullsql string
	var rows *sql.Rows
	var initclosest int
	var err error

	if self.gi == nil {
		return errors.New("GeoIP not loaded")
	}

	if req.Origin == nil {
		return errors.New("No origin specified")
	}

	// First, determine the geodata for all of the given IPs and
	// sort them into a map.
	for _, addr := range req.Candidates {
		loc = self.gi.GetLocationByIP(addr)

		if loc == nil {
			// No country data? Always return…
			res.Closest = append(res.Closest, addr)
		} else {
			var cc string = strings.ToUpper(
				loc.CountryCode)
			var detail *GeoProximityByIPDetail =
				new(GeoProximityByIPDetail)
			var ok bool

			detail.Ip = new(string)
			*detail.Ip = addr

			_, ok = addrlocations[cc]
			if !ok {
				addrlocations[cc] =
					make([]*GeoProximityByIPDetail,
					0)
			}

			addrlocations[cc] =
				append(addrlocations[cc], detail)
		}
	}

	initclosest = len(res.Closest)

	for cc, _ := range addrlocations {
		locdata = append(locdata, "'" + cc + "'")
	}

	fullsql = strings.Join(locdata, ",")

	if isRFC1918(*req.Origin) {
		origin = self.rfc1918_country
	} else {
		// Now let's figure out where the request came from.
		// TODO(tonnerre): Handle RFC1918 IPs.
		loc = self.gi.GetLocationByIP(*req.Origin)

		if loc == nil {
			// Fail open: return all IPs.
			// TODO(tonnerre): Filter out RFC1918 IPs.
			res.Closest = req.Candidates
			return nil
		}
		origin = strings.ToUpper(loc.CountryCode)
	}

	rows, err = self.conn.Query("SELECT s.iso_a2, distance(" +
		"s.the_geom, (SELECT g.the_geom FROM geoborders g " +
		"WHERE g.iso_a2 = $1 ) ) AS dist FROM geoborders s " +
		"WHERE s.iso_a2 IN ( " + fullsql + " ) ORDER BY " +
		"dist ASC", origin)
	if err != nil {
		return err
	}

	for rows.Next() {
		var country string
		var distance float64
		var detail *GeoProximityByIPDetail

		err = rows.Scan(&country, &distance)
		if err != nil {
			return err
		}

		if country != origin {
			distance += 0.01
		}

		// Go RPC hates 0 values :S
		if distance == 0 {
			distance += 0.001
		}

		for _, detail = range addrlocations[country] {
			detail.Distance = new(float64)
			*detail.Distance = distance

			if len(res.Closest) == initclosest {
				maxdistance = distance

				if req.MaxDistance != nil {
					maxdistance +=
						*req.MaxDistance
				}
			}

			if distance <= maxdistance {
				res.Closest = append(res.Closest,
					*detail.Ip)
			}

			if *req.DetailedResponse {
				res.FullMap = append(res.FullMap, detail)
			}
		}
	}

	return nil
}
