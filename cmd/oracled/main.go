// Package main runs `oracled` — the Lux oracle operator daemon.
//
// Standalone operator process (no luxd validator required). Fetches data
// from external sources, signs Observations, and submits them to O-Chain
// (oraclevm) via JSON-RPC.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		listenAddr = flag.String("listen", env("ORACLED_LISTEN", ":7800"), "HTTP listen address")
		oracleRPC  = flag.String("oracle-rpc", env("ORACLED_ORACLE_RPC", "http://127.0.0.1:9650/ext/bc/O/rpc"), "O-Chain (oraclevm) JSON-RPC URL")
		operatorID = flag.String("operator-id", env("ORACLED_OPERATOR_ID", ""), "this operator's NodeID")
		showVer    = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("oracled/1.0.0")
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *operatorID == "" {
		logger.Error("operator-id is required (set ORACLED_OPERATOR_ID or pass --operator-id)")
		os.Exit(2)
	}

	logger.Info("oracled starting", "listen", *listenAddr, "oracle", *oracleRPC, "operator", *operatorID)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Operator loop: fetch external data, sign Observations, submit to O-Chain.
	// Concrete fetchers (Bitcoin RPC, OP_NET indexer, Chainlink, Pyth, etc.)
	// plug in here. The chain VM is the source of truth for canonical
	// aggregated values; this daemon only emits operator observations.
	<-ctx.Done()
	logger.Info("oracled stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
