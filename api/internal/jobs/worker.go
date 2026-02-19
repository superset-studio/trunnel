package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/superset-studio/kapstan/api/internal/services"
)

// ValidateConnectionArgs are the arguments for the validate_connection job.
type ValidateConnectionArgs struct {
	ConnectionID uuid.UUID `json:"connection_id"`
}

func (ValidateConnectionArgs) Kind() string { return "validate_connection" }

// ValidateConnectionWorker validates a single connection's credentials.
type ValidateConnectionWorker struct {
	river.WorkerDefaults[ValidateConnectionArgs]
	connService *services.ConnectionService
}

func (w *ValidateConnectionWorker) Work(ctx context.Context, job *river.Job[ValidateConnectionArgs]) error {
	_, err := w.connService.ValidateConnectionByID(ctx, job.Args.ConnectionID)
	return err
}

// PeriodicValidateAllArgs triggers periodic validation of all connections.
type PeriodicValidateAllArgs struct{}

func (PeriodicValidateAllArgs) Kind() string { return "periodic_validate_all_connections" }

// PeriodicValidateAllWorker enqueues individual validation jobs for every connection.
type PeriodicValidateAllWorker struct {
	river.WorkerDefaults[PeriodicValidateAllArgs]
	connService *services.ConnectionService
	client      *river.Client[pgx.Tx]
}

func (w *PeriodicValidateAllWorker) Work(ctx context.Context, _ *river.Job[PeriodicValidateAllArgs]) error {
	conns, err := w.connService.ListAllConnections(ctx)
	if err != nil {
		return err
	}

	for _, conn := range conns {
		_, err := w.client.Insert(ctx, ValidateConnectionArgs{ConnectionID: conn.ID}, nil)
		if err != nil {
			slog.Error("failed to enqueue validate_connection", slog.String("connection_id", conn.ID.String()), slog.String("error", err.Error()))
		}
	}

	return nil
}

// NewJobClient creates a River client with connection validation workers registered.
func NewJobClient(pool *pgxpool.Pool, connService *services.ConnectionService) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()

	river.AddWorker(workers, &ValidateConnectionWorker{connService: connService})

	// We'll set the periodic worker's client after creation.
	periodicWorker := &PeriodicValidateAllWorker{connService: connService}
	river.AddWorker(workers, periodicWorker)

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return PeriodicValidateAllArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
		},
	})
	if err != nil {
		return nil, err
	}

	periodicWorker.client = client

	return client, nil
}
