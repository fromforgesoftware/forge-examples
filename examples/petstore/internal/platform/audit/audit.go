// Package audit wires a go-kit audit.Sink for petstore services. Producers
// (catalog, adoptions) emit a neutral kit audit.Event; the concrete sink is
// chosen by configuration:
//
//	AUDIT_SINK=stdout  (default) — log events as JSON to stdout, for dev.
//	AUDIT_SINK=talos              — forward to Talos over gRPC (TALOS_GRPC_ADDR),
//	                                via talos's go-kit audit.Sink adapter.
//
// Emitting through the kit Sink port means the producers never import a
// concrete backend.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	kitaudit "github.com/fromforgesoftware/go-kit/audit"
	"github.com/fromforgesoftware/go-kit/monitoring/logger"
	talosaudit "github.com/fromforgesoftware/talos/audit"
	talos "github.com/fromforgesoftware/talos/pkg/client"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Closer is an optional clean-up hook returned by the talos sink (to close the
// gRPC connection). The stdout sink returns a no-op closer.
type Closer func() error

// NewSink builds the kit audit.Sink selected by AUDIT_SINK, returning a Closer
// to release any backing resources (e.g. the Talos gRPC connection).
//
//	stdout (default): logs each event as JSON; nothing to dial, so a real
//	                  deployment can run without Talos.
//	talos:            dials TALOS_GRPC_ADDR and wraps the talos/audit adapter.
func NewSink() (kitaudit.Sink, Closer, error) {
	switch os.Getenv("AUDIT_SINK") {
	case "talos":
		addr := os.Getenv("TALOS_GRPC_ADDR")
		if addr == "" {
			return nil, nil, fmt.Errorf("AUDIT_SINK=talos requires TALOS_GRPC_ADDR")
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("dial talos: %w", err)
		}
		// talos/audit.NewSink adapts the kit audit.Sink port to the Talos client.
		sink := talosaudit.NewSink(talos.New(conn))
		return sink, conn.Close, nil
	default: // "stdout" or unset
		return &stdoutSink{log: logger.New()}, func() error { return nil }, nil
	}
}

// stdoutSink writes each audit event as a JSON line. It stands in for a durable
// sink during local development.
type stdoutSink struct {
	log logger.Logger
}

func (s *stdoutSink) Emit(ctx context.Context, e kitaudit.Event) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	s.log.InfoContext(ctx, "audit", "event", json.RawMessage(line))
	return nil
}
