// Copyright 2016 - 2023 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to and
// read from XLAM / XLSM / XLSX / XLTM / XLTX files. Supports reading and
// writing spreadsheet documents generated by Microsoft Excel™ 2007 and later.
// Supports complex components by high compatibility, and provided streaming
// API for generating or reading data from a worksheet with huge amounts of
// data. This library needs Go version 1.16 or later.

package excelize

import (
	"archive/zip"
	"bytes"
	"container/list"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ReadZipReader extract spreadsheet with given options.
func (f *File) ReadZipReader(r *zip.Reader) (map[string][]byte, int, error) {
	var (
		err     error
		docPart = map[string]string{
			"[content_types].xml":  defaultXMLPathContentTypes,
			"xl/sharedstrings.xml": defaultXMLPathSharedStrings,
		}
		fileList   = make(map[string][]byte, len(r.File))
		worksheets int
		unzipSize  int64
	)
	for _, v := range r.File {
		fileSize := v.FileInfo().Size()
		unzipSize += fileSize
		if unzipSize > f.options.UnzipSizeLimit {
			return fileList, worksheets, newUnzipSizeLimitError(f.options.UnzipSizeLimit)
		}
		fileName := strings.ReplaceAll(v.Name, "\\", "/")
		if partName, ok := docPart[strings.ToLower(fileName)]; ok {
			fileName = partName
		}
		if strings.EqualFold(fileName, defaultXMLPathSharedStrings) && fileSize > f.options.UnzipXMLSizeLimit {
			tempFile, err := f.unzipToTemp(v)
			if tempFile != "" {
				f.tempFiles.Store(fileName, tempFile)
			}
			if err == nil {
				continue
			}
		}
		if strings.HasPrefix(strings.ToLower(fileName), "xl/worksheets/sheet") {
			worksheets++
			if fileSize > f.options.UnzipXMLSizeLimit && !v.FileInfo().IsDir() {
				tempFile, err := f.unzipToTemp(v)
				if tempFile != "" {
					f.tempFiles.Store(fileName, tempFile)
				}
				if err == nil {
					continue
				}
			}
		}
		if fileList[fileName], err = readFile(v); err != nil {
			return nil, 0, err
		}
	}
	return fileList, worksheets, nil
}

// unzipToTemp unzip the zip entity to the system temporary directory and
// returned the unzipped file path.
func (f *File) unzipToTemp(zipFile *zip.File) (string, error) {
	tmp, err := os.CreateTemp(os.TempDir(), "excelize-")
	if err != nil {
		return "", err
	}
	rc, err := zipFile.Open()
	if err != nil {
		return tmp.Name(), err
	}
	if _, err = io.Copy(tmp, rc); err != nil {
		return tmp.Name(), err
	}
	if err = rc.Close(); err != nil {
		return tmp.Name(), err
	}
	return tmp.Name(), tmp.Close()
}

// readXML provides a function to read XML content as bytes.
func (f *File) readXML(name string) []byte {
	if content, _ := f.Pkg.Load(name); content != nil {
		return content.([]byte)
	}
	if content, ok := f.streams[name]; ok {
		return content.rawData.buf.Bytes()
	}
	return []byte{}
}

// readBytes read file as bytes by given path.
func (f *File) readBytes(name string) []byte {
	content := f.readXML(name)
	if len(content) != 0 {
		return content
	}
	file, err := f.readTemp(name)
	if err != nil {
		return content
	}
	content, _ = io.ReadAll(file)
	f.Pkg.Store(name, content)
	_ = file.Close()
	return content
}

// readTemp read file from system temporary directory by given path.
func (f *File) readTemp(name string) (file *os.File, err error) {
	path, ok := f.tempFiles.Load(name)
	if !ok {
		return
	}
	file, err = os.Open(path.(string))
	return
}

// saveFileList provides a function to update given file content in file list
// of spreadsheet.
func (f *File) saveFileList(name string, content []byte) {
	f.Pkg.Store(name, append([]byte(xml.Header), content...))
}

// Read file content as string in an archive file.
func readFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	dat := make([]byte, 0, file.FileInfo().Size())
	buff := bytes.NewBuffer(dat)
	_, _ = io.Copy(buff, rc)
	return buff.Bytes(), rc.Close()
}

