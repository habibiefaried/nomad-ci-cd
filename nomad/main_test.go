package nomad

import (
	"testing"
)

func TestSubmitJob(t *testing.T) {
	t.Log(jobGeneration())
	err := SubmitJob("http://localhost:4646")
	t.Fatalf(err)
}
