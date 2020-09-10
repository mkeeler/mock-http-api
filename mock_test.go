package mockapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	// mockapi "github.com/mkeeler/mock-http-api"
)

// This test will pass as all the requisite API calls are made.
func TestMyAPI(t *testing.T) {
	m := NewMockAPI(t)
	// http.Get will add both of the headers but we don't want to care about them.
	m.SetFilteredHeaders([]string{
		"Accept-Encoding",
		"User-Agent",
	})

	// This sets up an expectation that a GET request to /my/endpoint will be made and that it should
	// return a 200 status code with the provided map sent back JSON encoded as the body of the response
	call := m.WithJSONReply(NewMockRequest("GET", "/my/endpoint"), 200, map[string]string{
		"foo": "bar",
	})

	// This sets the call to be required to happen exactly once
	call.Once()

	// This makes the HTTP request to the mock HTTP server
	resp, err := http.Get(fmt.Sprintf("%s/my/endpoint", m.URL()))
	if err != nil {
		t.Fatalf("Error issuing GET of /my/endpoint: %v", err)
	}

	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)

	var output map[string]string
	if err := dec.Decode(&output); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if val, ok := output["foo"]; !ok || val != "bar" {
		t.Fatalf("Didn't get the expected response")
	}
}
