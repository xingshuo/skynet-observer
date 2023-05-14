package main

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type ChartData struct {
	serieData []*SerieData
	XAxis     []string
}

type SerieData struct {
	name  string
	YAxis []opts.LineData
}

func NewChartLine(title, subTitle string, chartData *ChartData) *charts.Line {
	// create a new line instance
	line := charts.NewLine()
	selected := make(map[string]bool)
	// Put data into instance
	line.SetXAxis(chartData.XAxis)
	for _, d := range chartData.serieData {
		selected[d.name] = true
		line.AddSeries(d.name, d.YAxis)
	}
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subTitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true, Orient: "vertical", X: "right", Y: "left", Selected: selected, SelectedMode: "multiple",
		}),
	)
	line.SetSeriesOptions(
		charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
	)

	return line
}
