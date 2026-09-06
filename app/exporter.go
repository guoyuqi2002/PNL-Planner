package main

import (
	"fmt"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
)

// colLetter converts a column index (0-based) to a letter (A, B, ..., Z, AA, AB, ...)
func colLetter(idx int) string {
	result := ""
	for {
		result = string(rune('A'+idx%26)) + result
		idx = idx/26 - 1
		if idx < 0 {
			break
		}
	}
	return result
}

// Style colors
var headerFill = "D9E1F2" // light blue header
var totalFill = "FFF2CC"  // light yellow totals

// Borders (shared by all styles)
func stdBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
}

// strPtr converts a string to a pointer (used for CustomNumFmt)
func strPtr(s string) *string { return &s }

// ============================================================
// Split table style factories (Microsoft YaHei 10, centered) -
// all use CustomNumFmt (excelize built-in NumFmt does not work here)
// ============================================================

// splitHeaderStyle is the split table header (bold + light blue)
func splitHeaderStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Family: "Microsoft YaHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    stdBorder(),
	})
	return s
}

// splitTotalStyle is the split table total (bold + light yellow)
func splitTotalStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Family: "Microsoft YaHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{totalFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    stdBorder(),
	})
	return s
}

// splitTotalMoney is the total-row money style (bold + CNY #,##0.00)
func splitTotalMoney(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 10, Family: "Microsoft YaHei"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{totalFill}, Pattern: 1},
		CustomNumFmt: strPtr(`¥#,##0.00`),
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:       stdBorder(),
	})
	return s
}

// splitTotalRate is the total-row rate style (bold + 0.00%)
func splitTotalRate(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 10, Family: "Microsoft YaHei"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{totalFill}, Pattern: 1},
		CustomNumFmt: strPtr(`0.00%`),
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:       stdBorder(),
	})
	return s
}

// splitTotalInt is the total-row integer style (bold + 0)
func splitTotalInt(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 10, Family: "Microsoft YaHei"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{totalFill}, Pattern: 1},
		CustomNumFmt: strPtr(`0`),
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:       stdBorder(),
	})
	return s
}

// splitMoneyStyle is the money style (CNY #,##0.00)
func splitMoneyStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10, Family: "Microsoft YaHei"},
		CustomNumFmt: strPtr(`¥#,##0.00`),
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:       stdBorder(),
	})
	return s
}

// splitRateStyle is the percentage style (0.00%)
func splitRateStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10, Family: "Microsoft YaHei"},
		CustomNumFmt: strPtr(`0.00%`),
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:       stdBorder(),
	})
	return s
}

// splitIntStyle is the integer style (order qty etc.)
func splitIntStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10, Family: "Microsoft YaHei"},
		CustomNumFmt: strPtr(`0`),
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:       stdBorder(),
	})
	return s
}

// splitTextStyle is the plain-text style (shop / SKU name etc.)
func splitTextStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Microsoft YaHei"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    stdBorder(),
	})
	return s
}

// ============================================================
// Drill-down table style factories (Microsoft YaHei 11)
// ============================================================

// pivotHeaderStyle is the drill-down header (bold + light blue)
func pivotHeaderStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: "Microsoft YaHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerFill}, Pattern: 1},
		Border:    stdBorder(),
	})
	return s
}

// pivotAmountStyle is the drill-down amount style (integer #,##0, not centered)
func pivotAmountStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: "Microsoft YaHei"},
		CustomNumFmt: strPtr(`#,##0`),
		Border:       stdBorder(),
	})
	return s
}

// pivotRateStyle is the drill-down rate style (0.0%, not centered)
func pivotRateStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: "Microsoft YaHei"},
		CustomNumFmt: strPtr(`0.0%`),
		Border:       stdBorder(),
	})
	return s
}

// pivotTextStyle is the drill-down plain-text style (fee item names etc., not centered)
func pivotTextStyle(f *excelize.File) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 11, Family: "Microsoft YaHei"},
		Border: stdBorder(),
	})
	return s
}

