// Copyright 2016 - 2022 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// Package excelize providing a set of functions that allow you to write to and
// read from XLAM / XLSM / XLSX / XLTM / XLTX files. Supports reading and
// writing spreadsheet documents generated by Microsoft Excel™ 2007 and later.
// Supports complex components by high compatibility, and provided streaming
// API for generating or reading data from a worksheet with huge amounts of
// data. This library needs Go version 1.15 or later.
//
// See https://xuri.me/excelize for more information about this package.
package excelize

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html/charset"
)

// File define a populated spreadsheet file struct.
type File struct {
	sync.Mutex
	options          *Options
	xmlAttr          map[string][]xml.Attr
	checked          map[string]bool
	sheetMap         map[string]string
	streams          map[string]*StreamWriter
	tempFiles        sync.Map
	CalcChain        *xlsxCalcChain
	Comments         map[string]*xlsxComments
	ContentTypes     *xlsxTypes
	Drawings         sync.Map
	Path             string
	SharedStrings    *xlsxSST
	sharedStringsMap map[string]int
	sharedStringItem [][]uint
	sharedStringTemp *os.File
	Sheet            sync.Map
	SheetCount       int
	Styles           *xlsxStyleSheet
	Theme            *xlsxTheme
	DecodeVMLDrawing map[string]*decodeVmlDrawing
	VMLDrawing       map[string]*vmlDrawing
	WorkBook         *xlsxWorkbook
	Relationships    sync.Map
	Pkg              sync.Map
	CharsetReader    charsetTranscoderFn
}

type charsetTranscoderFn func(charset string, input io.Reader) (rdr io.Reader, err error)

// Options define the options for open and reading spreadsheet.
//
// MaxCalcIterations specifies the maximum iterations for iterative
// calculation, the default value is 0.
//
// Password specifies the password of the spreadsheet in plain text.
//
// RawCellValue specifies if apply the number format for the cell value or get
// the raw value.
//
// UnzipSizeLimit specifies the unzip size limit in bytes on open the
// spreadsheet, this value should be greater than or equal to
// UnzipXMLSizeLimit, the default size limit is 16GB.
//
// UnzipXMLSizeLimit specifies the memory limit on unzipping worksheet and
// shared string table in bytes, worksheet XML will be extracted to system
// temporary directory when the file size is over this value, this value
// should be less than or equal to UnzipSizeLimit, the default value is
// 16MB.
type Options struct {
	MaxCalcIterations uint
	Password          string
	RawCellValue      bool
	UnzipSizeLimit    int64
	UnzipXMLSizeLimit int64
}

// OpenFile take the name of an spreadsheet file and returns a populated
// spreadsheet file struct for it. For example, open spreadsheet with
// password protection:
//
//	f, err := excelize.OpenFile("Book1.xlsx", excelize.Options{Password: "password"})
//	if err != nil {
//	    return
//	}
//
// Close the file by Close function after opening the spreadsheet.
func OpenFile(filename string, opt ...Options) (*File, error) {
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	f, err := OpenReader(file, opt...)
	if err != nil {
		closeErr := file.Close()
		if closeErr == nil {
			return f, err
		}
		return f, closeErr
	}
	f.Path = filename
	return f, file.Close()
}

// newFile is object builder
func newFile() *File {
	return &File{
		options:          &Options{UnzipSizeLimit: UnzipSizeLimit, UnzipXMLSizeLimit: StreamChunkSize},
		xmlAttr:          make(map[string][]xml.Attr),
		checked:          make(map[string]bool),
		sheetMap:         make(map[string]string),
		tempFiles:        sync.Map{},
		Comments:         make(map[string]*xlsxComments),
		Drawings:         sync.Map{},
		sharedStringsMap: make(map[string]int),
		Sheet:            sync.Map{},
		DecodeVMLDrawing: make(map[string]*decodeVmlDrawing),
		VMLDrawing:       make(map[string]*vmlDrawing),
		Relationships:    sync.Map{},
		CharsetReader:    charset.NewReaderLabel,
	}
}

