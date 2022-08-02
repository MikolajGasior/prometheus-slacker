package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/template"
	"time"
)

type PrometheusSlacker struct {
	config Config
}

func (ps *PrometheusSlacker) GetConfig() *Config {
	return &(ps.config)
}

func (ps *PrometheusSlacker) Init(p string) {
	c, err := ioutil.ReadFile(p)
	if err != nil {
		log.Fatal("Error reading config file")
	}

	var cfg Config
	cfg.SetFromJSON(c)
	ps.config = cfg
}

func (ps *PrometheusSlacker) Run() int {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Syntax: prometheus-slacker <config.json path>\n")
		os.Exit(1)
	}
	ps.Init(os.Args[1])
	done := make(chan bool)
	go ps.startHttpd()
	<-done
	return 0
}

func (ps *PrometheusSlacker) startHttpd() {
	done := make(chan bool)
	go ps.startScrapper()
	go ps.startApi()
	<-done
}

func (ps PrometheusSlacker) getDelay() int {
	delay := ps.config.ScrapperMinutes
	if delay < 1 {
		delay = 1
	}
	return delay
}

func (ps PrometheusSlacker) sleep() {
	log.Print(fmt.Sprintf("Sleeping %d minutes...", ps.getDelay()))
	time.Sleep(time.Minute * time.Duration(ps.getDelay()))
}

func (ps PrometheusSlacker) sendMsg(webhook SlackWebhook, msg SlackMessage) {
	err := webhook.SendMessage(msg)
	if err != nil {
		log.Print("Error sending slack msg")
	}
}

func (ps *PrometheusSlacker) getMetricValueAndCompareWithThreshold(metric string, threshold string) (Metric, bool, error) {
	levelMetric := ps.config.Metrics[metric]
	query := ps.config.Metrics[metric].Query

	c, err := ps.GetMetricValue(query)
	if err != nil {
		levelMetric.LastValue = "-1"
		log.Print(fmt.Sprintf("Error getting metric value for %s", query))
		return levelMetric, false, err
	}

	levelMetric.LastValue = c
	levelMetric.Threshold = threshold

	leverage, err := ps.IsValueBiggerThanThreshold(c, threshold)
	if err != nil {
		log.Print(err.Error())
		return levelMetric, false, err
	}

	return levelMetric, leverage, nil
}

func (ps *PrometheusSlacker) getMetricValues() map[string]Metric {
	v := make(map[string]Metric)
	for name, metric := range ps.config.Metrics {
		newMetric := metric
		c, err := ps.GetMetricValue(newMetric.Query)
		if err == nil {
			newMetric.LastValue = c
		} else {
			newMetric.LastValue = ""
		}
		v[name] = newMetric
	}
	return v
}

func (ps *PrometheusSlacker) getCurrentLevelAndMetrics(metricsWithValues map[string]Metric) (int, map[int]map[string]Metric) {
	currentLevel := -1
	levelMetrics := make(map[int]map[string]Metric)
	for i, notificationLevel := range ps.config.NotificationLevels {
		levelMetrics[i] = make(map[string]Metric)
		metricThresholds := make(map[string]string)
		thresholdExceededMetrics := make(map[string]bool)
		leverageMetrics := make(map[string]bool)

		if i == 0 {
			currentLevel = i
		}

		for metric, threshold := range notificationLevel.LeverageMetrics {
			metricThresholds[metric] = threshold
			leverageMetrics[metric] = true
			leverage, err := ps.IsValueBiggerThanThreshold(metricsWithValues[metric].LastValue, threshold)
			if err != nil {
				log.Print(fmt.Errorf("Error with comparing values: %w", err.Error()))
				continue
			}
			if leverage {
				thresholdExceededMetrics[metric] = true
				// log.Print(fmt.Sprintf("%f <= %f so changing color to %s", threshold, metricsWithValues[metric].LastValue, color))
				currentLevel = i
			}
		}

		for name, metric := range metricsWithValues {
			newMetric := metric
			if leverageMetrics[name] == true {
				newMetric.Leverage = true
			}
			if metricThresholds[name] != "" {
				newMetric.Threshold = metricThresholds[name]
			}
			if thresholdExceededMetrics[name] == true {
				newMetric.ThresholdExceeded = true
			}
			levelMetrics[i][name] = newMetric
		}

		// log.Print(fmt.Sprintf("Current color is: %s\n", currentLevel))
	}
	return currentLevel, levelMetrics
}

