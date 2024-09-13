// Code generated by MockGen. DO NOT EDIT.
// Source: ../manager.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	goqite "github.com/maragudk/goqite"
	queue "github.com/retr0h/osapi/internal/queue"
)

// MockManager is a mock of Manager interface.
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager.
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance.
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// Count mocks base method.
func (m *MockManager) Count(ctx context.Context) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Count", ctx)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Count indicates an expected call of Count.
func (mr *MockManagerMockRecorder) Count(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Count", reflect.TypeOf((*MockManager)(nil).Count), ctx)
}

// Delete mocks base method.
func (m *MockManager) Delete(ctx context.Context, msgID goqite.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, msgID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockManagerMockRecorder) Delete(ctx, msgID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockManager)(nil).Delete), ctx, msgID)
}

// DeleteByID mocks base method.
func (m *MockManager) DeleteByID(ctx context.Context, messageID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteByID", ctx, messageID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteByID indicates an expected call of DeleteByID.
func (mr *MockManagerMockRecorder) DeleteByID(ctx, messageID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteByID", reflect.TypeOf((*MockManager)(nil).DeleteByID), ctx, messageID)
}

// Get mocks base method.
func (m *MockManager) Get(ctx context.Context) (*goqite.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx)
	ret0, _ := ret[0].(*goqite.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockManagerMockRecorder) Get(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockManager)(nil).Get), ctx)
}

// GetAll mocks base method.
func (m *MockManager) GetAll(ctx context.Context, limit, offset int) ([]queue.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll", ctx, limit, offset)
	ret0, _ := ret[0].([]queue.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll.
func (mr *MockManagerMockRecorder) GetAll(ctx, limit, offset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockManager)(nil).GetAll), ctx, limit, offset)
}

// GetByID mocks base method.
func (m *MockManager) GetByID(ctx context.Context, messageID string) (*queue.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, messageID)
	ret0, _ := ret[0].(*queue.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockManagerMockRecorder) GetByID(ctx, messageID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockManager)(nil).GetByID), ctx, messageID)
}

// Put mocks base method.
func (m *MockManager) Put(ctx context.Context, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Put", ctx, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Put indicates an expected call of Put.
func (mr *MockManagerMockRecorder) Put(ctx, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockManager)(nil).Put), ctx, data)
}

// SetupQueue mocks base method.
func (m *MockManager) SetupQueue() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetupQueue")
	ret0, _ := ret[0].(error)
	return ret0
}

// SetupQueue indicates an expected call of SetupQueue.
func (mr *MockManagerMockRecorder) SetupQueue() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupQueue", reflect.TypeOf((*MockManager)(nil).SetupQueue))
}

// SetupSchema mocks base method.
func (m *MockManager) SetupSchema(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetupSchema", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetupSchema indicates an expected call of SetupSchema.
func (mr *MockManagerMockRecorder) SetupSchema(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupSchema", reflect.TypeOf((*MockManager)(nil).SetupSchema), ctx)
}
