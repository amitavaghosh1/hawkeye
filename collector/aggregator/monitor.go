package aggregator

import (
	"fmt"
	"hawkeye/utils"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Trigger struct {
	Threshold      float32  `yaml:"threshold"`
	Text           *string  `yaml:"text,omitempty"`
	To             []string `yaml:"to"`
	RunEveryMinute int64    `yaml:"run_every"`
	Subject        string  `yaml:"subject"`
}

type Monitor struct {
	Metric            string    `yaml:"metric"`
	Type              string    `yaml:"type"`
	IntervalInSeconds int64     `yaml:"interval"`
	Notifier          string    `yaml:"notifier"`
	Triggers          []Trigger `yaml:"triggers"`
	Subject           *string   `yaml:"subject"`
}

type MonitorConfig struct {
	Version  int       `yaml:"version"`
	Monitors []Monitor `yaml:"monitors"`
}

func ReadMonitoringConfig(configFile string, serviceName string) []Monitor {
	b, err := ioutil.ReadFile(configFile)
	utils.CheckErr(err, "failed to read config file")

	monitorCfg := MonitorConfig{}

	err = yaml.Unmarshal(b, &monitorCfg)
	utils.CheckErr(err, "failed to unmarshal monitors config")

	monitors := []Monitor{}

	for _, monitor := range monitorCfg.Monitors {
		for _, trigger := range monitor.Triggers {
			if trigger.Subject == "" {
				trigger.Subject = monitor.GetSubject(serviceName)
			}
		}

		monitors = append(monitors, monitor)
	}

	return monitors
}

func (m Monitor) GetSubject(app string) string {
	subject := fmt.Sprintf("%s error limit exceeded in %s", m.Metric, app)
	if m.Subject != nil {
		subject = *m.Subject
	}

	return subject
}
