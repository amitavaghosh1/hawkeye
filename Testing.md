## local setup

To run this locally, you need to install `go-task`. 

### Standalone

Start the collector and agent separately.

- `task run.server`
- `task run.agent`

And to test metrics, you can modify in `cmd/cli/main.go`. You can run it using `task run.cmd`

### Embed

You can also embed both of them in application code. Please see, `example/main.go`.

An example has been provided using `gin`. To test the notifications for threshold breach. You need a benchmarking tool.

We will use `ab` here. `ab -n 10 -c 1 http://localhost:8080/400`