// SplitCellName splits cell name to column name and row number.
//
// Example:
//
//	excelize.SplitCellName("AK74") // return "AK", 74, nil
func SplitCellName(cell string) (string, int, error) {
	alpha := func(r rune) bool {
		return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') || (r == 36)
	}
	if strings.IndexFunc(cell, alpha) == 0 {
		i := strings.LastIndexFunc(cell, alpha)
		if i >= 0 && i < len(cell)-1 {
			col, rowStr := strings.ReplaceAll(cell[:i+1], "$", ""), cell[i+1:]
			if row, err := strconv.Atoi(rowStr); err == nil && row > 0 {
				return col, row, nil
			}
		}
	}
	return "", -1, newInvalidCellNameError(cell)
}

// JoinCellName joins cell name from column name and row number.
func JoinCellName(col string, row int) (string, error) {
	normCol := strings.Map(func(rune rune) rune {
		switch {
		case 'A' <= rune && rune <= 'Z':
			return rune
		case 'a' <= rune && rune <= 'z':
			return rune - 32
		}
		return -1
	}, col)
	if len(col) == 0 || len(col) != len(normCol) {
		return "", newInvalidColumnNameError(col)
	}
	if row < 1 {
		return "", newInvalidRowNumberError(row)
	}
	return normCol + strconv.Itoa(row), nil
}

// ColumnNameToNumber provides a function to convert Excel sheet column name
// (case-insensitive) to int. The function returns an error if column name
// incorrect.
//
// Example:
//
//	excelize.ColumnNameToNumber("AK") // returns 37, nil
func ColumnNameToNumber(name string) (int, error) {
	if len(name) == 0 {
		return -1, newInvalidColumnNameError(name)
	}
	col := 0
	multi := 1
	for i := len(name) - 1; i >= 0; i-- {
		r := name[i]
		if r >= 'A' && r <= 'Z' {
			col += int(r-'A'+1) * multi
		} else if r >= 'a' && r <= 'z' {
			col += int(r-'a'+1) * multi
		} else {
			return -1, newInvalidColumnNameError(name)
		}
		multi *= 26
	}
	if col > MaxColumns {
		return -1, ErrColumnNumber
	}
	return col, nil
}

// ColumnNumberToName provides a function to convert the integer to Excel
// sheet column title.
//
// Example:
//
//	excelize.ColumnNumberToName(37) // returns "AK", nil
func ColumnNumberToName(num int) (string, error) {
	if num < MinColumns || num > MaxColumns {
		return "", ErrColumnNumber
	}
	var col string
	for num > 0 {
		col = string(rune((num-1)%26+65)) + col
		num = (num - 1) / 26
	}
	return col, nil
}

// CellNameToCoordinates converts alphanumeric cell name to [X, Y] coordinates
// or returns an error.
//
// Example:
//
//	excelize.CellNameToCoordinates("A1") // returns 1, 1, nil
//	excelize.CellNameToCoordinates("Z3") // returns 26, 3, nil
func CellNameToCoordinates(cell string) (int, int, error) {
	colName, row, err := SplitCellName(cell)
	if err != nil {
		return -1, -1, newCellNameToCoordinatesError(cell, err)
	}
	if row > TotalRows {
		return -1, -1, ErrMaxRows
	}
	col, err := ColumnNameToNumber(colName)
	return col, row, err
}

// CoordinatesToCellName converts [X, Y] coordinates to alpha-numeric cell
// name or returns an error.
//
// Example:
//
//	excelize.CoordinatesToCellName(1, 1) // returns "A1", nil
//	excelize.CoordinatesToCellName(1, 1, true) // returns "$A$1", nil
func CoordinatesToCellName(col, row int, abs ...bool) (string, error) {
	if col < 1 || row < 1 {
		return "", newCoordinatesToCellNameError(col, row)
	}
	if row > TotalRows {
		return "", ErrMaxRows
	}
	sign := ""
	for _, a := range abs {
		if a {
			sign = "$"
		}
	}
	colName, err := ColumnNumberToName(col)
	return sign + colName + sign + strconv.Itoa(row), err
}

