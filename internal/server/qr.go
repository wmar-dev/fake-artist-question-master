package server

import (
	"net"
	"net/http"
	"strings"

	"rsc.io/qr"
)

// joinAddress returns the link that joins a game, and whether phones can
// reach it. Forwarded headers are ignored, so a client cannot change the
// address in a QR code; deployments behind a proxy set PublicURL instead.
func (s *Server) joinAddress(r *http.Request, code string) (string, bool) {
	if s.cfg.PublicURL != "" {
		return s.cfg.PublicURL + "/g/" + code, true
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/g/" + code, !isLocalOnly(r.Host)
}

func isLocalOnly(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

func (s *Server) qrImage(w http.ResponseWriter, r *http.Request) {
	g, err := s.store.Get(r.PathValue("code"))
	if err != nil {
		writeError(w, err)
		return
	}
	address, _ := s.joinAddress(r, g.Code())
	code, err := qr.Encode(address, qr.M)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(code.PNG())
}
