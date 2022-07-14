// Copyright 2016 - 2022 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to and
// read from XLAM / XLSM / XLSX / XLTM / XLTX files. Supports reading and
// writing spreadsheet documents generated by Microsoft Excel™ 2007 and later.
// Supports complex components by high compatibility, and provided streaming
// API for generating or reading data from a worksheet with huge amounts of
// data. This library needs Go version 1.15 or later.

package excelize

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// This section defines the currently supported chart types.
const (
	Area                        = "area"
	AreaStacked                 = "areaStacked"
	AreaPercentStacked          = "areaPercentStacked"
	Area3D                      = "area3D"
	Area3DStacked               = "area3DStacked"
	Area3DPercentStacked        = "area3DPercentStacked"
	Bar                         = "bar"
	BarStacked                  = "barStacked"
	BarPercentStacked           = "barPercentStacked"
	Bar3DClustered              = "bar3DClustered"
	Bar3DStacked                = "bar3DStacked"
	Bar3DPercentStacked         = "bar3DPercentStacked"
	Bar3DConeClustered          = "bar3DConeClustered"
	Bar3DConeStacked            = "bar3DConeStacked"
	Bar3DConePercentStacked     = "bar3DConePercentStacked"
	Bar3DPyramidClustered       = "bar3DPyramidClustered"
	Bar3DPyramidStacked         = "bar3DPyramidStacked"
	Bar3DPyramidPercentStacked  = "bar3DPyramidPercentStacked"
	Bar3DCylinderClustered      = "bar3DCylinderClustered"
	Bar3DCylinderStacked        = "bar3DCylinderStacked"
	Bar3DCylinderPercentStacked = "bar3DCylinderPercentStacked"
	Col                         = "col"
	ColStacked                  = "colStacked"
	ColPercentStacked           = "colPercentStacked"
	Col3D                       = "col3D"
	Col3DClustered              = "col3DClustered"
	Col3DStacked                = "col3DStacked"
	Col3DPercentStacked         = "col3DPercentStacked"
	Col3DCone                   = "col3DCone"
	Col3DConeClustered          = "col3DConeClustered"
	Col3DConeStacked            = "col3DConeStacked"
	Col3DConePercentStacked     = "col3DConePercentStacked"
	Col3DPyramid                = "col3DPyramid"
	Col3DPyramidClustered       = "col3DPyramidClustered"
	Col3DPyramidStacked         = "col3DPyramidStacked"
	Col3DPyramidPercentStacked  = "col3DPyramidPercentStacked"
	Col3DCylinder               = "col3DCylinder"
	Col3DCylinderClustered      = "col3DCylinderClustered"
	Col3DCylinderStacked        = "col3DCylinderStacked"
	Col3DCylinderPercentStacked = "col3DCylinderPercentStacked"
	Doughnut                    = "doughnut"
	Line                        = "line"
	Pie                         = "pie"
	Pie3D                       = "pie3D"
	PieOfPieChart               = "pieOfPie"
	BarOfPieChart               = "barOfPie"
	Radar                       = "radar"
	Scatter                     = "scatter"
	Surface3D                   = "surface3D"
	WireframeSurface3D          = "wireframeSurface3D"
	Contour                     = "contour"
	WireframeContour            = "wireframeContour"
	Bubble                      = "bubble"
	Bubble3D                    = "bubble3D"
)

