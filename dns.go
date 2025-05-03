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

	AAAA, foundAAAA := findRecord(records, "AAAA", "www")
	if !foundAAAA {
		return errors.New("www AAAA record not found")
	}
	A, foundA := findRecord(records, "A", "www")
	if !foundA {
		return errors.New("www A record not found")
	}

	AAAA.TTL = 60
	AAAA.Value = ipv6
	A.TTL = 60
	A.Value = ipv4
	
	slog.Info("updating dns records", "ipv4", ipv4, "ipv6", ipv6)

	for _, r := range []hetzner.Record{A, AAAA} {
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