// rangeRefToCoordinates provides a function to convert range reference to a
// pair of coordinates.
func rangeRefToCoordinates(ref string) ([]int, error) {
	rng := strings.Split(strings.ReplaceAll(ref, "$", ""), ":")
	if len(rng) < 2 {
		return nil, ErrParameterInvalid
	}
	return cellRefsToCoordinates(rng[0], rng[1])
}

// cellRefsToCoordinates provides a function to convert cell range to a
// pair of coordinates.
func cellRefsToCoordinates(firstCell, lastCell string) ([]int, error) {
	coordinates := make([]int, 4)
	var err error
	coordinates[0], coordinates[1], err = CellNameToCoordinates(firstCell)
	if err != nil {
		return coordinates, err
	}
	coordinates[2], coordinates[3], err = CellNameToCoordinates(lastCell)
	return coordinates, err
}

// sortCoordinates provides a function to correct the cell range, such
// correct C1:B3 to B1:C3.
func sortCoordinates(coordinates []int) error {
	if len(coordinates) != 4 {
		return ErrCoordinates
	}
	if coordinates[2] < coordinates[0] {
		coordinates[2], coordinates[0] = coordinates[0], coordinates[2]
	}
	if coordinates[3] < coordinates[1] {
		coordinates[3], coordinates[1] = coordinates[1], coordinates[3]
	}
	return nil
}

// coordinatesToRangeRef provides a function to convert a pair of coordinates
// to range reference.
func (f *File) coordinatesToRangeRef(coordinates []int, abs ...bool) (string, error) {
	if len(coordinates) != 4 {
		return "", ErrCoordinates
	}
	firstCell, err := CoordinatesToCellName(coordinates[0], coordinates[1], abs...)
	if err != nil {
		return "", err
	}
	lastCell, err := CoordinatesToCellName(coordinates[2], coordinates[3], abs...)
	if err != nil {
		return "", err
	}
	return firstCell + ":" + lastCell, err
}

// getDefinedNameRefTo convert defined name to reference range.
func (f *File) getDefinedNameRefTo(definedNameName, currentSheet string) (refTo string) {
	var workbookRefTo, worksheetRefTo string
	for _, definedName := range f.GetDefinedName() {
		if definedName.Name == definedNameName {
			// worksheet scope takes precedence over scope workbook when both definedNames exist
			if definedName.Scope == "Workbook" {
				workbookRefTo = definedName.RefersTo
			}
			if definedName.Scope == currentSheet {
				worksheetRefTo = definedName.RefersTo
			}
		}
	}
	refTo = workbookRefTo
	if worksheetRefTo != "" {
		refTo = worksheetRefTo
	}
	return
}

// flatSqref convert reference sequence to cell reference list.
func (f *File) flatSqref(sqref string) (cells map[int][][]int, err error) {
	var coordinates []int
	cells = make(map[int][][]int)
	for _, ref := range strings.Fields(sqref) {
		rng := strings.Split(ref, ":")
		switch len(rng) {
		case 1:
			var col, row int
			col, row, err = CellNameToCoordinates(rng[0])
			if err != nil {
				return
			}
			cells[col] = append(cells[col], []int{col, row})
		case 2:
			if coordinates, err = rangeRefToCoordinates(ref); err != nil {
				return
			}
			_ = sortCoordinates(coordinates)
			for c := coordinates[0]; c <= coordinates[2]; c++ {
				for r := coordinates[1]; r <= coordinates[3]; r++ {
					cells[c] = append(cells[c], []int{c, r})
				}
			}
		}
	}
	return
}

// inCoordinates provides a method to check if a coordinate is present in
// coordinates array, and return the index of its location, otherwise
// return -1.
func inCoordinates(a [][]int, x []int) int {
	for idx, n := range a {
		if x[0] == n[0] && x[1] == n[1] {
			return idx
		}
	}
	return -1
}

// inStrSlice provides a method to check if an element is present in an array,
// and return the index of its location, otherwise return -1.
func inStrSlice(a []string, x string, caseSensitive bool) int {
	for idx, n := range a {
		if !caseSensitive && strings.EqualFold(x, n) {
			return idx
		}
		if x == n {
			return idx
		}
	}
	return -1
}

