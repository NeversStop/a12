// Copyright 2016 - 2019 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to
// and read from XLSX files. Support reads and writes XLSX file generated by
// Microsoft Excel™ 2007 and later. Support save file without losing original
// charts of XLSX. This library needs Go version 1.10 or later.

package excelize

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
)

// SetDocProps provides a function to set document core properties. The
// properties that can be set are:
//
//     Property       | Description
//    ----------------+-----------------------------------------------------------------------------
//     Title          | The name given to the resource.
//                    |
//     Subject        | The topic of the content of the resource.
//                    |
//     Creator        | An entity primarily responsible for making the content of the resource.
//                    |
//     Keywords       | A delimited set of keywords to support searching and indexing. This is
//                    | typically a list of terms that are not available elsewhere in the properties.
//                    |
//     Description    | An explanation of the content of the resource.
//                    |
//     LastModifiedBy | The user who performed the last modification. The identification is
//                    | environment-specific.
//                    |
//     Language       | The language of the intellectual content of the resource.
//                    |
//     Identifier     | An unambiguous reference to the resource within a given context.
//                    |
//     Revision       | The topic of the content of the resource.
//                    |
//     ContentStatus  | The status of the content. For example: Values might include "Draft",
//                    | "Reviewed" and "Final"
//                    |
//     Category       | A categorization of the content of this package.
//                    |
//     Version        | The version number. This value is set by the user or by the application.
//
// For example:
//
//    err := f.SetDocProps(&excelize.DocProperties{
//        Category:       "category",
//        ContentStatus:  "Draft",
//        Created:        "2019-06-04T22:00:10Z",
//        Creator:        "Go Excelize",
//        Description:    "This file created by Go Excelize",
//        Identifier:     "xlsx",
//        Keywords:       "Spreadsheet",
//        LastModifiedBy: "Go Author",
//        Modified:       "2019-06-04T22:00:10Z",
//        Revision:       "0",
//        Subject:        "Test Subject",
//        Title:          "Test Title",
//        Language:       "en-US",
//        Version:        "1.0.0",
//    })
//
func (f *File) SetDocProps(docProperties *DocProperties) (err error) {
	var (
		core               *decodeCoreProperties
		newProps           *xlsxCoreProperties
		fields             []string
		output             []byte
		immutable, mutable reflect.Value
		field, val         string
	)

	core = new(decodeCoreProperties)
	if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML("docProps/core.xml")))).
		Decode(core); err != nil && err != io.EOF {
		err = fmt.Errorf("xml decode error: %s", err)
		return
	}
	newProps, err = &xlsxCoreProperties{
		Dc:             NameSpaceDublinCore,
		Dcterms:        NameSpaceDublinCoreTerms,
		Dcmitype:       NameSpaceDublinCoreMetadataIntiative,
		XSI:            NameSpaceXMLSchemaInstance,
		Title:          core.Title,
		Subject:        core.Subject,
		Creator:        core.Creator,
		Keywords:       core.Keywords,
		Description:    core.Description,
		LastModifiedBy: core.LastModifiedBy,
		Language:       core.Language,
		Identifier:     core.Identifier,
		Revision:       core.Revision,
		ContentStatus:  core.ContentStatus,
		Category:       core.Category,
		Version:        core.Version,
	}, nil
	newProps.Created.Text, newProps.Created.Type, newProps.Modified.Text, newProps.Modified.Type =
		core.Created.Text, core.Created.Type, core.Modified.Text, core.Modified.Type
	fields = []string{
		"Category", "ContentStatus", "Creator", "Description", "Identifier", "Keywords",
		"LastModifiedBy", "Revision", "Subject", "Title", "Language", "Version",
	}
	immutable, mutable = reflect.ValueOf(*docProperties), reflect.ValueOf(newProps).Elem()
	for _, field = range fields {
		if val = immutable.FieldByName(field).String(); val != "" {
			mutable.FieldByName(field).SetString(val)
		}
	}
	if docProperties.Created != "" {
		newProps.Created.Text = docProperties.Created
	}
	if docProperties.Modified != "" {
		newProps.Modified.Text = docProperties.Modified
	}
	output, err = xml.Marshal(newProps)
	f.saveFileList("docProps/core.xml", output)

	return
}

// GetDocProps provides a function to get document core properties.
func (f *File) GetDocProps() (ret *DocProperties, err error) {
	var core = new(decodeCoreProperties)

	if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(f.readXML("docProps/core.xml")))).
		Decode(core); err != nil && err != io.EOF {
		err = fmt.Errorf("xml decode error: %s", err)
		return
	}
	ret, err = &DocProperties{
		Category:       core.Category,
		ContentStatus:  core.ContentStatus,
		Created:        core.Created.Text,
		Creator:        core.Creator,
		Description:    core.Description,
		Identifier:     core.Identifier,
		Keywords:       core.Keywords,
		LastModifiedBy: core.LastModifiedBy,
		Modified:       core.Modified.Text,
		Revision:       core.Revision,
		Subject:        core.Subject,
		Title:          core.Title,
		Language:       core.Language,
		Version:        core.Version,
	}, nil

	return
}