// ============================================================
// Legacy compatibility helpers (transitional)
// ============================================================
func newStyle(f *excelize.File, bold bool, size int, fill string) int {
	s, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: bold, Size: float64(size), Family: "Microsoft YaHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{fill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    stdBorder(),
	})
	return s
}
func moneyStyle(f *excelize.File) int  { return splitMoneyStyle(f) }
func rateStyle(f *excelize.File) int   { return splitRateStyle(f) }
func borderStyle(f *excelize.File) int { return splitTextStyle(f) }

// ExportSplitTable exports the interest e-commerce business plan split table.
// Returns the saved file path.
func ExportSplitTable(yearMonth string) (string, error) {
	rows := QueryByMonth(yearMonth)
	f := excelize.NewFile()
	defer f.Close()

	headerStyle := splitHeaderStyle(f)
	totalStyle := splitTotalStyle(f)
	totalMoney := splitTotalMoney(f)
	totalRate := splitTotalRate(f)
	totalInt := splitTotalInt(f)
	moneySt := splitMoneyStyle(f)
	rateSt := splitRateStyle(f)
	textSt := splitTextStyle(f)
	_ = moneySt
	_ = totalMoney
	_ = totalRate
	_ = totalInt

	// ==== Sheet 1: Interest summary ====
	f.SetSheetName("Sheet1", "Summary")
	f.SetCellValue("Summary", "A1", "Interest E-commerce Estimated P&L Summary")
	// Header row 3
	for i, col := range SummaryCols {
		cell := fmt.Sprintf("%s3", colLetter(i))
		f.SetCellValue("Summary", cell, col)
		f.SetCellStyle("Summary", cell, cell, headerStyle)
	}

	// Compute each dimension
	allRecs := []ComputeResult{}
	for _, r := range rows {
		allRecs = append(allRecs, computeFromRow(r))
	}
	dimFilter := func(dept, ch string) []ComputeResult {
		var out []ComputeResult
		for i, r := range rows {
			if dept != "" && r.Department != dept {
				continue
			}
			if ch != "" && r.Channel != ch {
				continue
			}
			out = append(out, allRecs[i])
		}
		return out
	}
	type summaryRow struct {
		label1, label2 string
		recs           []ComputeResult
		isTotal        bool
	}
	var sRows []summaryRow
	sRows = append(sRows, summaryRow{"Interest Summary", "", allRecs, true})
	for _, dept := range Departments {
		sRows = append(sRows, summaryRow{dept + " Summary", "", dimFilter(dept, ""), false})
	}
	for _, ch := range Channels {
		sRows = append(sRows, summaryRow{ch, ch + " Total", dimFilter("", ch), true})
		for _, dept := range Departments {
			sRows = append(sRows, summaryRow{"", dept, dimFilter(dept, ch), false})
		}
	}

	// Column types: 0=text 1=money (CNY) 2=rate (%) 3=integer
	colType := []int{0, 0, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2}

	r := 4
	for _, sr := range sRows {
		agg := Aggregate(sr.recs)
		cr := agg["company_rev"]
		vals := make([]interface{}, 22)
		vals[0], vals[1] = sr.label1, sr.label2
		vals[2] = agg["platform_rev_gsv"] // summary column C: platform revenue gross GSV
		vals[3] = cr
		vals[4] = agg["std_cost_total"]
		vals[5] = agg["gross_rate"]
		vals[6] = agg["tp12"]
		vals[7] = agg["tp22"]
		vals[8] = agg["cp12"]
		vals[9] = agg["cp51"]
		vals[10] = agg["cp52"]
		vals[11] = agg["cp53"]
		vals[12] = 0.0
		if cr != 0 {
			vals[13] = agg["ap"] / cr
			vals[14] = agg["tp_total"] / cr
			vals[15] = agg["cp_total"] / cr
			vals[16] = agg["pp"] / cr
			vals[17] = agg["mp"] / cr
			vals[18] = agg["fp"] / cr // FP rate (auto: gross profit x 0.0156)
			vals[19] = agg["ae"] / cr // AE rate (company revenue x 1.08%)
			vals[20] = agg["assess_net"] / cr
			vals[21] = agg["op_net"] / cr
		}
		for i, v := range vals {
			cell := fmt.Sprintf("%s%d", colLetter(i), r)
			f.SetCellValue("Summary", cell, v)
			// Apply style by column type
			var st int
			if sr.isTotal {
				st = totalStyle
			} else {
				switch colType[i] {
				case 2:
					st = rateSt
				case 1:
					st = moneySt
				default:
					st = textSt
				}
			}
			f.SetCellStyle("Summary", cell, cell, st)
		}
		r++
	}

	// ==== Sheets 2-5: channel split ====
	intSt := splitIntStyle(f)
	for _, ch := range Channels {
		sheetName := ch + " Split"
		f.NewSheet(sheetName)
		rowIdx := 1
		for _, dept := range Departments {
			modes := GetModes(dept, ch)
			for _, mode := range modes {
				title := GetBlockTitle(dept, mode)
				// Title row
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), title)
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), textSt)
				rowIdx += 2 // leave a blank row
				// Header row
				for i, col := range SplitCols {
					cell := fmt.Sprintf("%s%d", colLetter(i), rowIdx)
					f.SetCellValue(sheetName, cell, col)
					f.SetCellStyle(sheetName, cell, cell, headerStyle)
				}
				rowIdx++
				// Shop details
				blockRecs := []ComputeResult{}
				shops := GetShops(ch)
				blockFirstRow := rowIdx // block's first shop row (including blank placeholders)
				hasAnyData := false
				for _, shop := range shops {
					shopRows := []ConfigRow{}
					for _, rw := range rows {
						if rw.Department == dept && rw.Channel == ch && rw.Mode == mode && rw.Shop == shop {
							shopRows = append(shopRows, rw)
						}
					}
					if len(shopRows) == 0 {
						// Blank placeholder row (plain text style)
						f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), shop)
						for c := 1; c < 17; c++ {
							cell := fmt.Sprintf("%s%d", colLetter(c), rowIdx)
							f.SetCellStyle(sheetName, cell, cell, textSt)
						}
						rowIdx++
						continue
					}
					hasAnyData = true
					first := true
					for _, rw := range shopRows {
						rec := computeFromRow(rw)
						blockRecs = append(blockRecs, rec)
						// Standard cost E: unit standard cost (per piece), computed for
						// single and multiple materials
						stdCostVal := getComboUnitCost(rw.SubItems)
						setSplitRow(f, sheetName, rowIdx, shop, first, rw.ComboName, rec, rw, stdCostVal, moneySt, rateSt, intSt, textSt)
						rowIdx++
						first = false
					}
				}
				// Total row (SUM range = all shop rows of the block, including blank placeholders)
				if hasAnyData {
					firstData := blockFirstRow
					lastData := rowIdx - 1
					f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), "Total")
					f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), totalStyle)
					f.SetCellStyle(sheetName, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), totalStyle)
					// Money columns (bold + CNY)
					moneyCols := []string{"C", "D", "E", "F", "H", "I", "J", "K", "L", "P"}
					for _, c := range moneyCols {
						cell := fmt.Sprintf("%s%d", c, rowIdx)
						f.SetCellFormula(sheetName, cell, fmt.Sprintf("SUM(%s%d:%s%d)", c, firstData, c, lastData))
						f.SetCellStyle(sheetName, cell, cell, totalMoney)
					}
					// Discount rate G = F/D (bold + %)
					cellG := fmt.Sprintf("G%d", rowIdx)
					f.SetCellFormula(sheetName, cellG, fmt.Sprintf("IF(D%d=0,0,F%d/D%d)", rowIdx, rowIdx, rowIdx))
					f.SetCellStyle(sheetName, cellG, cellG, totalRate)
					// Order qty O = SUM (bold + integer)
					cellO := fmt.Sprintf("O%d", rowIdx)
					f.SetCellFormula(sheetName, cellO, fmt.Sprintf("SUM(O%d:O%d)", firstData, lastData))
					f.SetCellStyle(sheetName, cellO, cellO, totalInt)
					// AOV N = D/O (bold + CNY)
					cellN := fmt.Sprintf("N%d", rowIdx)
					f.SetCellFormula(sheetName, cellN, fmt.Sprintf("IF(O%d=0,0,D%d/O%d)", rowIdx, rowIdx, rowIdx))
					f.SetCellStyle(sheetName, cellN, cellN, totalMoney)
					// Gross margin Q = (D-P)/D (bold + %)
					cellQ := fmt.Sprintf("Q%d", rowIdx)
					f.SetCellFormula(sheetName, cellQ, fmt.Sprintf("IF(D%d=0,0,(D%d-P%d)/D%d)", rowIdx, rowIdx, rowIdx, rowIdx))
					f.SetCellStyle(sheetName, cellQ, cellQ, totalRate)
					// M (Non-HQ) empty + standard cost E is already a money amount
					f.SetCellStyle(sheetName, fmt.Sprintf("M%d", rowIdx), fmt.Sprintf("M%d", rowIdx), totalStyle)
				} else {
					f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), "Total")
					f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("Q%d", rowIdx), totalStyle)
				}
				rowIdx += 2
			}
		}
	}

	// Save
	ts := time.Now().Format("0102_1504")
	filename := fmt.Sprintf("Split_%s_%s.xlsx", yearMonth, ts)
	dir := exportDir()
	path := dir + "/" + filename
	if err := saveExcelWithCalc(f, path); err != nil {
		return "", err
	}
	return path, nil
}

