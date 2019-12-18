// Copyright 2016 - 2019 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// Package excelize providing a set of functions that allow you to write to
// and read from XLSX files. Support reads and writes XLSX file generated by
// Microsoft Excel™ 2007 and later. Support save file without losing original
// charts of XLSX. This library needs Go version 1.10 or later.
//
// See https://xuri.me/excelize for more information about this package.
package excelize

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

// File define a populated XLSX file struct.
type File struct {
	checked          map[string]bool
	sheetMap         map[string]string
	CalcChain        *xlsxCalcChain
	Comments         map[string]*xlsxComments
	ContentTypes     *xlsxTypes
	Drawings         map[string]*xlsxWsDr
	Path             string
	SharedStrings    *xlsxSST
	Sheet            map[string]*xlsxWorksheet
	SheetCount       int
	Styles           *xlsxStyleSheet
	Theme            *xlsxTheme
	DecodeVMLDrawing map[string]*decodeVmlDrawing
	VMLDrawing       map[string]*vmlDrawing
	WorkBook         *xlsxWorkbook
	Relationships    map[string]*xlsxRelationships
	XLSX             map[string][]byte
}

// OpenFile take the name of an XLSX file and returns a populated XLSX file
// struct for it.
func OpenFile(filename string) (*File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	f, err := OpenReader(file)
	if err != nil {
		return nil, err
	}
	f.Path = filename
	return f, nil
}

// OpenReader take an io.Reader and return a populated XLSX file.
func OpenReader(r io.Reader) (*File, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		identifier := []byte{
			// checking protect workbook by [MS-OFFCRYPTO] - v20181211 3.1 FeatureIdentifier
			0x3c, 0x00, 0x00, 0x00, 0x4d, 0x00, 0x69, 0x00, 0x63, 0x00, 0x72, 0x00, 0x6f, 0x00, 0x73, 0x00,
			0x6f, 0x00, 0x66, 0x00, 0x74, 0x00, 0x2e, 0x00, 0x43, 0x00, 0x6f, 0x00, 0x6e, 0x00, 0x74, 0x00,
			0x61, 0x00, 0x69, 0x00, 0x6e, 0x00, 0x65, 0x00, 0x72, 0x00, 0x2e, 0x00, 0x44, 0x00, 0x61, 0x00,
			0x74, 0x00, 0x61, 0x00, 0x53, 0x00, 0x70, 0x00, 0x61, 0x00, 0x63, 0x00, 0x65, 0x00, 0x73, 0x00,
			0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		}
		if bytes.Contains(b, identifier) {
			return nil, errors.New("not support encrypted file currently")
		}
		return nil, err
	}

	file, sheetCount, err := ReadZipReader(zr)
	if err != nil {
		return nil, err
	}
	f := &File{
		checked:          make(map[string]bool),
		Comments:         make(map[string]*xlsxComments),
		Drawings:         make(map[string]*xlsxWsDr),
		Sheet:            make(map[string]*xlsxWorksheet),
		SheetCount:       sheetCount,
		DecodeVMLDrawing: make(map[string]*decodeVmlDrawing),
		VMLDrawing:       make(map[string]*vmlDrawing),
		Relationships:    make(map[string]*xlsxRelationships),
		XLSX:             file,
	}
	f.CalcChain = f.calcChainReader()
	f.sheetMap = f.getSheetMap()
	f.Styles = f.stylesReader()
	f.Theme = f.themeReader()
	return f, nil
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
		style, _ := f.NewStyle(`{"number_format": ` + strconv.Itoa(format) + `}`)
		f.SetCellStyle(sheet, axis, axis, style)
	}
	return err
}

