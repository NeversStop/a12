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
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// StreamWriter defined the type of stream writer.
type StreamWriter struct {
	File            *File
	Sheet           string
	SheetID         int
	sheetWritten    bool
	cols            string
	worksheet       *xlsxWorksheet
	rawData         bufferedWriter
	mergeCellsCount int
	mergeCells      strings.Builder
	tableParts      string
}

// NewStreamWriter return stream writer struct by given worksheet name for
// generate new worksheet with large amounts of data. Note that after set
// rows, you must call the 'Flush' method to end the streaming writing process
// and ensure that the order of line numbers is ascending, the normal mode
// functions and stream mode functions can't be work mixed to writing data on
// the worksheets, you can't get cell value when in-memory chunks data over
// 16MB. For example, set data for worksheet of size 102400 rows x 50 columns
// with numbers and style:
//
//	file := excelize.NewFile()
//	streamWriter, err := file.NewStreamWriter("Sheet1")
//	if err != nil {
//	    fmt.Println(err)
//	}
//	styleID, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "#777777"}})
//	if err != nil {
//	    fmt.Println(err)
//	}
//	if err := streamWriter.SetRow("A1", []interface{}{excelize.Cell{StyleID: styleID, Value: "Data"}},
//	    excelize.RowOpts{Height: 45, Hidden: false}); err != nil {
//	    fmt.Println(err)
//	}
//	for rowID := 2; rowID <= 102400; rowID++ {
//	    row := make([]interface{}, 50)
//	    for colID := 0; colID < 50; colID++ {
//	        row[colID] = rand.Intn(640000)
//	    }
//	    cell, _ := excelize.CoordinatesToCellName(1, rowID)
//	    if err := streamWriter.SetRow(cell, row); err != nil {
//	        fmt.Println(err)
//	    }
//	}
//	if err := streamWriter.Flush(); err != nil {
//	    fmt.Println(err)
//	}
//	if err := file.SaveAs("Book1.xlsx"); err != nil {
//	    fmt.Println(err)
//	}
//
// Set cell value and cell formula for a worksheet with stream writer:
//
//	err := streamWriter.SetRow("A1", []interface{}{
//	    excelize.Cell{Value: 1},
//	    excelize.Cell{Value: 2},
//	    excelize.Cell{Formula: "SUM(A1,B1)"}});
//
// Set cell value and rows style for a worksheet with stream writer:
//
//	err := streamWriter.SetRow("A1", []interface{}{
//	    excelize.Cell{Value: 1}},
//	    excelize.RowOpts{StyleID: styleID, Height: 20, Hidden: false});
func (f *File) NewStreamWriter(sheet string) (*StreamWriter, error) {
	sheetID := f.getSheetID(sheet)
	if sheetID == -1 {
		return nil, newNoExistSheetError(sheet)
	}
	sw := &StreamWriter{
		File:    f,
		Sheet:   sheet,
		SheetID: sheetID,
	}
	var err error
	sw.worksheet, err = f.workSheetReader(sheet)
	if err != nil {
		return nil, err
	}

	sheetXMLPath, _ := f.getSheetXMLPath(sheet)
	if f.streams == nil {
		f.streams = make(map[string]*StreamWriter)
	}
	f.streams[sheetXMLPath] = sw

	_, _ = sw.rawData.WriteString(xml.Header + `<worksheet` + templateNamespaceIDMap)
	bulkAppendFields(&sw.rawData, sw.worksheet, 2, 5)
	return sw, err
}