// setSplitRow writes one split table detail row (for generating new tables).
// Input columns get values, derived columns get formulas (same as the sample), keeping the
// in-sheet calculation chain correct.
// Columns: A(0) Shop  B(1) Product  C(2) Platform revenue  D(3) Company revenue [formula]
// E(4) Standard cost  F(5) Discount amount [formula]  G(6) Discount rate  H(7) Traffic
// I(8) Influencer commission [formula]  J(9) Agency commission  K(10) Slot  L(11) Service fee
// M(12) Non-HQ  N(13) AOV  O(14) Order qty [formula]  P(15) Standard cost total [formula]
// Q(16) Gross margin [formula]
func setSplitRow(f *excelize.File, sheet string, row int, shop string, first bool,
	comboName string, rec ComputeResult, rw ConfigRow, stdCostVal interface{},
	moneySt, rateSt, intSt, textSt int) {
	r := fmt.Sprintf("%d", row)
	// Input columns: write value + style
	setVal := func(col int, val interface{}, styleType int) {
		cell := fmt.Sprintf("%s%d", colLetter(col), row)
		f.SetCellValue(sheet, cell, val)
		var st int
		switch styleType {
		case 1:
			st = moneySt
		case 2:
			st = rateSt
		case 3:
			st = intSt
		default:
			st = textSt
		}
		f.SetCellStyle(sheet, cell, cell, st)
	}
	// Derived columns: write formula + style
	setForm := func(col int, formula string, styleType int) {
		cell := fmt.Sprintf("%s%d", colLetter(col), row)
		f.SetCellFormula(sheet, cell, formula)
		var st int
		switch styleType {
		case 1:
			st = moneySt
		case 2:
			st = rateSt
		case 3:
			st = intSt
		default:
			st = textSt
		}
		f.SetCellStyle(sheet, cell, cell, st)
	}
	// A Shop / B Product name
	if first {
		setVal(0, shop, 0)
	} else {
		setVal(0, "", 0)
	}
	setVal(1, comboName, 0)
	// C Platform revenue (gross GSV, input) - back to gross
	setVal(2, rec.PlatformRevGsv, 1)
	// D Company revenue (formula: C/1.13/(1-G)) - gross C divided by 1.13 to excl. tax
	setForm(3, "C"+r+"/1.13/(1-G"+r+")", 1)
	// E Standard cost (input, unit cost)
	setVal(4, stdCostVal, 1)
	// F Discount amount (formula: (D*1.13-C)/1.13)
	setForm(5, "(D"+r+"*1.13-C"+r+")/1.13", 1)
	// G Discount rate (input)
	setVal(6, rec.DiscountRate, 2)
	// H Traffic spend (input)
	if rec.Cp51 != 0 {
		setVal(7, rec.Cp51, 1)
	} else {
		setVal(7, "", 0)
	}
	// I Influencer commission (formula: gross C * commission rate)
	if rec.PlatformRevGsv != 0 && (rec.Cp52-rw.Cp52Slot) != 0 {
		cp52Rate := (rec.Cp52 - rw.Cp52Slot) / rec.PlatformRevGsv
		setForm(8, fmt.Sprintf("C%s*%g", r, cp52Rate), 1)
	} else {
		setVal(8, "", 0)
	}
	// J Agency commission (input)
	if rec.Cp53 != 0 {
		setVal(9, rec.Cp53, 1)
	} else {
		setVal(9, "", 0)
	}
	// K Slot fee (input)
	if rw.Cp52Slot != 0 {
		setVal(10, rw.Cp52Slot, 1)
	} else {
		setVal(10, "", 0)
	}
	// L Service fee (input)
	if rec.Tp12 != 0 {
		setVal(11, rec.Tp12, 1)
	} else {
		setVal(11, "", 0)
	}
	// M Non-HQ (empty)
	setVal(12, "", 0)
	// N AOV (input)
	setVal(13, rec.Aov, 1)
	// O Order qty (formula: IFERROR(D/N,"0"))
	setForm(14, fmt.Sprintf(`IFERROR(D%s/N%s,"0")`, r, r), 3)
	// P Standard cost total (formula: E*O)
	setForm(15, fmt.Sprintf("E%s*O%s", r, r), 1)
	// Q Gross margin (formula: (D-P)/D)
	setForm(16, fmt.Sprintf("(D%s-P%s)/D%s", r, r, r), 2)
}