// workSheetReader provides a function to get the pointer to the structure
// after deserialization by given worksheet name.
func (f *File) workSheetReader(sheet string) (xlsx *xlsxWorksheet, err error) {
	var (
		name string
		ok bool
		decoder *xml.Decoder
	)

	if name, ok = f.sheetMap[trimSheetName(sheet)]; !ok {
		err = fmt.Errorf("sheet %s is not exist", sheet)
		return
	}
	if xlsx = f.Sheet[name]; f.Sheet[name] == nil {
		xlsx = new(xlsxWorksheet)
		decoder = xml.NewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(name))))
		decoder.CharsetReader = CharsetReader
		if err = decoder.Decode(xlsx); err != nil {
			return xlsx, err
		}
		if f.checked == nil {
			f.checked = make(map[string]bool)
		}
		if ok = f.checked[name]; !ok {
			checkSheet(xlsx)
			checkRow(xlsx)
			f.checked[name] = true
		}
		f.Sheet[name] = xlsx
	}
	return
}

// checkSheet provides a function to fill each row element and make that is
// continuous in a worksheet of XML.
func checkSheet(xlsx *xlsxWorksheet) {
	row := len(xlsx.SheetData.Row)
	if row >= 1 {
		lastRow := xlsx.SheetData.Row[row-1].R
		if lastRow >= row {
			row = lastRow
		}
	}
	sheetData := xlsxSheetData{}
	existsRows := map[int]int{}
	for k := range xlsx.SheetData.Row {
		existsRows[xlsx.SheetData.Row[k].R] = k
	}
	for i := 0; i < row; i++ {
		_, ok := existsRows[i+1]
		if ok {
			sheetData.Row = append(sheetData.Row, xlsx.SheetData.Row[existsRows[i+1]])
		} else {
			sheetData.Row = append(sheetData.Row, xlsxRow{
				R: i + 1,
			})
		}
	}
	xlsx.SheetData = sheetData
}

// addRels provides a function to add relationships by given XML path,
// relationship type, target and target mode.
func (f *File) addRels(relPath, relType, target, targetMode string) int {
	rels := f.relsReader(relPath)
	rID := 0
	if rels == nil {
		rels = &xlsxRelationships{}
	}
	rID = len(rels.Relationships) + 1
	var ID bytes.Buffer
	ID.WriteString("rId")
	ID.WriteString(strconv.Itoa(rID))
	rels.Relationships = append(rels.Relationships, xlsxRelationship{
		ID:         ID.String(),
		Type:       relType,
		Target:     target,
		TargetMode: targetMode,
	})
	f.Relationships[relPath] = rels
	return rID
}