func (ps *PrometheusSlacker) newSlackMessage(srcMsg SlackMessage) SlackMessage {
	msg := SlackMessage{}
	b, err := json.Marshal(srcMsg)
	if err != nil {
		log.Print(fmt.Errorf("Error with Marshal slack msg from config: %w", err.Error()))
		return msg
	}

	_ = json.Unmarshal(b, &msg)
	if err != nil {
		log.Print(fmt.Errorf("Error with Unmarshal slack msg from config: %w", err.Error()))
	}

	return msg
}

func (ps *PrometheusSlacker) getWebhookAndMsgForNotificationLevelSlackWebhooks(notificationLevel NotificationLevel, w string, levelMetrics map[string]Metric) (SlackWebhook, SlackMessage) {
	webhook := ps.config.SlackWebhooks[w]
	msg := ps.newSlackMessage(notificationLevel.SlackMessage)
	if webhook.ShowDetails[notificationLevel.Color] == true {
		metrics := make([]Metric, 0)
		if len(levelMetrics) > 0 {
			for _, metric := range levelMetrics {
				metrics = append(metrics, metric)
			}
		}

		for _, block := range msg.DetailBlocks {
			blockCopy := block
			if blockCopy.Type == "section" && blockCopy.Text.Type == "mrkdwn" {
				tmpl, err := template.New("metrics").Parse(blockCopy.Text.Text)
				var tpl bytes.Buffer
				if err == nil {
					err = tmpl.Execute(&tpl, struct {
						Metrics *([]Metric)
					}{
						Metrics: &metrics,
					})
					if err == nil {
						blockCopy.Text.Text = tpl.String()
					}
				}
			}
			msg.Blocks = append(msg.Blocks, blockCopy)
		}
	}
	if webhook.ShowActions[notificationLevel.Color] == true {
		for _, block := range msg.ActionBlocks {
			blockCopy := block
			msg.Blocks = append(msg.Blocks, blockCopy)
		}
	}
	return webhook, msg
}

func (ps *PrometheusSlacker) scrap() {
	metricsWithValues := ps.getMetricValues()
	if metricsWithValues == nil || len(metricsWithValues) == 0 {
		return
	}

	currentLevel, levelMetrics := ps.getCurrentLevelAndMetrics(metricsWithValues)

	if currentLevel < 0 {
		return
	}

	for i, notificationLevel := range ps.config.NotificationLevels {
		if i != currentLevel {
			continue
		}

		for _, w := range notificationLevel.SlackWebhooks {
			webhook, msg := ps.getWebhookAndMsgForNotificationLevelSlackWebhooks(notificationLevel, w, levelMetrics[i])
			ps.sendMsg(webhook, msg)
		}
	}
}

func (ps *PrometheusSlacker) startScrapper() {
	for {
		ps.scrap()
		ps.sleep()
		continue
	}
}

func (ps *PrometheusSlacker) startApi() {
	router := mux.NewRouter()
	router.HandleFunc("/scrap", ps.getScrapHandler()).Methods("POST")
	log.Print("Starting daemon listening on " + ps.config.Port + "...")
	log.Fatal(http.ListenAndServe(":"+ps.config.Port, router))
}

func (ps PrometheusSlacker) GetMetricValue(metric string) (string, error) {
	res, err := http.Get(ps.config.PrometheusURL + "/api/v1/query?query=" + url.QueryEscape(metric))
	if err != nil {
		return "", err
	}
	c, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	var j map[string]interface{}
	err = json.Unmarshal(c, &j)
	if err != nil {
		log.Print("Error unmarshalling metric response")
		return "", nil
	}
	// data.result.0.value.1
	data := j["data"].(map[string]interface{})
	result := data["result"].([]interface{})
	value := result[0].(map[string]interface{})["value"].([]interface{})
	currentValStr := value[1].(string)
	// log.Print("Metric " + metric + " value " + currentValStr)

	return currentValStr, nil
}

func (ps *PrometheusSlacker) getScrapHandler() func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ps.scrap()

		w.WriteHeader(http.StatusOK)
		return
	}
	return http.HandlerFunc(fn)
}

func (ps PrometheusSlacker) IsValueBiggerThanThreshold(val string, threshold string) (bool, error) {
	currentVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Print("Error getting float from string for current val")
	}

	compareVal, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		log.Print("Error getting float from string for threshold val")
	}

	// log.Print(fmt.Sprintf("Comparing %f and %f ...", compareVal, currentVal))
	if compareVal <= currentVal {
		return true, nil
	}

	return false, nil
}

func NewPrometheusSlacker() *PrometheusSlacker {
	ps := &PrometheusSlacker{}
	return ps
}
