// Package grouplogger wraps a Stackdriver logging client to facilitate writing
// groups of log entries, similar to the default behavior in Google App Engine
// Standard.
//
//		var r *http.Request
//
//		ctx := context.Background()
//		cli, err := NewClient(ctx, "logging-parent")
//		if err != nil {
//			// Handle "failed to generate Stackdriver client."
//		}
//
//		logger := cli.Logger(r, "app_identifier", logging.CommonLabels(WithHostname(nil)))
//
//		logger.Info("Info log entry body.")
//		logger.Error("Error log entry body.")
//
//		logger.Close()
package grouplogger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/logging"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

const (
	// outerFormat is a format string for a GroupLogger's outer log name.
	outerFormat = "%v-request"
	// innerFormat is a format string for a GroupLogger's inner log name.
	innerFormat = "%v-app"
)

// Client adds different Logger generation to Stackdriver's logging.Client.
//
// It can be reused across multiple requests to generate a Logger for each one
// without repeating auth.
type Client struct {
	innerClient *logging.Client
}

// NewClient generates a new Client associated with the provided parent.
//
// Options are documented here:
// https://godoc.org/google.golang.org/api/option#ClientOption
func NewClient(ctx context.Context, parent string, opts ...option.ClientOption) (*Client, error) {
	client, err := logging.NewClient(ctx, parent, opts...)
	if err != nil {
		return &Client{}, err
	}
	return &Client{client}, nil
}

// Close waits for all opened GroupLoggers to be flushed and closes the client.
func (client *Client) Close() error {
	return client.innerClient.Close()
}

// Logger constructs and returns a new GroupLogger object for a new group of log
// entries corresponding to a request R.
//
// Logger options (labels, resources, etc.) are documented here:
// https://godoc.org/cloud.google.com/go/logging#LoggerOption
func (client *Client) Logger(r *http.Request, name string, opts ...logging.LoggerOption) *GroupLogger {
	outerLogger := client.innerClient.Logger(fmt.Sprintf(outerFormat, name), opts...)
	innerLogger := client.innerClient.Logger(fmt.Sprintf(innerFormat, name), opts...)
	// Use trace from request if available; otherwise generate a group ID.
	gl := &GroupLogger{r, getGroupID(r), outerLogger, innerLogger, nil}
	return gl
}

// Ping reports whether the client's connection to the logging service and the
// authentication configuration are valid. To accomplish this, Ping writes a log
// entry "ping" to a log named "ping".
func (client *Client) Ping(ctx context.Context) error {
	return client.innerClient.Ping(ctx)
}

// SetOnError sets the function that is called when an error occurs in a call to
// Log. This function should be called only once, before any method of Client is
// called.
//
// Detailed OnError behavior is documented here:
// https://godoc.org/cloud.google.com/go/logging#Client
func (client *Client) SetOnError(f func(err error)) {
	client.innerClient.OnError = f
}

// GroupLogger wraps two Stackdriver Logger clients. The OuterLogger is used to
// write the entries by which other entries are grouped: usually, these are
// requests. The InnerLogger is used to write the grouped (enclosed) entries.
//
// These groups are associated in the Stackdriver logging console by their
// GroupID.
//
// For the inner entries to appear grouped, either `LogOuterEntry` or
// `CloseWith` must be called.
type GroupLogger struct {
	Req          *http.Request
	GroupID      string
	OuterLogger  *logging.Logger
	InnerLogger  *logging.Logger
	InnerEntries []logging.Entry
}

// Close calls CloseWith without specifying statistics. It does not close the
// client that generated the GroupLogger.
//
// Latency, status, response size, etc. are set to 0 or nil.
func (gl *GroupLogger) Close() {
	gl.CloseWith(&logging.HTTPRequest{})
}

