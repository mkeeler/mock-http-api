package mockapi

type BodyType string

const (
	BodyTypeNone   BodyType = "none"
	BodyTypeJSON   BodyType = "json"
	BodyTypeString BodyType = "string"
	BodyTypeStream BodyType = "stream"
)

type ResponseType string

const (
	ResponseTypeJSON   ResponseType = "json"
	ResponseTypeString ResponseType = "string"
	ResponseTypeStream ResponseType = "stream"
	ResponseTypeFunc   ResponseType = "func"
)

// Endpoint represents an HTTP endpoint to be mocked
// This is mostly used by github.com/mkeeler/mock-http-/api/cmd/mock-expect-gen
// in order to generate expectation helpers for an HTTP API.
type Endpoint struct {
	// Path is the HTTP path this endpoint is served under
	Path string
	// Method is the HTTP Method used to invoke this API
	Method string
	// BodyType is what type of body to take as input
	BodyType BodyType
	// PathParameters are the parameters required to be in the path
	PathParameters []string
	// ResponseType is the type of Response that helpers should
	ResponseType ResponseType

	// Headers indicates that this endpoints operation is influenced by
	// headers which may be present and so the headers should be a part
	// of the expectation
	Headers bool

	// QueryParams indicates that this endpoints operation is influenced by
	// query params which may be present and so the params should be part
	// of the expectation
	QueryParams bool
}
