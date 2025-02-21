package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/librespeed/speedtest/config"
	"github.com/pires/go-proxyproto"
)

func startListener(ctx context.Context, conf *config.Config, r http.Handler) error {
	// See if systemd socket activation has been used when starting our process
	listeners, err := activation.Listeners()
	if err != nil {
		return fmt.Errorf("error whilst checking for systemd socket activation %s", err)
	}
	if p := conf.ProxyProtocolPort; !(p == "0" || p == "") {
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
	srv.Protocols = new(http.Protocols)
	srv.Protocols.SetHTTP1(true)

	var listenFn func() error
	// TLS and HTTP/2.
	if conf.EnableTLS {
		slog.Info("Use TLS connection")
		if conf.EnableHTTP2 {
			slog.Info("Use HTTP2 connection.")
			srv.Protocols.SetHTTP2(conf.EnableHTTP2)
		}
		listenFn = func() error {
			return srv.ServeTLS(listener, conf.TLSCertFile, conf.TLSKeyFile)
		}
	} else {
		if conf.EnableHTTP2 {
			slog.Info("Use HTTP2 connection.")
			srv.Protocols.SetUnencryptedHTTP2(conf.EnableHTTP2)
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