// This section defines the default value of chart properties.
var (
	chartView3DRotX = map[string]int{
		Area:                        0,
		AreaStacked:                 0,
		AreaPercentStacked:          0,
		Area3D:                      15,
		Area3DStacked:               15,
		Area3DPercentStacked:        15,
		Bar:                         0,
		BarStacked:                  0,
		BarPercentStacked:           0,
		Bar3DClustered:              15,
		Bar3DStacked:                15,
		Bar3DPercentStacked:         15,
		Bar3DConeClustered:          15,
		Bar3DConeStacked:            15,
		Bar3DConePercentStacked:     15,
		Bar3DPyramidClustered:       15,
		Bar3DPyramidStacked:         15,
		Bar3DPyramidPercentStacked:  15,
		Bar3DCylinderClustered:      15,
		Bar3DCylinderStacked:        15,
		Bar3DCylinderPercentStacked: 15,
		Col:                         0,
		ColStacked:                  0,
		ColPercentStacked:           0,
		Col3D:                       15,
		Col3DClustered:              15,
		Col3DStacked:                15,
		Col3DPercentStacked:         15,
		Col3DCone:                   15,
		Col3DConeClustered:          15,
		Col3DConeStacked:            15,
		Col3DConePercentStacked:     15,
		Col3DPyramid:                15,
		Col3DPyramidClustered:       15,
		Col3DPyramidStacked:         15,
		Col3DPyramidPercentStacked:  15,
		Col3DCylinder:               15,
		Col3DCylinderClustered:      15,
		Col3DCylinderStacked:        15,
		Col3DCylinderPercentStacked: 15,
		Doughnut:                    0,
		Line:                        0,
		Pie:                         0,
		Pie3D:                       30,
		PieOfPieChart:               0,
		BarOfPieChart:               0,
		Radar:                       0,
		Scatter:                     0,
		Surface3D:                   15,
		WireframeSurface3D:          15,
		Contour:                     90,
		WireframeContour:            90,
	}
	chartView3DRotY = map[string]int{
		Area:                        0,
		AreaStacked:                 0,
		AreaPercentStacked:          0,
		Area3D:                      20,
		Area3DStacked:               20,
		Area3DPercentStacked:        20,
		Bar:                         0,
		BarStacked:                  0,
		BarPercentStacked:           0,
		Bar3DClustered:              20,
		Bar3DStacked:                20,
		Bar3DPercentStacked:         20,
		Bar3DConeClustered:          20,
		Bar3DConeStacked:            20,
		Bar3DConePercentStacked:     20,
		Bar3DPyramidClustered:       20,
		Bar3DPyramidStacked:         20,
		Bar3DPyramidPercentStacked:  20,
		Bar3DCylinderClustered:      20,
		Bar3DCylinderStacked:        20,
		Bar3DCylinderPercentStacked: 20,
		Col:                         0,
		ColStacked:                  0,
		ColPercentStacked:           0,
		Col3D:                       20,
		Col3DClustered:              20,
		Col3DStacked:                20,
		Col3DPercentStacked:         20,
		Col3DCone:                   20,
		Col3DConeClustered:          20,
		Col3DConeStacked:            20,
		Col3DConePercentStacked:     20,
		Col3DPyramid:                20,
		Col3DPyramidClustered:       20,
		Col3DPyramidStacked:         20,
		Col3DPyramidPercentStacked:  20,
		Col3DCylinder:               20,
		Col3DCylinderClustered:      20,
		Col3DCylinderStacked:        20,
		Col3DCylinderPercentStacked: 20,
		Doughnut:                    0,
		Line:                        0,
		Pie:                         0,
		Pie3D:                       0,
		PieOfPieChart:               0,
		BarOfPieChart:               0,
		Radar:                       0,
		Scatter:                     0,
		Surface3D:                   20,
		WireframeSurface3D:          20,
		Contour:                     0,
		WireframeContour:            0,
	}
	plotAreaChartOverlap = map[string]int{
		BarStacked:        100,
		BarPercentStacked: 100,
		ColStacked:        100,
		ColPercentStacked: 100,
	}
	chartView3DPerspective = map[string]int{
		Contour:          0,
		WireframeContour: 0,
	}
	chartView3DRAngAx = map[string]int{
		Area:                        0,
		AreaStacked:                 0,
		AreaPercentStacked:          0,
		Area3D:                      1,
		Area3DStacked:               1,
		Area3DPercentStacked:        1,
		Bar:                         0,
		BarStacked:                  0,
		BarPercentStacked:           0,
		Bar3DClustered:              1,
		Bar3DStacked:                1,
		Bar3DPercentStacked:         1,
		Bar3DConeClustered:          1,
		Bar3DConeStacked:            1,
		Bar3DConePercentStacked:     1,
		Bar3DPyramidClustered:       1,
		Bar3DPyramidStacked:         1,
		Bar3DPyramidPercentStacked:  1,
		Bar3DCylinderClustered:      1,
		Bar3DCylinderStacked:        1,
		Bar3DCylinderPercentStacked: 1,
		Col:                         0,
		ColStacked:                  0,
		ColPercentStacked:           0,
		Col3D:                       1,
		Col3DClustered:              1,
		Col3DStacked:                1,
		Col3DPercentStacked:         1,
		Col3DCone:                   1,
		Col3DConeClustered:          1,
		Col3DConeStacked:            1,
		Col3DConePercentStacked:     1,
		Col3DPyramid:                1,
		Col3DPyramidClustered:       1,
		Col3DPyramidStacked:         1,
		Col3DPyramidPercentStacked:  1,
		Col3DCylinder:               1,
		Col3DCylinderClustered:      1,
		Col3DCylinderStacked:        1,
		Col3DCylinderPercentStacked: 1,
		Doughnut:                    0,
		Line:                        0,
		Pie:                         0,
		Pie3D:                       0,
		PieOfPieChart:               0,
		BarOfPieChart:               0,
		Radar:                       0,
		Scatter:                     0,
		Surface3D:                   0,
		WireframeSurface3D:          0,
		Contour:                     0,
		Bubble:                      0,
		Bubble3D:                    0,
	}
	chartLegendPosition = map[string]string{
		"bottom":    "b",
		"left":      "l",
		"right":     "r",
		"top":       "t",
		"top_right": "tr",
	}
	chartValAxNumFmtFormatCode = map[string]string{
		Area:                        "General",
		AreaStacked:                 "General",
		AreaPercentStacked:          "0%",
		Area3D:                      "General",
		Area3DStacked:               "General",
		Area3DPercentStacked:        "0%",
		Bar:                         "General",
		BarStacked:                  "General",
		BarPercentStacked:           "0%",
		Bar3DClustered:              "General",
		Bar3DStacked:                "General",
		Bar3DPercentStacked:         "0%",
		Bar3DConeClustered:          "General",
		Bar3DConeStacked:            "General",
		Bar3DConePercentStacked:     "0%",
		Bar3DPyramidClustered:       "General",
		Bar3DPyramidStacked:         "General",
		Bar3DPyramidPercentStacked:  "0%",
		Bar3DCylinderClustered:      "General",
		Bar3DCylinderStacked:        "General",
		Bar3DCylinderPercentStacked: "0%",
		Col:                         "General",
		ColStacked:                  "General",
		ColPercentStacked:           "0%",
		Col3D:                       "General",
		Col3DClustered:              "General",
		Col3DStacked:                "General",
		Col3DPercentStacked:         "0%",
		Col3DCone:                   "General",
		Col3DConeClustered:          "General",
		Col3DConeStacked:            "General",
		Col3DConePercentStacked:     "0%",
		Col3DPyramid:                "General",
		Col3DPyramidClustered:       "General",
		Col3DPyramidStacked:         "General",
		Col3DPyramidPercentStacked:  "0%",
		Col3DCylinder:               "General",
		Col3DCylinderClustered:      "General",
		Col3DCylinderStacked:        "General",
		Col3DCylinderPercentStacked: "0%",
		Doughnut:                    "General",
		Line:                        "General",
		Pie:                         "General",
		Pie3D:                       "General",
		PieOfPieChart:               "General",
		BarOfPieChart:               "General",
		Radar:                       "General",
		Scatter:                     "General",
		Surface3D:                   "General",
		WireframeSurface3D:          "General",
		Contour:                     "General",
		WireframeContour:            "General",
		Bubble:                      "General",
		Bubble3D:                    "General",
	}
	chartValAxCrossBetween = map[string]string{
		Area:                        "midCat",
		AreaStacked:                 "midCat",
		AreaPercentStacked:          "midCat",
		Area3D:                      "midCat",
		Area3DStacked:               "midCat",
		Area3DPercentStacked:        "midCat",
		Bar:                         "between",
		BarStacked:                  "between",
		BarPercentStacked:           "between",
		Bar3DClustered:              "between",
		Bar3DStacked:                "between",
		Bar3DPercentStacked:         "between",
		Bar3DConeClustered:          "between",
		Bar3DConeStacked:            "between",
		Bar3DConePercentStacked:     "between",
		Bar3DPyramidClustered:       "between",
		Bar3DPyramidStacked:         "between",
		Bar3DPyramidPercentStacked:  "between",
		Bar3DCylinderClustered:      "between",
		Bar3DCylinderStacked:        "between",
		Bar3DCylinderPercentStacked: "between",
		Col:                         "between",
		ColStacked:                  "between",
		ColPercentStacked:           "between",
		Col3D:                       "between",
		Col3DClustered:              "between",
		Col3DStacked:                "between",
		Col3DPercentStacked:         "between",
		Col3DCone:                   "between",
		Col3DConeClustered:          "between",
		Col3DConeStacked:            "between",
		Col3DConePercentStacked:     "between",
		Col3DPyramid:                "between",
		Col3DPyramidClustered:       "between",
		Col3DPyramidStacked:         "between",
		Col3DPyramidPercentStacked:  "between",
		Col3DCylinder:               "between",
		Col3DCylinderClustered:      "between",
		Col3DCylinderStacked:        "between",
		Col3DCylinderPercentStacked: "between",
		Doughnut:                    "between",
		Line:                        "between",
		Pie:                         "between",
		Pie3D:                       "between",
		PieOfPieChart:               "between",
		BarOfPieChart:               "between",
		Radar:                       "between",
		Scatter:                     "between",
		Surface3D:                   "midCat",
		WireframeSurface3D:          "midCat",
		Contour:                     "midCat",
		WireframeContour:            "midCat",
		Bubble:                      "midCat",
		Bubble3D:                    "midCat",
	}
	plotAreaChartGrouping = map[string]string{
		Area:                        "standard",
		AreaStacked:                 "stacked",
		AreaPercentStacked:          "percentStacked",
		Area3D:                      "standard",
		Area3DStacked:               "stacked",
		Area3DPercentStacked:        "percentStacked",
		Bar:                         "clustered",
		BarStacked:                  "stacked",
		BarPercentStacked:           "percentStacked",
		Bar3DClustered:              "clustered",
		Bar3DStacked:                "stacked",
		Bar3DPercentStacked:         "percentStacked",
		Bar3DConeClustered:          "clustered",
		Bar3DConeStacked:            "stacked",
		Bar3DConePercentStacked:     "percentStacked",
		Bar3DPyramidClustered:       "clustered",
		Bar3DPyramidStacked:         "stacked",
		Bar3DPyramidPercentStacked:  "percentStacked",
		Bar3DCylinderClustered:      "clustered",
		Bar3DCylinderStacked:        "stacked",
		Bar3DCylinderPercentStacked: "percentStacked",
		Col:                         "clustered",
		ColStacked:                  "stacked",
		ColPercentStacked:           "percentStacked",
		Col3D:                       "standard",
		Col3DClustered:              "clustered",
		Col3DStacked:                "stacked",
		Col3DPercentStacked:         "percentStacked",
		Col3DCone:                   "standard",
		Col3DConeClustered:          "clustered",
		Col3DConeStacked:            "stacked",
		Col3DConePercentStacked:     "percentStacked",
		Col3DPyramid:                "standard",
		Col3DPyramidClustered:       "clustered",
		Col3DPyramidStacked:         "stacked",
		Col3DPyramidPercentStacked:  "percentStacked",
		Col3DCylinder:               "standard",
		Col3DCylinderClustered:      "clustered",
		Col3DCylinderStacked:        "stacked",
		Col3DCylinderPercentStacked: "percentStacked",
		Line:                        "standard",
	}
	plotAreaChartBarDir = map[string]string{
		Bar:                         "bar",
		BarStacked:                  "bar",
		BarPercentStacked:           "bar",
		Bar3DClustered:              "bar",
		Bar3DStacked:                "bar",
		Bar3DPercentStacked:         "bar",
		Bar3DConeClustered:          "bar",
		Bar3DConeStacked:            "bar",
		Bar3DConePercentStacked:     "bar",
		Bar3DPyramidClustered:       "bar",
		Bar3DPyramidStacked:         "bar",
		Bar3DPyramidPercentStacked:  "bar",
		Bar3DCylinderClustered:      "bar",
		Bar3DCylinderStacked:        "bar",
		Bar3DCylinderPercentStacked: "bar",
		Col:                         "col",
		ColStacked:                  "col",
		ColPercentStacked:           "col",
		Col3D:                       "col",
		Col3DClustered:              "col",
		Col3DStacked:                "col",
		Col3DPercentStacked:         "col",
		Col3DCone:                   "col",
		Col3DConeStacked:            "col",
		Col3DConeClustered:          "col",
		Col3DConePercentStacked:     "col",
		Col3DPyramid:                "col",
		Col3DPyramidClustered:       "col",
		Col3DPyramidStacked:         "col",
		Col3DPyramidPercentStacked:  "col",
		Col3DCylinder:               "col",
		Col3DCylinderClustered:      "col",
		Col3DCylinderStacked:        "col",
		Col3DCylinderPercentStacked: "col",
		Line:                        "standard",
	}
	orientation = map[bool]string{
		true:  "maxMin",
		false: "minMax",
	}
	catAxPos = map[bool]string{
		true:  "t",
		false: "b",
	}
	valAxPos = map[bool]string{
		true:  "r",
		false: "l",
	}
	valTickLblPos = map[string]string{
		Contour:          "none",
		WireframeContour: "none",
	}
)