// OpenReader read data stream from io.Reader and return a populated
// spreadsheet file.
func OpenReader(r io.Reader, opt ...Options) (*File, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f := newFile()
	f.options = parseOptions(opt...)
	if f.options.UnzipSizeLimit == 0 {
		f.options.UnzipSizeLimit = UnzipSizeLimit
		if f.options.UnzipXMLSizeLimit > f.options.UnzipSizeLimit {
			f.options.UnzipSizeLimit = f.options.UnzipXMLSizeLimit
		}
	}
	if f.options.UnzipXMLSizeLimit == 0 {
		f.options.UnzipXMLSizeLimit = StreamChunkSize
		if f.options.UnzipSizeLimit < f.options.UnzipXMLSizeLimit {
			f.options.UnzipXMLSizeLimit = f.options.UnzipSizeLimit
		}
	}
	if f.options.UnzipXMLSizeLimit > f.options.UnzipSizeLimit {
		return nil, ErrOptionsUnzipSizeLimit
	}
	if bytes.Contains(b, oleIdentifier) {
		if b, err = Decrypt(b, f.options); err != nil {
			return nil, ErrWorkbookFileFormat
		}
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		if len(f.options.Password) > 0 {
			return nil, ErrWorkbookPassword
		}
		return nil, err
	}
	file, sheetCount, err := f.ReadZipReader(zr)
	if err != nil {
		return nil, err
	}
	f.SheetCount = sheetCount
	for k, v := range file {
		f.Pkg.Store(k, v)
	}
	f.CalcChain = f.calcChainReader()
	f.sheetMap = f.getSheetMap()
	f.Styles = f.stylesReader()
	f.Theme = f.themeReader()
	return f, nil
}

// parseOptions provides a function to parse the optional settings for open
// and reading spreadsheet.
func parseOptions(opts ...Options) *Options {
	opt := &Options{}
	for _, o := range opts {
		opt = &o
	}
	return opt
}

// CharsetTranscoder Set user defined codepage transcoder function for open
// XLSX from non UTF-8 encoding.
func (f *File) CharsetTranscoder(fn charsetTranscoderFn) *File { f.CharsetReader = fn; return f }

// Creates new XML decoder with charset reader.
func (f *File) xmlNewDecoder(rdr io.Reader) (ret *xml.Decoder) {
	ret = xml.NewDecoder(rdr)
	ret.CharsetReader = f.CharsetReader
	return
}

// setDefaultTimeStyle provides a function to set default numbers format for
// time.Time type cell value by given worksheet name, cell coordinates and
// number format code.
func (f *File) setDefaultTimeStyle(sheet, axis string, format int) error {
	s, err := f.GetCellStyle(sheet, axis)
	if err != nil {
		return err
	}
	if s == 0 {
		style, _ := f.NewStyle(&Style{NumFmt: format})
		err = f.SetCellStyle(sheet, axis, axis, style)
	}
	return err
}

func (f *File) workSheetReader(sheet string) (ws *xlsxWorksheet, err error) {
	return nil, nil
}

func (f *File) SRworkSheetReader(sheet string) (ws *SRxlsxWorksheet, err error) {
	return f.SRWorkSheetReader(sheet, false)
}

// workSheetReader provides a function to get the pointer to the structure
// after deserialization by given worksheet name.
func (f *File) SRWorkSheetReader(sheet string, skipCheckRow bool) (ws *SRxlsxWorksheet, err error) {
	f.Lock()
	defer f.Unlock()
	var (
		name string
		ok   bool
	)

	if name, ok = f.getSheetXMLPath(sheet); !ok {
		err = newNoExistSheetError(sheet)
		return
	}

	if worksheet, ok := f.Sheet.Load(name); ok && worksheet != nil {
		ws = worksheet.(*SRxlsxWorksheet)
		return
	}

	for _, sheetType := range []string{"xl/chartsheets", "xl/dialogsheet", "xl/macrosheet"} {
		if strings.HasPrefix(name, sheetType) {
			err = newNotWorksheetError(sheet)
			return
		}
	}

	ws = new(SRxlsxWorksheet)

	if _, ok := f.xmlAttr[name]; !ok {
		d := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readBytes(name))))
		f.xmlAttr[name] = append(f.xmlAttr[name], getRootElement(d)...)
	}

	if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readBytes(name)))).
		Decode(ws); err != nil && err != io.EOF {
		err = newDecodeXMLError(err)
		return
	}

	err = nil
	if f.checked == nil {
		f.checked = make(map[string]bool)
	}
	if ok = f.checked[name]; !ok {
		SRcheckSheet(ws)
		if !skipCheckRow {
			if err = SRcheckRow(ws); err != nil {
				return
			}
		}
		f.checked[name] = true
	}

	f.Sheet.Store(name, ws)
	return
}

// checkSheet provides a function to fill each row element and make that is
// continuous in a worksheet of XML.
func checkSheet(ws *xlsxWorksheet) {
	var row int
	var r0 xlsxRow
	for i, r := range ws.SheetData.Row {
		if i == 0 && r.R == 0 {
			r0 = r
			ws.SheetData.Row = ws.SheetData.Row[1:]
			continue
		}
		if r.R != 0 && r.R > row {
			row = r.R
			continue
		}
		if r.R != row {
			row++
		}
	}
	sheetData := xlsxSheetData{Row: make([]xlsxRow, row)}
	row = 0
	for _, r := range ws.SheetData.Row {
		if r.R == row && row > 0 {
			sheetData.Row[r.R-1].C = append(sheetData.Row[r.R-1].C, r.C...)
			continue
		}
		if r.R != 0 {
			sheetData.Row[r.R-1] = r
			row = r.R
			continue
		}
		row++
		r.R = row
		sheetData.Row[row-1] = r
	}
	for i := 1; i <= row; i++ {
		sheetData.Row[i-1].R = i
	}
	checkSheetR0(ws, &sheetData, &r0)
}