// inFloat64Slice provides a method to check if an element is present in a
// float64 array, and return the index of its location, otherwise return -1.
func inFloat64Slice(a []float64, x float64) int {
	for idx, n := range a {
		if x == n {
			return idx
		}
	}
	return -1
}

// boolPtr returns a pointer to a bool with the given value.
func boolPtr(b bool) *bool { return &b }

// intPtr returns a pointer to an int with the given value.
func intPtr(i int) *int { return &i }

// uintPtr returns a pointer to an int with the given value.
func uintPtr(i uint) *uint { return &i }

// float64Ptr returns a pointer to a float64 with the given value.
func float64Ptr(f float64) *float64 { return &f }

// stringPtr returns a pointer to a string with the given value.
func stringPtr(s string) *string { return &s }

// Value extracts string data type text from a attribute value.
func (avb *attrValString) Value() string {
	if avb != nil && avb.Val != nil {
		return *avb.Val
	}
	return ""
}

// Value extracts boolean data type value from a attribute value.
func (avb *attrValBool) Value() bool {
	if avb != nil && avb.Val != nil {
		return *avb.Val
	}
	return false
}

// Value extracts float64 data type numeric from a attribute value.
func (attr *attrValFloat) Value() float64 {
	if attr != nil && attr.Val != nil {
		return *attr.Val
	}
	return 0
}

// MarshalXML convert the boolean data type to literal values 0 or 1 on
// serialization.
func (avb attrValBool) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	attr := xml.Attr{
		Name: xml.Name{
			Space: start.Name.Space,
			Local: "val",
		},
		Value: "0",
	}
	if avb.Val != nil {
		if *avb.Val {
			attr.Value = "1"
		} else {
			attr.Value = "0"
		}
	}
	start.Attr = []xml.Attr{attr}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// UnmarshalXML convert the literal values true, false, 1, 0 of the XML
// attribute to boolean data type on deserialization.
func (avb *attrValBool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err != nil {
			return err
		}
		found := false
		switch t.(type) {
		case xml.StartElement:
			return ErrAttrValBool
		case xml.EndElement:
			found = true
		}
		if found {
			break
		}
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "val" {
			if attr.Value == "" {
				val := true
				avb.Val = &val
			} else {
				val, err := strconv.ParseBool(attr.Value)
				if err != nil {
					return err
				}
				avb.Val = &val
			}
			return nil
		}
	}
	defaultVal := true
	avb.Val = &defaultVal
	return nil
}

// MarshalXML encodes ext element with specified namespace attributes on
// serialization.
func (ext xlsxExt) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = ext.xmlns
	return e.EncodeElement(decodeExt{URI: ext.URI, Content: ext.Content}, start)
}

// UnmarshalXML extracts ext element attributes namespace by giving XML decoder
// on deserialization.
func (ext *xlsxExt) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "uri" {
			continue
		}
		if attr.Name.Space == "xmlns" {
			attr.Name.Space = ""
			attr.Name.Local = "xmlns:" + attr.Name.Local
		}
		ext.xmlns = append(ext.xmlns, attr)
	}
	e := &decodeExt{}
	if err := d.DecodeElement(&e, &start); err != nil {
		return err
	}
	ext.URI, ext.Content = e.URI, e.Content
	return nil
}

// namespaceStrictToTransitional provides a method to convert Strict and
// Transitional namespaces.
func namespaceStrictToTransitional(content []byte) []byte {
	namespaceTranslationDic := map[string]string{
		StrictNameSpaceDocumentPropertiesVariantTypes: NameSpaceDocumentPropertiesVariantTypes.Value,
		StrictNameSpaceDrawingMLMain:                  NameSpaceDrawingMLMain,
		StrictNameSpaceExtendedProperties:             NameSpaceExtendedProperties,
		StrictNameSpaceSpreadSheet:                    NameSpaceSpreadSheet.Value,
		StrictSourceRelationship:                      SourceRelationship.Value,
		StrictSourceRelationshipChart:                 SourceRelationshipChart,
		StrictSourceRelationshipComments:              SourceRelationshipComments,
		StrictSourceRelationshipExtendProperties:      SourceRelationshipExtendProperties,
		StrictSourceRelationshipImage:                 SourceRelationshipImage,
		StrictSourceRelationshipOfficeDocument:        SourceRelationshipOfficeDocument,
	}
	for s, n := range namespaceTranslationDic {
		content = bytesReplace(content, []byte(s), []byte(n), -1)
	}
	return content
}

