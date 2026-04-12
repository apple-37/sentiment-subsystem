package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongo-exporter/internal/config"
)

// Exporter connects to MongoDB, reads documents, and writes them as JSONL.
type Exporter struct {
	cfg    *config.Config
	client *mongo.Client
	logger *slog.Logger
}

// New creates an Exporter and establishes the MongoDB connection.
func New(cfg *config.Config, logger *slog.Logger) (*Exporter, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.MongoDB.TimeoutSeconds)*time.Second,
	)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoDB.URI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	// Verify the connection is alive.
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	logger.Info("connected to mongodb",
		"uri", cfg.MongoDB.URI,
		"database", cfg.MongoDB.Database,
		"collection", cfg.MongoDB.Collection,
	)

	return &Exporter{cfg: cfg, client: client, logger: logger}, nil
}

// Close disconnects from MongoDB.
func (e *Exporter) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.client.Disconnect(ctx); err != nil {
		e.logger.Warn("error disconnecting from mongodb", "error", err)
	}
}

// Export reads all documents from the configured collection and writes JSONL
// to the configured output path atomically (write to temp, rename on success).
func (e *Exporter) Export(ctx context.Context) error {
	start := time.Now()
	e.logger.Info("export started",
		"database", e.cfg.MongoDB.Database,
		"collection", e.cfg.MongoDB.Collection,
		"output", e.cfg.Output.FilePath,
	)

	opCtx, cancel := context.WithTimeout(ctx, time.Duration(e.cfg.MongoDB.TimeoutSeconds)*time.Second)
	defer cancel()

	count, err := e.writeJSONL(opCtx)
	if err != nil {
		return err
	}

	e.logger.Info("export finished",
		"documents", count,
		"elapsed", time.Since(start).String(),
		"output", e.cfg.Output.FilePath,
	)
	return nil
}

// writeJSONL streams documents from MongoDB into a temp file, then renames it.
func (e *Exporter) writeJSONL(ctx context.Context) (int64, error) {
	// Ensure the output directory exists.
	dir := filepath.Dir(e.cfg.Output.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, fmt.Errorf("create output directory: %w", err)
	}

	tmpPath := e.cfg.Output.FilePath + e.cfg.Output.TempSuffix
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open temp file: %w", err)
	}

	count, writeErr := e.streamDocuments(ctx, f)

	// Always close before rename, even on error.
	if closeErr := f.Close(); closeErr != nil {
		e.logger.Warn("failed to close temp file", "error", closeErr)
	}

	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return 0, writeErr
	}

	// Atomic rename: other subsystems reading the file never see a partial write.
	if err := os.Rename(tmpPath, e.cfg.Output.FilePath); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("rename temp to output: %w", err)
	}

	return count, nil
}

// streamDocuments cursors through all documents and encodes each as a JSON line.
func (e *Exporter) streamDocuments(ctx context.Context, w io.Writer) (int64, error) {
	coll := e.client.Database(e.cfg.MongoDB.Database).Collection(e.cfg.MongoDB.Collection)

	// bson.D{} = no filter → fetch all documents.
	cursor, err := coll.Find(ctx, bson.D{}, options.Find().SetBatchSize(500))
	if err != nil {
		return 0, fmt.Errorf("mongodb find: %w", err)
	}
	defer cursor.Close(ctx)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	var count int64
	for cursor.Next(ctx) {
		// Decode into a generic ordered map so field order is preserved.
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return count, fmt.Errorf("decode document #%d: %w", count+1, err)
		}

		if err := enc.Encode(doc); err != nil {
			return count, fmt.Errorf("encode document #%d: %w", count+1, err)
		}
		count++
	}

	if err := cursor.Err(); err != nil {
		return count, fmt.Errorf("cursor error: %w", err)
	}
	return count, nil
}

// RunScheduled blocks and runs Export on every configured interval until ctx is cancelled.
func (e *Exporter) RunScheduled(ctx context.Context) {
	// Run immediately on startup, then on each tick.
	if err := e.Export(ctx); err != nil {
		e.logger.Error("scheduled export failed", "error", err)
	}

	ticker := time.NewTicker(e.cfg.Scheduler.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			if err := e.Export(ctx); err != nil {
				// Log but don't exit — retry on the next tick.
				e.logger.Error("scheduled export failed", "error", err)
			}
		}
	}
}
