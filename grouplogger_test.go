package grouplogger

import (
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/logging"
)

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
