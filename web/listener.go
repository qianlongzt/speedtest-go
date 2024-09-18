package web

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/go-chi/chi/v5"
	"github.com/librespeed/speedtest/config"
	"github.com/pires/go-proxyproto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func startListener(ctx context.Context, conf *config.Config, r *chi.Mux) error {
	// See if systemd socket activation has been used when starting our process
	listeners, err := activation.Listeners()
	if err != nil {
		return fmt.Errorf("error whilst checking for systemd socket activation %s", err)
	}
	if conf.ProxyProtocolPort != "0" {
		return errors.New("proxyprotocol_port is deprecated, use enable_proxyprotocol")
	}

	var listener net.Listener

	switch len(listeners) {
	case 0:
		addr := net.JoinHostPort(conf.BindAddress, conf.Port)
		slog.Info("Starting backend server on", "address", addr)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}
		listener = l
	case 1:
		slog.Info("Starting backend server on inherited file descriptor via systemd socket activation")
		if conf.BindAddress != "" || conf.Port != "" {
			slog.Error("Both an address/port has been specified in the config AND externally configured socket activation has been detected")
			slog.Error("Please deconfigure socket activation (e.g. in systemd unit files), or set both 'bind_address' and 'listen_port' to ''")
			return errors.New("configure either 'bind_address' and 'listen_port' or systemd socket activation")
		}
		listener = listeners[0]
	default:
		return fmt.Errorf("asked to listen on %d sockets via systemd activation.  Sorry we currently only support listening on 1 socket", len(listeners))
	}

	if conf.EnableProxyprotocol {
		slog.Info("use proxy protocol listener")
		pl := &proxyproto.Listener{
			Listener: listener,
		}
		if allow := conf.ProxyprotocolAllowedIPs; len(allow) != 0 {
			slog.Info("allowed proxy protocol from", slog.Any("ips", allow))
			pl.Policy = proxyproto.MustStrictWhiteListPolicy(allow)
		}
		listener = pl
		defer pl.Close()
	}

	srv := &http.Server{
		Handler: r,
	}
	var listenFn func() error
	// TLS and HTTP/2.
	if conf.EnableTLS {
		slog.Info("Use TLS connection")

		if !(conf.EnableHTTP2) {
			// If TLSNextProto is not nil, HTTP/2 support is not enabled automatically.
			srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
		}
		listenFn = func() error {
			return srv.ServeTLS(listener, conf.TLSCertFile, conf.TLSKeyFile)
		}
	} else {
		if conf.EnableHTTP2 {
			slog.Info("Use HTTP2 connection.")
			h2s := &http2.Server{}
			srv.Handler = h2c.NewHandler(r, h2s)
			err = http2.ConfigureServer(srv, h2s)
			if err != nil {
				return fmt.Errorf("http2.ConfigureServer: %s", err)
			}
		}
		listenFn = func() error {
			return srv.Serve(listener)
		}
	}

	go func() {
		err := listenFn()
		slog.Info("http server closed")
		if err != nil && err != http.ErrServerClosed {
			panic(fmt.Errorf("failed to listen: %w", err))
		}
	}()
	<-ctx.Done()
	slog.Info("http server shutting down")
	err = srv.Shutdown(ctx)
	slog.Info("http server shutdown finished")

	return err
}
