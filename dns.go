package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"fmt"
)

type DNSService struct {
	HetznerClient *hcloud.Client
	Zone          string
	Token         string
}

func NewDNSService(token string, hetznerCloudToken string, zone string) (*DNSService, error) {
	if token == "" {
		return nil, ErrNoToken
	}
	if hetznerCloudToken == "" {
		return nil, ErrNoHetznerCloudToken
	}
	if zone == "" {
		return nil, ErrNoZone
	}

	s := new(DNSService)
	s.Zone = zone
	s.Token = token
	s.HetznerClient = hcloud.NewClient(
		hcloud.WithToken(hetznerCloudToken),
	)
	return s, nil
}

var ErrNoToken = errors.New("no token")
var ErrNoZone = errors.New("no zone")

var ErrNoHetznerCloudToken = errors.New("no hetzner api token")

var ErrBadToken = errors.New("bad token")

var DefaultTTL = new(60)

const (
	TypeAAAA = hcloud.ZoneRRSetTypeAAAA
	TypeA    = hcloud.ZoneRRSetTypeA
)

func singleRecord(value string) []hcloud.ZoneRRSetRecord {
	return []hcloud.ZoneRRSetRecord{
		{
			Value:   value,
			Comment: "set by hetzner-ddns DynDNS service",
		},
	}
}

func recordsValueString(records []hcloud.ZoneRRSetRecord) string {
	var b strings.Builder

	b.WriteString("(")
	for i, r := range records {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(r.Value)
	}
	b.WriteString(")")
	return b.String()
}

func (service *DNSService) UpdateDomain(token string, ipv4 string, ipv6 string) error {
	if token == "" {
		return ErrNoToken
	}
	if token != service.Token {
		return ErrBadToken
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone := &hcloud.Zone{
		Name: service.Zone,
	}
	records, _, err := service.HetznerClient.Zone.ListRRSets(ctx, zone, hcloud.ZoneRRSetListOpts{})
	if err != nil {
		return err
	}

	AAAA, rerr := mustRecord(records, TypeAAAA, "@")
	err = errors.Join(err, rerr)
	AAAAWildcard, rerr := mustRecord(records, TypeAAAA, "*")
	err = errors.Join(err, rerr)
	A, rerr := mustRecord(records, TypeA, "@")
	err = errors.Join(err, rerr)
	AWildcard, rerr := mustRecord(records, TypeA, "*")
	err = errors.Join(err, rerr)

	if err != nil {
		slog.Error("failed to find records", "err", err)
	}

	recordsToPatch := make([]*hcloud.ZoneRRSet, 0)
	if ipv6 != "" {
		AAAA.TTL = DefaultTTL
		AAAA.Records = singleRecord(ipv6)
		recordsToPatch = append(recordsToPatch, AAAA)

		AAAAWildcard.TTL = DefaultTTL
		AAAAWildcard.Records = singleRecord(ipv6)

		recordsToPatch = append(recordsToPatch, AAAAWildcard)
	}

	if ipv4 != "" {
		A.TTL = DefaultTTL
		A.Records = singleRecord(ipv4)
		recordsToPatch = append(recordsToPatch, A)

		AWildcard.TTL = DefaultTTL
		AWildcard.Records = singleRecord(ipv4)
		recordsToPatch = append(recordsToPatch, AWildcard)
	}

	for _, r := range recordsToPatch {
		slog.Info("set dns record", "type", r.Type, "value", recordsValueString(r.Records))

		_, res, err := service.HetznerClient.Zone.SetRRSetRecords(ctx, r, hcloud.ZoneRRSetSetRecordsOpts{
			Records: r.Records,
		})
		slog.Info("set dns record response", "status", res.Status)
		if err != nil {
			return err
		}
	}

	return nil
}

func mustRecord(records []*hcloud.ZoneRRSet, recordType hcloud.ZoneRRSetType, name string) (r *hcloud.ZoneRRSet, err error) {

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
