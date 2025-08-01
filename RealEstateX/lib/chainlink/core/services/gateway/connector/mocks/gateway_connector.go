// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	api "github.com/smartcontractkit/chainlink/v2/core/services/gateway/api"
	connector "github.com/smartcontractkit/chainlink/v2/core/services/gateway/connector"

	context "context"

	mock "github.com/stretchr/testify/mock"

	url "net/url"
)

// GatewayConnector is an autogenerated mock type for the GatewayConnector type
type GatewayConnector struct {
	mock.Mock
}

type GatewayConnector_Expecter struct {
	mock *mock.Mock
}

func (_m *GatewayConnector) EXPECT() *GatewayConnector_Expecter {
	return &GatewayConnector_Expecter{mock: &_m.Mock}
}

// AddHandler provides a mock function with given fields: methods, handler
func (_m *GatewayConnector) AddHandler(methods []string, handler connector.GatewayConnectorHandler) error {
	ret := _m.Called(methods, handler)

	if len(ret) == 0 {
		panic("no return value specified for AddHandler")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]string, connector.GatewayConnectorHandler) error); ok {
		r0 = rf(methods, handler)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_AddHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddHandler'
type GatewayConnector_AddHandler_Call struct {
	*mock.Call
}

// AddHandler is a helper method to define mock.On call
//   - methods []string
//   - handler connector.GatewayConnectorHandler
func (_e *GatewayConnector_Expecter) AddHandler(methods interface{}, handler interface{}) *GatewayConnector_AddHandler_Call {
	return &GatewayConnector_AddHandler_Call{Call: _e.mock.On("AddHandler", methods, handler)}
}

func (_c *GatewayConnector_AddHandler_Call) Run(run func(methods []string, handler connector.GatewayConnectorHandler)) *GatewayConnector_AddHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]string), args[1].(connector.GatewayConnectorHandler))
	})
	return _c
}

func (_c *GatewayConnector_AddHandler_Call) Return(_a0 error) *GatewayConnector_AddHandler_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_AddHandler_Call) RunAndReturn(run func([]string, connector.GatewayConnectorHandler) error) *GatewayConnector_AddHandler_Call {
	_c.Call.Return(run)
	return _c
}

// AwaitConnection provides a mock function with given fields: ctx, gatewayID
func (_m *GatewayConnector) AwaitConnection(ctx context.Context, gatewayID string) error {
	ret := _m.Called(ctx, gatewayID)

	if len(ret) == 0 {
		panic("no return value specified for AwaitConnection")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, gatewayID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_AwaitConnection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AwaitConnection'
type GatewayConnector_AwaitConnection_Call struct {
	*mock.Call
}

// AwaitConnection is a helper method to define mock.On call
//   - ctx context.Context
//   - gatewayID string
func (_e *GatewayConnector_Expecter) AwaitConnection(ctx interface{}, gatewayID interface{}) *GatewayConnector_AwaitConnection_Call {
	return &GatewayConnector_AwaitConnection_Call{Call: _e.mock.On("AwaitConnection", ctx, gatewayID)}
}

func (_c *GatewayConnector_AwaitConnection_Call) Run(run func(ctx context.Context, gatewayID string)) *GatewayConnector_AwaitConnection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *GatewayConnector_AwaitConnection_Call) Return(_a0 error) *GatewayConnector_AwaitConnection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_AwaitConnection_Call) RunAndReturn(run func(context.Context, string) error) *GatewayConnector_AwaitConnection_Call {
	_c.Call.Return(run)
	return _c
}