// parseFormatChartSet provides a function to parse the format settings of the
// chart with default value.
func parseFormatChartSet(formatSet string) (*formatChart, error) {
	format := formatChart{
		Dimension: formatChartDimension{
			Width:  480,
			Height: 290,
		},
		Format: formatPicture{
			FPrintsWithSheet: true,
			XScale:           1,
			YScale:           1,
		},
		Legend: formatChartLegend{
			Position: "bottom",
		},
		Title: formatChartTitle{
			Name: " ",
		},
		VaryColors:   true,
		ShowBlanksAs: "gap",
	}
	err := json.Unmarshal([]byte(formatSet), &format)
	return &format, err
}

// AddChart provides the method to add chart in a sheet by given chart format
// set (such as offset, scale, aspect ratio setting and print settings) and
// properties set. For example, create 3D clustered column chart with data
// Sheet1!$E$1:$L$15:
//
//    package main
//
//    import (
//        "fmt"
//
//        "github.com/xuri/excelize/v2"
//    )
//
//    func main() {
//        categories := map[string]string{
//            "A2": "Small", "A3": "Normal", "A4": "Large",
//            "B1": "Apple", "C1": "Orange", "D1": "Pear"}
//        values := map[string]int{
//            "B2": 2, "C2": 3, "D2": 3, "B3": 5, "C3": 2, "D3": 4, "B4": 6, "C4": 7, "D4": 8}
//        f, err := excelize.NewFile()
//        if err != nil {
//            fmt.Println(err)
//            return
//        }
//        for k, v := range categories {
//            f.SetCellValue("Sheet1", k, v)
//        }
//        for k, v := range values {
//            f.SetCellValue("Sheet1", k, v)
//        }
//        if err := f.AddChart("Sheet1", "E1", `{
//            "type": "col3DClustered",
//            "series": [
//            {
//                "name": "Sheet1!$A$2",
//                "categories": "Sheet1!$B$1:$D$1",
//                "values": "Sheet1!$B$2:$D$2"
//            },
//            {
//                "name": "Sheet1!$A$3",
//                "categories": "Sheet1!$B$1:$D$1",
//                "values": "Sheet1!$B$3:$D$3"
//            },
//            {
//                "name": "Sheet1!$A$4",
//                "categories": "Sheet1!$B$1:$D$1",
//                "values": "Sheet1!$B$4:$D$4"
//            }],
//            "title":
//            {
//                "name": "Fruit 3D Clustered Column Chart"
//            },
//            "legend":
//            {
//                "none": false,
//                "position": "bottom",
//                "show_legend_key": false
//            },
//            "plotarea":
//            {
//                "show_bubble_size": true,
//                "show_cat_name": false,
//                "show_leader_lines": false,
//                "show_percent": true,
//                "show_series_name": true,
//                "show_val": true
//            },
//            "show_blanks_as": "zero",
//            "x_axis":
//            {
//                "reverse_order": true
//            },
//            "y_axis":
//            {
//                "maximum": 7.5,
//                "minimum": 0.5
//            }
//        }`); err != nil {
//            fmt.Println(err)
//            return
//        }
//        // Save spreadsheet by the given path.
//        if err := f.SaveAs("Book1.xlsx"); err != nil {
//            fmt.Println(err)
//        }
//    }
//
// The following shows the type of chart supported by excelize:
//
//     Type                        | Chart
//    -----------------------------+------------------------------
//     area                        | 2D area chart
//     areaStacked                 | 2D stacked area chart
//     areaPercentStacked          | 2D 100% stacked area chart
//     area3D                      | 3D area chart
//     area3DStacked               | 3D stacked area chart
//     area3DPercentStacked        | 3D 100% stacked area chart
//     bar                         | 2D clustered bar chart
//     barStacked                  | 2D stacked bar chart
//     barPercentStacked           | 2D 100% stacked bar chart
//     bar3DClustered              | 3D clustered bar chart
//     bar3DStacked                | 3D stacked bar chart
//     bar3DPercentStacked         | 3D 100% stacked bar chart
//     bar3DConeClustered          | 3D cone clustered bar chart
//     bar3DConeStacked            | 3D cone stacked bar chart
//     bar3DConePercentStacked     | 3D cone percent bar chart
//     bar3DPyramidClustered       | 3D pyramid clustered bar chart
//     bar3DPyramidStacked         | 3D pyramid stacked bar chart
//     bar3DPyramidPercentStacked  | 3D pyramid percent stacked bar chart
//     bar3DCylinderClustered      | 3D cylinder clustered bar chart
//     bar3DCylinderStacked        | 3D cylinder stacked bar chart
//     bar3DCylinderPercentStacked | 3D cylinder percent stacked bar chart
//     col                         | 2D clustered column chart
//     colStacked                  | 2D stacked column chart
//     colPercentStacked           | 2D 100% stacked column chart
//     col3DClustered              | 3D clustered column chart
//     col3D                       | 3D column chart
//     col3DStacked                | 3D stacked column chart
//     col3DPercentStacked         | 3D 100% stacked column chart
//     col3DCone                   | 3D cone column chart
//     col3DConeClustered          | 3D cone clustered column chart
//     col3DConeStacked            | 3D cone stacked column chart
//     col3DConePercentStacked     | 3D cone percent stacked column chart
//     col3DPyramid                | 3D pyramid column chart
//     col3DPyramidClustered       | 3D pyramid clustered column chart
//     col3DPyramidStacked         | 3D pyramid stacked column chart
//     col3DPyramidPercentStacked  | 3D pyramid percent stacked column chart
//     col3DCylinder               | 3D cylinder column chart
//     col3DCylinderClustered      | 3D cylinder clustered column chart
//     col3DCylinderStacked        | 3D cylinder stacked column chart
//     col3DCylinderPercentStacked | 3D cylinder percent stacked column chart
//     doughnut                    | doughnut chart
//     line                        | line chart
//     pie                         | pie chart
//     pie3D                       | 3D pie chart
//     pieOfPie                    | pie of pie chart
//     barOfPie                    | bar of pie chart
//     radar                       | radar chart
//     scatter                     | scatter chart
//     surface3D                   | 3D surface chart
//     wireframeSurface3D          | 3D wireframe surface chart
//     contour                     | contour chart
//     wireframeContour            | wireframe contour chart
//     bubble                      | bubble chart
//     bubble3D                    | 3D bubble chart
//
// In Excel a chart series is a collection of information that defines which data is plotted such as values, axis labels and formatting.
//
// The series options that can be set are:
//
//    name
//    categories
//    values
//    line
//    marker
//
// name: Set the name for the series. The name is displayed in the chart legend and in the formula bar. The name property is optional and if it isn't supplied it will default to Series 1..n. The name can also be a formula such as Sheet1!$A$1
//
// categories: This sets the chart category labels. The category is more or less the same as the X axis. In most chart types the categories property is optional and the chart will just assume a sequential series from 1..n.
//
// values: This is the most important property of a series and is the only mandatory option for every chart object. This option links the chart with the worksheet data that it displays.
//
// line: This sets the line format of the line chart. The line property is optional and if it isn't supplied it will default style. The options that can be set is width. The range of width is 0.25pt - 999pt. If the value of width is outside the range, the default width of the line is 2pt.
//
// marker: This sets the marker of the line chart and scatter chart. The range of optional field 'size' is 2-72 (default value is 5). The enumeration value of optional field 'symbol' are (default value is 'auto'):
//
//    circle
//    dash
//    diamond
//    dot
//    none
//    picture
//    plus
//    square
//    star
//    triangle
//    x
//    auto
//
// Set properties of the chart legend. The options that can be set are:
//
//    none
//    position
//    show_legend_key
//
// none: Specified if show the legend without overlapping the chart. The default value is 'false'.
//
// position: Set the position of the chart legend. The default legend position is right. This parameter only takes effect when 'none' is false. The available positions are:
//
//    top
//    bottom
//    left
//    right
//    top_right
//
// show_legend_key: Set the legend keys shall be shown in data labels. The default value is false.
//
// Set properties of the chart title. The properties that can be set are:
//
//    title
//
// name: Set the name (title) for the chart. The name is displayed above the chart. The name can also be a formula such as Sheet1!$A$1 or a list with a sheetname. The name property is optional. The default is to have no chart title.
//
// Specifies how blank cells are plotted on the chart by show_blanks_as. The default value is gap. The options that can be set are:
//
//    gap
//    span
//    zero
//
// gap: Specifies that blank values shall be left as a gap.
//
// span: Specifies that blank values shall be spanned with a line.
//
// zero: Specifies that blank values shall be treated as zero.
//
// Specifies that each data marker in the series has a different color by vary_colors. The default value is true.
//
// Set chart offset, scale, aspect ratio setting and print settings by format, same as function AddPicture.
//
// Set the position of the chart plot area by plotarea. The properties that can be set are:
//
//    show_bubble_size
//    show_cat_name
//    show_leader_lines
//    show_percent
//    show_series_name
//    show_val
//
// show_bubble_size: Specifies the bubble size shall be shown in a data label. The show_bubble_size property is optional. The default value is false.
//
// show_cat_name: Specifies that the category name shall be shown in the data label. The show_cat_name property is optional. The default value is true.
//
// show_leader_lines: Specifies leader lines shall be shown for data labels. The show_leader_lines property is optional. The default value is false.
//
// show_percent: Specifies that the percentage shall be shown in a data label. The show_percent property is optional. The default value is false.
//
// show_series_name: Specifies that the series name shall be shown in a data label. The show_series_name property is optional. The default value is false.
//
// show_val: Specifies that the value shall be shown in a data label. The show_val property is optional. The default value is false.
//
// Set the primary horizontal and vertical axis options by x_axis and y_axis. The properties of x_axis that can be set are:
//
//    none
//    major_grid_lines
//    minor_grid_lines
//    tick_label_skip
//    reverse_order
//    maximum
//    minimum
//
// The properties of y_axis that can be set are:
//
//    none
//    major_grid_lines
//    minor_grid_lines
//    major_unit
//    reverse_order
//    maximum
//    minimum
//
// none: Disable axes.
//
// major_grid_lines: Specifies major gridlines.
//
// minor_grid_lines: Specifies minor gridlines.
//
// major_unit: Specifies the distance between major ticks. Shall contain a positive floating-point number. The major_unit property is optional. The default value is auto.
//
// tick_label_skip: Specifies how many tick labels to skip between label that is drawn. The tick_label_skip property is optional. The default value is auto.
//
// reverse_order: Specifies that the categories or values on reverse order (orientation of the chart). The reverse_order property is optional. The default value is false.
//
// maximum: Specifies that the fixed maximum, 0 is auto. The maximum property is optional. The default value is auto.
//
// minimum: Specifies that the fixed minimum, 0 is auto. The minimum property is optional. The default value is auto.
//
// Set chart size by dimension property. The dimension property is optional. The default width is 480, and height is 290.
//
// combo: Specifies the create a chart that combines two or more chart types
// in a single chart. For example, create a clustered column - line chart with
// data Sheet1!$E$1:$L$15:
//
//    package main
//
//    import (
//        "fmt"
//
//        "github.com/xuri/excelize/v2"
//    )
//
//    func main() {
//        categories := map[string]string{
//            "A2": "Small", "A3": "Normal", "A4": "Large",
//            "B1": "Apple", "C1": "Orange", "D1": "Pear"}
//        values := map[string]int{
//            "B2": 2, "C2": 3, "D2": 3, "B3": 5, "C3": 2, "D3": 4, "B4": 6, "C4": 7, "D4": 8}
//        f, err := excelize.NewFile()
//        if err != nil {
//            fmt.Println(err)
//            return
//        }
//        for k, v := range categories {
//            f.SetCellValue("Sheet1", k, v)
//        }
//        for k, v := range values {
//            f.SetCellValue("Sheet1", k, v)
//        }
//        if err := f.AddChart("Sheet1", "E1", `{
//            "type": "col",
//            "series": [
//            {
//                "name": "Sheet1!$A$2",
//                "categories": "",
//                "values": "Sheet1!$B$2:$D$2"
//            },
//            {
//                "name": "Sheet1!$A$3",
//                "categories": "Sheet1!$B$1:$D$1",
//                "values": "Sheet1!$B$3:$D$3"
//            }],
//            "format":
//            {
//                "x_scale": 1.0,
//                "y_scale": 1.0,
//                "x_offset": 15,
//                "y_offset": 10,
//                "print_obj": true,
//                "lock_aspect_ratio": false,
//                "locked": false
//            },
//            "title":
//            {
//                "name": "Clustered Column - Line Chart"
//            },
//            "legend":
//            {
//                "position": "left",
//                "show_legend_key": false
//            },
//            "plotarea":
//            {
//                "show_bubble_size": true,
//                "show_cat_name": false,
//                "show_leader_lines": false,
//                "show_percent": true,
//                "show_series_name": true,
//                "show_val": true
//            }
//        }`, `{
//            "type": "line",
//            "series": [
//            {
//                "name": "Sheet1!$A$4",
//                "categories": "Sheet1!$B$1:$D$1",
//                "values": "Sheet1!$B$4:$D$4",
//                "marker":
//                {
//                    "symbol": "none",
//                    "size": 10
//                }
//            }],
//            "format":
//            {
//                "x_scale": 1,
//                "y_scale": 1,
//                "x_offset": 15,
//                "y_offset": 10,
//                "print_obj": true,
//                "lock_aspect_ratio": false,
//                "locked": false
//            },
//            "legend":
//            {
//                "position": "right",
//                "show_legend_key": false
//            },
//            "plotarea":
//            {
//                "show_bubble_size": true,
//                "show_cat_name": false,
//                "show_leader_lines": false,
//                "show_percent": true,
//                "show_series_name": true,
//                "show_val": true
//            }
//        }`); err != nil {
//            fmt.Println(err)
//            return
//        }
//        // Save spreadsheet file by the given path.
//        if err := f.SaveAs("Book1.xlsx"); err != nil {
//            fmt.Println(err)
//        }
//    }
//
func (f *File) AddChart(sheet, cell, format string, combo ...string) error {
	if !f.IsValid() {
		return ErrIncompleteFileSetup
	}

	// Read sheet data.
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	formatSet, comboCharts, err := f.getFormatChart(format, combo)
	if err != nil {
		return err
	}
	// Add first picture for given sheet, create xl/drawings/ and xl/drawings/_rels/ folder.
	drawingID := f.countDrawings() + 1
	chartID := f.countCharts() + 1
	drawingXML := "xl/drawings/drawing" + strconv.Itoa(drawingID) + ".xml"
	drawingID, drawingXML, err = f.prepareDrawing(ws, drawingID, sheet, drawingXML)
	if err != nil {
		return err
	}
	drawingRels := "xl/drawings/_rels/drawing" + strconv.Itoa(drawingID) + ".xml.rels"
	drawingRID, err := f.addRels(drawingRels, SourceRelationshipChart, "../charts/chart"+strconv.Itoa(chartID)+".xml", "")
	if err != nil {
		return err
	}
	err = f.addDrawingChart(sheet, drawingXML, cell, formatSet.Dimension.Width, formatSet.Dimension.Height, drawingRID, &formatSet.Format)
	if err != nil {
		return err
	}
	err = f.addChart(formatSet, comboCharts)
	if err != nil {
		return err
	}
	f.addContentTypePart(chartID, "chart")
	f.addContentTypePart(drawingID, "drawings")
	f.addSheetNameSpace(sheet, SourceRelationship)
	return err
}

