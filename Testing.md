## local setup

To run this locally, you need to install `go-task`. 

### Standalone

Start the collector and agent separately.

- `task run.server`
- `task run.agent`

And to test metrics, you can modify in `cmd/cli/main.go`. You can run it using `task run.cmd`

### Example API integration

You can also embed both of them in application code. Please see, `example/main.go`.

An example has been provided using `gin`. To test the notifications for threshold breach. You need a benchmarking tool.

- Run the server with `task run.example`
- Using apache benchmark run `ab -n 10 -c 1 http://localhost:8080/400`
- In your logs you will be able to see an email

You can pass your own `MailingService` implementation. The interface is:

```
type MailingService interface {
	Send(ctx context.Context, cfg MailerConfig) error
}

type MailerConfig struct {
	Subject    string
	Body       string
	Recipients []string
	Sender     string
	CC         []string
	Bcc        []string
}
```

