// Code generated by MockGen. DO NOT EDIT.
// Source: ../types.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	load "github.com/retr0h/osapi/internal/provider/system/load"
)

// MockProvider is a mock of Provider interface.
type MockProvider struct {
	ctrl     *gomock.Controller
	recorder *MockProviderMockRecorder
}

// MockProviderMockRecorder is the mock recorder for MockProvider.
type MockProviderMockRecorder struct {
	mock *MockProvider
}

// NewMockProvider creates a new mock instance.
func NewMockProvider(ctrl *gomock.Controller) *MockProvider {
	mock := &MockProvider{ctrl: ctrl}
	mock.recorder = &MockProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProvider) EXPECT() *MockProviderMockRecorder {
	return m.recorder
}

// GetAverageStats mocks base method.
func (m *MockProvider) GetAverageStats() (*load.AverageStats, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAverageStats")
	ret0, _ := ret[0].(*load.AverageStats)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAverageStats indicates an expected call of GetAverageStats.
func (mr *MockProviderMockRecorder) GetAverageStats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAverageStats", reflect.TypeOf((*MockProvider)(nil).GetAverageStats))
}
