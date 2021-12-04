package excelize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjustMergeCells(t *testing.T) {
	f := NewFile()
	// testing adjustAutoFilter with illegal cell coordinates.
	assert.EqualError(t, f.adjustMergeCells(&xlsxWorksheet{
		MergeCells: &xlsxMergeCells{
			Cells: []*xlsxMergeCell{
				{
					Ref: "A:B1",
				},
			},
		},
	}, rows, 0, 0), `cannot convert cell "A" to coordinates: invalid cell name "A"`)
	assert.EqualError(t, f.adjustMergeCells(&xlsxWorksheet{
		MergeCells: &xlsxMergeCells{
			Cells: []*xlsxMergeCell{
				{
					Ref: "A1:B",
				},
			},
		},
	}, rows, 0, 0), `cannot convert cell "B" to coordinates: invalid cell name "B"`)
	assert.NoError(t, f.adjustMergeCells(&xlsxWorksheet{
		MergeCells: &xlsxMergeCells{
			Cells: []*xlsxMergeCell{
				{
					Ref: "A1:B1",
				},
			},
		},
	}, rows, 1, -1))
	assert.NoError(t, f.adjustMergeCells(&xlsxWorksheet{
		MergeCells: &xlsxMergeCells{
			Cells: []*xlsxMergeCell{
				{
					Ref: "A1:A2",
				},
			},
		},
	}, columns, 1, -1))
}

func TestAdjustMergeCellsPerformance(t *testing.T) {
	var cases []struct {
		lable  string
		ws     *xlsxWorksheet
		dir    adjustDirection
		num    int
		offset int
		expect string
	}
	f := NewFile()

	// testing insert
	cases = []struct {
		lable  string
		ws     *xlsxWorksheet
		dir    adjustDirection
		num    int
		offset int
		expect string
	}{
		{
			lable: "insert row on ref",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    rows,
			num:    2,
			offset: 1,
			expect: "A3:B4",
		},
		{
			lable: "insert row on bottom of ref",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    rows,
			num:    3,
			offset: 1,
			expect: "A2:B4",
		},
		{
			lable: "insert column on the left",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    columns,
			num:    1,
			offset: 1,
			expect: "B2:C3",
		},
	}
	for _, c := range cases {
		assert.NoError(t, f.adjustMergeCells(c.ws, c.dir, c.num, 1))
		assert.Equal(t, c.expect, c.ws.MergeCells.Cells[0].Ref, c.lable)
	}

	// testing delete
	cases = []struct {
		lable  string
		ws     *xlsxWorksheet
		dir    adjustDirection
		num    int
		offset int
		expect string
	}{
		{
			lable: "delete row on top of ref",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    rows,
			num:    2,
			offset: -1,
			expect: "A2:B2",
		},
		{
			lable: "delete row on bottom of ref",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    rows,
			num:    3,
			offset: -1,
			expect: "A2:B2",
		},
		{
			lable: "delete column on the ref left",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    columns,
			num:    1,
			offset: -1,
			expect: "A2:A3",
		},
		{
			lable: "delete column on the ref right",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B3",
						},
					},
				},
			},
			dir:    columns,
			num:    2,
			offset: -1,
			expect: "A2:A3",
		},
	}
	for _, c := range cases {
		assert.NoError(t, f.adjustMergeCells(c.ws, c.dir, c.num, -1))
		assert.Equal(t, c.expect, c.ws.MergeCells.Cells[0].Ref, c.lable)
	}

	// testing delete one row/column
	cases = []struct {
		lable  string
		ws     *xlsxWorksheet
		dir    adjustDirection
		num    int
		offset int
		expect string
	}{
		{
			lable: "delete one row ref",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "A2:B2",
						},
					},
				},
			},
			dir:    rows,
			num:    2,
			offset: -1,
		},
		{
			lable: "delete one column ref",
			ws: &xlsxWorksheet{
				MergeCells: &xlsxMergeCells{
					Cells: []*xlsxMergeCell{
						{
							Ref: "B2:B2",
						},
					},
				},
			},
			dir:    columns,
			num:    2,
			offset: -1,
		},
	}
	for _, c := range cases {
		assert.NoError(t, f.adjustMergeCells(c.ws, c.dir, c.num, -1))
		assert.Equal(t, 0, len(c.ws.MergeCells.Cells), c.lable)
	}

}

func TestAdjustAutoFilter(t *testing.T) {
	f := NewFile()
	assert.NoError(t, f.adjustAutoFilter(&xlsxWorksheet{
		SheetData: xlsxSheetData{
			Row: []xlsxRow{{Hidden: true, R: 2}},
		},
		AutoFilter: &xlsxAutoFilter{
			Ref: "A1:A3",
		},
	}, rows, 1, -1))
	// testing adjustAutoFilter with illegal cell coordinates.
	assert.EqualError(t, f.adjustAutoFilter(&xlsxWorksheet{
		AutoFilter: &xlsxAutoFilter{
			Ref: "A:B1",
		},
	}, rows, 0, 0), `cannot convert cell "A" to coordinates: invalid cell name "A"`)
	assert.EqualError(t, f.adjustAutoFilter(&xlsxWorksheet{
		AutoFilter: &xlsxAutoFilter{
			Ref: "A1:B",
		},
	}, rows, 0, 0), `cannot convert cell "B" to coordinates: invalid cell name "B"`)
}

func TestAdjustHelper(t *testing.T) {
	f := NewFile()
	f.NewSheet("Sheet2")
	f.Sheet.Store("xl/worksheets/sheet1.xml", &xlsxWorksheet{
		MergeCells: &xlsxMergeCells{Cells: []*xlsxMergeCell{{Ref: "A:B1"}}}})
	f.Sheet.Store("xl/worksheets/sheet2.xml", &xlsxWorksheet{
		AutoFilter: &xlsxAutoFilter{Ref: "A1:B"}})
	// testing adjustHelper with illegal cell coordinates.
	assert.EqualError(t, f.adjustHelper("Sheet1", rows, 0, 0), `cannot convert cell "A" to coordinates: invalid cell name "A"`)
	assert.EqualError(t, f.adjustHelper("Sheet2", rows, 0, 0), `cannot convert cell "B" to coordinates: invalid cell name "B"`)
	// testing adjustHelper on not exists worksheet.
	assert.EqualError(t, f.adjustHelper("SheetN", rows, 0, 0), "sheet SheetN is not exist")
}

func TestAdjustCalcChain(t *testing.T) {
	f := NewFile()
	f.CalcChain = &xlsxCalcChain{
		C: []xlsxCalcChainC{
			{R: "B2", I: 2}, {R: "B2", I: 1},
		},
	}
	assert.NoError(t, f.InsertCol("Sheet1", "A"))
	assert.NoError(t, f.InsertRow("Sheet1", 1))

	f.CalcChain.C[1].R = "invalid coordinates"
	assert.EqualError(t, f.InsertCol("Sheet1", "A"), `cannot convert cell "invalid coordinates" to coordinates: invalid cell name "invalid coordinates"`)
	f.CalcChain = nil
	assert.NoError(t, f.InsertCol("Sheet1", "A"))
}
