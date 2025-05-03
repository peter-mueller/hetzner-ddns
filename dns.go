package main

import (
	"errors"
	"log"
	"log/slog"
	"git.p3r.dev/hetzner-ddns/hetzner"
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

	AAAA, foundAAAA := findRecord(records, "AAAA", "@")
	if !foundAAAA {
		return errors.New("@ AAAA record not found")
	}
	A, foundA := findRecord(records, "A", "@")
	if !foundA {
		return errors.New("@ A record not found")
	}
	
	recordsToPatch := make([]hetzner.Record, 0)
	if ipv6 != "" {
		AAAA.TTL = 60
		AAAA.Value = ipv6
		recordsToPatch = append(recordsToPatch, AAAA)
	}
	
	if ipv4 != "" {
		AAAA.TTL = 60
		AAAA.Value = ipv4
		recordsToPatch = append(recordsToPatch, A)
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

func findRecord(records []hetzner.Record, recordType string, name string) (r hetzner.Record, found bool) {
	for _, r := range records {
		if r.Type != recordType {
			continue
		}
		if r.Name != name {
			continue
		}
		return r, true
	}
	return r, false
}
