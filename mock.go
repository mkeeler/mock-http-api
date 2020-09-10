package mockapi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// checkError ensures that the error value is nil
// If the t parameter is non nil we will fail and
// abort the current test if the err is non-nil. If
// t is nil then we will panic if the err is non-nil.
func checkError(t TestingT, err error) {
	if t != nil {
		require.NoError(t, err)
	} else {
		if err != nil {
			panic(err)
		}
	}
}

// TestingT is the interface encompassing all this libraries unconditional
// uses of methods typically found on the testing.T type.
type TestingT interface {
	mock.TestingT
}

// CleanerT is the interface that may optionally be implemented for Go 1.14
// compatibility in addition to generic testing.T compatibility with older versions.
type CleanerT interface {
	TestingT
	Cleanup(func())
}

// MockRequest is the container for all the elements pertaining to an expected API
// request.
type MockRequest struct {
	method      string
	path        string
	body        interface{}
	headers     map[string]string
	queryParams map[string]string
}

// NewMockRequest will create a new MockRequest. Other With* methods
// can then be called to build out the other parts of the expected request
func NewMockRequest(method, path string) *MockRequest {
	return &MockRequest{
		method: method,
		path:   path,
	}
}

func (r *MockRequest) WithBody(body interface{}) *MockRequest {
	r.body = body
	return r
}

// WithHeaders will set these headers to be expected in the request
func (r *MockRequest) WithHeaders(headers map[string]string) *MockRequest {
	r.headers = headers
	return r
}

// WithQueryParams will set these query params to be expected in the request
func (r *MockRequest) WithQueryParams(params map[string]string) *MockRequest {
	r.queryParams = params
	return r
}

// MockResponse is the type of function that the mock HTTP server is expecting
// to be used to handle setting up the response. This function should write
// a status code and maybe a body
type MockResponse func(http.ResponseWriter, *http.Request)

// MockAPI is the container holding all the bits necessary to provide a mocked HTTP
// API.
type MockAPI struct {
	s *httptest.Server
	t TestingT

	filteredHeaders map[string]struct{}
	filteredParams  map[string]struct{}

	m mock.Mock
}

// NewMockAPI creates a MockAPI. If `t` supports the Go 1.14 Cleanup function
// then a cleanup routine will be setup to close the MockAPI when the test
// completes. This will teardown the HTTP server and assert that all the
// required HTTP calls were made. If not using Go 1.14 then the caller
// should ensure that Close() is called in order to properly shut things down.
func NewMockAPI(t TestingT) *MockAPI {
	mapi := MockAPI{t: t}
	mapi.m.Test(t)
	mapi.s = httptest.NewServer(&mapi)

	if cleanupT, canUseCleanup := t.(CleanerT); canUseCleanup {
		cleanupT.Cleanup(mapi.Close)
	}

	return &mapi
}

// SetFilteredHeaders sets a list of headers that shouldn't be taken into
// account when recording an API call.
func (m *MockAPI) SetFilteredHeaders(headers []string) {
	hdrMap := make(map[string]struct{})
	for _, hdr := range headers {
		hdrMap[hdr] = struct{}{}
	}
	m.filteredHeaders = hdrMap
}

// SetFilteredQueryParams sets a list of query params that shouldn't be taken into
// account when recording an API call.
func (m *MockAPI) SetFilteredQueryParams(params []string) {
	paramMap := make(map[string]struct{})
	for _, param := range params {
		paramMap[param] = struct{}{}
	}
	m.filteredParams = paramMap
}

// URL returns the URL the HTTP server is listening on. It will have the
// form described for the httptest.Server's URL field
// https://pkg.go.dev/net/http/httptest#Server
func (m *MockAPI) URL() string {
	return m.s.URL
}

// ServeHTTP implements the HTTP.Handler interface
func (m *MockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body interface{}

	if r.Body != nil {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			body = bodyBytes

			var bodyMap map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
				body = bodyMap
			}
		}
	}

	var headers map[string]string
	for hdr, values := range r.Header {
		if _, ok := m.filteredHeaders[hdr]; ok {
			continue
		}
		if headers == nil {
			headers = make(map[string]string)
		}
		headers[hdr] = values[0]
		m.t.Errorf("multi-value header was unexpected")
	}

	var params map[string]string
	for param, values := range r.URL.Query() {
		if _, ok := m.filteredParams[param]; ok {
			continue
		}
		if params == nil {
			params = make(map[string]string)
		}
		params[param] = values[0]
		m.t.Errorf("multi-value query param was unexpected")
	}

	ret := m.m.Called(r.Method, r.URL.Path, headers, params, body)

	if replyFn, ok := ret.Get(0).(MockResponse); ok {
		replyFn(w, r)
		return
	}
}