// ExportPivotTable exports the influencer distribution budget drill-down table (per department)
func ExportPivotTable(department string) (string, error) {
	rows := QueryByDepartment(department)
	f := excelize.NewFile()
	defer f.Close()

	sheetName := department + " Budget Pivot"
	f.SetSheetName("Sheet1", sheetName)

	headerStyle := pivotHeaderStyle(f)
	amountSt := pivotAmountStyle(f)
	rateSt := pivotRateStyle(f)
	textSt := pivotTextStyle(f)
	months := AllMonthsFY2026()

	// Dynamic blocks: top level = department name + the channel blocks that department
	// actually has
	topBlock := PivotTopBlockName(department)
	blocks := append([]string{topBlock}, PivotChannelBlocks(department)...)
	rowIdx := 1
	for _, blockLabel := range blocks {
		// 3 header rows
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), headerStyle)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), "Fee Item")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), blockLabel)
		f.SetCellStyle(sheetName, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("C%d", rowIdx), headerStyle)
		// R1: don't repeat the block name past column D (the channel name appears only once
		// in column B); just apply the header style
		col := 4 // from column D
		for range months {
			cell := fmt.Sprintf("%s%d", colLetter(col-1), rowIdx)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
			col += 2
		}
		// Odd columns of R1 (rate columns) also get the header style
		col = 5
		for range months {
			cell := fmt.Sprintf("%s%d", colLetter(col-1), rowIdx)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
			col += 2
		}
		rowIdx++
		// R2: fiscal-year total + each month (short month label)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), "FY Total")
		f.SetCellStyle(sheetName, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("C%d", rowIdx), headerStyle)
		col = 4
		for _, m := range months {
			t, _ := time.Parse("2006-01", m)
			monthLabel := t.Format("Jan")
			cell := fmt.Sprintf("%s%d", colLetter(col-1), rowIdx)
			f.SetCellValue(sheetName, cell, monthLabel)
			// Merge 2 columns (amount + rate) to show the month
			f.SetCellStyle(sheetName, cell, fmt.Sprintf("%s%d", colLetter(col), rowIdx), headerStyle)
			col += 2
		}
		rowIdx++
		// R3: Amount / Rate column titles
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), "Amount")
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), "Rate")
		f.SetCellStyle(sheetName, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("C%d", rowIdx), headerStyle)
		col = 4
		for i := 0; i < len(months); i++ {
			amtCell := fmt.Sprintf("%s%d", colLetter(col-1), rowIdx)
			rateCell := fmt.Sprintf("%s%d", colLetter(col), rowIdx)
			f.SetCellValue(sheetName, amtCell, "Amount")
			f.SetCellValue(sheetName, rateCell, "Rate")
			f.SetCellStyle(sheetName, amtCell, amtCell, headerStyle)
			f.SetCellStyle(sheetName, rateCell, rateCell, headerStyle)
			col += 2
		}
		rowIdx++

		// Filter this block's data (top-level summary = all / channel block = filter by Channel)
		var blockRows []ConfigRow
		isTopBlock := blockLabel == topBlock
		if isTopBlock {
			blockRows = rows // all channels
		} else {
			for _, r := range rows {
				if r.Channel == blockLabel {
					blockRows = append(blockRows, r)
				}
			}
		}

		// Group by month
		byMonth := map[string][]ComputeResult{}
		for _, r := range blockRows {
			rec := computeFromRow(r)
			byMonth[r.YearMonth] = append(byMonth[r.YearMonth], rec)
		}
		// Full year
		var yearRecs []ComputeResult
		for _, recs := range byMonth {
			yearRecs = append(yearRecs, recs...)
		}
		yearAgg := Aggregate(yearRecs)
		yearCr := yearAgg["company_rev"]
		monthAggs := map[string]map[string]float64{}
		for m, recs := range byMonth {
			monthAggs[m] = Aggregate(recs)
		}

		// 11 fee items
		feeItems := []struct {
			name     string
			yearVal  float64
			yearRate interface{}
			monthVal func(m string) float64
			monthRate func(m string) interface{}
		}{
			{"Revenue", yearAgg["company_rev"], nil,
				func(m string) float64 { return monthAggs[m]["company_rev"] },
				func(m string) interface{} { return nil }},
			{"Gross Profit", yearAgg["gross_profit"], yearAgg["gross_rate"],
				func(m string) float64 { return monthAggs[m]["gross_profit"] },
				func(m string) interface{} { return monthAggs[m]["gross_rate"] }},
			{"AP", yearAgg["ap"], safeDiv(yearAgg["ap"], yearCr),
				func(m string) float64 { return monthAggs[m]["ap"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["ap"], monthAggs[m]["company_rev"]) }},
			{"TP", yearAgg["tp_total"], safeDiv(yearAgg["tp_total"], yearCr),
				func(m string) float64 { return monthAggs[m]["tp_total"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["tp_total"], monthAggs[m]["company_rev"]) }},
			{"CP", yearAgg["cp_total"], safeDiv(yearAgg["cp_total"], yearCr),
				func(m string) float64 { return monthAggs[m]["cp_total"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["cp_total"], monthAggs[m]["company_rev"]) }},
			{"PP", yearAgg["pp"], safeDiv(yearAgg["pp"], yearCr),
				func(m string) float64 { return monthAggs[m]["pp"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["pp"], monthAggs[m]["company_rev"]) }},
			{"MP", yearAgg["mp"], safeDiv(yearAgg["mp"], yearCr),
				func(m string) float64 { return monthAggs[m]["mp"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["mp"], monthAggs[m]["company_rev"]) }},
			{"FP", yearAgg["fp"], safeDiv(yearAgg["fp"], yearCr),
				func(m string) float64 { return monthAggs[m]["fp"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["fp"], monthAggs[m]["company_rev"]) }},
			{"Assessed Net Profit", yearAgg["assess_net"], safeDiv(yearAgg["assess_net"], yearCr),
				func(m string) float64 { return monthAggs[m]["assess_net"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["assess_net"], monthAggs[m]["company_rev"]) }},
			{"Non-HQ Support", 0, 0.0,
				func(m string) float64 { return 0 },
				func(m string) interface{} { return 0.0 }},
			{"Operating Net Profit", yearAgg["op_net"], safeDiv(yearAgg["op_net"], yearCr),
				func(m string) float64 { return monthAggs[m]["op_net"] },
				func(m string) interface{} { return safeDiv(monthAggs[m]["op_net"], monthAggs[m]["company_rev"]) }},
		}

		for _, fi := range feeItems {
			feeRow := rowIdx // current fee-item row number
			isAssess := fi.name == "Assessed Net Profit"
			isOpNet := fi.name == "Operating Net Profit"
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", feeRow), fi.name)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", feeRow), fmt.Sprintf("A%d", feeRow), textSt)
			// Fiscal-year total - Assessed/Operating Net Profit use formulas
			if isAssess {
				// B (assessed) = B(Gross Profit) - B(AP) - B(TP) - B(CP) - B(PP) - B(MP) - B(FP)
				// Gross Profit=feeRow-7, AP=feeRow-6, TP=feeRow-5, CP=feeRow-4, PP=feeRow-3, MP=feeRow-2, FP=feeRow-1
				f.SetCellFormula(sheetName, fmt.Sprintf("B%d", feeRow),
					fmt.Sprintf("B%d-B%d-B%d-B%d-B%d-B%d-B%d",
						feeRow-7, feeRow-6, feeRow-5, feeRow-4, feeRow-3, feeRow-2, feeRow-1))
			} else if isOpNet {
				// B (operating) = B(Assessed Net Profit) + B(Non-HQ Support)
				f.SetCellFormula(sheetName, fmt.Sprintf("B%d", feeRow),
					fmt.Sprintf("B%d+B%d", feeRow-2, feeRow-1))
			} else if fi.yearVal != 0 {
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", feeRow), roundWan(fi.yearVal))
			}
			f.SetCellStyle(sheetName, fmt.Sprintf("B%d", feeRow), fmt.Sprintf("B%d", feeRow), amountSt)
			// C rate (full year)
			if isAssess {
				// C (assessed rate) = B(assessed) / B(Revenue); Revenue row = feeRow-8
				f.SetCellFormula(sheetName, fmt.Sprintf("C%d", feeRow),
					fmt.Sprintf("IF(B%d=0,0,B%d/B%d)", feeRow, feeRow, feeRow-8))
				f.SetCellStyle(sheetName, fmt.Sprintf("C%d", feeRow), fmt.Sprintf("C%d", feeRow), rateSt)
			} else if isOpNet {
				// C (operating rate) = B(operating) / B(Revenue); Revenue row = feeRow-10
				f.SetCellFormula(sheetName, fmt.Sprintf("C%d", feeRow),
					fmt.Sprintf("IF(B%d=0,0,B%d/B%d)", feeRow, feeRow, feeRow-10))
				f.SetCellStyle(sheetName, fmt.Sprintf("C%d", feeRow), fmt.Sprintf("C%d", feeRow), rateSt)
			} else if fi.yearRate != nil {
				f.SetCellValue(sheetName, fmt.Sprintf("C%d", feeRow), fi.yearRate)
				f.SetCellStyle(sheetName, fmt.Sprintf("C%d", feeRow), fmt.Sprintf("C%d", feeRow), rateSt)
			} else {
				f.SetCellStyle(sheetName, fmt.Sprintf("C%d", feeRow), fmt.Sprintf("C%d", feeRow), textSt)
			}
			// Each month - from column D, 2 columns per month
			col := 4
			for mi, m := range months {
				amtCol := colLetter(col - 1)
				rateCol := colLetter(col)
				amtCell := fmt.Sprintf("%s%d", amtCol, feeRow)
				rateCell := fmt.Sprintf("%s%d", rateCol, feeRow)
				hasMonthData := false
				if recs, ok := monthAggs[m]; ok && len(recs) > 0 {
					hasMonthData = true
				}
				if hasMonthData {
					if isAssess {
						// Monthly assessed net = Gross Profit - AP - TP - CP - PP - MP - FP (same column)
						f.SetCellFormula(sheetName, amtCell,
							fmt.Sprintf("%s%d-%s%d-%s%d-%s%d-%s%d-%s%d-%s%d",
								amtCol, feeRow-7, amtCol, feeRow-6, amtCol, feeRow-5,
								amtCol, feeRow-4, amtCol, feeRow-3, amtCol, feeRow-2, amtCol, feeRow-1))
					} else if isOpNet {
						// Monthly operating net = monthly assessed net + monthly Non-HQ Support
						f.SetCellFormula(sheetName, amtCell,
							fmt.Sprintf("%s%d+%s%d", amtCol, feeRow-2, amtCol, feeRow-1))
					} else {
						mv := fi.monthVal(m)
						if mv != 0 {
							f.SetCellValue(sheetName, amtCell, roundWan(mv))
						}
					}
					f.SetCellStyle(sheetName, amtCell, amtCell, amountSt)
					// Monthly rate
					if isAssess {
						// Monthly assessed rate = monthly assessed / monthly revenue (Revenue row = feeRow-8)
						f.SetCellFormula(sheetName, rateCell,
							fmt.Sprintf("IF(%s%d=0,0,%s%d/%s%d)", amtCol, feeRow-8, amtCol, feeRow, amtCol, feeRow-8))
						f.SetCellStyle(sheetName, rateCell, rateCell, rateSt)
					} else if isOpNet {
						// Monthly operating rate = monthly operating / monthly revenue (Revenue row = feeRow-10)
						f.SetCellFormula(sheetName, rateCell,
							fmt.Sprintf("IF(%s%d=0,0,%s%d/%s%d)", amtCol, feeRow-10, amtCol, feeRow, amtCol, feeRow-10))
						f.SetCellStyle(sheetName, rateCell, rateCell, rateSt)
					} else {
						mr := fi.monthRate(m)
						if mr != nil {
							f.SetCellValue(sheetName, rateCell, mr)
							f.SetCellStyle(sheetName, rateCell, rateCell, rateSt)
						} else {
							f.SetCellStyle(sheetName, rateCell, rateCell, textSt)
						}
					}
				} else {
					// No data for this month: formula rows are still written (referencing
					// empty cells = 0), plain rows stay empty
					if isAssess {
						f.SetCellFormula(sheetName, amtCell,
							fmt.Sprintf("%s%d-%s%d-%s%d-%s%d-%s%d-%s%d-%s%d",
								amtCol, feeRow-7, amtCol, feeRow-6, amtCol, feeRow-5,
								amtCol, feeRow-4, amtCol, feeRow-3, amtCol, feeRow-2, amtCol, feeRow-1))
						f.SetCellStyle(sheetName, amtCell, amtCell, amountSt)
						f.SetCellFormula(sheetName, rateCell,
							fmt.Sprintf("IF(%s%d=0,0,%s%d/%s%d)", amtCol, feeRow-8, amtCol, feeRow, amtCol, feeRow-8))
						f.SetCellStyle(sheetName, rateCell, rateCell, rateSt)
					} else if isOpNet {
						f.SetCellFormula(sheetName, amtCell,
							fmt.Sprintf("%s%d+%s%d", amtCol, feeRow-2, amtCol, feeRow-1))
						f.SetCellStyle(sheetName, amtCell, amtCell, amountSt)
						f.SetCellFormula(sheetName, rateCell,
							fmt.Sprintf("IF(%s%d=0,0,%s%d/%s%d)", amtCol, feeRow-10, amtCol, feeRow, amtCol, feeRow-10))
						f.SetCellStyle(sheetName, rateCell, rateCell, rateSt)
					} else {
						f.SetCellStyle(sheetName, amtCell, amtCell, amountSt)
						f.SetCellStyle(sheetName, rateCell, rateCell, textSt)
					}
				}
				_ = mi
				col += 2
			}
			rowIdx++
		}
		rowIdx += 2 // blank row between blocks
	}

	ts := time.Now().Format("0102_1504")
	filename := fmt.Sprintf("Pivot_%s_%s.xlsx", department, ts)
	dir := exportDir()
	path := dir + "/" + filename
	if err := saveExcelWithCalc(f, path); err != nil {
		return "", err
	}
	return path, nil
}