// AddTable creates an Excel table for the StreamWriter using the given
// cell range and format set. For example, create a table of A1:D5:
//
//	err := sw.AddTable("A1", "D5", "")
//
// Create a table of F2:H6 with format set:
//
//	err := sw.AddTable("F2", "H6", `{
//	    "table_name": "table",
//	    "table_style": "TableStyleMedium2",
//	    "show_first_column": true,
//	    "show_last_column": true,
//	    "show_row_stripes": false,
//	    "show_column_stripes": true
//	}`)
//
// Note that the table must be at least two lines including the header. The
// header cells must contain strings and must be unique.
//
// Currently, only one table is allowed for a StreamWriter. AddTable must be
// called after the rows are written but before Flush.
//
// See File.AddTable for details on the table format.
func (sw *StreamWriter) AddTable(hCell, vCell, opts string) error {
	options, err := parseTableOptions(opts)
	if err != nil {
		return err
	}

	coordinates, err := cellRefsToCoordinates(hCell, vCell)
	if err != nil {
		return err
	}
	_ = sortCoordinates(coordinates)

	// Correct the minimum number of rows, the table at least two lines.
	if coordinates[1] == coordinates[3] {
		coordinates[3]++
	}

	// Correct table reference range, such correct C1:B3 to B1:C3.
	ref, err := sw.File.coordinatesToRangeRef(coordinates)
	if err != nil {
		return err
	}

	// create table columns using the first row
	tableHeaders, err := sw.getRowValues(coordinates[1], coordinates[0], coordinates[2])
	if err != nil {
		return err
	}
	tableColumn := make([]*xlsxTableColumn, len(tableHeaders))
	for i, name := range tableHeaders {
		tableColumn[i] = &xlsxTableColumn{
			ID:   i + 1,
			Name: name,
		}
	}

	tableID := sw.File.countTables() + 1

	name := options.TableName
	if name == "" {
		name = "Table" + strconv.Itoa(tableID)
	}

	table := xlsxTable{
		XMLNS:       NameSpaceSpreadSheet.Value,
		ID:          tableID,
		Name:        name,
		DisplayName: name,
		Ref:         ref,
		AutoFilter: &xlsxAutoFilter{
			Ref: ref,
		},
		TableColumns: &xlsxTableColumns{
			Count:       len(tableColumn),
			TableColumn: tableColumn,
		},
		TableStyleInfo: &xlsxTableStyleInfo{
			Name:              options.TableStyle,
			ShowFirstColumn:   options.ShowFirstColumn,
			ShowLastColumn:    options.ShowLastColumn,
			ShowRowStripes:    options.ShowRowStripes,
			ShowColumnStripes: options.ShowColumnStripes,
		},
	}

	sheetRelationshipsTableXML := "../tables/table" + strconv.Itoa(tableID) + ".xml"
	tableXML := strings.ReplaceAll(sheetRelationshipsTableXML, "..", "xl")

	// Add first table for given sheet.
	sheetPath := sw.File.sheetMap[trimSheetName(sw.Sheet)]
	sheetRels := "xl/worksheets/_rels/" + strings.TrimPrefix(sheetPath, "xl/worksheets/") + ".rels"
	rID := sw.File.addRels(sheetRels, SourceRelationshipTable, sheetRelationshipsTableXML, "")

	sw.tableParts = fmt.Sprintf(`<tableParts count="1"><tablePart r:id="rId%d"></tablePart></tableParts>`, rID)

	sw.File.addContentTypePart(tableID, "table")

	b, _ := xml.Marshal(table)
	sw.File.saveFileList(tableXML, b)
	return nil
}

// Extract values from a row in the StreamWriter.
func (sw *StreamWriter) getRowValues(hRow, hCol, vCol int) (res []string, err error) {
	res = make([]string, vCol-hCol+1)

	r, err := sw.rawData.Reader()
	if err != nil {
		return nil, err
	}

	dec := sw.File.xmlNewDecoder(r)
	for {
		token, err := dec.Token()
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return nil, err
		}
		startElement, ok := getRowElement(token, hRow)
		if !ok {
			continue
		}
		// decode cells
		var row xlsxRow
		if err := dec.DecodeElement(&row, &startElement); err != nil {
			return nil, err
		}
		for _, c := range row.C {
			col, _, err := CellNameToCoordinates(c.R)
			if err != nil {
				return nil, err
			}
			if col < hCol || col > vCol {
				continue
			}
			res[col-hCol] = c.V
		}
		return res, nil
	}
}

// Check if the token is an XLSX row with the matching row number.
func getRowElement(token xml.Token, hRow int) (startElement xml.StartElement, ok bool) {
	startElement, ok = token.(xml.StartElement)
	if !ok {
		return
	}
	ok = startElement.Name.Local == "row"
	if !ok {
		return
	}
	ok = false
	for _, attr := range startElement.Attr {
		if attr.Name.Local != "r" {
			continue
		}
		row, _ := strconv.Atoi(attr.Value)
		if row == hRow {
			ok = true
			return
		}
	}
	return
}

