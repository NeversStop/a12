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
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

// SetWorkbookProps provides a function to sets workbook properties.
func (f *File) SetWorkbookProps(opts *WorkbookPropsOptions) error {
	wb := f.workbookReader()
	if wb.WorkbookPr == nil {
		wb.WorkbookPr = new(xlsxWorkbookPr)
	}
	if opts == nil {
		return nil
	}
	if opts.Date1904 != nil {
		wb.WorkbookPr.Date1904 = *opts.Date1904
	}
	if opts.FilterPrivacy != nil {
		wb.WorkbookPr.FilterPrivacy = *opts.FilterPrivacy
	}
	if opts.CodeName != nil {
		wb.WorkbookPr.CodeName = *opts.CodeName
	}
	return nil
}

// GetWorkbookProps provides a function to gets workbook properties.
func (f *File) GetWorkbookProps() (WorkbookPropsOptions, error) {
	wb, opts := f.workbookReader(), WorkbookPropsOptions{}
	if wb.WorkbookPr != nil {
		opts.Date1904 = boolPtr(wb.WorkbookPr.Date1904)
		opts.FilterPrivacy = boolPtr(wb.WorkbookPr.FilterPrivacy)
		opts.CodeName = stringPtr(wb.WorkbookPr.CodeName)
	}
	return opts, nil
}

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
func (f *File) getWorkbookPath() (path string) {
	if rels := f.relsReader("_rels/.rels"); rels != nil {
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
func (f *File) getWorkbookRelsPath() (path string) {
	wbPath := f.getWorkbookPath()
	wbDir := filepath.Dir(wbPath)
	if wbDir == "." {
		path = "_rels/" + filepath.Base(wbPath) + ".rels"
		return
	}
	path = strings.TrimPrefix(filepath.Dir(wbPath)+"/_rels/"+filepath.Base(wbPath)+".rels", "/")
	return
}

// workbookReader provides a function to get the pointer to the workbook.xml
// structure after deserialization.
func (f *File) workbookReader() *xlsxWorkbook {
	var err error
	if f.WorkBook == nil {
		wbPath := f.getWorkbookPath()
		f.WorkBook = new(xlsxWorkbook)
		if _, ok := f.xmlAttr[wbPath]; !ok {
			d := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(wbPath))))
			f.xmlAttr[wbPath] = append(f.xmlAttr[wbPath], getRootElement(d)...)
			f.addNameSpaces(wbPath, SourceRelationship)
		}
		if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML(wbPath)))).
			Decode(f.WorkBook); err != nil && err != io.EOF {
			log.Printf("xml decode error: %s", err)
		}
	}
	return f.WorkBook
}

// workBookWriter provides a function to save workbook.xml after serialize
// structure.
func (f *File) workBookWriter() {
	if f.WorkBook != nil {
		if f.WorkBook.DecodeAlternateContent != nil {
			f.WorkBook.AlternateContent = &xlsxAlternateContent{
				Content: f.WorkBook.DecodeAlternateContent.Content,
				XMLNSMC: SourceRelationshipCompatibility.Value,
			}
		}
		f.WorkBook.DecodeAlternateContent = nil
		output, _ := xml.Marshal(f.WorkBook)
		f.saveFileList(f.getWorkbookPath(), replaceRelationshipsBytes(f.replaceNameSpaceBytes(f.getWorkbookPath(), output)))
	}
}