// Close will stop the HTTP server and also assert that all expected HTTP invocations
// have happened.
func (m *MockAPI) Close() {
	m.s.Close()
	m.m.AssertExpectations(m.t)
}

// WithRequest will setup an expectation for an API call to be made. Its is the responsibility of the
// passed in response function to set the HTTP status code and write out any body.
// The body may of the MockRequest passed in may be either nil, a []byte or a map[string]interface{}.
// During processing of the HTTP request, the entire body will be read. If the len is not greater than 0,
// then nil will be recorded as the body. If the len is greater than 0 an attempt to JSON decode the body
// contents into a map[string]interface{} is made. If successful the map is recorded as the body, if
// unsuccessful then the raw []byte is recorded as the body.
func (m *MockAPI) WithRequest(req *MockRequest, resp MockResponse) *MockAPICall {
	c := m.m.On("ServeHTTP", req.method, req.path, req.headers, req.queryParams, req.body).Return(resp)
	return &MockAPICall{c: c}
}

func (m *MockAPI) DefaultHandler(response func(http.ResponseWriter, *http.Request)) *MockAPICall {
	c := m.m.On("ServeHTTP", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(response).Times(0)
	return &MockAPICall{c: c}
}

// WithNoResponseBody will setup an expectation for an API call to be made. The supplied status code will
// be used for the responses reply but no response body will be written.
func (m *MockAPI) WithNoResponseBody(req *MockRequest, status int) *MockAPICall {
	return m.WithRequest(req, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	})
}

// WithTxtReply will setup an expectation for an API call to be made. The supplied status code will
// be use for the responses reply and the reply object will be JSON encoded and written to the response. If there is
// an error in JSON encoding it will fail the test object passed into the NewMockAPI constructor if that
// was non-nil and if it was nil, will panic. The method, path and body parameters are the same as for
// the Request method.
func (m *MockAPI) WithJSONReply(req *MockRequest, status int, reply interface{}) *MockAPICall {
	return m.WithRequest(req, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)

		fmt.Printf("reply: %v\n", reply)
		if reply == nil {
			return
		}

		enc := json.NewEncoder(w)
		err := enc.Encode(reply)
		if m.t != nil {
			require.NoError(m.t, err)
		} else {
			panic(err)
		}
	})
}

// WithTxtReply will setup an expectation for an API call to be made. The supplied status code will
// be use for the responses reply and the reply string will be written to the response.
func (m *MockAPI) WithTxtReply(req *MockRequest, status int, reply string) *MockAPICall {
	return m.WithRequest(req, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(reply))
	})
}

// WithStreamingReply will setup an expectation for an API call to be made. The supplied status code will
// be used for the responses reply and the reply readers content will be copied as the response body.
func (m *MockAPI) WithStreamingReply(req *MockRequest, status int, reply io.Reader) *MockAPICall {
	return m.WithRequest(req, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)

		if reply == nil {
			return
		}

		_, err := io.Copy(w, reply)
		checkError(m.t, err)
	})
}

// AssertExpectations will assert that all expected API invocations have happened and fail
// the test if any required calls did not happen.
func (m *MockAPI) AssertExpectations(t TestingT) {
	if t == nil {
		// cannot actually do anything as the AssertExpectations requires a t.
		// potentially we could reuse the t value on the MockAPI but that doesn't
		// seem very clean. If you want to use that one then its probably best to just
		// defer m.Close() and let us call AssertExpectations that way.
		return
	}
	m.m.AssertExpectations(t)
}

// MockAPICall is a wrapper around the github.com/stretchr/testify/mock.Call
// type. It provides a smaller interface that is more suitable for use with
// the MockAPI type and should prevent some accidental issues.
type MockAPICall struct {
	c *mock.Call
}

// Maybe marks this API call as optional.
func (m *MockAPICall) Maybe() *MockAPICall {
	m.c.Maybe()
	return m
}

// Once marks this API call as being expected to occur exactly once.
func (m *MockAPICall) Once() *MockAPICall {
	m.c.Once()
	return m
}

// Times marks this API call as being expected to occur the specified number of times.
func (m *MockAPICall) Times(i int) *MockAPICall {
	m.c.Times(i)
	return m
}

// Twice marks this API call as being expected to occur exactly twice
func (m *MockAPICall) Twice() *MockAPICall {
	m.c.Twice()
	return m
}

// WaitUntil sets the channel that will block the sending back an HTTP response
// to this Call. This happens prior to setting the status code as well as writing
// out any of the reply (before the function passed to MockAPI.Request is called)
func (m *MockAPICall) WaitUntil(w <-chan time.Time) *MockAPICall {
	m.c.WaitUntil(w)
	return m
}
