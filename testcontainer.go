package gko

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"google.golang.org/appengine/aetest"
)

// TestContainer is test container.
type TestContainer struct {
	t     *testing.T
	inst  aetest.Instance
	tests []testing.InternalTest
}

// NewTestContiner return test container.
func NewTestContiner(t *testing.T, appID string, stronglyConsistent bool) (*TestContainer, error) {
	if appID == "" {
		appID = "tetapp"
	}

	inst, err := aetest.NewInstance(&aetest.Options{AppID: appID, StronglyConsistentDatastore: stronglyConsistent})
	if err != nil {
		return nil, err
	}
	return &TestContainer{
		t:    t,
		inst: inst,
	}, nil
}

// AddTest add test to container.
func (tc *TestContainer) AddTest(f func(t *testing.T)) {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	tc.tests = append(tc.tests, testing.InternalTest{Name: name, F: f})
}

// RunTest run test in container.
func (tc *TestContainer) RunTest() {
	defer func() {
		if err := tc.inst.Close(); err != nil {
			tc.t.Fatal(err)
		}
	}()

	const (
		success = "SUCCESS"
		faild   = "FAILED"
	)
	type testResult struct {
		name, ret string
	}
	trs := make([]*testResult, len(tc.tests))

	tc.t.Logf("App Engine Test start to run at %s\n", time.Now())
	for i, test := range tc.tests {
		name := test.Name
		tr := &testResult{name, success}
		ok := tc.t.Run(name, test.F)
		if !ok {
			tr.ret = faild
		}
		trs[i] = tr
	}

	tc.t.Log("Test Result")
	for _, tr := range trs {
		tc.t.Logf("Test %s: %s\n", tr.name, tr.ret)
	}

	tc.t.Logf("App Engine Test end at %s\n", time.Now())
}
