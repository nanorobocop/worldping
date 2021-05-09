package main

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/apsdehal/go-logger"
	"github.com/golang/mock/gomock"
	"github.com/nanorobocop/worldping/mocks"
	"github.com/nanorobocop/worldping/pkg/types"
)

// mockgen -destination=mocks/mock_db.go -package=mocks github.com/nanorobocop/worldping/db DB

func TestInitizlize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := mocks.NewMockDB(mockCtrl)
	mockEnv := &envStruct{dbConn: mockDB}
	mockEnv.log, _ = logger.New("worldping", 0, os.Stdout)

	mockDB.EXPECT().Open().Return(nil).Times(1)
	mockDB.EXPECT().Ping().Return(nil).Times(1)
	mockDB.EXPECT().CreateTable().Return(nil).Times(1)

	mockEnv.initialize()

}

func TestGetTasks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := mocks.NewMockDB(mockCtrl)
	mockEnv := &envStruct{
		dbConn: mockDB,
	}
	mockEnv.log, _ = logger.New("worldping", 0, os.Stdout)

	tests := []struct {
		ip      uint32
		err     error
		expTask types.Task
	}{
		{
			ip:      111,
			err:     nil,
			expTask: types.Task{IP: 111},
		},
		{
			ip:      0,
			err:     errors.New("Some error"),
			expTask: types.Task{IP: 0},
		},
		{
			ip:      4278190080,
			err:     nil,
			expTask: types.Task{IP: 4278190080},
		},
	}

	for i, test := range tests {

		t.Logf("[TEST] %d: %+v", i, test)

		mockDB.EXPECT().GetOldestIP().Return(test.ip, test.err).Times(1)

		var cancel context.CancelFunc
		mockEnv.ctx = context.Background()
		mockEnv.ctx, cancel = context.WithCancel(mockEnv.ctx)

		tasksCh := make(chan types.Task, 1)

		go mockEnv.getTasks(tasksCh)

		actual := <-tasksCh
		cancel()

		if actual != test.expTask {
			t.Errorf("[TEST FAILED] Incorrect task generated")
			t.Errorf("Expected: %+v", test.expTask)
			t.Errorf("Actual  : %+v", actual)
		}
	}
}

func TestGetLoad(t *testing.T) {
	var cancel context.CancelFunc
	mockEnv := &envStruct{}
	mockEnv.ctx = context.Background()
	mockEnv.ctx, cancel = context.WithCancel(mockEnv.ctx)

	go time.AfterFunc(1000*time.Millisecond, cancel)

	loadCh := make(chan float64, 1)
	mockEnv.getLoad(loadCh)

	load := <-loadCh

	if load < 0 || load > 10 {
		t.Errorf("Test FAILed, load is out of range [0; 10]: %f", load)
	}
}

type mockPinger struct {
	mockErr error
}

func (p mockPinger) Ping(*net.IPAddr, time.Duration) (time.Duration, error) {
	return time.Second, p.mockErr
}

func (p mockPinger) Close() {}

func TestPingf(t *testing.T) {
	guard := make(chan struct{}, 1)
	resultCh := make(chan types.Task, 1)

	steps := []struct {
		ip      uint32
		ping    bool
		fakeErr error
	}{
		{
			ip:      uint32(0),
			ping:    false,
			fakeErr: errors.New("some error"),
		},
		{
			ip:      uint32(1),
			ping:    true,
			fakeErr: nil,
		},
	}

	for i, step := range steps {
		guard <- struct{}{}
		mockEnv := &envStruct{pinger: mockPinger{mockErr: step.fakeErr}}
		mockEnv.log, _ = logger.New("worldping", 0, os.Stdout)

		t.Logf("Step %d: %+v", i, step)
		mockEnv.pingf(step.ip, resultCh, guard)
		actualResult := <-resultCh
		if actualResult.Ping != step.ping {
			t.Errorf("TEST FAILED: expected %v, actual %v", step.ping, actualResult)
		}
	}

}

func TestSchedule(t *testing.T) {
	taskCh := make(chan types.Task, 1)
	resultCh := make(chan types.Task)
	loadCh := make(chan float64, 1)

	mockEnv := &envStruct{pinger: mockPinger{mockErr: nil}}
	mockEnv.log, _ = logger.New("worldping", 0, os.Stdout)
	mockEnv.ctx = context.Background()
	var cancel context.CancelFunc
	mockEnv.ctx, cancel = context.WithCancel(mockEnv.ctx)

	go mockEnv.schedule(taskCh, resultCh, loadCh)

	loadCh <- 1
	loadCh <- 1001
	taskCh <- types.Task{IP: uint32(0)}

	cancel()
}

func TestSendStat(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDB := mocks.NewMockDB(mockCtrl)
	// mockDB.EXPECT().Save(types.Task{}).Return(nil).Times(1)

	resultCh := make(chan types.Task)

	mockEnv := &envStruct{dbConn: mockDB}
	mockEnv.log, _ = logger.New("worldping", 0, os.Stdout)
	mockEnv.ctx = context.Background()
	var cancel context.CancelFunc
	mockEnv.ctx, cancel = context.WithCancel(mockEnv.ctx)
	mockEnv.wg.Add(1)

	go mockEnv.sendStat(resultCh)

	cancel()
}