// AddChartSheet provides the method to create a chartsheet by given chart
// format set (such as offset, scale, aspect ratio setting and print settings)
// and properties set. In Excel a chartsheet is a worksheet that only contains
// a chart.
func (f *File) AddChartSheet(sheet, format string, combo ...string) error {
	if !f.IsValid() {
		return ErrIncompleteFileSetup
	}

	// Check if the worksheet already exists
	if f.GetSheetIndex(sheet) != -1 {
		return ErrExistsWorksheet
	}
	formatSet, comboCharts, err := f.getFormatChart(format, combo)
	if err != nil {
		return err
	}
	cs := xlsxChartsheet{
		SheetViews: &xlsxChartsheetViews{
			SheetView: []*xlsxChartsheetView{{ZoomScaleAttr: 100, ZoomToFitAttr: true}},
		},
	}
	f.SheetCount++
	wb := f.workbookReader()
	sheetID := 0
	for _, v := range wb.Sheets.Sheet {
		if v.SheetID > sheetID {
			sheetID = v.SheetID
		}
	}
	sheetID++
	path := "xl/chartsheets/sheet" + strconv.Itoa(sheetID) + ".xml"
	f.sheetMap[trimSheetName(sheet)] = path
	f.Sheet.Store(path, nil)
	drawingID := f.countDrawings() + 1
	chartID := f.countCharts() + 1
	drawingXML := "xl/drawings/drawing" + strconv.Itoa(drawingID) + ".xml"
	err = f.prepareChartSheetDrawing(&cs, drawingID, sheet)
	if err != nil {
		return err
	}
	drawingRels := "xl/drawings/_rels/drawing" + strconv.Itoa(drawingID) + ".xml.rels"
	drawingRID, err := f.addRels(drawingRels, SourceRelationshipChart, "../charts/chart"+strconv.Itoa(chartID)+".xml", "")
	if err != nil {
		return err
	}
	err = f.addSheetDrawingChart(drawingXML, drawingRID, &formatSet.Format)
	if err != nil {
		return err
	}
	err = f.addChart(formatSet, comboCharts)
	if err != nil {
		return err
	}
	f.addContentTypePart(chartID, "chart")
	f.addContentTypePart(sheetID, "chartsheet")
	f.addContentTypePart(drawingID, "drawings")
	wrp, err := f.getWorkbookRelsPath()
	if err != nil {
		return err
	}
	// Update workbook.xml.rels
	rID, err := f.addRels(wrp, SourceRelationshipChartsheet, fmt.Sprintf("/xl/chartsheets/sheet%d.xml", sheetID), "")
	if err != nil {
		return err
	}
	// Update workbook.xml
	f.setWorkbook(sheet, sheetID, rID)
	chartsheet, err := xml.Marshal(cs)
	if err != nil {
		return err
	}
	f.addSheetNameSpace(sheet, NameSpaceSpreadSheet)
	f.saveFileList(path, replaceRelationshipsBytes(f.replaceNameSpaceBytes(path, chartsheet)))
	return err
}

