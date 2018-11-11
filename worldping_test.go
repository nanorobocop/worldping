package main

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nanorobocop/worldping/mocks"
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
