package protocols

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
)

// Datagram protocol
// <METRIC_NAME>:<VALUE>|<TYPE>|@<SAMPLE_RATE>|#<TAG_KEY_1>:<TAG_VALUE_1>,<TAG_2>

// Parser logic
// lexes := splitBy("|")
// <metric_details> = lexes[0]
// <type> = lexes[1]
// rest of split, lexes[i]
// startsWith('@') = sample rate
// startsWith('#') = tags
// parse each tag = splitBy(",")
// [key, value] = splitBy(":")

type MetricType int

const (
	MetricTypeNone MetricType = iota
	MetricTypeCounter
	MetricTypeGauge
	MetricTypeHistogram

	MetricInvalid
)

const (
	_counterMetric   = "c"
	_gaugeMetric     = "g"
	_histogramMetric = "h"
)

var _metricTypeMap = []string{"", _counterMetric, _gaugeMetric, _histogramMetric}

func Is(got string, expected MetricType) bool {
	if expected >= MetricInvalid {
		log.Println(expected)
		return false
	}

	val := _metricTypeMap[expected]
	return val == got
}

type MetricTag map[string]string

type ExtraData struct {
	SampleRate float64
	TagList    MetricTag
}

type Metric struct {
	Name  string
	Value float32
	Type  MetricType
	ExtraData
}

func (m Metric) MetricType() string {
	switch m.Type {
	case MetricTypeCounter:
		return _counterMetric
	case MetricTypeGauge:
		return _gaugeMetric
	case MetricTypeHistogram:
		return _histogramMetric
	default:
		return ""
	}
}

var ErrInvalidDatagramProtocol = errors.New("invalid_datagram")

func ParseDatagram(ctx context.Context, value string) (m Metric, err error) {
	grams := strings.Split(value, "|")
	if len(grams) < 2 {
		err = ErrInvalidDatagramProtocol
		return
	}

	m, err = parseMetricInfo(ctx, grams[0])
	if err != nil {
		return
	}
	mType, err := parseMetricType(ctx, grams[1])
	if err != nil {
		return m, err
	}

	m.Type = mType

	hasMoreData := len(grams) > 2
	if hasMoreData {
		m.ExtraData = parseExtra(ctx, grams...)
	}

	return
}

var (
	ErrMalformedMetricKey     = errors.New("malformed_metric_details")
	ErrInvalidMetricValueType = errors.New("invalid_type_for_metric_value")
)

func parseMetricInfo(ctx context.Context, metricInfo string) (m Metric, err error) {
	splits := strings.Split(metricInfo, ":")
	if len(splits) != 2 {
		err = ErrMalformedMetricKey
		return
	}

	metricVal, err := strconv.ParseFloat(splits[1], 32)
	if err != nil {
		err = ErrInvalidMetricValueType
		return
	}

	m = Metric{Name: splits[0], Value: float32(metricVal)}
	return
}

var (
	ErrUnsupportedMetricType = errors.New("unsupported_metric_type")
)

func parseMetricType(ctx context.Context, mType string) (MetricType, error) {
	switch mType {
	case _counterMetric:
		return MetricTypeCounter, nil
	case _gaugeMetric:
		return MetricTypeGauge, nil
	case _histogramMetric:
		return MetricTypeHistogram, nil
	}

	return MetricTypeNone, ErrUnsupportedMetricType
}

func parseExtra(ctx context.Context, rest ...string) ExtraData {
	ex := ExtraData{}

	for _, gram := range rest {
		if strings.HasPrefix(gram, "@") {
			sampleRate, err := strconv.ParseFloat(strings.TrimPrefix(gram, "@"), 64)
			if err == nil {
				ex.SampleRate = sampleRate
			}

			continue
		}

		if strings.HasPrefix(gram, "#") {
			tagStr := strings.TrimPrefix(gram, "#")
			tagMap := map[string]string{}

			for _, tag := range strings.Split(tagStr, ",") {
				kvp := strings.Split(tag, ":")
				if len(kvp) == 2 {
					tagMap[kvp[0]] = kvp[1]
				}
			}

			ex.TagList = tagMap
		}
	}

	return ex
}