// Cell can be used directly in StreamWriter.SetRow to specify a style and
// a value.
type Cell struct {
	StyleID int
	Formula string
	Value   interface{}
}

// RowOpts define the options for the set row, it can be used directly in
// StreamWriter.SetRow to specify the style and properties of the row.
type RowOpts struct {
	Height  float64
	Hidden  bool
	StyleID int
}

// marshalAttrs prepare attributes of the row.
func (r *RowOpts) marshalAttrs() (attrs string, err error) {
	if r == nil {
		return
	}
	if r.Height > MaxRowHeight {
		err = ErrMaxRowHeight
		return
	}
	if r.StyleID > 0 {
		attrs += fmt.Sprintf(` s="%d" customFormat="true"`, r.StyleID)
	}
	if r.Height > 0 {
		attrs += fmt.Sprintf(` ht="%v" customHeight="true"`, r.Height)
	}
	if r.Hidden {
		attrs += ` hidden="true"`
	}
	return
}

// parseRowOpts provides a function to parse the optional settings for
// *StreamWriter.SetRow.
func parseRowOpts(opts ...RowOpts) *RowOpts {
	options := &RowOpts{}
	for _, opt := range opts {
		options = &opt
	}
	return options
}

// SetRow writes an array to stream rows by giving a worksheet name, starting
// coordinate and a pointer to an array of values. Note that you must call the
// 'Flush' method to end the streaming writing process.
//
// As a special case, if Cell is used as a value, then the Cell.StyleID will be
// applied to that cell.
func (sw *StreamWriter) SetRow(cell string, values []interface{}, opts ...RowOpts) error {
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return err
	}
	if !sw.sheetWritten {
		if len(sw.cols) > 0 {
			_, _ = sw.rawData.WriteString("<cols>" + sw.cols + "</cols>")
		}
		_, _ = sw.rawData.WriteString(`<sheetData>`)
		sw.sheetWritten = true
	}
	options := parseRowOpts(opts...)
	attrs, err := options.marshalAttrs()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(&sw.rawData, `<row r="%d"%s>`, row, attrs)
	for i, val := range values {
		if val == nil {
			continue
		}
		ref, err := CoordinatesToCellName(col+i, row)
		if err != nil {
			return err
		}
		c := xlsxC{R: ref, S: options.StyleID}
		if v, ok := val.(Cell); ok {
			c.S = v.StyleID
			val = v.Value
			setCellFormula(&c, v.Formula)
		} else if v, ok := val.(*Cell); ok && v != nil {
			c.S = v.StyleID
			val = v.Value
			setCellFormula(&c, v.Formula)
		}
		if err = sw.setCellValFunc(&c, val); err != nil {
			_, _ = sw.rawData.WriteString(`</row>`)
			return err
		}
		writeCell(&sw.rawData, c)
	}
	_, _ = sw.rawData.WriteString(`</row>`)
	return sw.rawData.Sync()
}

// SetColWidth provides a function to set the width of a single column or
// multiple columns for the StreamWriter. Note that you must call
// the 'SetColWidth' function before the 'SetRow' function. For example set
// the width column B:C as 20:
//
//	err := streamWriter.SetColWidth(2, 3, 20)
func (sw *StreamWriter) SetColWidth(min, max int, width float64) error {
	if sw.sheetWritten {
		return ErrStreamSetColWidth
	}
	if min < MinColumns || min > MaxColumns || max < MinColumns || max > MaxColumns {
		return ErrColumnNumber
	}
	if width > MaxColumnWidth {
		return ErrColumnWidth
	}
	if min > max {
		min, max = max, min
	}
	sw.cols += fmt.Sprintf(`<col min="%d" max="%d" width="%f" customWidth="1"/>`, min, max, width)
	return nil
}

// MergeCell provides a function to merge cells by a given range reference for
// the StreamWriter. Don't create a merged cell that overlaps with another
// existing merged cell.
func (sw *StreamWriter) MergeCell(hCell, vCell string) error {
	_, err := cellRefsToCoordinates(hCell, vCell)
	if err != nil {
		return err
	}
	sw.mergeCellsCount++
	_, _ = sw.mergeCells.WriteString(`<mergeCell ref="`)
	_, _ = sw.mergeCells.WriteString(hCell)
	_, _ = sw.mergeCells.WriteString(`:`)
	_, _ = sw.mergeCells.WriteString(vCell)
	_, _ = sw.mergeCells.WriteString(`"/>`)
	return nil
}

