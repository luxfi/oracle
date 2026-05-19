// Package main runs `oracled` — the Lux oracle operator daemon.
//
// Standalone operator process (no luxd validator required). Fetches data
// from external sources, signs Observations with ML-DSA-65 by default
// (FIPS 204, NIST Level 3), and submits them to O-Chain (oraclevm) via
// JSON-RPC. Cross-operator gossip (Observation / OracleRecord) rides the
// intra-Lux ZAP plane on a separate port.
//
// External fetchers (Bitcoin RPC, Ethereum RPC, Pyth, Chainlink, market
// data APIs) keep using their native transport and authentication.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/luxfi/oracle/pkg/zaptransport"
)

const defaultZAPPort = 7810

func main() {
	var (
		listenAddr = flag.String("listen", env("ORACLED_LISTEN", ":7800"), "HTTP listen address")
		zapPort    = flag.Int("zap-port", envInt("ORACLED_ZAP_PORT", defaultZAPPort), "intra-Lux ZAP operator-plane port (0 = disabled)")
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

	logger.Info("oracled starting",
		"listen", *listenAddr,
		"zapPort", *zapPort,
		"oracle", *oracleRPC,
		"operator", *operatorID,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start the intra-Lux ZAP listener if enabled. External RPC fetchers
	// don't go through this plane — they keep their native transport.
	var zn *zaptransport.Node
	if *zapPort > 0 {
		var err error
		zn, err = zaptransport.New(zaptransport.Config{
			NodeID: *operatorID,
			Port:   *zapPort,
			Logger: logger,
		})
		if err != nil {
			logger.Error("zap init", "err", err)
			os.Exit(1)
		}
		zn.HandleObservation(func(_ context.Context, from string, body []byte) error {
			logger.Debug("zap: observation received", "from", from, "bytes", len(body))
			return nil
		})
		zn.HandleRecord(func(_ context.Context, from string, body []byte) error {
			logger.Debug("zap: record received", "from", from, "bytes", len(body))
			return nil
		})
		if err := zn.Start(); err != nil {
			logger.Error("zap start", "err", err)
			os.Exit(1)
		}
		logger.Info("oracle zap listener started", "port", *zapPort)
		defer zn.Stop()
	}

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

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
