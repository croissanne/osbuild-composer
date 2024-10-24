// +build integration

package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osbuild/osbuild-composer/jobqueue/dbjobqueue"
	"github.com/stretchr/testify/require"
)

const url = "postgres://postgres:foobar@localhost:5432/osbuildcomposer"

func TestDBJobQueueMaintenance(t *testing.T) {
	dbMaintenance, err := newDB(url)
	require.NoError(t, err)
	defer dbMaintenance.Close()
	q, err := dbjobqueue.New(url)
	require.NoError(t, err)
	defer q.Close()

	_, err = dbMaintenance.Conn.Exec(context.Background(), "DELETE FROM jobs")
	require.NoError(t, err)

	t.Run("testJobsUptoByType", func(t *testing.T) {
		testJobsUptoByType(t, dbMaintenance, q)
	})
	t.Run("testDeleteJobResult", func(t *testing.T) {
		testDeleteJobResult(t, dbMaintenance, q)
	})

}

func setFinishedAt(t *testing.T, d db, id uuid.UUID, finished time.Time) {
	started := finished.Add(-time.Second)
	queued := started.Add(-time.Second)
	_, err := d.Conn.Exec(context.Background(), "UPDATE jobs SET queued_at = $1, started_at = $2, finished_at = $3, result = '{\"result\": \"success\" }' WHERE id = $4", queued, started, finished, id)
	require.NoError(t, err)
}

func testJobsUptoByType(t *testing.T, d db, q *dbjobqueue.DBJobQueue) {
	date80 := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	date85 := time.Date(1985, time.January, 1, 0, 0, 0, 0, time.UTC)
	date90 := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)

	id80, err := q.Enqueue("octopus", nil, nil, "")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id80)
	_, _, _, _, _, err = q.Dequeue(context.Background(), []string{"octopus"}, []string{""})
	require.NoError(t, err)
	err = q.FinishJob(id80, nil)
	require.NoError(t, err)
	setFinishedAt(t, d, id80, date80)

	id85, err := q.Enqueue("octopus", nil, nil, "")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id85)
	_, _, _, _, _, err = q.Dequeue(context.Background(), []string{"octopus"}, []string{""})
	require.NoError(t, err)
	err = q.FinishJob(id85, nil)
	require.NoError(t, err)
	setFinishedAt(t, d, id85, date85)

	ids, err := d.JobsUptoByType([]string{"octopus"}, date85)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{id80}, ids["octopus"])

	ids, err = d.JobsUptoByType([]string{"octopus"}, date90)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{id80, id85}, ids["octopus"])
}

func testDeleteJobResult(t *testing.T, d db, q *dbjobqueue.DBJobQueue) {
	id, err := q.Enqueue("octopus", nil, nil, "")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)
	_, _, _, _, _, err = q.Dequeue(context.Background(), []string{"octopus"}, []string{""})
	require.NoError(t, err)

	type Result struct {
		Result string `json:"result"`
	}
	result := Result{
		"deleteme",
	}

	res, err := json.Marshal(result)
	require.NoError(t, err)
	err = q.FinishJob(id, res)
	require.NoError(t, err)

	_, _, r, _, _, _, _, _, err := q.JobStatus(id)
	require.NoError(t, err)

	var r1 Result
	require.NoError(t, json.Unmarshal(r, &r1))
	require.Equal(t, result, r1)

	rows, err := d.DeleteJobResult([]uuid.UUID{id})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)

	_, _, r, _, _, _, _, _, err = q.JobStatus(id)
	require.NoError(t, err)
	require.Nil(t, r)
}