func safeDiv(a, b float64) interface{} {
	if b == 0 {
		return 0.0
	}
	return a / b
}

// yuanToWan converts CNY to 10k CNY (exact value kept; display digits controlled by NumFmt)
func yuanToWan(yuan float64) float64 {
	return yuan / 10000
}

// roundWan is kept for legacy callers (deprecated; use yuanToWan uniformly)
func roundWan(yuan float64) float64 {
	return yuanToWan(yuan)
}

// exportDir is the temp directory (no longer creates a Desktop folder automatically; the
// system temp directory is used instead). The final save location is chosen by the user
// via the SaveFileDialog.
var exportDirCache string

func exportDir() string {
	if exportDirCache != "" {
		return exportDirCache
	}
	exportDirCache = os.TempDir()
	return exportDirCache
}

// enableFullCalc forces a full formula recalculation when the workbook is opened.
// This fixes the excelize issue where formulas written by excelize don't recalculate
// when Excel opens the file (cells only update after double-clicking).
func enableFullCalc(f *excelize.File) {
	t := true
	_ = f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &t})
}

// saveExcelWithCalc saves the Excel file with recalculation-on-open enabled
func saveExcelWithCalc(f *excelize.File, path string) error {
	enableFullCalc(f)
	return f.SaveAs(path)
}