func SRcheckSheet(ws *SRxlsxWorksheet) {
	var row int
	var r0 SRxlsxRow
	for i, r := range ws.SheetData.Row {
		if i == 0 && r.R == 0 {
			r0 = r
			ws.SheetData.Row = ws.SheetData.Row[1:]
			continue
		}
		if r.R != 0 && r.R > row {
			row = r.R
			continue
		}
		if r.R != row {
			row++
		}
	}
	sheetData := SRxlsxSheetData{Row: make([]SRxlsxRow, row)}
	row = 0
	for _, r := range ws.SheetData.Row {
		if r.R == row && row > 0 {
			sheetData.Row[r.R-1].C = append(sheetData.Row[r.R-1].C, r.C...)
			continue
		}
		if r.R != 0 {
			sheetData.Row[r.R-1] = r
			row = r.R
			continue
		}
		row++
		r.R = row
		sheetData.Row[row-1] = r
	}
	for i := 1; i <= row; i++ {
		sheetData.Row[i-1].R = i
	}
	SRcheckSheetR0(ws, &sheetData, &r0)
}

// checkSheetR0 handle the row element with r="0" attribute, cells in this row
// could be disorderly, the cell in this row can be used as the value of
// which cell is empty in the normal rows.
func checkSheetR0(ws *xlsxWorksheet, sheetData *xlsxSheetData, r0 *xlsxRow) {
	for _, cell := range r0.C {
		if col, row, err := CellNameToCoordinates(cell.R); err == nil {
			rows, rowIdx := len(sheetData.Row), row-1
			for r := rows; r < row; r++ {
				sheetData.Row = append(sheetData.Row, xlsxRow{R: r + 1})
			}
			columns, colIdx := len(sheetData.Row[rowIdx].C), col-1
			for c := columns; c < col; c++ {
				sheetData.Row[rowIdx].C = append(sheetData.Row[rowIdx].C, xlsxC{})
			}
			if !sheetData.Row[rowIdx].C[colIdx].hasValue() {
				sheetData.Row[rowIdx].C[colIdx] = cell
			}
		}
	}
	ws.SheetData = *sheetData
}

func SRcheckSheetR0(ws *SRxlsxWorksheet, sheetData *SRxlsxSheetData, r0 *SRxlsxRow) {
	for _, cell := range r0.C {
		if col, row, err := CellNameToCoordinates(cell.R); err == nil {
			rows, rowIdx := len(sheetData.Row), row-1
			for r := rows; r < row; r++ {
				sheetData.Row = append(sheetData.Row, SRxlsxRow{R: r + 1})
			}
			columns, colIdx := len(sheetData.Row[rowIdx].C), col-1
			for c := columns; c < col; c++ {
				sheetData.Row[rowIdx].C = append(sheetData.Row[rowIdx].C, SRxlsxC{})
			}
			if !sheetData.Row[rowIdx].C[colIdx].hasValue() {
				sheetData.Row[rowIdx].C[colIdx] = cell
			}
		}
	}
	ws.SheetData = *sheetData
}

// setRels provides a function to set relationships by given relationship ID,
// XML path, relationship type, target and target mode.
func (f *File) setRels(rID, relPath, relType, target, targetMode string) int {
	rels := f.relsReader(relPath)
	if rels == nil || rID == "" {
		return f.addRels(relPath, relType, target, targetMode)
	}
	rels.Lock()
	defer rels.Unlock()
	var ID int
	for i, rel := range rels.Relationships {
		if rel.ID == rID {
			rels.Relationships[i].Type = relType
			rels.Relationships[i].Target = target
			rels.Relationships[i].TargetMode = targetMode
			ID, _ = strconv.Atoi(strings.TrimPrefix(rID, "rId"))
			break
		}
	}
	return ID
}

