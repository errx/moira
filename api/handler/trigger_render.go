package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/checker"
)

type metricFormat int

// Supported metrics format types
const (
	PNG metricFormat = iota
	PNGr
	JSON
)

var (
	valueRaisingThresholdTemplate = []string{
		"alpha(areaBetween(lineWidth(group(threshold(%.f, color='red'),threshold(%.f, color='red')),2)),0.2)",
		"alpha(areaBetween(lineWidth(group(threshold(%.f, color='yellow'),threshold(%.f, color='yellow')),2)),0.2)",
		}
	valueFallingThresholdTemplate = []string{
		"alpha(lineWidth(threshold(%.f, color='red'),2),0.2)",
		"alpha(lineWidth(threshold(%.f, color='yellow'),2),0.2)",
		}
)

func renderTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	rawFrom := middleware.GetFromStr(request)
	rawTo := middleware.GetToStr(request)
	from := date.DateParamToEpoch(rawFrom, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse from: %s", rawFrom)))
		return
	}
	to := date.DateParamToEpoch(rawTo, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse to: %v", rawTo)))
		return
	}
	format, err := getMetricFormat(request)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	tts, trigger, err := controller.GetTriggerEvaluationResult(database, int64(from), int64(to), triggerID)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}

	var metricsData = make([]*expr.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}

	switch format {
	case JSON:
		json := expr.MarshalJSON(metricsData)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(json)
	case PNG:
		thresholdData, err := getThresholdData(trigger, tts)
		if err != nil {
			render.Render(writer, request, api.ErrorRender(err))
			return
		}
		for _, th := range thresholdData {
			metricsData = append(metricsData, th...)
		}
		params := getPictureParams()
		params.Title = trigger.Name
		png := expr.MarshalPNG(params, metricsData)
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(png)
	case PNGr:
		png := expr.MarshalPNGRequest(request, metricsData)
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(png)
	default:
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("inexpected metrics format")))
	}
}

func getMetricFormat(request *http.Request) (metricFormat, error) {
	format := request.URL.Query().Get("format")
	if format == "" {
		return JSON, nil
	}
	switch format {
	case "json":
		return JSON, nil
	case "png":
		return PNG, nil
	case "pngr":
		return PNGr, nil
	default:
		return JSON, fmt.Errorf("invalid format type: %s", format)
	}
}

func getPictureParams() expr.PictureParams {
	params := expr.DefaultParams
	params.Width = 586
	params.Height = 380
	params.LeftWidth = 2
	params.BgColor = "1f1d1d"
	params.MinorGridLineColor = "1f1d1d"
	params.MajorGridLineColor = "grey"
	params.AreaAlpha = 0.2
	params.AreaMode = expr.AreaModeAll
	return params
}

func getThresholdData(trigger *moira.Trigger, timeSeries *checker.TriggerTimeSeries) ([][]*expr.MetricData, error) {
	metricsMap := make(map[expr.MetricRequest][]*expr.MetricData)
	thresholdSeriesList := make([][]*expr.MetricData, 2)

	startTime := timeSeries.Main[0].StartTime
	stopTime := timeSeries.Main[0].StopTime
	limitValue := getLimitValue(timeSeries)
	thresholdExpressions := getThresholdExpressions(*trigger.WarnValue, *trigger.ErrorValue, limitValue)

	for _, thresholdExpression := range thresholdExpressions {
		parsedThreshold, _, _ := expr.ParseExpr(thresholdExpression)
		thresholdSeries, err := expr.EvalExpr(parsedThreshold, startTime, stopTime, metricsMap)
		if err != nil {
			return nil, fmt.Errorf("can't evaluate thresholds for trigger %s: %s", trigger.ID, err.Error())
		}
		thresholdSeriesList = append(thresholdSeriesList, thresholdSeries)
	}
	return thresholdSeriesList, nil
}

func getThresholdExpressions(warnValue float64, errorValue float64, limitValue float64) []string {
	var thresholdTemplate []string
	switch {
	case warnValue < errorValue:
		thresholdTemplate = valueRaisingThresholdTemplate
		return []string{
			fmt.Sprintf(thresholdTemplate[0], errorValue, limitValue),
			fmt.Sprintf(thresholdTemplate[1], warnValue, errorValue),
		}
	default:
		thresholdTemplate = valueFallingThresholdTemplate
		return []string{
			fmt.Sprintf(thresholdTemplate[0], errorValue),
			fmt.Sprintf(thresholdTemplate[1], warnValue),
		}
	}
}

func getLimitValue(timeSeries *checker.TriggerTimeSeries) float64 {
	var metricsData *expr.MetricData
	var LimitValue float64
	for _, ts := range timeSeries.Main{
		metricsData = &ts.MetricData
		for _, val := range metricsData.Values{
			if val > LimitValue{
				LimitValue = val
			}
		}
	}
	return LimitValue
}
