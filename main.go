package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

type server struct {
	port       int
	dnsService DNSService
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

	s.dnsService.Token = os.Getenv("HETZNERDDNS_TOKEN")
	s.dnsService.HetznerClient.AuthAPIToken = os.Getenv("HETZNERDDNS_HETZNER_TOKEN")

	http.HandleFunc("GET /update", s.update)
	http.ListenAndServe(":"+strconv.Itoa(s.port), nil)
}

func (s *server) update(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var (
		token = query.Get("token")
		ipv4  = query.Get("ipv4")
		ipv6  = query.Get("ipv6")
	)
	err := s.dnsService.UpdateDomain(token, ipv4, ipv6)

	switch {
	case errors.Is(err, ErrBadToken):
		slog.Info("unauthorized", "err", err.Error())
		http.Error(w, "bad token", http.StatusUnauthorized)
		return
	case errors.Is(err, ErrNoToken):
		slog.Info("unauthorized", "err", err.Error())
		http.Error(w, "no token", http.StatusUnauthorized)
		return
	case err != nil:
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