// setCellFormula provides a function to set formula of a cell.
func setCellFormula(c *xlsxC, formula string) {
	if formula != "" {
		c.F = &xlsxF{Content: formula}
	}
}

// setCellValFunc provides a function to set value of a cell.
func (sw *StreamWriter) setCellValFunc(c *xlsxC, val interface{}) (err error) {
	switch val := val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		err = setCellIntFunc(c, val)
	case float32:
		c.T, c.V = setCellFloat(float64(val), -1, 32)
	case float64:
		c.T, c.V = setCellFloat(val, -1, 64)
	case string:
		c.T, c.V, c.XMLSpace = setCellStr(val)
	case []byte:
		c.T, c.V, c.XMLSpace = setCellStr(string(val))
	case time.Duration:
		c.T, c.V = setCellDuration(val)
	case time.Time:
		var isNum bool
		date1904, wb := false, sw.File.workbookReader()
		if wb != nil && wb.WorkbookPr != nil {
			date1904 = wb.WorkbookPr.Date1904
		}
		c.T, c.V, isNum, err = setCellTime(val, date1904)
		if isNum && c.S == 0 {
			style, _ := sw.File.NewStyle(&Style{NumFmt: 22})
			c.S = style
		}
	case bool:
		c.T, c.V = setCellBool(val)
	case nil:
		c.T, c.V, c.XMLSpace = setCellStr("")
	case []RichTextRun:
		sst := sw.File.sharedStringsReader()
		si := xlsxSI{}
		si.R = transformRichTextRun(val)
		sst.SI = append(sst.SI, si)
		sst.Count++
		sst.UniqueCount++
		c.T, c.V = "s", strconv.Itoa(len(sst.SI)-1)
	default:
		c.T, c.V, c.XMLSpace = setCellStr(fmt.Sprint(val))
	}
	return err
}

func transformRichTextRun(runs []RichTextRun) []xlsxR {
	var result = make([]xlsxR, 0, len(runs))
	for _, textRun := range runs {
		run := xlsxR{T: &xlsxT{}}
		_, run.T.Val, run.T.Space = setCellStr(textRun.Text)
		fnt := textRun.Font
		if fnt != nil {
			run.RPr = newRpr(fnt)
		}
		result = append(result, run)
	}
	return result
}

// setCellIntFunc is a wrapper of SetCellInt.
func setCellIntFunc(c *xlsxC, val interface{}) (err error) {
	switch val := val.(type) {
	case int:
		c.T, c.V = setCellInt(val)
	case int8:
		c.T, c.V = setCellInt(int(val))
	case int16:
		c.T, c.V = setCellInt(int(val))
	case int32:
		c.T, c.V = setCellInt(int(val))
	case int64:
		c.T, c.V = setCellInt(int(val))
	case uint:
		c.T, c.V = setCellInt(int(val))
	case uint8:
		c.T, c.V = setCellInt(int(val))
	case uint16:
		c.T, c.V = setCellInt(int(val))
	case uint32:
		c.T, c.V = setCellInt(int(val))
	case uint64:
		c.T, c.V = setCellInt(int(val))
	default:
	}
	return
}

func writeCell(buf *bufferedWriter, c xlsxC) {
	_, _ = buf.WriteString(`<c`)
	if c.XMLSpace.Value != "" {
		fmt.Fprintf(buf, ` xml:%s="%s"`, c.XMLSpace.Name.Local, c.XMLSpace.Value)
	}
	fmt.Fprintf(buf, ` r="%s"`, c.R)
	if c.S != 0 {
		fmt.Fprintf(buf, ` s="%d"`, c.S)
	}
	if c.T != "" {
		fmt.Fprintf(buf, ` t="%s"`, c.T)
	}
	_, _ = buf.WriteString(`>`)
	if c.F != nil {
		_, _ = buf.WriteString(`<f>`)
		_ = xml.EscapeText(buf, []byte(c.F.Content))
		_, _ = buf.WriteString(`</f>`)
	}
	if c.V != "" {
		_, _ = buf.WriteString(`<v>`)
		_ = xml.EscapeText(buf, []byte(c.V))
		_, _ = buf.WriteString(`</v>`)
	}
	_, _ = buf.WriteString(`</c>`)
}