// ChallengeResponse provides a mock function with given fields: ctx, _a1, challenge
func (_m *GatewayConnector) ChallengeResponse(ctx context.Context, _a1 *url.URL, challenge []byte) ([]byte, error) {
	ret := _m.Called(ctx, _a1, challenge)

	if len(ret) == 0 {
		panic("no return value specified for ChallengeResponse")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *url.URL, []byte) ([]byte, error)); ok {
		return rf(ctx, _a1, challenge)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *url.URL, []byte) []byte); ok {
		r0 = rf(ctx, _a1, challenge)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *url.URL, []byte) error); ok {
		r1 = rf(ctx, _a1, challenge)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GatewayConnector_ChallengeResponse_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ChallengeResponse'
type GatewayConnector_ChallengeResponse_Call struct {
	*mock.Call
}

// ChallengeResponse is a helper method to define mock.On call
//   - ctx context.Context
//   - _a1 *url.URL
//   - challenge []byte
func (_e *GatewayConnector_Expecter) ChallengeResponse(ctx interface{}, _a1 interface{}, challenge interface{}) *GatewayConnector_ChallengeResponse_Call {
	return &GatewayConnector_ChallengeResponse_Call{Call: _e.mock.On("ChallengeResponse", ctx, _a1, challenge)}
}

func (_c *GatewayConnector_ChallengeResponse_Call) Run(run func(ctx context.Context, _a1 *url.URL, challenge []byte)) *GatewayConnector_ChallengeResponse_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*url.URL), args[2].([]byte))
	})
	return _c
}

func (_c *GatewayConnector_ChallengeResponse_Call) Return(_a0 []byte, _a1 error) *GatewayConnector_ChallengeResponse_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *GatewayConnector_ChallengeResponse_Call) RunAndReturn(run func(context.Context, *url.URL, []byte) ([]byte, error)) *GatewayConnector_ChallengeResponse_Call {
	_c.Call.Return(run)
	return _c
}

// Close provides a mock function with no fields
func (_m *GatewayConnector) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type GatewayConnector_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *GatewayConnector_Expecter) Close() *GatewayConnector_Close_Call {
	return &GatewayConnector_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *GatewayConnector_Close_Call) Run(run func()) *GatewayConnector_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *GatewayConnector_Close_Call) Return(_a0 error) *GatewayConnector_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_Close_Call) RunAndReturn(run func() error) *GatewayConnector_Close_Call {
	_c.Call.Return(run)
	return _c
}

// DonID provides a mock function with no fields
func (_m *GatewayConnector) DonID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for DonID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GatewayConnector_DonID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DonID'
type GatewayConnector_DonID_Call struct {
	*mock.Call
}

// DonID is a helper method to define mock.On call
func (_e *GatewayConnector_Expecter) DonID() *GatewayConnector_DonID_Call {
	return &GatewayConnector_DonID_Call{Call: _e.mock.On("DonID")}
}

func (_c *GatewayConnector_DonID_Call) Run(run func()) *GatewayConnector_DonID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *GatewayConnector_DonID_Call) Return(_a0 string) *GatewayConnector_DonID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_DonID_Call) RunAndReturn(run func() string) *GatewayConnector_DonID_Call {
	_c.Call.Return(run)
	return _c
}

// GatewayIDs provides a mock function with no fields
func (_m *GatewayConnector) GatewayIDs() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GatewayIDs")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// GatewayConnector_GatewayIDs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GatewayIDs'
type GatewayConnector_GatewayIDs_Call struct {
	*mock.Call
}

// GatewayIDs is a helper method to define mock.On call
func (_e *GatewayConnector_Expecter) GatewayIDs() *GatewayConnector_GatewayIDs_Call {
	return &GatewayConnector_GatewayIDs_Call{Call: _e.mock.On("GatewayIDs")}
}

func (_c *GatewayConnector_GatewayIDs_Call) Run(run func()) *GatewayConnector_GatewayIDs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *GatewayConnector_GatewayIDs_Call) Return(_a0 []string) *GatewayConnector_GatewayIDs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_GatewayIDs_Call) RunAndReturn(run func() []string) *GatewayConnector_GatewayIDs_Call {
	_c.Call.Return(run)
	return _c
}

// HealthReport provides a mock function with no fields
func (_m *GatewayConnector) HealthReport() map[string]error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for HealthReport")
	}

	var r0 map[string]error
	if rf, ok := ret.Get(0).(func() map[string]error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]error)
		}
	}

	return r0
}

// GatewayConnector_HealthReport_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HealthReport'
type GatewayConnector_HealthReport_Call struct {
	*mock.Call
}

