package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
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

func triggerMetrics(router chi.Router) {
	router.With(middleware.DateRange("-10minutes", "now")).Get("/", getTriggerMetrics)
	router.Delete("/", deleteTriggerMetric)
}

func deleteTriggerMetric(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	metricName := request.URL.Query().Get("name")
	if metricName == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("metric name can not be empty")))
		return
	}
	if err := controller.DeleteTriggerMetric(database, metricName, triggerID); err != nil {
		render.Render(writer, request, err)
	}
}

func getTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
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

	switch format {
	case JSON:
		triggerMetrics, err := controller.GetTriggerMetricsJSON(database, int64(from), int64(to), triggerID)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		if err := render.Render(writer, request, triggerMetrics); err != nil {
			render.Render(writer, request, api.ErrorRender(err))
		}
		return
	case PNG:
		pic, err := controller.GetTriggerMetricsPNG(database, int64(from), int64(to), triggerID)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(pic)
		return
	default:
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("unknown metric format type")))
		return
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
