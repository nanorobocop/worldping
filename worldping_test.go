package main

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nanorobocop/worldping/mocks"
	"github.com/nanorobocop/worldping/task"
)

func TestInitizlize(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := mocks.NewMockDB(mockCtrl)
	mockEnv := &envStruct{dbConn: mockDB}

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