// getFormatChart provides a function to check format set of the chart and
// create chart format.
func (f *File) getFormatChart(format string, combo []string) (*formatChart, []*formatChart, error) {
	var comboCharts []*formatChart
	formatSet, err := parseFormatChartSet(format)
	if err != nil {
		return formatSet, comboCharts, err
	}
	for _, comboFormat := range combo {
		comboChart, err := parseFormatChartSet(comboFormat)
		if err != nil {
			return formatSet, comboCharts, err
		}
		if _, ok := chartValAxNumFmtFormatCode[comboChart.Type]; !ok {
			return formatSet, comboCharts, newUnsupportedChartType(comboChart.Type)
		}
		comboCharts = append(comboCharts, comboChart)
	}
	if _, ok := chartValAxNumFmtFormatCode[formatSet.Type]; !ok {
		return formatSet, comboCharts, newUnsupportedChartType(formatSet.Type)
	}
	return formatSet, comboCharts, err
}

// DeleteChart provides a function to delete chart in XLSX by given worksheet
// and cell name.
func (f *File) DeleteChart(sheet, cell string) (err error) {
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return
	}
	col--
	row--
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return
	}
	if ws.Drawing == nil {
		return
	}
	drawingXML := strings.ReplaceAll(f.getSheetRelationshipsTargetByID(sheet, ws.Drawing.RID), "..", "xl")
	return f.deleteDrawing(col, row, drawingXML, "Chart")
}

// countCharts provides a function to get chart files count storage in the
// folder xl/charts.
func (f *File) countCharts() int {
	count := 0
	f.Pkg.Range(func(k, v interface{}) bool {
		if strings.Contains(k.(string), "xl/charts/chart") {
			count++
		}
		return true
	})
	return count
}

// ptToEMUs provides a function to convert pt to EMUs, 1 pt = 12700 EMUs. The
// range of pt is 0.25pt - 999pt. If the value of pt is outside the range, the
// default EMUs will be returned.
func (f *File) ptToEMUs(pt float64) int {
	if 0.25 > pt || pt > 999 {
		return 25400
	}
	return int(12700 * pt)
}
