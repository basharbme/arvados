// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&DispatcherSuite{})

type DispatcherSuite struct {
	ctx        context.Context
	cancel     context.CancelFunc
	cluster    *arvados.Cluster
	stubDriver *test.StubDriver
	disp       *dispatcher
}

func (s *DispatcherSuite) SetUpTest(c *check.C) {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ctx = ctxlog.Context(s.ctx, ctxlog.TestLogger(c))
	dispatchpub, _ := test.LoadTestKey(c, "test/sshkey_dispatch")
	dispatchprivraw, err := ioutil.ReadFile("test/sshkey_dispatch")
	c.Assert(err, check.IsNil)

	_, hostpriv := test.LoadTestKey(c, "test/sshkey_vm")
	s.stubDriver = &test.StubDriver{
		HostKey:                   hostpriv,
		AuthorizedKeys:            []ssh.PublicKey{dispatchpub},
		ErrorRateDestroy:          0.1,
		MinTimeBetweenCreateCalls: time.Millisecond,
	}

	s.cluster = &arvados.Cluster{
		ManagementToken: "test-management-token",
		Containers: arvados.ContainersConfig{
			DispatchPrivateKey: string(dispatchprivraw),
			StaleLockTimeout:   arvados.Duration(5 * time.Millisecond),
			CloudVMs: arvados.CloudVMsConfig{
				Driver:               "test",
				SyncInterval:         arvados.Duration(10 * time.Millisecond),
				TimeoutIdle:          arvados.Duration(150 * time.Millisecond),
				TimeoutBooting:       arvados.Duration(150 * time.Millisecond),
				TimeoutProbe:         arvados.Duration(15 * time.Millisecond),
				TimeoutShutdown:      arvados.Duration(5 * time.Millisecond),
				MaxCloudOpsPerSecond: 500,
				PollInterval:         arvados.Duration(5 * time.Millisecond),
				ProbeInterval:        arvados.Duration(5 * time.Millisecond),
				MaxProbesPerSecond:   1000,
				TimeoutSignal:        arvados.Duration(3 * time.Millisecond),
				TimeoutTERM:          arvados.Duration(20 * time.Millisecond),
				ResourceTags:         map[string]string{"testtag": "test value"},
				TagKeyPrefix:         "test:",
			},
		},
		InstanceTypes: arvados.InstanceTypeMap{
			test.InstanceType(1).Name:  test.InstanceType(1),
			test.InstanceType(2).Name:  test.InstanceType(2),
			test.InstanceType(3).Name:  test.InstanceType(3),
			test.InstanceType(4).Name:  test.InstanceType(4),
			test.InstanceType(6).Name:  test.InstanceType(6),
			test.InstanceType(8).Name:  test.InstanceType(8),
			test.InstanceType(16).Name: test.InstanceType(16),
		},
	}
	arvadostest.SetServiceURL(&s.cluster.Services.DispatchCloud, "http://localhost:/")
	arvadostest.SetServiceURL(&s.cluster.Services.Controller, "https://"+os.Getenv("ARVADOS_API_HOST")+"/")

	arvClient, err := arvados.NewClientFromConfig(s.cluster)
	c.Check(err, check.IsNil)

	s.disp = &dispatcher{
		Cluster:   s.cluster,
		Context:   s.ctx,
		ArvClient: arvClient,
		AuthToken: arvadostest.AdminToken,
	}
	// Test cases can modify s.cluster before calling
	// initialize(), and then modify private state before calling
	// go run().
}

func (s *DispatcherSuite) TearDownTest(c *check.C) {
	s.cancel()
	s.disp.Close()
}

