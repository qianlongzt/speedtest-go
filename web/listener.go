package web

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/go-chi/chi/v5"
	"github.com/librespeed/speedtest/config"
	"github.com/pires/go-proxyproto"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func startListener(conf *config.Config, r *chi.Mux) error {
	// See if systemd socket activation has been used when starting our process
	listeners, err := activation.Listeners()
	if err != nil {
		panic(fmt.Errorf("error whilst checking for systemd socket activation %s", err))
	}
	if conf.ProxyProtocolPort != "0" {
		panic("proxyprotocol_port is deprecated, use enable_proxyprotocol")
	}

	var listener net.Listener

	switch len(listeners) {
	case 0:
		addr := net.JoinHostPort(conf.BindAddress, conf.Port)
		log.Infof("Starting backend server on %s", addr)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			panic(fmt.Errorf("failed to listen on %s: %w", addr, err))
		}
		listener = l
	case 1:
		log.Info("Starting backend server on inherited file descriptor via systemd socket activation")
		if conf.BindAddress != "" || conf.Port != "" {
			log.Errorf("Both an address/port (%s:%s) has been specificed in the config AND externally configured socket activation has been detected", conf.BindAddress, conf.Port)
			log.Fatal(`Please deconfigure socket activation (e.g. in systemd unit files), or set both 'bind_address' and 'listen_port' to ''`)
		}
		listener = listeners[0]
	default:
		log.Fatalf("Asked to listen on %d sockets via systemd activation.  Sorry we currently only support listening on 1 socket.", len(listeners))
	}

	if conf.EnableProxyprotocol {
		log.Infof("use proxy protocol listener")
		pl := &proxyproto.Listener{
			Listener: listener,
		}
		if allow := conf.ProxyprotocolAllowedIPs; len(allow) != 0 {
			log.Infof("allowed proxy protocol ips: %v", allow)
			pl.Policy = proxyproto.MustStrictWhiteListPolicy(allow)
		}
		listener = pl
		defer pl.Close()
	}

	srv := &http.Server{
		Handler: r,
	}
	// TLS and HTTP/2.
	if conf.EnableTLS {
		log.Info("Use TLS connection.")

		if !(conf.EnableHTTP2) {
			// If TLSNextProto is not nil, HTTP/2 support is not enabled automatically.
			srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
		}
		err = srv.ServeTLS(listener, conf.TLSCertFile, conf.TLSKeyFile)
	} else {
		if conf.EnableHTTP2 {
			log.Info("Use HTTP2 connection.")
			h2s := &http2.Server{}
			srv.Handler = h2c.NewHandler(r, h2s)
			err = http2.ConfigureServer(srv, h2s)
			if err != nil {
				log.Fatalf("http2.ConfigureServer: %s", err)
			}
		}
		err = srv.Serve(listener)
	}

	return err
}