// replaceWorkSheetsRelationshipsNameSpaceBytes provides a function to replace
// xl/worksheets/sheet%d.xml XML tags to self-closing for compatible Microsoft
// Office Excel 2007.
func replaceWorkSheetsRelationshipsNameSpaceBytes(workbookMarshal []byte) []byte {
	var oldXmlns = []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	var newXmlns = []byte(`<worksheet xr:uid="{00000000-0001-0000-0000-000000000000}" xmlns:xr="http://schemas.microsoft.com/office/spreadsheetml/2014/revision" xmlns:xr3="http://schemas.microsoft.com/office/spreadsheetml/2016/revision3" xmlns:xr2="http://schemas.microsoft.com/office/spreadsheetml/2015/revision2" xmlns:xr6="http://schemas.microsoft.com/office/spreadsheetml/2016/revision6" xmlns:xr10="http://schemas.microsoft.com/office/spreadsheetml/2016/revision10" xmlns:x14="http://schemas.microsoft.com/office/spreadsheetml/2009/9/main" xmlns:x14ac="http://schemas.microsoft.com/office/spreadsheetml/2009/9/ac" xmlns:x15="http://schemas.microsoft.com/office/spreadsheetml/2010/11/main" mc:Ignorable="x14ac xr xr2 xr3 xr6 xr10 x15" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" xmlns:mx="http://schemas.microsoft.com/office/mac/excel/2008/main" xmlns:mv="urn:schemas-microsoft-com:mac:vml" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	workbookMarshal = bytes.Replace(workbookMarshal, oldXmlns, newXmlns, -1)
	return workbookMarshal
}

// replaceStyleRelationshipsNameSpaceBytes provides a function to replace
// xl/styles.xml XML tags to self-closing for compatible Microsoft Office
// Excel 2007.
func replaceStyleRelationshipsNameSpaceBytes(contentMarshal []byte) []byte {
	var oldXmlns = []byte(`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	var newXmlns = []byte(`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" mc:Ignorable="x14ac x16r2 xr xr9" xmlns:x14ac="http://schemas.microsoft.com/office/spreadsheetml/2009/9/ac" xmlns:x16r2="http://schemas.microsoft.com/office/spreadsheetml/2015/02/main" xmlns:xr="http://schemas.microsoft.com/office/spreadsheetml/2014/revision" xmlns:xr9="http://schemas.microsoft.com/office/spreadsheetml/2016/revision9">`)
	contentMarshal = bytes.Replace(contentMarshal, oldXmlns, newXmlns, -1)
	return contentMarshal
}

// UpdateLinkedValue fix linked values within a spreadsheet are not updating in
// Office Excel 2007 and 2010. This function will be remove value tag when met a
// cell have a linked value. Reference
// https://social.technet.microsoft.com/Forums/office/en-US/e16bae1f-6a2c-4325-8013-e989a3479066/excel-2010-linked-cells-not-updating
//
// Notice: after open XLSX file Excel will be update linked value and generate
// new value and will prompt save file or not.
//
// For example:
//
//    <row r="19" spans="2:2">
//        <c r="B19">
//            <f>SUM(Sheet2!D2,Sheet2!D11)</f>
//            <v>100</v>
//         </c>
//    </row>
//
// to
//
//    <row r="19" spans="2:2">
//        <c r="B19">
//            <f>SUM(Sheet2!D2,Sheet2!D11)</f>
//        </c>
//    </row>
//
func (f *File) UpdateLinkedValue() error {
	for _, name := range f.GetSheetMap() {
		xlsx, err := f.workSheetReader(name)
		if err != nil {
			return err
		}
		for indexR := range xlsx.SheetData.Row {
			for indexC, col := range xlsx.SheetData.Row[indexR].C {
				if col.F != nil && col.V != "" {
					xlsx.SheetData.Row[indexR].C[indexC].V = ""
					xlsx.SheetData.Row[indexR].C[indexC].T = ""
				}
			}
		}
	}
	return nil
}

// AddVBAProject provides the method to add vbaProject.bin file which contains
// functions and/or macros. The file extension should be .xlsm. For example:
//
//    err := f.SetSheetPrOptions("Sheet1", excelize.CodeName("Sheet1"))
//    if err != nil {
//        fmt.Println(err)
//    }
//    err = f.AddVBAProject("vbaProject.bin")
//    if err != nil {
//        fmt.Println(err)
//    }
//    err = f.SaveAs("macros.xlsm")
//    if err != nil {
//        fmt.Println(err)
//    }
//
func (f *File) AddVBAProject(bin string) error {
	var err error
	// Check vbaProject.bin exists first.
	if _, err = os.Stat(bin); os.IsNotExist(err) {
		return err
	}
	if path.Ext(bin) != ".bin" {
		return errors.New("unsupported VBA project extension")
	}
	f.setContentTypePartVBAProjectExtensions()
	wb := f.relsReader("xl/_rels/workbook.xml.rels")
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
	file, _ := ioutil.ReadFile(bin)
	f.XLSX["xl/vbaProject.bin"] = file
	return err
}

// setContentTypePartVBAProjectExtensions provides a function to set the
// content type for relationship parts and the main document part.
func (f *File) setContentTypePartVBAProjectExtensions() {
	var ok bool
	content := f.contentTypesReader()
	for _, v := range content.Defaults {
		if v.Extension == "bin" {
			ok = true
		}
	}
	for idx, o := range content.Overrides {
		if o.PartName == "/xl/workbook.xml" {
			content.Overrides[idx].ContentType = "application/vnd.ms-excel.sheet.macroEnabled.main+xml"
		}
	}
	if !ok {
		content.Defaults = append(content.Defaults, xlsxDefault{
			Extension:   "bin",
			ContentType: "application/vnd.ms-office.vbaProject",
		})
	}
}
