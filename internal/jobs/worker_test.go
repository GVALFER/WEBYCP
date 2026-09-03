package jobs_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

type prober struct{}

func (prober) Probe(context.Context, string) error { return nil }

func TestWorkerRunsNodeProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	node, err := store.EnsureLocal(ctx, "test", "/tmp/test-agent.sock")
	if err != nil {
		t.Fatal(err)
	}

	nodeService := nodes.NewService(store, prober{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := jobs.NewWorker(store, store, logger)
	worker.Handle(jobs.KindNodeProbe, func(ctx context.Context, job jobs.Job) error {
		return nodeService.Probe(ctx, job.NodeID)
	})
	service := jobs.NewService(store, worker.Notify)
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()

	job, err := service.QueueProbe(ctx, node.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, _, err := service.Job(ctx, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == "succeeded" {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("job did not complete, status = %s", current.Status)
		case <-ticker.C:
		}
	}

	currentNode, err := nodeService.Node(ctx, node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if currentNode.Status != "online" || currentNode.LastSeenAt == nil {
		t.Fatalf("node was not marked online: %+v", currentNode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}
