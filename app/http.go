package app

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/cors"
)

type Headers [][]string

func (h Headers) Len() int           { return len(h) }
func (h Headers) Less(i, j int) bool { return h[i][0] < h[j][0] }
func (h Headers) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func SprintHeader(hdr http.Header) func() (s string) {
	return func() (s string) {
		var sections Headers
		s += "\n"
		for i := range hdr {
			sect := []string{i}
			sect = append(sect, hdr[i]...)
			sections = append(sections, sect)
		}
		sort.Sort(sections)
		for i := range sections {
			for j := range sections[i] {
				s += "\"" + sections[i][j] + "\" "
			}
			s += "\n"
		}
		return
	}
}

// ServeHTTP implements http.Handler interface.
//
// This is the main starting function of the relay. This launches
// HandleWebsocket which runs the message handling main loop.
func (rl *Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.T.C(SprintHeader(r.Header))
	select {
	case <-rl.Ctx.Done():
		log.W.Ln("shutting down")
		return
	default:
	}
	if r.Header.Get("Upgrade") == "websocket" {
		rl.HandleWebsocket(getServiceBaseURL(r))(w, r)
	} else if r.Header.Get("Accept") == "application/nostr+json" {
		cors.AllowAll().Handler(http.HandlerFunc(rl.HandleNIP11)).
			ServeHTTP(w, r)
	} else {
		rl.serveMux.ServeHTTP(w, r)
	}
}

func getServiceBaseURL(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if host == "localhost" {
			proto = "http"
		} else if strings.Index(host, ":") != -1 {
			// has a port number
			proto = "http"
		} else if _, err := strconv.Atoi(strings.ReplaceAll(host, ".",
			"")); chk.E(err) {
			// it's a naked IP
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + host
}
