# prometheus-slacker

**This project is no longer maintained**

This tool queries Prometheus for metrics and sends messages on Slack (using
Webhooks) when thresholds are exceeded.

It's not really any sort of alerting system. The code was created to send hourly
updates on Slack about status of infrastructure and some website and sale
metrics during Black Friday, Xmas etc.

Application is meant to be run on the internal network and cannot be exposed to
the public.

Also, it runs a simple HTTP daemon (listening on `port` config value) which can
be used to send any Slack message to all the webhooks.

### Example
`config-sample.json` is a sample app configuration which could be a good
starting point.

#### Notification levels
In the configuration file, notification levels have to be defined. For example,
these could be green, amber and red. Green might mean your infrastructure is all
right, amber might mean that some services respond slower than usual and red
would mean that action needs to be taken.

Application starts with first notification level, in this case green and then
iterates through all next levels and checks if thresholds defined for them are
exceeded. For example, if "amber" notification level has below
`leverage_metrics` defined, level will be leveraged from "green" to "amber"
when `app1_lb_avg_resp_time` metric value is higher than `0.05`.

```
"leverage_metrics": {
  "app1_lb_avg_resp_time": "0.05"
}
```

In the configuration, Slack webhooks have to be defined for each notification
level. For example, you might only want to send "green" to general channel, not
devops.

#### Metrics
Prometheus metric definition has two fields: query and name under they are
displayed. Following sample configuration file, `query` is a Prometheus query.

```
"metrics": {
  "app1_lb_avg_resp_time": {
	"display_name": "Some App LB Avg Resp Time",
	"query": "aws_applicationelb_target_response_time_average{load_balancer=\"app/prod-frontend/111\"}"
  },
  "app2_lb_avg_resp_time": {
	"display_name": "Another App LB Avg Resp Time",
	"query": "aws_applicationelb_target_response_time_average{load_balancer=\"app/prod-frontend/222\"}"
  },
  ...
},
```

#### Slack message
Message that is sent on Slack is built from blocks (see Slack docs). In config
file these are `blocks`, `detail_blocks` and `action_blocks`.

`blocks` needs to be defined always and that is part of the message that is
always sent.

In addition to that, Slack message can be extended with "detail blocks" and
"action blocks". Every notification level (see "green", "amber", "red" in
earlier sections) can have different message, meaning you have to define
`blocks`, `detail_blocks`, `action_blocks` for each of the "colors". Also,
every Slack webhook can receive a message with or without `detail_blocks` or
`action_blocks`.

The idea behind such split is that the first required "blocks" contain general
status information, "detail blocks" contain metrics with their values and
thresholds and finally "action blocks" contain buttons (eg. that open Grafana
dashboard). Now, you might want to send all the details to devops channel but
when sending message to other people in the company you can skip the metric
values (details) and additional action buttons/links.

Following `config-sample.json`, this is how `slack_message` section can look
like. Notice that in `detail_blocks`, the `text` value uses Go template with
`.Metrics` which contains list of metrics with its `DisplayName`, `LastValue`,
`Threshold` (for that notification level), `ThresholdExceeded` and `Leverage`
values. Last two are booleans and `ThresholdExceeded` is self-explanatory and
`Leverage` means that the metric leverages to current notification level (but
this happened when `ThresholdExceeded` is true).

```
"slack_message": {
  "blocks": [
	{
	  "type": "section",
	  "text": {
		"type": "mrkdwn",
		"text": ":large_yellow_circle: Warning! Some systems are working slightly slower"
	  }
	}
  ],
  "detail_blocks": [
	{
	  "type": "divider"
	},
	{
	  "type": "section",
	  "text": {
		"type": "mrkdwn",
		"text": "*Threshold exceeded metrics:*\n{{ range .Metrics }}{{ if and .Leverage .ThresholdExceeded }}{{ .DisplayName }} {{ .LastValue }} > *{{ .Threshold }}*\n{{ end }}{{ end }}"
	  }
	},
	{
	  "type": "section",
	  "text": {
		"type": "mrkdwn",
		"text": "*Other leverage metrics:*\n{{ range .Metrics }}{{ if and .Leverage (not .ThresholdExceeded) }}{{ .DisplayName }} {{ .LastValue }} < *{{ .Threshold }}*\n{{ end }}{{ end }}"
	  }
	},
	{
	  "type": "section",
	  "text": {
		"type": "mrkdwn",
		"text": "*Rest of metrics:*\n{{ range .Metrics }}{{ if not .Leverage }}{{ .DisplayName }} {{ .LastValue }}\n{{ end }}{{ end }}"
	  }
	}
  ],
  "action_blocks": [
	{
	  "type": "divider"
	},
	{
	  "type": "section",
	  "text": {
		"type": "mrkdwn",
		"text": "Check Grafana dashboard"
	  },
	  "accessory": {
		"type": "button",
		"text": {
		  "type": "plain_text",
		  "text": "Click Me",
		  "emoji": true
		},
		"value": "prom_uk",
		"url": "https://grafana.example.com",
		"action_id": "button-action"
	  }
	}
  ]
},
```

#### Slack webhooks
Here is an example of webhooks definition. Notice `show_details` and
`show_actions` keys which determine for which notification levels
`detail_blocks` and `action_blocks` should be added.

```
"slack_webhooks": {
  "general": {
	"url": "https://hooks.slack.com/services/hook1",
	"show_details": {
	  "amber": false,
	  "red": false
	},
	"show_actions": {
	  "amber": false,
	  "red": true
	}
  },
  "dev": {
	"url": "https://hooks.slack.com/services/hook2",
	"show_details": {
	  "amber": true,
	  "red": true
	},
	"show_actions": {
	  "amber": true,
	  "red": true
	}
  }
},
```

### Building and running
To compile the code just run `go build -o prometheus-slacker *.go`.

Alternatively, to run straight away: `go run *.go ./config.json`