// DispatchToStubDriver checks that the dispatcher wires everything
// together effectively. It uses a real scheduler and worker pool with
// a fake queue and cloud driver. The fake cloud driver injects
// artificial errors in order to exercise a variety of code paths.
func (s *DispatcherSuite) TestDispatchToStubDriver(c *check.C) {
	Drivers["test"] = s.stubDriver
	s.disp.setupOnce.Do(s.disp.initialize)
	queue := &test.Queue{
		ChooseType: func(ctr *arvados.Container) (arvados.InstanceType, error) {
			return ChooseInstanceType(s.cluster, ctr)
		},
	}
	for i := 0; i < 200; i++ {
		queue.Containers = append(queue.Containers, arvados.Container{
			UUID:     test.ContainerUUID(i + 1),
			State:    arvados.ContainerStateQueued,
			Priority: int64(i%20 + 1),
			RuntimeConstraints: arvados.RuntimeConstraints{
				RAM:   int64(i%3+1) << 30,
				VCPUs: i%8 + 1,
			},
		})
	}
	s.disp.queue = queue

	var mtx sync.Mutex
	done := make(chan struct{})
	waiting := map[string]struct{}{}
	for _, ctr := range queue.Containers {
		waiting[ctr.UUID] = struct{}{}
	}
	finishContainer := func(ctr arvados.Container) {
		mtx.Lock()
		defer mtx.Unlock()
		if _, ok := waiting[ctr.UUID]; !ok {
			c.Errorf("container completed twice: %s", ctr.UUID)
			return
		}
		delete(waiting, ctr.UUID)
		if len(waiting) == 0 {
			close(done)
		}
	}
	executeContainer := func(ctr arvados.Container) int {
		finishContainer(ctr)
		return int(rand.Uint32() & 0x3)
	}
	n := 0
	s.stubDriver.Queue = queue
	s.stubDriver.SetupVM = func(stubvm *test.StubVM) {
		n++
		stubvm.Boot = time.Now().Add(time.Duration(rand.Int63n(int64(5 * time.Millisecond))))
		stubvm.CrunchRunDetachDelay = time.Duration(rand.Int63n(int64(10 * time.Millisecond)))
		stubvm.ExecuteContainer = executeContainer
		stubvm.CrashRunningContainer = finishContainer
		switch n % 7 {
		case 0:
			stubvm.Broken = time.Now().Add(time.Duration(rand.Int63n(90)) * time.Millisecond)
		case 1:
			stubvm.CrunchRunMissing = true
		case 2:
			stubvm.ReportBroken = time.Now().Add(time.Duration(rand.Int63n(200)) * time.Millisecond)
		default:
			stubvm.CrunchRunCrashRate = 0.1
		}
	}

	start := time.Now()
	go s.disp.run()
	err := s.disp.CheckHealth()
	c.Check(err, check.IsNil)

	select {
	case <-done:
		c.Logf("containers finished (%s), waiting for instances to shutdown and queue to clear", time.Since(start))
	case <-time.After(10 * time.Second):
		c.Fatalf("timed out; still waiting for %d containers: %q", len(waiting), waiting)
	}

	deadline := time.Now().Add(5 * time.Second)
	for range time.NewTicker(10 * time.Millisecond).C {
		insts, err := s.stubDriver.InstanceSets()[0].Instances(nil)
		c.Check(err, check.IsNil)
		queue.Update()
		ents, _ := queue.Entries()
		if len(ents) == 0 && len(insts) == 0 {
			break
		}
		if time.Now().After(deadline) {
			c.Fatalf("timed out with %d containers (%v), %d instances (%+v)", len(ents), ents, len(insts), insts)
		}
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+s.cluster.ManagementToken)
	resp := httptest.NewRecorder()
	s.disp.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*driver_operations{error="0",operation="Create"} [^0].*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*driver_operations{error="0",operation="List"} [^0].*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*driver_operations{error="0",operation="Destroy"} [^0].*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*driver_operations{error="1",operation="Create"} [^0].*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*driver_operations{error="1",operation="List"} 0\n.*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*instances_disappeared{state="shutdown"} [^0].*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*instances_disappeared{state="unknown"} 0\n.*`)
}

func (s *DispatcherSuite) TestAPIPermissions(c *check.C) {
	s.cluster.ManagementToken = "abcdefgh"
	Drivers["test"] = s.stubDriver
	s.disp.setupOnce.Do(s.disp.initialize)
	s.disp.queue = &test.Queue{}
	go s.disp.run()

	for _, token := range []string{"abc", ""} {
		req := httptest.NewRequest("GET", "/arvados/v1/dispatch/instances", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp := httptest.NewRecorder()
		s.disp.ServeHTTP(resp, req)
		if token == "" {
			c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusForbidden)
		}
	}
}

func (s *DispatcherSuite) TestAPIDisabled(c *check.C) {
	s.cluster.ManagementToken = ""
	Drivers["test"] = s.stubDriver
	s.disp.setupOnce.Do(s.disp.initialize)
	s.disp.queue = &test.Queue{}
	go s.disp.run()

	for _, token := range []string{"abc", ""} {
		req := httptest.NewRequest("GET", "/arvados/v1/dispatch/instances", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp := httptest.NewRecorder()
		s.disp.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusForbidden)
	}
}

func (s *DispatcherSuite) TestInstancesAPI(c *check.C) {
	s.cluster.ManagementToken = "abcdefgh"
	s.cluster.Containers.CloudVMs.TimeoutBooting = arvados.Duration(time.Second)
	Drivers["test"] = s.stubDriver
	s.disp.setupOnce.Do(s.disp.initialize)
	s.disp.queue = &test.Queue{}
	go s.disp.run()

	type instance struct {
		Instance             string
		WorkerState          string `json:"worker_state"`
		Price                float64
		LastContainerUUID    string `json:"last_container_uuid"`
		ArvadosInstanceType  string `json:"arvados_instance_type"`
		ProviderInstanceType string `json:"provider_instance_type"`
	}
	type instancesResponse struct {
		Items []instance
	}
	getInstances := func() instancesResponse {
		req := httptest.NewRequest("GET", "/arvados/v1/dispatch/instances", nil)
		req.Header.Set("Authorization", "Bearer abcdefgh")
		resp := httptest.NewRecorder()
		s.disp.ServeHTTP(resp, req)
		var sr instancesResponse
		c.Check(resp.Code, check.Equals, http.StatusOK)
		err := json.Unmarshal(resp.Body.Bytes(), &sr)
		c.Check(err, check.IsNil)
		return sr
	}

	sr := getInstances()
	c.Check(len(sr.Items), check.Equals, 0)

	ch := s.disp.pool.Subscribe()
	defer s.disp.pool.Unsubscribe(ch)
	ok := s.disp.pool.Create(test.InstanceType(1))
	c.Check(ok, check.Equals, true)
	<-ch

	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		sr = getInstances()
		if len(sr.Items) > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	c.Assert(len(sr.Items), check.Equals, 1)
	c.Check(sr.Items[0].Instance, check.Matches, "stub.*")
	c.Check(sr.Items[0].WorkerState, check.Equals, "booting")
	c.Check(sr.Items[0].Price, check.Equals, 0.123)
	c.Check(sr.Items[0].LastContainerUUID, check.Equals, "")
	c.Check(sr.Items[0].ProviderInstanceType, check.Equals, test.InstanceType(1).ProviderType)
	c.Check(sr.Items[0].ArvadosInstanceType, check.Equals, test.InstanceType(1).Name)
}