// HealthReport is a helper method to define mock.On call
func (_e *GatewayConnector_Expecter) HealthReport() *GatewayConnector_HealthReport_Call {
	return &GatewayConnector_HealthReport_Call{Call: _e.mock.On("HealthReport")}
}

func (_c *GatewayConnector_HealthReport_Call) Run(run func()) *GatewayConnector_HealthReport_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *GatewayConnector_HealthReport_Call) Return(_a0 map[string]error) *GatewayConnector_HealthReport_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_HealthReport_Call) RunAndReturn(run func() map[string]error) *GatewayConnector_HealthReport_Call {
	_c.Call.Return(run)
	return _c
}

// Name provides a mock function with no fields
func (_m *GatewayConnector) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GatewayConnector_Name_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Name'
type GatewayConnector_Name_Call struct {
	*mock.Call
}

// Name is a helper method to define mock.On call
func (_e *GatewayConnector_Expecter) Name() *GatewayConnector_Name_Call {
	return &GatewayConnector_Name_Call{Call: _e.mock.On("Name")}
}

func (_c *GatewayConnector_Name_Call) Run(run func()) *GatewayConnector_Name_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *GatewayConnector_Name_Call) Return(_a0 string) *GatewayConnector_Name_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_Name_Call) RunAndReturn(run func() string) *GatewayConnector_Name_Call {
	_c.Call.Return(run)
	return _c
}

// NewAuthHeader provides a mock function with given fields: ctx, _a1
func (_m *GatewayConnector) NewAuthHeader(ctx context.Context, _a1 *url.URL) ([]byte, error) {
	ret := _m.Called(ctx, _a1)

	if len(ret) == 0 {
		panic("no return value specified for NewAuthHeader")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *url.URL) ([]byte, error)); ok {
		return rf(ctx, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *url.URL) []byte); ok {
		r0 = rf(ctx, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *url.URL) error); ok {
		r1 = rf(ctx, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GatewayConnector_NewAuthHeader_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NewAuthHeader'
type GatewayConnector_NewAuthHeader_Call struct {
	*mock.Call
}

// NewAuthHeader is a helper method to define mock.On call
//   - ctx context.Context
//   - _a1 *url.URL
func (_e *GatewayConnector_Expecter) NewAuthHeader(ctx interface{}, _a1 interface{}) *GatewayConnector_NewAuthHeader_Call {
	return &GatewayConnector_NewAuthHeader_Call{Call: _e.mock.On("NewAuthHeader", ctx, _a1)}
}

func (_c *GatewayConnector_NewAuthHeader_Call) Run(run func(ctx context.Context, _a1 *url.URL)) *GatewayConnector_NewAuthHeader_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*url.URL))
	})
	return _c
}

func (_c *GatewayConnector_NewAuthHeader_Call) Return(_a0 []byte, _a1 error) *GatewayConnector_NewAuthHeader_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *GatewayConnector_NewAuthHeader_Call) RunAndReturn(run func(context.Context, *url.URL) ([]byte, error)) *GatewayConnector_NewAuthHeader_Call {
	_c.Call.Return(run)
	return _c
}

// Ready provides a mock function with no fields
func (_m *GatewayConnector) Ready() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Ready")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_Ready_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Ready'
type GatewayConnector_Ready_Call struct {
	*mock.Call
}

// Ready is a helper method to define mock.On call
func (_e *GatewayConnector_Expecter) Ready() *GatewayConnector_Ready_Call {
	return &GatewayConnector_Ready_Call{Call: _e.mock.On("Ready")}
}

func (_c *GatewayConnector_Ready_Call) Run(run func()) *GatewayConnector_Ready_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *GatewayConnector_Ready_Call) Return(_a0 error) *GatewayConnector_Ready_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_Ready_Call) RunAndReturn(run func() error) *GatewayConnector_Ready_Call {
	_c.Call.Return(run)
	return _c
}

