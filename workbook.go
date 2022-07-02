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
	"path/filepath"
	"strconv"
	"strings"
)

// WorkbookPrOption is an option of a view of a workbook. See SetWorkbookPrOptions().
type WorkbookPrOption interface {
	setWorkbookPrOption(pr *xlsxWorkbookPr)
}

// WorkbookPrOptionPtr is a writable WorkbookPrOption. See GetWorkbookPrOptions().
type WorkbookPrOptionPtr interface {
	WorkbookPrOption
	getWorkbookPrOption(pr *xlsxWorkbookPr)
}

type (
	// Date1904 is an option used for WorkbookPrOption, that indicates whether
	// to use a 1900 or 1904 date system when converting serial date-times in
	// the workbook to dates
	Date1904 bool
	// FilterPrivacy is an option used for WorkbookPrOption
	FilterPrivacy bool
)

// setWorkbook update workbook property of the spreadsheet. Maximum 31
// characters are allowed in sheet title.
func (f *File) setWorkbook(name string, sheetID, rid int) {
	content := f.workbookReader()
	content.Sheets.Sheet = append(content.Sheets.Sheet, xlsxSheet{
		Name:    trimSheetName(name),
		SheetID: sheetID,
		ID:      "rId" + strconv.Itoa(rid),
	})
}

// getWorkbookPath provides a function to get the path of the workbook.xml in
// the spreadsheet.
func (f *File) getWorkbookPath() (path string, err error) {
	var rels *xlsxRelationships
	rels, err = f.relsReader("_rels/.rels")
	if err != nil {
		return
	}
	if rels != nil {
		rels.Lock()
		defer rels.Unlock()
		for _, rel := range rels.Relationships {
			if rel.Type == SourceRelationshipOfficeDocument {
				path = strings.TrimPrefix(rel.Target, "/")
				return
			}
		}
	}
	return
}

// getWorkbookRelsPath provides a function to get the path of the workbook.xml.rels
// in the spreadsheet.
func (f *File) getWorkbookRelsPath() (path string, err error) {
	var wbPath string
	wbPath, err = f.getWorkbookPath()
	if err != nil {
		return
	}
	wbDir := filepath.Dir(wbPath)
	if wbDir == "." {
		path = "_rels/" + filepath.Base(wbPath) + ".rels"
		return
	}
	path = strings.TrimPrefix(filepath.Dir(wbPath)+"/_rels/"+filepath.Base(wbPath)+".rels", "/")
	return
}

// NewWorkbookReader provides a function to get the pointer to the workbook.xml
// structure after deserialization.
func (f *File) NewWorkbookReader() (*xlsxWorkbook, error) {
	var err error
	wbPath, err := f.getWorkbookPath()
	if err != nil {
		return nil, err
	}
	f.WorkBook = new(xlsxWorkbook)
	if _, ok := f.xmlAttr[wbPath]; !ok {
		d := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(wbPath))))
		f.xmlAttr[wbPath] = append(f.xmlAttr[wbPath], getRootElement(d)...)
		f.addNameSpaces(wbPath, SourceRelationship)
	}
	if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(wbPath)))).
		Decode(f.WorkBook); err != nil && err != io.EOF {
		return nil, fmt.Errorf("xml decode error: %w", err)
	}
	return f.WorkBook, err
}

// workbookReader provides a function to get the pointer to WorkBook.
func (f *File) workbookReader() *xlsxWorkbook {
	return f.WorkBook
}

// workBookWriter provides a function to save workbook.xml after serialize
// structure.
func (f *File) workBookWriter() error {
	if f.WorkBook != nil {
		if f.WorkBook.DecodeAlternateContent != nil {
			f.WorkBook.AlternateContent = &xlsxAlternateContent{
				Content: f.WorkBook.DecodeAlternateContent.Content,
				XMLNSMC: SourceRelationshipCompatibility.Value,
			}
		}
		f.WorkBook.DecodeAlternateContent = nil
		output, err := xml.Marshal(f.WorkBook)
		if err != nil {
			return err
		}
		wp, err := f.getWorkbookPath()
		if err != nil {
			return err
		}
		f.saveFileList(wp, replaceRelationshipsBytes(f.replaceNameSpaceBytes(wp, output)))
	}
	return nil
}

// SetWorkbookPrOptions provides a function to sets workbook properties.
//
// Available options:
//   Date1904(bool)
//   FilterPrivacy(bool)
//   CodeName(string)
func (f *File) SetWorkbookPrOptions(opts ...WorkbookPrOption) error {
	wb := f.workbookReader()
	pr := wb.WorkbookPr
	if pr == nil {
		pr = new(xlsxWorkbookPr)
		wb.WorkbookPr = pr
	}
	for _, opt := range opts {
		opt.setWorkbookPrOption(pr)
	}
	return nil
}

// setWorkbookPrOption implements the WorkbookPrOption interface.
func (o Date1904) setWorkbookPrOption(pr *xlsxWorkbookPr) {
	pr.Date1904 = bool(o)
}

// setWorkbookPrOption implements the WorkbookPrOption interface.
func (o FilterPrivacy) setWorkbookPrOption(pr *xlsxWorkbookPr) {
	pr.FilterPrivacy = bool(o)
}

// setWorkbookPrOption implements the WorkbookPrOption interface.
func (o CodeName) setWorkbookPrOption(pr *xlsxWorkbookPr) {
	pr.CodeName = string(o)
}

// GetWorkbookPrOptions provides a function to gets workbook properties.
//
// Available options:
//   Date1904(bool)
//   FilterPrivacy(bool)
//   CodeName(string)
func (f *File) GetWorkbookPrOptions(opts ...WorkbookPrOptionPtr) error {
	wb := f.workbookReader()
	pr := wb.WorkbookPr
	for _, opt := range opts {
		opt.getWorkbookPrOption(pr)
	}
	return nil
}

// getWorkbookPrOption implements the WorkbookPrOption interface and get the
// date1904 of the workbook.
func (o *Date1904) getWorkbookPrOption(pr *xlsxWorkbookPr) {
	if pr == nil {
		*o = false
		return
	}
	*o = Date1904(pr.Date1904)
}

// getWorkbookPrOption implements the WorkbookPrOption interface and get the
// filter privacy of the workbook.
func (o *FilterPrivacy) getWorkbookPrOption(pr *xlsxWorkbookPr) {
	if pr == nil {
		*o = false
		return
	}
	*o = FilterPrivacy(pr.FilterPrivacy)
}

// getWorkbookPrOption implements the WorkbookPrOption interface and get the
// code name of the workbook.
func (o *CodeName) getWorkbookPrOption(pr *xlsxWorkbookPr) {
	if pr == nil {
		*o = ""
		return
	}
	*o = CodeName(pr.CodeName)
}
