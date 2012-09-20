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

package geocolo

import (
	"database/sql"
	"fmt"

	_ "github.com/bmizerany/pq"
)

type GeoProximityService struct {
	conn *sql.DB
}

func NewGeoProximityService(
	config *GeoProximityServiceConfig) (*GeoProximityService, error) {
	var c *sql.DB
	var err error
	var dsn string

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

	return &GeoProximityService{
		conn: c,
	}, nil
}

func (self *GeoProximityService) GetProximity(req GeoProximityRequest,
	res *GeoProximityResponse) error {
	var rows *sql.Rows
	var err error

	if len(req.Candidates) > 0 {
		rows, err = self.conn.Query("SELECT s.iso_a2, distance("+
			"s.the_geom, (SELECT g.the_geom FROM geoborders g "+
			"WHERE g.iso_a2 = ?)) AS dist FROM geoborders s "+
			"WHERE s.iso_a2 IN (?) ORDER BY dist ASC;",
			*req.Origin, req.Candidates)
	} else {
		rows, err = self.conn.Query("SELECT s.iso_a2, distance("+
			"s.the_geom, (SELECT g.the_geom FROM geoborders g "+
			"WHERE g.iso_a2 = ?)) AS dist FROM geoborders s "+
			"ORDER BY dist ASC;", *req.Origin, req.Candidates)
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

		if res.Closest == nil {
			res.Closest = detail.Country
		}

		if *req.DetailedResponse {
			res.FullMap = append(res.FullMap, detail)
		}
	}

	return nil
}
