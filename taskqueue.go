package gko

import (
	"net/url"

	"golang.org/x/net/context"
	"google.golang.org/appengine/taskqueue"
)

// Path is path of task.
type Path string

// POSTTask return new post task.
func (p Path) POSTTask(params url.Values) *taskqueue.Task {
	return taskqueue.NewPOSTTask(string(p), params)
}

// Queue is name of task queue.
type Queue string

// Add add task to task queue.
func (q Queue) Add(ctx context.Context, task *taskqueue.Task) (*taskqueue.Task, error) {
	return taskqueue.Add(ctx, task, string(q))
}

// AddMulti add multi task to task queue.
func (q Queue) AddMulti(ctx context.Context, tasks []*taskqueue.Task) ([]*taskqueue.Task, error) {
	return taskqueue.AddMulti(ctx, tasks, string(q))
}
