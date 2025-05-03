package main

import (
	"errors"
	"log"
	"log/slog"
	"git.p3r.dev/hetzner-ddns/hetzner"
	
	"fmt"
	
)

type DNSService struct {
	HetznerClient hetzner.Client
	Token         string
}

var ErrNoToken = errors.New("no token")
var ErrBadToken = errors.New("bad token")

func (service *DNSService) UpdateDomain(token string, ipv4 string, ipv6 string) error {
	if token == "" {
		return ErrNoToken
	}
	if token != service.Token {
		return ErrBadToken
	}

	records, err := service.HetznerClient.GetAllRecords()
	if err != nil {
		log.Fatal(err)
	}


	AAAA, err := mustRecord(records, "AAAA", "@")
	err = errors.Join(err, err)
	AAAAWildcard, err := mustRecord(records, "AAAA", "*")
	err = errors.Join(err, err)
	A, err := mustRecord(records, "A", "@")
	err = errors.Join(err, err)
	AWildcard, err := mustRecord(records, "A", "*")
	err = errors.Join(err, err)
	
	recordsToPatch := make([]hetzner.Record, 0)
	if ipv6 != "" {
		AAAA.TTL = 60
		AAAA.Value = ipv6
		recordsToPatch = append(recordsToPatch, AAAA)
		
		AAAAWildcard.TTL = 60
		AAAAWildcard.Value = ipv6
		recordsToPatch = append(recordsToPatch, AAAAWildcard) 
	}
	
	if ipv4 != "" {
		AAAA.TTL = 60
		AAAA.Value = ipv4
		recordsToPatch = append(recordsToPatch, A)
		
		AWildcard.TTL = 60
		AWildcard.Value = ipv4
		recordsToPatch = append(recordsToPatch, AWildcard) 
	}


	for _, r := range recordsToPatch {		
	    slog.Info("updating dns record", "type", r.Type, "value", r.Value)
		err = service.HetznerClient.UpdateRecord(r)
		if err != nil {
			return err
		}
	}

	return nil
}

func mustRecord(records []hetzner.Record, recordType string, name string) (r hetzner.Record, err error) {
	for _, r := range records {
		if r.Type != recordType {
			continue
		}
		if r.Name != name {
			continue
		}
		return r, nil
	}
	return r, fmt.Errorf("record with type %s and name %s not found", recordType, name)
}
