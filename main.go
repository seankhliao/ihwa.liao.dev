package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.seankhliao.com/svcrunner/v2/basehttp"
	"go.seankhliao.com/webstyle"
)

//go:embed index.md
var rawIndex []byte

func main() {
	fset := flag.NewFlagSet("ihwa.liao.dev", flag.ExitOnError)
	conf := &basehttp.Config{}
	conf.SetFlags(fset)
	fset.Parse(os.Args[1:])

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	svr := basehttp.New(ctx, conf)

	err := run(ctx, svr)
	if err != nil {
		svr.O.L.LogAttrs(ctx, slog.LevelError, "server exit", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context, svr *basehttp.Server) error {
	t0 := time.Now()
	index, err := webstyle.NewRenderer(webstyle.TemplateCompact).RenderBytes(rawIndex, webstyle.Data{})
	if err != nil {
		return fmt.Errorf("render index: %w", err)
	}
	svr.Mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	svr.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := svr.O.T.Start(r.Context(), "handle request")
		defer span.End()

		svr.O.L.LogAttrs(ctx, slog.LevelDebug, "handle request", slog.Group("http.request",
			slog.String("proto", r.Proto),
			slog.String("method", r.Method),
			slog.String("host", r.Host),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
			slog.Any("headers", r.Header),
		))

		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		http.ServeContent(w, r, "index.html", t0, bytes.NewReader(index))
	})

	return svr.Run(ctx)
}