// bytesReplace replace source bytes with given target.
func bytesReplace(s, source, target []byte, n int) []byte {
	if n == 0 {
		return s
	}

	if len(source) < len(target) {
		return bytes.Replace(s, source, target, n)
	}

	if n < 0 {
		n = len(s)
	}

	var wid, i, j, w int
	for i, j = 0, 0; i < len(s) && j < n; j++ {
		wid = bytes.Index(s[i:], source)
		if wid < 0 {
			break
		}

		w += copy(s[w:], s[i:i+wid])
		w += copy(s[w:], target)
		i += wid + len(source)
	}

	w += copy(s[w:], s[i:])
	return s[:w]
}

// genSheetPasswd provides a method to generate password for worksheet
// protection by given plaintext. When an Excel sheet is being protected with
// a password, a 16-bit (two byte) long hash is generated. To verify a
// password, it is compared to the hash. Obviously, if the input data volume
// is great, numerous passwords will match the same hash. Here is the
// algorithm to create the hash value:
//
// take the ASCII values of all characters shift left the first character 1 bit,
// the second 2 bits and so on (use only the lower 15 bits and rotate all higher bits,
// the highest bit of the 16-bit value is always 0 [signed short])
// XOR all these values
// XOR the count of characters
// XOR the constant 0xCE4B
func genSheetPasswd(plaintext string) string {
	var password int64 = 0x0000
	var charPos uint = 1
	for _, v := range plaintext {
		value := int64(v) << charPos
		charPos++
		rotatedBits := value >> 15 // rotated bits beyond bit 15
		value &= 0x7fff            // first 15 bits
		password ^= value | rotatedBits
	}
	password ^= int64(len(plaintext))
	password ^= 0xCE4B
	return strings.ToUpper(strconv.FormatInt(password, 16))
}

// getRootElement extract root element attributes by given XML decoder.
func getRootElement(d *xml.Decoder) []xml.Attr {
	tokenIdx := 0
	for {
		token, _ := d.Token()
		if token == nil {
			break
		}
		switch startElement := token.(type) {
		case xml.StartElement:
			tokenIdx++
			if tokenIdx == 1 {
				return startElement.Attr
			}
		}
	}
	return nil
}

// genXMLNamespace generate serialized XML attributes with a multi namespace
// by given element attributes.
func genXMLNamespace(attr []xml.Attr) string {
	var rootElement string
	for _, v := range attr {
		if lastSpace := getXMLNamespace(v.Name.Space, attr); lastSpace != "" {
			if lastSpace == NameSpaceXML {
				lastSpace = "xml"
			}
			rootElement += fmt.Sprintf("%s:%s=\"%s\" ", lastSpace, v.Name.Local, v.Value)
			continue
		}
		rootElement += fmt.Sprintf("%s=\"%s\" ", v.Name.Local, v.Value)
	}
	return strings.TrimSpace(rootElement) + ">"
}

// getXMLNamespace extract XML namespace from specified element name and attributes.
func getXMLNamespace(space string, attr []xml.Attr) string {
	for _, attribute := range attr {
		if attribute.Value == space {
			return attribute.Name.Local
		}
	}
	return space
}