// Flush ending the streaming writing process.
func (sw *StreamWriter) Flush() error {
	if !sw.sheetWritten {
		_, _ = sw.rawData.WriteString(`<sheetData>`)
		sw.sheetWritten = true
	}
	_, _ = sw.rawData.WriteString(`</sheetData>`)
	bulkAppendFields(&sw.rawData, sw.worksheet, 8, 15)
	mergeCells := strings.Builder{}
	if sw.mergeCellsCount > 0 {
		_, _ = mergeCells.WriteString(`<mergeCells count="`)
		_, _ = mergeCells.WriteString(strconv.Itoa(sw.mergeCellsCount))
		_, _ = mergeCells.WriteString(`">`)
		_, _ = mergeCells.WriteString(sw.mergeCells.String())
		_, _ = mergeCells.WriteString(`</mergeCells>`)
	}
	_, _ = sw.rawData.WriteString(mergeCells.String())
	bulkAppendFields(&sw.rawData, sw.worksheet, 17, 38)
	_, _ = sw.rawData.WriteString(sw.tableParts)
	bulkAppendFields(&sw.rawData, sw.worksheet, 40, 40)
	_, _ = sw.rawData.WriteString(`</worksheet>`)
	if err := sw.rawData.Flush(); err != nil {
		return err
	}

	sheetPath := sw.File.sheetMap[trimSheetName(sw.Sheet)]
	sw.File.Sheet.Delete(sheetPath)
	delete(sw.File.checked, sheetPath)
	sw.File.Pkg.Delete(sheetPath)

	return nil
}

// bulkAppendFields bulk-appends fields in a worksheet by specified field
// names order range.
func bulkAppendFields(w io.Writer, ws *xlsxWorksheet, from, to int) {
	s := reflect.ValueOf(ws).Elem()
	enc := xml.NewEncoder(w)
	for i := 0; i < s.NumField(); i++ {
		if from <= i && i <= to {
			_ = enc.Encode(s.Field(i).Interface())
		}
	}
}

// bufferedWriter uses a temp file to store an extended buffer. Writes are
// always made to an in-memory buffer, which will always succeed. The buffer
// is written to the temp file with Sync, which may return an error.
// Therefore, Sync should be periodically called and the error checked.
type bufferedWriter struct {
	tmp *os.File
	buf bytes.Buffer
}

// Write to the in-memory buffer. The err is always nil.
func (bw *bufferedWriter) Write(p []byte) (n int, err error) {
	return bw.buf.Write(p)
}

// WriteString wite to the in-memory buffer. The err is always nil.
func (bw *bufferedWriter) WriteString(p string) (n int, err error) {
	return bw.buf.WriteString(p)
}

// Reader provides read-access to the underlying buffer/file.
func (bw *bufferedWriter) Reader() (io.Reader, error) {
	if bw.tmp == nil {
		return bytes.NewReader(bw.buf.Bytes()), nil
	}
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	fi, err := bw.tmp.Stat()
	if err != nil {
		return nil, err
	}
	// os.File.ReadAt does not affect the cursor position and is safe to use here
	return io.NewSectionReader(bw.tmp, 0, fi.Size()), nil
}

// Sync will write the in-memory buffer to a temp file, if the in-memory
// buffer has grown large enough. Any error will be returned.
func (bw *bufferedWriter) Sync() (err error) {
	// Try to use local storage
	if bw.buf.Len() < StreamChunkSize {
		return nil
	}
	if bw.tmp == nil {
		bw.tmp, err = ioutil.TempFile(os.TempDir(), "excelize-")
		if err != nil {
			// can not use local storage
			return nil
		}
	}
	return bw.Flush()
}

// Flush the entire in-memory buffer to the temp file, if a temp file is being
// used.
func (bw *bufferedWriter) Flush() error {
	if bw.tmp == nil {
		return nil
	}
	_, err := bw.buf.WriteTo(bw.tmp)
	if err != nil {
		return err
	}
	bw.buf.Reset()
	return nil
}

// Close the underlying temp file and reset the in-memory buffer.
func (bw *bufferedWriter) Close() error {
	bw.buf.Reset()
	if bw.tmp == nil {
		return nil
	}
	defer os.Remove(bw.tmp.Name())
	return bw.tmp.Close()
}
