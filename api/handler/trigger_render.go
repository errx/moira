package handler

import (
	"fmt"
	"image/color"
	"math"
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
	PNGr
	JSON
)

//TODO дефолт в carbon api
//TODO избавиться от RGBA
//TODO публичный GetPictureParams
var def = expr.PictureParams{
	Width:      330,
	Height:     250,
	Margin:     10,
	LogBase:    0,
	FgColor:    color.RGBA{0xff, 0xff, 0xff, 0xff},
	BgColor:    color.RGBA{0x00, 0x00, 0x00, 0xff},
	MajorLine:  color.RGBA{0xc8, 0x96, 0xc8, 0xff},
	MinorLine:  color.RGBA{0xaf, 0xaf, 0xaf, 0xff},
	FontName:   "Sans",
	FontSize:   10,
	FontBold:   expr.FontWeightNormal,
	FontItalic: expr.FontSlantNormal,

	GraphOnly:  false,
	HideLegend: false,
	HideGrid:   false,
	HideAxes:   false,
	HideYAxis:  false,
	HideXAxis:  false,
	YAxisSide:  expr.YAxisSideLeft,

	Title:       "",
	Vtitle:      "",
	VtitleRight: "",

	Tz: time.Local,

	ConnectedLimit: math.MaxUint32,
	LineMode:       expr.LineModeSlope,
	AreaMode:       expr.AreaModeNone,
	AreaAlpha:      math.NaN(),
	PieMode:        expr.PieModeAverage,
	LineWidth:      1.2,
	ColorList:      []string{"blue", "green", "red", "purple", "brown", "yellow", "aqua", "grey", "magenta", "pink", "gold", "rose"}, //TODO там есть переменная

	YMin:    math.NaN(),
	YMax:    math.NaN(),
	YStep:   math.NaN(),
	XMin:    math.NaN(),
	XMax:    math.NaN(),
	XStep:   math.NaN(),
	XFormat: "",
	MinorY:  1,

	UniqueLegend:   false,
	DrawNullAsZero: false,
	DrawAsInfinite: false,

	YMinLeft:    math.NaN(),
	YMinRight:   math.NaN(),
	YMaxLeft:    math.NaN(),
	YMaxRight:   math.NaN(),
	YStepL:      math.NaN(),
	YStepR:      math.NaN(),
	YLimitLeft:  math.NaN(),
	YLimitRight: math.NaN(),

	YUnitSystem: "si",
	YDivisors:   []float64{4, 5, 6},

	RightWidth:  1.2,
	RightDashed: false,
	RightColor:  "",
	LeftWidth:   1.2,
	LeftDashed:  false,
	LeftColor:   "",

	MajorGridLineColor: "white",
	MinorGridLineColor: "grey",
}

func getPictureParams() expr.PictureParams {
	p := def
	p.Width = 586
	p.Height = 380
	p.LeftWidth = 2
	p.BgColor = color.RGBA{R: 0x1f, G: 0x1d, B: 0x1d, A: byte(255)}
	p.MinorGridLineColor = "1f1d1d"
	p.MajorGridLineColor = "grey"
	return p
}

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