// CloseWith decorates the group's base request with the GroupID and with the
// maximum severity of the inner entries logged so far. It does not close the
// client that generated the GroupLogger.
//
// If LogOuterEntry is not called, nothing from this group will appear in
// the outer log.
func (gl *GroupLogger) CloseWith(stats *logging.HTTPRequest) {
	stats.Request = gl.Req
	entry := logging.Entry{
		Trace:       gl.GroupID,
		Severity:    gl.getMaxSeverity(),
		HTTPRequest: stats,
	}
	gl.LogOuterEntry(entry)
}

// LogInnerEntry pushes an inner log entry for the group, decorated with the
// GroupID.
func (gl *GroupLogger) LogInnerEntry(entry logging.Entry) {
	entry.Trace = gl.GroupID
	gl.InnerLogger.Log(entry)
	gl.InnerEntries = append(gl.InnerEntries, entry)
}

// LogOuterEntry pushes the top-level log entry for the group, decorated
// with the GroupID.
//
// For the group to be grouped in the GCP logging console, ENTRY must have
// entry.HTTPRequest set.
func (gl *GroupLogger) LogOuterEntry(entry logging.Entry) {
	entry.Trace = gl.GroupID
	gl.OuterLogger.Log(entry)
}

// Log logs the payload as an inner entry with severity logging.Default.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Log(e logging.Entry) {
	gl.LogInnerEntry(e)
}

// Default logs the payload as an inner entry with severity logging.Default.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Default(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Default,
		Payload:  payload,
	})
}

// Debug logs the payload as an inner entry with severity logging.Debug.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Debug(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Debug,
		Payload:  payload,
	})
}

// Info logs the payload as an inner entry with severity logging.Info.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Info(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Info,
		Payload:  payload,
	})
}

// Notice logs the payload as an inner entry with severity logging.Notice.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Notice(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Notice,
		Payload:  payload,
	})
}

// Warning logs the payload as an inner entry with severity logging.Warning.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Warning(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Warning,
		Payload:  payload,
	})
}

// Error logs the payload as an inner entry with severity logging.Error.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Error(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Error,
		Payload:  payload,
	})
}

// Critical logs the payload as an inner entry with severity logging.Critical.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Critical(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Critical,
		Payload:  payload,
	})
}

// Alert logs the payload as an inner entry with severity logging.Alert.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Alert(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Alert,
		Payload:  payload,
	})
}

// Emergency logs the payload as an inner entry with severity logging.Emergency.
// Payload must be JSON-marshalable.
func (gl *GroupLogger) Emergency(payload interface{}) {
	gl.LogInnerEntry(logging.Entry{
		Severity: logging.Emergency,
		Payload:  payload,
	})
}

// getMaxSeverity returns the highest severity among the inner entries logged by
// a grouplogger.
// Logging severities are specified in Stackdriver documentation.
func (gl *GroupLogger) getMaxSeverity() logging.Severity {
	max := logging.Default
	for _, entry := range gl.InnerEntries {
		if entry.Severity > max {
			max = entry.Severity
		}
	}
	return max
}

// getGroupID selects an ID by which the group will be grouped in the Google
// Cloud Logging console.
//
// If the `X-Cloud-Trace-Context` header is set in the request by GCP
// middleware, then that trace ID is used.
//
// Otherwise, a pseudorandom UUID is used.
func getGroupID(r *http.Request) string {
	// If the trace header exists, use the trace.
	if id := r.Header.Get("X-Cloud-Trace-Context"); id != "" {
		return id
	}
	// Otherwise, generate a random group ID.
	return uuid.New().String()
}

var detectedHost struct {
	hostname string
	once     sync.Once
}

// WithHostname adds the hostname to a labels map. Useful for setting common
// labels: logging.CommonLabels(WithHostname(labels))
func WithHostname(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	detectedHost.once.Do(func() {
		if metadata.OnGCE() {
			instanceName, err := metadata.InstanceName()
			if err == nil {
				detectedHost.hostname = instanceName
			}
		} else {
			hostname, err := os.Hostname()
			if err == nil {
				detectedHost.hostname = hostname
			}
		}
	})
	labels["hostname"] = detectedHost.hostname
	return labels
}
