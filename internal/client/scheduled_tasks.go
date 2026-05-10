package client

import (
	"context"
	"fmt"
	"net/http"
)

// ScheduledTask represents a scheduled task on an application or service.
type ScheduledTask struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	Frequency string `json:"frequency"`
	Enabled   bool   `json:"enabled"`
}

// CreateScheduledTaskInput holds the fields required to create a scheduled task.
type CreateScheduledTaskInput struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	Frequency string `json:"frequency"`
}

// UpdateScheduledTaskInput holds the fields that can be updated on a scheduled task.
type UpdateScheduledTaskInput struct {
	Name      *string `json:"name,omitempty"`
	Command   *string `json:"command,omitempty"`
	Frequency *string `json:"frequency,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

type createScheduledTaskResponse struct {
	UUID string `json:"uuid"`
}

// ListScheduledTasks returns all scheduled tasks for a parent resource.
// parentType must be "applications" or "services".
func (c *Client) ListScheduledTasks(ctx context.Context, parentType, parentUUID string) ([]ScheduledTask, error) {
	var tasks []ScheduledTask
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/%s/%s/scheduled-tasks", parentType, parentUUID), nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// CreateScheduledTask creates a new scheduled task on a parent resource.
// parentType must be "applications" or "services". Returns the UUID of the created task.
func (c *Client) CreateScheduledTask(ctx context.Context, parentType, parentUUID string, input CreateScheduledTaskInput) (string, error) {
	var resp createScheduledTaskResponse
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/%s/%s/scheduled-tasks", parentType, parentUUID), input, &resp, http.StatusCreated); err != nil {
		return "", err
	}
	return resp.UUID, nil
}

// UpdateScheduledTask updates an existing scheduled task.
// parentType must be "applications" or "services".
func (c *Client) UpdateScheduledTask(ctx context.Context, parentType, parentUUID, taskUUID string, input UpdateScheduledTaskInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/%s/%s/scheduled-tasks/%s", parentType, parentUUID, taskUUID), input, nil)
}

// DeleteScheduledTask deletes a scheduled task.
// parentType must be "applications" or "services".
func (c *Client) DeleteScheduledTask(ctx context.Context, parentType, parentUUID, taskUUID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/%s/%s/scheduled-tasks/%s", parentType, parentUUID, taskUUID), nil, nil)
}

// TaskExecution represents a single execution of a scheduled task.
type TaskExecution struct {
	UUID      string `json:"uuid"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"created_at"`
}

// ListTaskExecutions returns all executions for a scheduled task.
// parentType must be "applications" or "services".
func (c *Client) ListTaskExecutions(ctx context.Context, parentType, parentUUID, taskUUID string) ([]TaskExecution, error) {
	var execs []TaskExecution
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/%s/%s/scheduled-tasks/%s/executions", parentType, parentUUID, taskUUID), nil, &execs); err != nil {
		return nil, err
	}
	return execs, nil
}