// replaceNameSpaceBytes provides a function to replace the XML root element
// attribute by the given component part path and XML content.
func (f *File) replaceNameSpaceBytes(path string, contentMarshal []byte) []byte {
	sourceXmlns := []byte(`xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	targetXmlns := []byte(templateNamespaceIDMap)
	if attrs, ok := f.xmlAttr.Load(path); ok {
		targetXmlns = []byte(genXMLNamespace(attrs.([]xml.Attr)))
	}
	return bytesReplace(contentMarshal, sourceXmlns, bytes.ReplaceAll(targetXmlns, []byte(" mc:Ignorable=\"r\""), []byte{}), -1)
}

// addNameSpaces provides a function to add an XML attribute by the given
// component part path.
func (f *File) addNameSpaces(path string, ns xml.Attr) {
	exist := false
	mc := false
	ignore := -1
	if attrs, ok := f.xmlAttr.Load(path); ok {
		for i, attr := range attrs.([]xml.Attr) {
			if attr.Name.Local == ns.Name.Local && attr.Name.Space == ns.Name.Space {
				exist = true
			}
			if attr.Name.Local == "Ignorable" && getXMLNamespace(attr.Name.Space, attrs.([]xml.Attr)) == "mc" {
				ignore = i
			}
			if attr.Name.Local == "mc" && attr.Name.Space == "xmlns" {
				mc = true
			}
		}
	}
	if !exist {
		attrs, _ := f.xmlAttr.Load(path)
		if attrs == nil {
			attrs = []xml.Attr{}
		}
		attrs = append(attrs.([]xml.Attr), ns)
		f.xmlAttr.Store(path, attrs)
		if !mc {
			attrs = append(attrs.([]xml.Attr), SourceRelationshipCompatibility)
			f.xmlAttr.Store(path, attrs)
		}
		if ignore == -1 {
			attrs = append(attrs.([]xml.Attr), xml.Attr{
				Name:  xml.Name{Local: "Ignorable", Space: "mc"},
				Value: ns.Name.Local,
			})
			f.xmlAttr.Store(path, attrs)
			return
		}
		f.setIgnorableNameSpace(path, ignore, ns)
	}
}

// setIgnorableNameSpace provides a function to set XML namespace as ignorable
// by the given attribute.
func (f *File) setIgnorableNameSpace(path string, index int, ns xml.Attr) {
	ignorableNS := []string{"c14", "cdr14", "a14", "pic14", "x14", "xdr14", "x14ac", "dsp", "mso14", "dgm14", "x15", "x12ac", "x15ac", "xr", "xr2", "xr3", "xr4", "xr5", "xr6", "xr7", "xr8", "xr9", "xr10", "xr11", "xr12", "xr13", "xr14", "xr15", "x15", "x16", "x16r2", "mo", "mx", "mv", "o", "v"}
	xmlAttrs, _ := f.xmlAttr.Load(path)
	if inStrSlice(strings.Fields(xmlAttrs.([]xml.Attr)[index].Value), ns.Name.Local, true) == -1 && inStrSlice(ignorableNS, ns.Name.Local, true) != -1 {
		xmlAttrs.([]xml.Attr)[index].Value = strings.TrimSpace(fmt.Sprintf("%s %s", xmlAttrs.([]xml.Attr)[index].Value, ns.Name.Local))
		f.xmlAttr.Store(path, xmlAttrs)
	}
}

// addSheetNameSpace add XML attribute for worksheet.
func (f *File) addSheetNameSpace(sheet string, ns xml.Attr) {
	name, _ := f.getSheetXMLPath(sheet)
	f.addNameSpaces(name, ns)
}

// isNumeric determines whether an expression is a valid numeric type and get
// the precision for the numeric.
func isNumeric(s string) (bool, int, float64) {
	if strings.Contains(s, "_") {
		return false, 0, 0
	}
	var decimal big.Float
	_, ok := decimal.SetString(s)
	if !ok {
		return false, 0, 0
	}
	var noScientificNotation string
	flt, _ := decimal.Float64()
	noScientificNotation = strconv.FormatFloat(flt, 'f', -1, 64)
	return true, len(strings.ReplaceAll(noScientificNotation, ".", "")), flt
}

var (
	bstrExp       = regexp.MustCompile(`_x[a-fA-F\d]{4}_`)
	bstrEscapeExp = regexp.MustCompile(`x[a-fA-F\d]{4}_`)
)

// bstrUnmarshal parses the binary basic string, this will trim escaped string
// literal which not permitted in an XML 1.0 document. The basic string
// variant type can store any valid Unicode character. Unicode's characters
// that cannot be directly represented in XML as defined by the XML 1.0
// specification, shall be escaped using the Unicode numerical character
// representation escape character format _xHHHH_, where H represents a
// hexadecimal character in the character's value. For example: The Unicode
// character 8 is not permitted in an XML 1.0 document, so it shall be
// escaped as _x0008_. To store the literal form of an escape sequence, the
// initial underscore shall itself be escaped (i.e. stored as _x005F_). For
// example: The string literal _x0008_ would be stored as _x005F_x0008_.
func bstrUnmarshal(s string) (result string) {
	matches, l, cursor := bstrExp.FindAllStringSubmatchIndex(s, -1), len(s), 0
	for _, match := range matches {
		result += s[cursor:match[0]]
		subStr := s[match[0]:match[1]]
		if subStr == "_x005F_" {
			cursor = match[1]
			result += "_"
			continue
		}
		if bstrExp.MatchString(subStr) {
			cursor = match[1]
			v, _ := strconv.Unquote(`"\u` + s[match[0]+2:match[1]-1] + `"`)
			result += v
		}
	}
	if cursor < l {
		result += s[cursor:]
	}
	return result
}

// bstrMarshal encode the escaped string literal which not permitted in an XML
// 1.0 document.
func bstrMarshal(s string) (result string) {
	matches, l, cursor := bstrExp.FindAllStringSubmatchIndex(s, -1), len(s), 0
	for _, match := range matches {
		result += s[cursor:match[0]]
		subStr := s[match[0]:match[1]]
		if subStr == "_x005F_" {
			cursor = match[1]
			if match[1]+6 <= l && bstrEscapeExp.MatchString(s[match[1]:match[1]+6]) {
				_, err := strconv.Unquote(`"\u` + s[match[1]+1:match[1]+5] + `"`)
				if err == nil {
					result += subStr + "x005F" + subStr
					continue
				}
			}
			result += subStr + "x005F_"
			continue
		}
		if bstrExp.MatchString(subStr) {
			cursor = match[1]
			if _, err := strconv.Unquote(`"\u` + s[match[0]+2:match[1]-1] + `"`); err == nil {
				result += "_x005F" + subStr
				continue
			}
		}
	}
	if cursor < l {
		result += s[cursor:]
	}
	return result
}

// newRat converts decimals to rational fractions with the required precision.
func newRat(n float64, iterations int64, prec float64) *big.Rat {
	x := int64(math.Floor(n))
	y := n - float64(x)
	rat := continuedFraction(y, 1, iterations, prec)
	return rat.Add(rat, new(big.Rat).SetInt64(x))
}

// continuedFraction returns rational from decimal with the continued fraction
// algorithm.
func continuedFraction(n float64, i int64, limit int64, prec float64) *big.Rat {
	if i >= limit || n <= prec {
		return big.NewRat(0, 1)
	}
	inverted := 1 / n
	y := int64(math.Floor(inverted))
	x := inverted - float64(y)
	ratY := new(big.Rat).SetInt64(y)
	ratNext := continuedFraction(x, i+1, limit, prec)
	res := ratY.Add(ratY, ratNext)
	res = res.Inv(res)
	return res
}

// Stack defined an abstract data type that serves as a collection of elements.
type Stack struct {
	list *list.List
}

// NewStack create a new stack.
func NewStack() *Stack {
	l := list.New()
	return &Stack{l}
}

// Push a value onto the top of the stack.
func (stack *Stack) Push(value interface{}) {
	stack.list.PushBack(value)
}

// Pop the top item of the stack and return it.
func (stack *Stack) Pop() interface{} {
	e := stack.list.Back()
	if e != nil {
		stack.list.Remove(e)
		return e.Value
	}
	return nil
}

// Peek view the top item on the stack.
func (stack *Stack) Peek() interface{} {
	e := stack.list.Back()
	if e != nil {
		return e.Value
	}
	return nil
}

// Len return the number of items in the stack.
func (stack *Stack) Len() int {
	return stack.list.Len()
}

// Empty the stack.
func (stack *Stack) Empty() bool {
	return stack.list.Len() == 0
}
