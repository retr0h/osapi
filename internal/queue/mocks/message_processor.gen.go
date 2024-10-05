// Code generated by MockGen. DO NOT EDIT.
// Source: ../message_processor.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	goqite "github.com/maragudk/goqite"
)

// MockMessageProcessor is a mock of MessageProcessor interface.
type MockMessageProcessor struct {
	ctrl     *gomock.Controller
	recorder *MockMessageProcessorMockRecorder
}

// MockMessageProcessorMockRecorder is the mock recorder for MockMessageProcessor.
type MockMessageProcessorMockRecorder struct {
	mock *MockMessageProcessor
}

// NewMockMessageProcessor creates a new mock instance.
func NewMockMessageProcessor(ctrl *gomock.Controller) *MockMessageProcessor {
	mock := &MockMessageProcessor{ctrl: ctrl}
	mock.recorder = &MockMessageProcessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageProcessor) EXPECT() *MockMessageProcessorMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockMessageProcessor) Delete(ctx context.Context, id goqite.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockMessageProcessorMockRecorder) Delete(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockMessageProcessor)(nil).Delete), ctx, id)
}

// Receive mocks base method.
func (m *MockMessageProcessor) Receive(ctx context.Context) (*goqite.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Receive", ctx)
	ret0, _ := ret[0].(*goqite.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Receive indicates an expected call of Receive.
func (mr *MockMessageProcessorMockRecorder) Receive(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Receive", reflect.TypeOf((*MockMessageProcessor)(nil).Receive), ctx)
}

// Send mocks base method.
func (m *MockMessageProcessor) Send(ctx context.Context, message goqite.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", ctx, message)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockMessageProcessorMockRecorder) Send(ctx, message interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockMessageProcessor)(nil).Send), ctx, message)
}