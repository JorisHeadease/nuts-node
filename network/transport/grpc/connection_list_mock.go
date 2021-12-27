// Code generated by MockGen. DO NOT EDIT.
// Source: network/transport/grpc/connection_list.go

// Package grpc is a generated GoMock package.
package grpc

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockConnectionList is a mock of ConnectionList interface.
type MockConnectionList struct {
	ctrl     *gomock.Controller
	recorder *MockConnectionListMockRecorder
}

// MockConnectionListMockRecorder is the mock recorder for MockConnectionList.
type MockConnectionListMockRecorder struct {
	mock *MockConnectionList
}

// NewMockConnectionList creates a new mock instance.
func NewMockConnectionList(ctrl *gomock.Controller) *MockConnectionList {
	mock := &MockConnectionList{ctrl: ctrl}
	mock.recorder = &MockConnectionListMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnectionList) EXPECT() *MockConnectionListMockRecorder {
	return m.recorder
}

// All mocks base method.
func (m *MockConnectionList) All() []Connection {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "All")
	ret0, _ := ret[0].([]Connection)
	return ret0
}

// All indicates an expected call of All.
func (mr *MockConnectionListMockRecorder) All() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "All", reflect.TypeOf((*MockConnectionList)(nil).All))
}

// Get mocks base method.
func (m *MockConnectionList) Get(query ...Predicate) Connection {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range query {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Get", varargs...)
	ret0, _ := ret[0].(Connection)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockConnectionListMockRecorder) Get(query ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockConnectionList)(nil).Get), query...)
}
