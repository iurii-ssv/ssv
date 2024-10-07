// Code generated by MockGen. DO NOT EDIT.
// Source: ./base_handler.go
//
// Generated by this command:
//
//	mockgen -package=duties -destination=./base_handler_mock.go -source=./base_handler.go
//

// Package duties is a generated GoMock package.
package duties

import (
	context "context"
	reflect "reflect"

	networkconfig "github.com/ssvlabs/ssv/networkconfig"
	slotticker "github.com/ssvlabs/ssv/operator/slotticker"
	gomock "go.uber.org/mock/gomock"
	zap "go.uber.org/zap"
)

// MockdutyHandler is a mock of dutyHandler interface.
type MockdutyHandler struct {
	ctrl     *gomock.Controller
	recorder *MockdutyHandlerMockRecorder
}

// MockdutyHandlerMockRecorder is the mock recorder for MockdutyHandler.
type MockdutyHandlerMockRecorder struct {
	mock *MockdutyHandler
}

// NewMockdutyHandler creates a new mock instance.
func NewMockdutyHandler(ctrl *gomock.Controller) *MockdutyHandler {
	mock := &MockdutyHandler{ctrl: ctrl}
	mock.recorder = &MockdutyHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockdutyHandler) EXPECT() *MockdutyHandlerMockRecorder {
	return m.recorder
}

// HandleDuties mocks base method.
func (m *MockdutyHandler) HandleDuties(arg0 context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HandleDuties", arg0)
}

// HandleDuties indicates an expected call of HandleDuties.
func (mr *MockdutyHandlerMockRecorder) HandleDuties(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleDuties", reflect.TypeOf((*MockdutyHandler)(nil).HandleDuties), arg0)
}

// HandleInitialDuties mocks base method.
func (m *MockdutyHandler) HandleInitialDuties(arg0 context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HandleInitialDuties", arg0)
}

// HandleInitialDuties indicates an expected call of HandleInitialDuties.
func (mr *MockdutyHandlerMockRecorder) HandleInitialDuties(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleInitialDuties", reflect.TypeOf((*MockdutyHandler)(nil).HandleInitialDuties), arg0)
}

// IndicesChangeFeed mocks base method.
func (m *MockdutyHandler) IndicesChangeFeed() chan struct{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IndicesChangeFeed")
	ret0, _ := ret[0].(chan struct{})
	return ret0
}

// IndicesChangeFeed indicates an expected call of IndicesChangeFeed.
func (mr *MockdutyHandlerMockRecorder) IndicesChangeFeed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IndicesChangeFeed", reflect.TypeOf((*MockdutyHandler)(nil).IndicesChangeFeed))
}

// Name mocks base method.
func (m *MockdutyHandler) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockdutyHandlerMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockdutyHandler)(nil).Name))
}

// ReorgFeed mocks base method.
func (m *MockdutyHandler) ReorgFeed() chan ReorgEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReorgFeed")
	ret0, _ := ret[0].(chan ReorgEvent)
	return ret0
}

// ReorgFeed indicates an expected call of ReorgFeed.
func (mr *MockdutyHandlerMockRecorder) ReorgFeed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReorgFeed", reflect.TypeOf((*MockdutyHandler)(nil).ReorgFeed))
}

// Setup mocks base method.
func (m *MockdutyHandler) Setup(name string, logger *zap.Logger, beaconNode BeaconNode, executionClient ExecutionClient, network networkconfig.NetworkConfig, validatorProvider ValidatorProvider, validatorController ValidatorController, dutiesExecutor DutiesExecutor, slotTickerProvider slotticker.Provider) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Setup", name, logger, beaconNode, executionClient, network, validatorProvider, validatorController, dutiesExecutor, slotTickerProvider)
}

// Setup indicates an expected call of Setup.
func (mr *MockdutyHandlerMockRecorder) Setup(name, logger, beaconNode, executionClient, network, validatorProvider, validatorController, dutiesExecutor, slotTickerProvider any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Setup", reflect.TypeOf((*MockdutyHandler)(nil).Setup), name, logger, beaconNode, executionClient, network, validatorProvider, validatorController, dutiesExecutor, slotTickerProvider)
}
