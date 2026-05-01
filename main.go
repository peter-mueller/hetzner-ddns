package main

import (
	"log"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"strconv"
)

type server struct {
	port       int
	dnsService *DNSService
}

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
	}))
	slog.SetDefault(l)

	var s server
	port, ok := lookupEnvInt("HETZNERDDNS_PORT")
	if ok {
		s.port = port
	} else {
		s.port = 8080
	}

	token := os.Getenv("HETZNERDDNS_TOKEN")
	hetznerApiToken := os.Getenv("HETZNERDDNS_HETZNER_TOKEN")
	zone := os.Getenv("HETZNERDDNS_ZONE")
	dnsService, err := NewDNSService(token, hetznerApiToken, zone)
	if err != nil {
		slog.Error("failed to init DNSService", "err", err)
		os.Exit(1)
	}
	s.dnsService = dnsService

	http.HandleFunc("GET /update", s.update)
	http.ListenAndServe(":"+strconv.Itoa(s.port), nil)
}

func (s *server) update(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var (
		token        = query.Get("token")
		ipv4         = query.Get("ipv4")
		ipv6prefix   = query.Get("ipv6prefix")
		ipv6instance = query.Get("ipv6instance")
	)

	prefix, err := netip.ParsePrefix(ipv6prefix)
	if err != nil {
		slog.Error("bad ipv6prefix", "ipv6prefix", ipv6prefix)
		http.Error(w, "bad ipv6prefix", 400)
		return
	}
	addr16 := prefix.Addr().As16()
	instance, err := netip.ParseAddr(ipv6instance)
	if err != nil {
		slog.Error("bad ipv6instance", "ipv6instance", ipv6instance)
		http.Error(w, "bad ipv6instance", 400)
		return
	}
	instance16 := instance.As16()
	copy(addr16[8:], instance16[8:])
	ipv6 := netip.AddrFrom16(addr16).String()

	err = s.dnsService.UpdateDomain(token, ipv4, ipv6)

	if err != nil {
		slog.Error("interal error", "err", err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func lookupEnvInt(name string) (int, bool) {
	text, ok := os.LookupEnv(name)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(text)
	if err != nil {
		log.Fatalf("cannot parse env %s as int: %s", name, err)
	}
	return i, true
}
