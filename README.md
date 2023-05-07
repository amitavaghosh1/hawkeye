# hawkeye

metric monitoring and reporting system.

Depends on:
- go
- go-task (for development)
- redis
- notification-service


This works with RPC invocation. Golang supports `jsonrpc`, which means, we can use other languages to make `RPC` call.

## Overview

The parts of the system involves:
- A RPC Server, that aggregates metric from socket and processes them
- An Agent, which monitors metrics to be tracked and notifies when SLA is breached
- The Notification Agent, which runs every set interval (in config)
- The client side intrumenter to send metrics to channel

#### RPC Server

The RPC server listens on a unix domain socket. And it listens for messages forver, until system is reboot or crashes.
This can be run independently as a separate process. Or can be invoked from code.

For implementation details check `cmd/client/main.go`. This is also the standalone version.

#### Agent

The Agent read the monitoring config from an yaml file. And based on the config value,
it runs an aggregator at every `interval` (in seconds), and gets the aggregated count for the metric key.

Once this `count` breaches the `threshold` in config, it sends a notification.

For implementation details check `cmd/agent/main.go`. This is also the standalone version.

Requirements:

- The standalone version requires `.env` to be present in your working directory.
- The monitoring config needs to be present in `monitors.yaml` in root project dir.

- The code invoked version takes some required `AppConfig` and can be invoked in a routine.
- The coded version also the monitors.yaml file path in config


#### Notifier

The notification agent, is an interface, but here we are using emails to notify.

This agent has a bit of an weird implementation. Since we don't have any complex monitor notification tracking,
I didn't want to send duplicate email notifications. 

So for now, the approach is to make the notifier run at every `run_every` (in minutes).

This does create a problem of missed notification, but for now, we can set the value to a lesser number.

#### Client library

The client library right now, when it receives a metric send request, uses golang's `rpc.Go`, to send
the metric asynchronously

But we can definitely have a buffered channel and consume in batches.


### Caveats:

In monitor config (`monitors.yaml`) each trigger, there is a separate list of recipients, this was built to notify users at different error levels.

Right now, for each trigger there is a separate monitoring channel that gets setup. In most of our use case, I think, we mostly need 2 levels.
But, this does create the possibility of too many go routines. This can be changed, by modifying the application code such that,

it loops through each trigger and then check the threshold. But each `trigger` will have its own notification medium.

This requires a new config modification. We will solve this with `version: 2` in config file, if required.

