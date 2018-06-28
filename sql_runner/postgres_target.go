//
// Copyright (c) 2015-2017 Snowplow Analytics Ltd. All rights reserved.
//
// This program is licensed to you under the Apache License Version 2.0,
// and you may not use this file except in compliance with the Apache License Version 2.0.
// You may obtain a copy of the Apache License Version 2.0 at http://www.apache.org/licenses/LICENSE-2.0.
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the Apache License Version 2.0 is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the Apache License Version 2.0 for the specific language governing permissions and limitations there under.
//
package main

import (
	"crypto/tls"
	"github.com/go-pg/pg"
	"net"
	"time"
	"log"
	"github.com/go-pg/pg/orm"
)

// For Redshift queries
const (
	dialTimeout = 10 * time.Second
	readTimeout = 8 * time.Hour // TODO: make this user configurable
)

type PostgresTarget struct {
	Target
	Client *pg.DB
}

func NewPostgresTarget(target Target) *PostgresTarget {
	var tlsConfig *tls.Config
	if target.Ssl == true {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	db := pg.Connect(&pg.Options{
		Addr:        target.Host + ":" + target.Port,
		User:        target.Username,
		Password:    target.Password,
		Database:    target.Database,
		TLSConfig:   tlsConfig,
		DialTimeout: dialTimeout,
		ReadTimeout: readTimeout,
		Dialer: func(network, addr string) (net.Conn, error) {
			cn, err := net.DialTimeout(network, addr, dialTimeout)
			if err != nil {
				return nil, err
			}
			return cn, cn.(*net.TCPConn).SetKeepAlive(true)
		},
	})

	return &PostgresTarget{target, db}
}

func (pt PostgresTarget) GetTarget() Target {
	return pt.Target
}

// Run a query against the target
func (pt PostgresTarget) RunQuery(query ReadyQuery, dryRun bool, dropOutput bool) QueryStatus {
	var err error = nil
	var res orm.Result
	if dryRun {
		return QueryStatus{query, query.Path, 0, nil}
	}

	affected := 0
	if dropOutput {
		var results Results
		res, err = pt.Client.Query(&results, query.Script)
		if err == nil {
			affected = res.RowsAffected()
		}
		if len(results) > 0 {
			log.Printf("QUERY OUTPUT: %s\n", results)
		} else {
			log.Println("QUERY OUTPUT: No output returned.")
		}
	} else {
		res, err = pt.Client.Exec(query.Script)
		if err == nil {
			affected = res.RowsAffected()
		}
	}

	return QueryStatus{query, query.Path, affected, err}
}
