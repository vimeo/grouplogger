# grouplogger

[![GoDoc](https://godoc.org/github.com/vimeo/grouplogger?status.svg)](https://godoc.org/github.com/vimeo/grouplogger)

Grouplogger is a specialized Stackdriver logging client for writing groups of log entries. Group entries by contexts that make sense for your application––e.g. by Stackdriver Trace, HTTP request, or Pub/Sub message.

([Stackdriver documentation: Viewing related request log entries](https://cloud.google.com/appengine/docs/flexible/go/writing-application-logs#related-app-logs))

```go
// Stackdriver requires a non-nil http.Request.
u, err := url.Parse("http://notareal.website/search?q=hihihi")
if err != nil {
  log.Fatal(err)
}
req := &http.Request{
  Header: http.Header{
    "X-Cloud-Trace-Context": []string{"your-trace-id"},
  },
  URL: u,
}

cli, err := grouplogger.NewClient(context.Background(), "your-project-id")
if err != nil {
  log.Fatal(err)
}

logger := cli.Logger(req, "logname")

logger.Info("Entry with Info severity.")
logger.Notice(map[string][]string{
"Words": []string{"structured", "data", "in", "entries"},
})
logger.Warning("Look out! Entry with Warning severity.")

logger.Close()

err = cli.Close()
log.Fatal(err)
```

<img alt="screen shot 2018-07-31 at 12 33 06 pm" src="https://user-images.githubusercontent.com/4955943/43481638-8330b71e-94d4-11e8-9288-cc16d48bf062.png">
