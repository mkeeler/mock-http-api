package mockapi

type BodyFormat string

const (
	BodyFormatNone   BodyFormat = "none"
	BodyFormatJSON   BodyFormat = "json"
	BodyFormatString BodyFormat = "string"
	BodyFormatStream BodyFormat = "stream"
)

type ResponseFormat string

const (
	ResponseFormatJSON   ResponseFormat = "json"
	ResponseFormatString ResponseFormat = "string"
	ResponseFormatStream ResponseFormat = "stream"
	ResponseFormatFunc   ResponseFormat = "func"
)

// Endpoint represents an HTTP endpoint to be mocked.
// This is mostly used by github.com/mkeeler/mock-http-api/cmd/mock-expect-gen
// in order to generate expectation helpers for an HTTP API.
type Endpoint struct {
	// Path is the HTTP path this endpoint is served under
	Path string
	// Method is the HTTP Method used to invoke this API
	Method string
	// BodyFormat is what format of body to take as input
	BodyFormat BodyFormat
	// BodyType is the golang type of the Body
	BodyType string

	// PathParameters are the parameters required to be in the path
	PathParameters []string
	// ResponseFormat is the format of Response that helpers should
	ResponseFormat ResponseFormat
	// ResponseType is the golang type of the Response
	ResponseType string
	// Headers indicates that this endpoints operation is influenced by
	// headers which may be present and so the headers should be a part
	// of the expectation
	Headers bool
	// QueryParams indicates that this endpoints operation is influenced by
	// query params which may be present and so the params should be part
	// of the expectation
	QueryParams bool
}
