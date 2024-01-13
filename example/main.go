package main

import (
	"context"
	"fmt"

	"github.com/balchua/durabletask-pg-backend/backend/postgres"
	"github.com/microsoft/durabletask-go/backend"
	"github.com/microsoft/durabletask-go/task"
)

func main() {
	r := task.NewTaskRegistry()
	r.AddOrchestrator(ActivitySequenceOrchestrator)
	r.AddActivity(SayHelloActivity)

	ctx := context.Background()
	client, worker := Init(ctx, r)
	defer worker.Shutdown(ctx)

	id, err := client.ScheduleNewOrchestration(ctx, ActivitySequenceOrchestrator)
	if err != nil {
		panic(err)
	}
	metadata, err := client.WaitForOrchestrationCompletion(ctx, id)
	if err != nil {
		panic(err)
	}
	fmt.Printf("orchestration completed: %v\n", metadata)
}

// Init creates and initializes an in-memory client and worker pair with default configuration.
func Init(ctx context.Context, r *task.TaskRegistry) (backend.TaskHubClient, backend.TaskHubWorker) {
	logger := backend.DefaultLogger()

	be := postgres.NewPostgresBackend(postgres.WithHost("localhost"),
		postgres.WithPassword("admin123"),
		postgres.WithDBName("testdb"),
		postgres.WithUser("postgresadmin"),
		postgres.WithLogger(logger))

	executor := task.NewTaskExecutor(r)
	orchestrationWorker := backend.NewOrchestrationWorker(be, executor, logger)
	activityWorker := backend.NewActivityTaskWorker(be, executor, logger)
	taskHubWorker := backend.NewTaskHubWorker(be, orchestrationWorker, activityWorker, logger)
	if err := taskHubWorker.Start(ctx); err != nil {
		panic(err)
	}
	taskHubClient := backend.NewTaskHubClient(be)
	return taskHubClient, taskHubWorker
}

// ActivitySequenceOrchestrator makes three activity calls in sequence and results the results
// as an array.
func ActivitySequenceOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var helloTokyo string
	if err := ctx.CallActivity(SayHelloActivity, task.WithActivityInput("Tokyo")).Await(&helloTokyo); err != nil {
		return nil, err
	}
	var helloLondon string
	if err := ctx.CallActivity(SayHelloActivity, task.WithActivityInput("London")).Await(&helloLondon); err != nil {
		return nil, err
	}
	var helloSeattle string
	if err := ctx.CallActivity(SayHelloActivity, task.WithActivityInput("Seattle")).Await(&helloSeattle); err != nil {
		return nil, err
	}
	return []string{helloTokyo, helloLondon, helloSeattle}, nil
}

// SayHelloActivity can be called by an orchestrator function and will return a friendly greeting.
func SayHelloActivity(ctx task.ActivityContext) (any, error) {
	var input string
	if err := ctx.GetInput(&input); err != nil {
		return "", err
	}
	return fmt.Sprintf("Hello, %s!", input), nil
}
