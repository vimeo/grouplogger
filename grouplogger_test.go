package grouplogger

import (
	"net/http"
	"testing"
)

const fake_uuid = "fake_uuid"

func mockUUIDFunc() string {
	return fake_uuid
}

func TestGetGroupIDWithRequestWithHeader(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://www.vimeo.com", nil)
	r.Header.Set("X-Cloud-Trace-Context", "123")
	id := getGroupID(r, newUUID)
	if id != "123" {
		t.Fatal(id)
	}
}

func TestGetGroupIDWithRequestWithoutHeader(t *testing.T) {
	id := getGroupID(&http.Request{}, mockUUIDFunc)
	if id != fake_uuid {
		t.Fatal(id)
	}
}

func TestGetGroupIDWithoutRequest(t *testing.T) {
	id := getGroupID(nil, mockUUIDFunc)
	if id != fake_uuid {
		t.Fatal(id)
	}
}
