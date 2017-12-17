package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

type metricFormat int

// Supported metrics format types
const (
	PNG metricFormat = iota
	JSON
)

func renderTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse from: %s", fromStr)))
		return
	}
	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse to: %v", toStr)))
		return
	}
	format, err := getMetricFormat(request)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	tts, err := controller.GetTriggerEvaluationResult(database, int64(from), int64(to), triggerID)
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
		png := expr.MarshalPNG(request, metricsData)
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
	default:
		return JSON, fmt.Errorf("invalid format type: %s", format)
	}
}

//// GetTriggerMetricsPNG gets all trigger metrics values, default values from: now - 10min, to: now
//func GetTriggerMetricsPNG(dataBase moira.Database, from, to int64, triggerID string) ([]byte, *api.ErrorResponse) {
//	trigger, err := dataBase.GetTrigger(triggerID)
//	if err != nil {
//		if err == database.ErrNil {
//			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger not found"))
//		}
//		return nil, api.ErrorInternalServer(err)
//	}
//
//	isSimpleTrigger := trigger.IsSimple()
//	for _, tar := range trigger.Targets {
//		result, err := target.EvaluateTarget(dataBase, tar, from, to, isSimpleTrigger)
//		if err != nil {
//			return nil, api.ErrorInternalServer(err)
//		}
//
//		var metricsData = make([]*expr.MetricData, 0, len(result.TimeSeries))
//		for _, ts := range result.TimeSeries {
//			metricsData = append(metricsData, &ts.MetricData)
//		}
//		return expr.MarshalPNG(&http.Request{}, metricsData), nil
//	}
//	return nil, nil
//}