// addRels provides a function to add relationships by given XML path,
// relationship type, target and target mode.
func (f *File) addRels(relPath, relType, target, targetMode string) int {
	uniqPart := map[string]string{
		SourceRelationshipSharedStrings: "/xl/sharedStrings.xml",
	}
	rels := f.relsReader(relPath)
	if rels == nil {
		rels = &xlsxRelationships{}
	}
	rels.Lock()
	defer rels.Unlock()
	var rID int
	for idx, rel := range rels.Relationships {
		ID, _ := strconv.Atoi(strings.TrimPrefix(rel.ID, "rId"))
		if ID > rID {
			rID = ID
		}
		if relType == rel.Type {
			if partName, ok := uniqPart[rel.Type]; ok {
				rels.Relationships[idx].Target = partName
				return rID
			}
		}
	}
	rID++
	var ID bytes.Buffer
	ID.WriteString("rId")
	ID.WriteString(strconv.Itoa(rID))
	rels.Relationships = append(rels.Relationships, xlsxRelationship{
		ID:         ID.String(),
		Type:       relType,
		Target:     target,
		TargetMode: targetMode,
	})
	f.Relationships.Store(relPath, rels)
	return rID
}

// UpdateLinkedValue fix linked values within a spreadsheet are not updating in
// Office Excel application. This function will be remove value tag when met a
// cell have a linked value. Reference
// https://social.technet.microsoft.com/Forums/office/en-US/e16bae1f-6a2c-4325-8013-e989a3479066/excel-2010-linked-cells-not-updating
//
// Notice: after opening generated workbook, Excel will update the linked value
// and generate a new value and will prompt to save the file or not.
//
// For example:
//
//	<row r="19" spans="2:2">
//	    <c r="B19">
//	        <f>SUM(Sheet2!D2,Sheet2!D11)</f>
//	        <v>100</v>
//	     </c>
//	</row>
//
// to
//
//	<row r="19" spans="2:2">
//	    <c r="B19">
//	        <f>SUM(Sheet2!D2,Sheet2!D11)</f>
//	    </c>
//	</row>
func (f *File) UpdateLinkedValue() error {
	wb := f.workbookReader()
	// recalculate formulas
	wb.CalcPr = nil
	for _, name := range f.GetSheetList() {
		ws, err := f.workSheetReader(name)
		if err != nil {
			if err.Error() == newNotWorksheetError(name).Error() {
				continue
			}
			return err
		}
		for indexR := range ws.SheetData.Row {
			for indexC, col := range ws.SheetData.Row[indexR].C {
				if col.F != nil && col.V != "" {
					ws.SheetData.Row[indexR].C[indexC].V = ""
					ws.SheetData.Row[indexR].C[indexC].T = ""
				}
			}
		}
	}
	return nil
}

// AddVBAProject provides the method to add vbaProject.bin file which contains
// functions and/or macros. The file extension should be .xlsm. For example:
//
//	if err := f.SetSheetPrOptions("Sheet1", excelize.CodeName("Sheet1")); err != nil {
//	    fmt.Println(err)
//	}
//	if err := f.AddVBAProject("vbaProject.bin"); err != nil {
//	    fmt.Println(err)
//	}
//	if err := f.SaveAs("macros.xlsm"); err != nil {
//	    fmt.Println(err)
//	}
func (f *File) AddVBAProject(bin string) error {
	var err error
	// Check vbaProject.bin exists first.
	if _, err = os.Stat(bin); os.IsNotExist(err) {
		return fmt.Errorf("stat %s: no such file or directory", bin)
	}
	if path.Ext(bin) != ".bin" {
		return ErrAddVBAProject
	}
	wb := f.relsReader(f.getWorkbookRelsPath())
	wb.Lock()
	defer wb.Unlock()
	var rID int
	var ok bool
	for _, rel := range wb.Relationships {
		if rel.Target == "vbaProject.bin" && rel.Type == SourceRelationshipVBAProject {
			ok = true
			continue
		}
		t, _ := strconv.Atoi(strings.TrimPrefix(rel.ID, "rId"))
		if t > rID {
			rID = t
		}
	}
	rID++
	if !ok {
		wb.Relationships = append(wb.Relationships, xlsxRelationship{
			ID:     "rId" + strconv.Itoa(rID),
			Target: "vbaProject.bin",
			Type:   SourceRelationshipVBAProject,
		})
	}
	file, _ := ioutil.ReadFile(filepath.Clean(bin))
	f.Pkg.Store("xl/vbaProject.bin", file)
	return err
}

// setContentTypePartProjectExtensions provides a function to set the content
// type for relationship parts and the main document part.
func (f *File) setContentTypePartProjectExtensions(contentType string) {
	var ok bool
	content := f.contentTypesReader()
	content.Lock()
	defer content.Unlock()
	for _, v := range content.Defaults {
		if v.Extension == "bin" {
			ok = true
		}
	}
	for idx, o := range content.Overrides {
		if o.PartName == "/xl/workbook.xml" {
			content.Overrides[idx].ContentType = contentType
		}
	}
	if !ok {
		content.Defaults = append(content.Defaults, xlsxDefault{
			Extension:   "bin",
			ContentType: ContentTypeVBA,
		})
	}
}
