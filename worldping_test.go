package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/apsdehal/go-logger"
	"github.com/golang/mock/gomock"
	"github.com/nanorobocop/worldping/mocks"
	"github.com/nanorobocop/worldping/task"
)

func TestInitizlize(t *testing.T) {
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
		expTask task.Task
	}{
		{
			ip:      111,
			err:     nil,
			expTask: task.Task{IP: 112},
		},
		{
			ip:      0,
			err:     errors.New("Some error"),
			expTask: task.Task{IP: 1},
		},
	}

	for i, test := range tests {

		t.Logf("[TEST] %d: %+v", i, test)

		mockDB.EXPECT().GetMaxIP().Return(test.ip, test.err).Times(1)

		var cancel context.CancelFunc
		mockEnv.ctx = context.Background()
		mockEnv.ctx, cancel = context.WithCancel(mockEnv.ctx)

		tasksCh := make(chan task.Task, 1)

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

func TestIPToStr(t *testing.T) {
	tests := []struct {
		ipInt uint32
		ipStr string
	}{
		{
			ipInt: 0,
			ipStr: "0.0.0.0",
		},
		{
			ipInt: 1,
			ipStr: "0.0.0.1",
		},
		{
			ipInt: 1234567890, // 0x499602D2
			ipStr: "73.150.2.210",
		},
		{
			ipInt: 4294967295,
			ipStr: "255.255.255.255",
		},
	}

	for i, test := range tests {
		t.Logf("Test %d, %+v", i, test)
		actual := ipToStr(test.ipInt)
		if actual != test.ipStr {
			t.Errorf("FAILED: %s", actual)
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