// SendToGateway provides a mock function with given fields: ctx, gatewayID, msg
func (_m *GatewayConnector) SendToGateway(ctx context.Context, gatewayID string, msg *api.Message) error {
	ret := _m.Called(ctx, gatewayID, msg)

	if len(ret) == 0 {
		panic("no return value specified for SendToGateway")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *api.Message) error); ok {
		r0 = rf(ctx, gatewayID, msg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_SendToGateway_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SendToGateway'
type GatewayConnector_SendToGateway_Call struct {
	*mock.Call
}

// SendToGateway is a helper method to define mock.On call
//   - ctx context.Context
//   - gatewayID string
//   - msg *api.Message
func (_e *GatewayConnector_Expecter) SendToGateway(ctx interface{}, gatewayID interface{}, msg interface{}) *GatewayConnector_SendToGateway_Call {
	return &GatewayConnector_SendToGateway_Call{Call: _e.mock.On("SendToGateway", ctx, gatewayID, msg)}
}

func (_c *GatewayConnector_SendToGateway_Call) Run(run func(ctx context.Context, gatewayID string, msg *api.Message)) *GatewayConnector_SendToGateway_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*api.Message))
	})
	return _c
}

func (_c *GatewayConnector_SendToGateway_Call) Return(_a0 error) *GatewayConnector_SendToGateway_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_SendToGateway_Call) RunAndReturn(run func(context.Context, string, *api.Message) error) *GatewayConnector_SendToGateway_Call {
	_c.Call.Return(run)
	return _c
}

// SignAndSendToGateway provides a mock function with given fields: ctx, gatewayID, msg
func (_m *GatewayConnector) SignAndSendToGateway(ctx context.Context, gatewayID string, msg *api.MessageBody) error {
	ret := _m.Called(ctx, gatewayID, msg)

	if len(ret) == 0 {
		panic("no return value specified for SignAndSendToGateway")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *api.MessageBody) error); ok {
		r0 = rf(ctx, gatewayID, msg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_SignAndSendToGateway_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SignAndSendToGateway'
type GatewayConnector_SignAndSendToGateway_Call struct {
	*mock.Call
}

// SignAndSendToGateway is a helper method to define mock.On call
//   - ctx context.Context
//   - gatewayID string
//   - msg *api.MessageBody
func (_e *GatewayConnector_Expecter) SignAndSendToGateway(ctx interface{}, gatewayID interface{}, msg interface{}) *GatewayConnector_SignAndSendToGateway_Call {
	return &GatewayConnector_SignAndSendToGateway_Call{Call: _e.mock.On("SignAndSendToGateway", ctx, gatewayID, msg)}
}

func (_c *GatewayConnector_SignAndSendToGateway_Call) Run(run func(ctx context.Context, gatewayID string, msg *api.MessageBody)) *GatewayConnector_SignAndSendToGateway_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*api.MessageBody))
	})
	return _c
}

func (_c *GatewayConnector_SignAndSendToGateway_Call) Return(_a0 error) *GatewayConnector_SignAndSendToGateway_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_SignAndSendToGateway_Call) RunAndReturn(run func(context.Context, string, *api.MessageBody) error) *GatewayConnector_SignAndSendToGateway_Call {
	_c.Call.Return(run)
	return _c
}

// Start provides a mock function with given fields: _a0
func (_m *GatewayConnector) Start(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GatewayConnector_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type GatewayConnector_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *GatewayConnector_Expecter) Start(_a0 interface{}) *GatewayConnector_Start_Call {
	return &GatewayConnector_Start_Call{Call: _e.mock.On("Start", _a0)}
}

func (_c *GatewayConnector_Start_Call) Run(run func(_a0 context.Context)) *GatewayConnector_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *GatewayConnector_Start_Call) Return(_a0 error) *GatewayConnector_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GatewayConnector_Start_Call) RunAndReturn(run func(context.Context) error) *GatewayConnector_Start_Call {
	_c.Call.Return(run)
	return _c
}

// NewGatewayConnector creates a new instance of GatewayConnector. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewGatewayConnector(t interface {
	mock.TestingT
	Cleanup(func())
}) *GatewayConnector {
	mock := &GatewayConnector{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
