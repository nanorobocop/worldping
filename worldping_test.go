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
		expTask task.Task
	}{
		{
			ip:      111,
			err:     nil,
			expTask: task.Task{IP: 111},
		},
		{
			ip:      0,
			err:     errors.New("Some error"),
			expTask: task.Task{IP: 0},
		},
		{
			ip:      4278190080,
			err:     nil,
			expTask: task.Task{IP: 4278190080},
		},
	}

	for i, test := range tests {

		t.Logf("[TEST] %d: %+v", i, test)

		mockDB.EXPECT().GetOldestIP().Return(test.ip, test.err).Times(1)

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
