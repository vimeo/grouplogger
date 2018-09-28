package grouplogger

import (
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/logging"
)

const fake_uuid = "fake_uuid"

func mockUUIDFunc() string {
	return fake_uuid
}

func TestGetGroupIDWithRequestWithHeader(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://www.vimeo.com", nil)
	r.Header.Set("X-Cloud-Trace-Context", "123")
	id := getGroupID(r, mockUUIDFunc)
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

func TestCloseWith(t *testing.T) {
	var outerEntry logging.Entry

	r, _ := http.NewRequest("GET", "https://www.vimeo.com", nil)

	gl := GroupLogger{
		Req:     r,
		GroupID: "fake_GroupID",
		OuterLogger: &mockLogger{
			LogFunc: func(e logging.Entry) {
				outerEntry = e
			},
		},
		InnerEntries: []logging.Entry{
			logging.Entry{
				Severity: logging.ParseSeverity("Info"),
			},
			logging.Entry{
				Severity: logging.ParseSeverity("Alert"),
			},
			logging.Entry{
				Severity: logging.ParseSeverity("Error"),
			},
		},
	}

	stats := logging.HTTPRequest{
		Latency: 1 * time.Second,
	}

	gl.CloseWith(&stats)

	if outerEntry.Severity.String() != "Alert" {
		t.Fatal(outerEntry.Severity.String())
	}

	if outerEntry.HTTPRequest.Latency != 1*time.Second {
		t.Fatal(outerEntry.HTTPRequest.Latency)
	}

	if outerEntry.HTTPRequest.Request.URL.String() != "https://www.vimeo.com" {
		t.Fatal(outerEntry.HTTPRequest.Request.URL.String())
	}
}
