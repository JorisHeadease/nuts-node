// Code generated by MockGen. DO NOT EDIT.
// Source: network/transport/v1/logic/senders.go

// Package logic is a generated GoMock package.
package logic

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	hash "github.com/nuts-foundation/nuts-node/crypto/hash"
	dag "github.com/nuts-foundation/nuts-node/network/dag"
	transport "github.com/nuts-foundation/nuts-node/network/transport"
	protobuf "github.com/nuts-foundation/nuts-node/network/transport/v1/protobuf"
)

// MockmessageSender is a mock of messageSender interface.
type MockmessageSender struct {
	ctrl     *gomock.Controller
	recorder *MockmessageSenderMockRecorder
}

// MockmessageSenderMockRecorder is the mock recorder for MockmessageSender.
type MockmessageSenderMockRecorder struct {
	mock *MockmessageSender
}

// NewMockmessageSender creates a new mock instance.
func NewMockmessageSender(ctrl *gomock.Controller) *MockmessageSender {
	mock := &MockmessageSender{ctrl: ctrl}
	mock.recorder = &MockmessageSenderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockmessageSender) EXPECT() *MockmessageSenderMockRecorder {
	return m.recorder
}

// broadcastAdvertHashes mocks base method.
func (m *MockmessageSender) broadcastAdvertHashes(blocks []dagBlock) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "broadcastAdvertHashes", blocks)
}

// broadcastAdvertHashes indicates an expected call of broadcastAdvertHashes.
func (mr *MockmessageSenderMockRecorder) broadcastAdvertHashes(blocks interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "broadcastAdvertHashes", reflect.TypeOf((*MockmessageSender)(nil).broadcastAdvertHashes), blocks)
}

// broadcastDiagnostics mocks base method.
func (m *MockmessageSender) broadcastDiagnostics(diagnostics transport.Diagnostics) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "broadcastDiagnostics", diagnostics)
}

// broadcastDiagnostics indicates an expected call of broadcastDiagnostics.
func (mr *MockmessageSenderMockRecorder) broadcastDiagnostics(diagnostics interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "broadcastDiagnostics", reflect.TypeOf((*MockmessageSender)(nil).broadcastDiagnostics), diagnostics)
}

// broadcastTransactionPayloadQuery mocks base method.
func (m *MockmessageSender) broadcastTransactionPayloadQuery(payloadHash hash.SHA256Hash) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "broadcastTransactionPayloadQuery", payloadHash)
}

// broadcastTransactionPayloadQuery indicates an expected call of broadcastTransactionPayloadQuery.
func (mr *MockmessageSenderMockRecorder) broadcastTransactionPayloadQuery(payloadHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "broadcastTransactionPayloadQuery", reflect.TypeOf((*MockmessageSender)(nil).broadcastTransactionPayloadQuery), payloadHash)
}

// sendTransactionList mocks base method.
func (m *MockmessageSender) sendTransactionList(peer transport.PeerID, transactions []dag.Transaction, date time.Time) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "sendTransactionList", peer, transactions, date)
}

// sendTransactionList indicates an expected call of sendTransactionList.
func (mr *MockmessageSenderMockRecorder) sendTransactionList(peer, transactions, date interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sendTransactionList", reflect.TypeOf((*MockmessageSender)(nil).sendTransactionList), peer, transactions, date)
}

// sendTransactionListQuery mocks base method.
func (m *MockmessageSender) sendTransactionListQuery(peer transport.PeerID, blockDate time.Time) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "sendTransactionListQuery", peer, blockDate)
}

// sendTransactionListQuery indicates an expected call of sendTransactionListQuery.
func (mr *MockmessageSenderMockRecorder) sendTransactionListQuery(peer, blockDate interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sendTransactionListQuery", reflect.TypeOf((*MockmessageSender)(nil).sendTransactionListQuery), peer, blockDate)
}

// sendTransactionPayload mocks base method.
func (m *MockmessageSender) sendTransactionPayload(peer transport.PeerID, payloadHash hash.SHA256Hash, data []byte) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "sendTransactionPayload", peer, payloadHash, data)
}

// sendTransactionPayload indicates an expected call of sendTransactionPayload.
func (mr *MockmessageSenderMockRecorder) sendTransactionPayload(peer, payloadHash, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sendTransactionPayload", reflect.TypeOf((*MockmessageSender)(nil).sendTransactionPayload), peer, payloadHash, data)
}

// sendTransactionPayloadQuery mocks base method.
func (m *MockmessageSender) sendTransactionPayloadQuery(peer transport.PeerID, payloadHash hash.SHA256Hash) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "sendTransactionPayloadQuery", peer, payloadHash)
}

// sendTransactionPayloadQuery indicates an expected call of sendTransactionPayloadQuery.
func (mr *MockmessageSenderMockRecorder) sendTransactionPayloadQuery(peer, payloadHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sendTransactionPayloadQuery", reflect.TypeOf((*MockmessageSender)(nil).sendTransactionPayloadQuery), peer, payloadHash)
}

// MockMessageGateway is a mock of MessageGateway interface.
type MockMessageGateway struct {
	ctrl     *gomock.Controller
	recorder *MockMessageGatewayMockRecorder
}

// MockMessageGatewayMockRecorder is the mock recorder for MockMessageGateway.
type MockMessageGatewayMockRecorder struct {
	mock *MockMessageGateway
}

// NewMockMessageGateway creates a new mock instance.
func NewMockMessageGateway(ctrl *gomock.Controller) *MockMessageGateway {
	mock := &MockMessageGateway{ctrl: ctrl}
	mock.recorder = &MockMessageGatewayMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageGateway) EXPECT() *MockMessageGatewayMockRecorder {
	return m.recorder
}

// Broadcast mocks base method.
func (m *MockMessageGateway) Broadcast(envelope *protobuf.NetworkMessage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Broadcast", envelope)
}

// Broadcast indicates an expected call of Broadcast.
func (mr *MockMessageGatewayMockRecorder) Broadcast(envelope interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Broadcast", reflect.TypeOf((*MockMessageGateway)(nil).Broadcast), envelope)
}

// Send mocks base method.
func (m *MockMessageGateway) Send(peer transport.PeerID, envelope *protobuf.NetworkMessage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Send", peer, envelope)
}

// Send indicates an expected call of Send.
func (mr *MockMessageGatewayMockRecorder) Send(peer, envelope interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockMessageGateway)(nil).Send), peer, envelope)
}
