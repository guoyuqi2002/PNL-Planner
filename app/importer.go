package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var titlePattern = regexp.MustCompile(`^(Influencer Distribution|Innovation & Custom|Channel Operations)\s*[-—]\s*(.+)$`)

// Split table column classification (aligned with the sample workbook's formula structure)
// Input columns (0-based): user data is written directly into these columns
//   A(0) Shop  B(1) Product  C(2) Platform revenue  E(4) Standard cost  G(6) Discount rate
//   H(7) Traffic  I(8) Influencer commission  J(9) Agency commission  K(10) Slot  L(11) Service fee  N(13) AOV
// Formula columns (derived): reuse the sample formulas, do not write values
//   D(3) Company revenue = C/1.13/(1-G)  F(5) Discount amount = (D*1.13-C)/1.13
//   O(14) Order qty = D/N  P(15) Standard cost total = E*O  Q(16) Gross margin = (D-P)/D
// Note: I (influencer commission) has formulas on rows with data in the sample but not on
// empty rows -> treated as an input column and written directly
var splitInputCols = map[int]bool{
	0: true, 1: true, 2: true, 4: true, 6: true,
	7: true, 8: true, 9: true, 10: true, 11: true, 13: true,
}

// ScanConfigItem is one scanned position
type ScanConfigItem struct {
	YearMonth  string   `json:"year_month"`
	Department string   `json:"department"`
	Channel    string   `json:"channel"`
	Mode       string   `json:"mode"`
	Shop       string   `json:"shop"`
	HasData    bool     `json:"has_data"`
	SkuCount   int      `json:"sku_count"`
	ExcelRow   int      `json:"excel_row"`
	SkuNames   []string `json:"sku_names"`
}

// ScanResult is the scan result
type ScanResult struct {
	YearMonth      string              `json:"year_month"`
	ShopsByChannel map[string][]string `json:"shops_by_channel"`
	Configs        []ScanConfigItem    `json:"configs"`
}

// parseSheetChannel extracts the channel from a sheet name
func parseSheetChannel(sheetName string) string {
	for _, ch := range Channels {
		if strings.HasPrefix(sheetName, ch) {
			return ch
		}
	}
	return ""
}

// isTitleRow checks whether a row is a "Department - Mode" title row
func isTitleRow(val string) (string, string, bool) {
	if val == "" {
		return "", "", false
	}
	m := titlePattern.FindStringSubmatch(strings.TrimSpace(val))
	if m == nil {
		return "", "", false
	}
	dept, mode := m[1], m[2]
	for _, d := range Departments {
		if d == dept {
			return dept, mode, true
		}
	}
	return "", "", false
}

// ScanSplitExcel scans the interest e-commerce split table
func ScanSplitExcel(filePath, yearMonth string) (*ScanResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if yearMonth == "" {
		yearMonth = ""
	}

	result := &ScanResult{
		YearMonth:      yearMonth,
		ShopsByChannel: map[string][]string{},
		Configs:        []ScanConfigItem{},
	}
	for _, ch := range Channels {
		result.ShopsByChannel[ch] = []string{}
	}

	for _, sheetName := range f.GetSheetList() {
		channel := parseSheetChannel(sheetName)
		if channel == "" {
			continue
		}
		rows, _ := f.GetRows(sheetName)
		var curDept, curMode string
		shopsSeen := map[string]bool{}

		for rowIdx := 0; rowIdx < len(rows); rowIdx++ {
			row := rows[rowIdx]
			var aVal string
			if len(row) > 0 {
				aVal = strings.TrimSpace(row[0])
			}
			var bVal string
			if len(row) > 1 {
				bVal = strings.TrimSpace(row[1])
			}

			// Detect title rows
			if dept, mode, ok := isTitleRow(aVal); ok {
				curDept, curMode = dept, mode
				continue
			}
			// Skip headers / totals / empty rows
			if curDept == "" || aVal == "" || aVal == "Shop" || aVal == "Total" {
				continue
			}
			// Tolerate dirty data (e.g. "a"): short values that are not known shops may be
			// dirty data. Handled leniently here: treat everything as a shop name.

			shop := aVal
			if !shopsSeen[shop] {
				shopsSeen[shop] = true
				if !contains(result.ShopsByChannel[channel], shop) {
					result.ShopsByChannel[channel] = append(result.ShopsByChannel[channel], shop)
				}
			}

			hasData := bVal != ""
			var skuNames []string
			if hasData {
				skuNames = append(skuNames, bVal)
				// Look at the following rows (A empty + B non-empty)
				for nr := rowIdx + 1; nr < len(rows); nr++ {
					nrow := rows[nr]
					var na, nb string
					if len(nrow) > 0 {
						na = strings.TrimSpace(nrow[0])
					}
					if len(nrow) > 1 {
						nb = strings.TrimSpace(nrow[1])
					}
					if na != "" {
						break
					}
					if nb != "" {
						skuNames = append(skuNames, nb)
					}
				}
			}

			result.Configs = append(result.Configs, ScanConfigItem{
				YearMonth:  yearMonth,
				Department: curDept, Channel: channel, Mode: curMode, Shop: shop,
				HasData: hasData, SkuCount: len(skuNames),
				ExcelRow: rowIdx + 1, SkuNames: skuNames,
			})
		}
	}
	return result, nil
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// FillExcel fills new data precisely into the original Excel file
func FillExcel(filePath, yearMonth string, newData []ConfigRow, overwriteMap map[string]bool) (string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Split table styles (Microsoft YaHei 10, centered)
	moneySt := splitMoneyStyle(f)
	rateSt := splitRateStyle(f)
	intSt := splitIntStyle(f)
	textSt := splitTextStyle(f)

	// Group by channel
	byChannel := map[string][]ConfigRow{}
	for _, d := range newData {
		byChannel[d.Channel] = append(byChannel[d.Channel], d)
	}

	for channel, chData := range byChannel {
		sheetName := channel + " Split"
		sheetIdx, _ := f.GetSheetIndex(sheetName)
		if sheetIdx == -1 {
			continue
		}
		// Scan the block structure
		blocks := scanBlocks(f, sheetName)
		// Fill in
		for _, d := range chData {
			key := fmt.Sprintf("%s|%s|%s|%s", d.Department, d.Channel, d.Mode, d.Shop)
			allowOverwrite := overwriteMap[key]

			var block *blockInfo
			for i := range blocks {
				if blocks[i].dept == d.Department && blocks[i].mode == d.Mode {
					block = &blocks[i]
					break
				}
			}
			if block == nil {
				continue
			}
			var shopInfo *shopPos
			for i := range block.shops {
				if block.shops[i].shop == d.Shop {
					shopInfo = &block.shops[i]
					break
				}
			}
			if shopInfo == nil {
				continue
			}

			existingCount := countExistingSku(f, sheetName, *shopInfo)
			if existingCount > 0 && !allowOverwrite {
				continue
			}

			// Clear + write
			clearShopRows(f, sheetName, *shopInfo)
			writeShopData(f, sheetName, *shopInfo, d, moneySt, rateSt, intSt, textSt)
		}
		// Recompute totals
		for i := range blocks {
			recalcTotal(f, sheetName, &blocks[i])
		}
	}

	ts := time.Now().Format("0102_1504")
	filename := fmt.Sprintf("FillResult_%s_%s.xlsx", yearMonth, ts)
	path := exportDir() + "/" + filename
	if err := saveExcelWithCalc(f, path); err != nil {
		return "", err
	}
	return path, nil
}

// ============================================================
// Auto fill: scan Excel positions -> read data from the in-memory monthly config ->
// compute -> fill in. The user does not need to enter any data on the import page;
// all data comes from the Tab-2 monthly config.
// ============================================================

// FillSplitAutoResult is the auto-fill result (details shown in the frontend)
type FillSplitAutoResult struct {
	OutputPath string       `json:"output_path"`
	YearMonth  string       `json:"year_month"`
	Filled     []FillRecord `json:"filled"`  // positions filled
	Skipped    []FillRecord `json:"skipped"` // positions skipped (no config or already has data)
	Stats      FillStats    `json:"stats"`
}

type FillRecord struct {
	Department string `json:"department"`
	Channel    string `json:"channel"`
	Mode       string `json:"mode"`
	Shop       string `json:"shop"`
	Reason     string `json:"reason"` // skip reason
}

type FillStats struct {
	TotalPositions int `json:"total_positions"` // total positions found in the Excel file
	ConfigFound    int `json:"config_found"`    // positions with config that were filled
	NoConfig       int `json:"no_config"`       // positions without config
	AlreadyFilled  int `json:"already_filled"`  // already had data and overwrite not selected
}

// FillSplitAuto auto-fills the split table (data comes entirely from the in-memory monthly config)
// yearMonth: which month's config; overwrite: whether to overwrite existing data in the Excel file
func FillSplitAuto(filePath, yearMonth string, overwrite bool) (*FillSplitAutoResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Split table styles (Microsoft YaHei 10, centered)
	moneySt := splitMoneyStyle(f)
	rateSt := splitRateStyle(f)
	intSt := splitIntStyle(f)
	textSt := splitTextStyle(f)

	result := &FillSplitAutoResult{
		YearMonth: yearMonth,
		Filled:    []FillRecord{},
		Skipped:   []FillRecord{},
	}
	totalPos, found, noCfg, already := 0, 0, 0, 0

	// Iterate each channel sheet
	for _, channel := range Channels {
		sheetName := channel + " Split"
		sheetIdx, _ := f.GetSheetIndex(sheetName)
		if sheetIdx == -1 {
			continue
		}
		blocks := scanBlocks(f, sheetName)

		for i := range blocks {
			block := &blocks[i]
			for j := range block.shops {
				sp := &block.shops[j]
				totalPos++

				// Excel already has data and overwrite not selected -> skip
				existingCount := countExistingSku(f, sheetName, *sp)
				if existingCount > 0 && !overwrite {
					already++
					result.Skipped = append(result.Skipped, FillRecord{
						Department: block.dept, Channel: channel,
						Mode: block.mode, Shop: sp.shop,
						Reason: fmt.Sprintf("Excel already has %d rows of data, overwrite not selected", existingCount),
					})
					continue
				}

				// Read this dimension's data from the in-memory monthly config
				rows := QueryByDimension(yearMonth, block.dept, channel, block.mode, sp.shop)
				if len(rows) == 0 {
					noCfg++
					result.Skipped = append(result.Skipped, FillRecord{
						Department: block.dept, Channel: channel,
						Mode: block.mode, Shop: sp.shop,
						Reason: fmt.Sprintf("Tab-2 monthly config has no data for this dimension (%s|%s|%s|%s|%s)",
							yearMonth, block.dept, channel, block.mode, sp.shop),
					})
					continue
				}

				// Clear + write (one row per config row; multiple SKUs occupy consecutive rows)
				clearShopRows(f, sheetName, *sp)
				for i, r := range rows {
					// Multiple SKUs: the first goes to sp.row, subsequent ones increment the row number
					sp2 := *sp
					sp2.row = sp.row + i
					writeShopData(f, sheetName, sp2, r, moneySt, rateSt, intSt, textSt)
				}
				found++
				result.Filled = append(result.Filled, FillRecord{
					Department: block.dept, Channel: channel,
					Mode: block.mode, Shop: sp.shop,
					Reason: fmt.Sprintf("filled %d SKU row(s)", len(rows)),
				})
			}
			// Recompute the block's totals
			recalcTotal(f, sheetName, block)
		}
	}

	result.Stats = FillStats{
		TotalPositions: totalPos, ConfigFound: found,
		NoConfig: noCfg, AlreadyFilled: already,
	}

	ts := time.Now().Format("0102_1504")
	filename := fmt.Sprintf("SplitAutoFill_%s_%s.xlsx", yearMonth, ts)
	path := exportDir() + "/" + filename
	if err := saveExcelWithCalc(f, path); err != nil {
		return nil, err
	}
	result.OutputPath = path
	return result, nil
}

type shopPos struct {
	shop   string
	row    int
	endRow int
}
type blockInfo struct {
	dept, mode string
	titleRow   int
	shops      []shopPos
	totalRow   int
}

func scanBlocks(f *excelize.File, sheet string) []blockInfo {
	var blocks []blockInfo
	var cur *blockInfo
	rows, _ := f.GetRows(sheet)
	for rowIdx := 0; rowIdx < len(rows); rowIdx++ {
		var aVal string
		if len(rows[rowIdx]) > 0 {
			aVal = strings.TrimSpace(rows[rowIdx][0])
		}
		if dept, mode, ok := isTitleRow(aVal); ok {
			if cur != nil {
				finalizeShops(cur, rows)
				blocks = append(blocks, *cur)
			}
			cur = &blockInfo{dept: dept, mode: mode, titleRow: rowIdx + 1}
			continue
		}
		if cur == nil {
			continue
		}
		if aVal != "" && aVal != "Shop" && aVal != "Total" {
			cur.shops = append(cur.shops, shopPos{shop: aVal, row: rowIdx + 1})
		}
	}
	if cur != nil {
		finalizeShops(cur, rows)
		blocks = append(blocks, *cur)
	}
	return blocks
}

func finalizeShops(b *blockInfo, rows [][]string) {
	for i := range b.shops {
		if i+1 < len(b.shops) {
			b.shops[i].endRow = b.shops[i+1].row - 1
		} else {
			// Find the total row
			for r := b.shops[i].row; r < len(rows); r++ {
				var a string
				if len(rows[r]) > 0 {
					a = strings.TrimSpace(rows[r][0])
				}
				if a == "Total" {
					b.shops[i].endRow = r
					b.totalRow = r + 1
					break
				}
			}
			if b.shops[i].endRow == 0 {
				b.shops[i].endRow = b.shops[i].row
			}
		}
	}
}

func countExistingSku(f *excelize.File, sheet string, sp shopPos) int {
	count := 0
	for r := sp.row; r <= sp.endRow; r++ {
		b, _ := f.GetCellValue(sheet, fmt.Sprintf("B%d", r))
		if b != "" {
			count++
		}
	}
	return count
}

// clearShopRows clears the input columns of a shop's detail rows (all formula columns kept)
// Input cells that contain formulas are also kept (to avoid breaking the formula chain on
// rows that have data in the sample)
func clearShopRows(f *excelize.File, sheet string, sp shopPos) {
	for r := sp.row; r <= sp.endRow; r++ {
		for c := 0; c <= 16; c++ { // A~Q
			// Only clear input columns; formula columns are kept
			if !splitInputCols[c] {
				continue
			}
			cell := fmt.Sprintf("%s%d", colLetter(c), r)
			// Keep cells that contain a formula (do not clear)
			if hasFormula(f, sheet, cell) {
				continue
			}
			f.SetCellValue(sheet, cell, nil)
		}
	}
}

// writeShopData writes one shop detail row (only fills input columns; formula columns reuse
// the sample's automatic calculation)
// Column layout per the sample:
//   A(0) Shop  B(1) Product  C(2) Platform revenue  D(3) Company revenue [formula]  E(4) Standard cost
//   F(5) Discount amount [formula]  G(6) Discount rate  H(7) Traffic  I(8) Influencer commission [formula]  J(9) Agency commission
//   K(10) Slot  L(11) Service fee  M(12) Non-HQ  N(13) AOV  O(14) Order qty [formula]
//   P(15) Standard cost total [formula]  Q(16) Gross margin [formula]
func writeShopData(f *excelize.File, sheet string, sp shopPos, d ConfigRow,
	moneySt, rateSt, intSt, textSt int) {
	rec := computeFromRow(d)
	// E (standard cost): unit standard cost (per piece), computed for single and multiple materials
	stdCostVal := getComboUnitCost(d.SubItems)
	// Only write input columns; formula columns (D/F/O/P/Q) keep the sample formulas untouched
	// Among the input columns: if a cell already has a formula (e.g. column I on rows with data
	// in the sample = C * rate), keep it, don't overwrite
	setInput := func(col int, val interface{}, styleType int) {
		cell := fmt.Sprintf("%s%d", colLetter(col), sp.row)
		// Keep formulas (the caller controls overwrite mode; formulas are protected here uniformly)
		if hasFormula(f, sheet, cell) {
			return
		}
		if val == nil || val == "" {
			f.SetCellValue(sheet, cell, "")
			f.SetCellStyle(sheet, cell, cell, textSt)
			return
		}
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
	// A Shop / B Product name (text)
	f.SetCellValue(sheet, fmt.Sprintf("A%d", sp.row), d.Shop)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", sp.row), fmt.Sprintf("A%d", sp.row), textSt)
	setInput(1, d.ComboName, 0)
	// C Platform revenue (gross GSV) - input (back to gross)
	setInput(2, rec.PlatformRevGsv, 1)
	// D Company revenue - formula column, skip
	// E Standard cost (CNY) - input (unit standard cost)
	setInput(4, stdCostVal, 1)
	// F Discount amount - formula column, skip
	// G Discount rate (%) - input
	setInput(6, rec.DiscountRate, 2)
	// H Traffic spend (CNY) - input
	if rec.Cp51 != 0 {
		setInput(7, rec.Cp51, 1)
	}
	// I Influencer commission (CNY) - input (empty sample rows have no formula, write the value
	// directly; rows with data have formulas handled by clearShopRows' input-column logic)
	setInput(8, rec.Cp52-d.Cp52Slot, 1)
	// J Agency commission (CNY) - input
	if rec.Cp53 != 0 {
		setInput(9, rec.Cp53, 1)
	}
	// K Slot fee (CNY) - input
	if d.Cp52Slot != 0 {
		setInput(10, d.Cp52Slot, 1)
	}
	// L Service fee (CNY) - input
	if rec.Tp12 != 0 {
		setInput(11, rec.Tp12, 1)
	}
	// M Non-HQ - leave empty
	// N AOV (CNY) - input
	setInput(13, rec.Aov, 1)
	// O Order qty - formula column (D/N), skip
	// P Standard cost total - formula column (E*O), skip
	// Q Gross margin - formula column (D-P)/D, skip
}

// recalcTotal handles the total row.
// Strategy: sample total rows already have SUM etc. formulas; after filling in details the
// formulas recompute automatically, no intervention needed. Only when the total row has no
// formulas (newly created table) do we write the formulas.
func recalcTotal(f *excelize.File, sheet string, b *blockInfo) {
	if b.totalRow == 0 {
		return
	}
	tr := b.totalRow
	// Check whether the total row already has a formula (in the sample, column C's total = SUM(...))
	testCell := fmt.Sprintf("C%d", tr)
	if hasFormula(f, sheet, testCell) {
		return // sample total-row formula intact, don't touch
	}
	// Newly created table: write the total formulas
	firstData, lastData := 0, 0
	if len(b.shops) > 0 {
		firstData = b.shops[0].row
		lastData = b.shops[len(b.shops)-1].endRow
	}
	if firstData == 0 || lastData == 0 {
		return
	}
	totalStyle := splitTotalStyle(f)
	totalMoney := splitTotalMoney(f)
	totalRate := splitTotalRate(f)
	totalInt := splitTotalInt(f)

	writeFormula := func(col string, formula string, style int) {
		cell := fmt.Sprintf("%s%d", col, tr)
		f.SetCellFormula(sheet, cell, formula)
		f.SetCellStyle(sheet, cell, cell, style)
	}
	f.SetCellValue(sheet, fmt.Sprintf("A%d", tr), "Total")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", tr), fmt.Sprintf("B%d", tr), totalStyle)
	// Money columns SUM
	for _, c := range []string{"C", "D", "F", "H", "I", "J", "K", "L", "P"} {
		writeFormula(c, fmt.Sprintf("SUM(%s%d:%s%d)", c, firstData, c, lastData), totalMoney)
	}
	writeFormula("G", fmt.Sprintf("IF(D%d=0,0,F%d/D%d)", tr, tr, tr), totalRate)
	writeFormula("O", fmt.Sprintf("SUM(O%d:O%d)", firstData, lastData), totalInt)
	writeFormula("N", fmt.Sprintf("IF(O%d=0,0,D%d/O%d)", tr, tr, tr), totalMoney)
	writeFormula("Q", fmt.Sprintf("IF(D%d=0,0,(D%d-P%d)/D%d)", tr, tr, tr, tr), totalRate)
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", tr), fmt.Sprintf("E%d", tr), totalStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("M%d", tr), fmt.Sprintf("M%d", tr), totalStyle)
}

func getCellFloat(f *excelize.File, sheet, cell string) float64 {
	v, err := f.GetCellValue(sheet, cell)
	if err != nil || v == "" {
		return 0
	}
	var n float64
	fmt.Sscanf(v, "%f", &n)
	return n
}

// hasFormula checks whether a cell already has a formula (cells with formulas are skipped
// when filling, to avoid breaking the formula chain)
func hasFormula(f *excelize.File, sheet, cell string) bool {
	formula, err := f.GetCellFormula(sheet, cell)
	if err != nil {
		return false
	}
	return formula != ""
}

// ============================================================
// Drill-down table (influencer distribution budget) scan + fill
// Structure: 4 blocks (Influencer Distribution / Douyin / Kuaishou / WeChat Channels) x
// (3 header rows + 11 fee-item rows)
// Horizontal: B/C = fiscal-year total, then 2 columns per month (amount, rate)
// Unit: 10k CNY (x10000 conversion on read/write)
// ============================================================

// PivotCell is one data cell of the drill-down table (channel x month x fee item)
type PivotCell struct {
	Block      string  `json:"block"`      // Influencer Distribution / Douyin / Kuaishou / WeChat Channels
	YearMonth  string  `json:"year_month"` // 2026-08
	FeeItem    string  `json:"fee_item"`   // Revenue / Gross Profit / AP / TP / ...
	Wan        float64 `json:"wan"`        // value in 10k CNY (original value)
	HasRate    bool    `json:"has_rate"`   // whether a rate column exists
	Rate       float64 `json:"rate"`       // rate (decimal, e.g. 0.071)
	ExcelRow   int     `json:"excel_row"`  // row number (1-based)
	AmountCol  int     `json:"amount_col"` // amount column index (0-based)
	RateCol    int     `json:"rate_col"`   // rate column index (0-based), -1 = none
}

// PivotScanResult is the drill-down table scan result
type PivotScanResult struct {
	Blocks    []string       `json:"blocks"`      // block order
	Months    []string       `json:"months"`      // months of the columns
	FeeItems  []string       `json:"fee_items"`   // fee-item order
	Cells     []PivotCell    `json:"cells"`       // all non-empty cells
	BlockRows map[string]int `json:"block_rows"`  // block -> first fee-item row number
}

// pivotFeeItems are the drill-down table fee items (fixed order, matching the template)
var pivotFeeItems = []string{
	"Revenue", "Gross Profit", "AP", "TP", "CP", "PP", "MP", "FP",
	"Assessed Net Profit", "Non-HQ Support", "Operating Net Profit",
}

// monthLabelToYM converts "Jul" -> "2026-07", "Jan" -> "2027-01" (FY2026 starts in July)
func monthLabelToYM(label string) string {
	t, err := time.Parse("Jan", strings.TrimSpace(label))
	if err != nil {
		return ""
	}
	m := int(t.Month())
	if m < 1 || m > 12 {
		return ""
	}
	if m >= 7 {
		return fmt.Sprintf("2026-%02d", m)
	}
	return fmt.Sprintf("2027-%02d", m)
}

// isPivotBlockLabel checks whether a value is a drill-down table block label.
// Accepts channel names (Douyin / Kuaishou / WeChat Channels / Xiaohongshu) plus each
// department's top-level summary name (Influencer Distribution / Innovation & Custom /
// Channel Operations)
func isPivotBlockLabel(v string) (string, bool) {
	v = strings.TrimSpace(v)
	// Channel blocks
	for _, ch := range Channels {
		if v == ch {
			return v, true
		}
	}
	// Department top-level summary block names
	for _, dept := range Departments {
		if v == PivotTopBlockName(dept) {
			return v, true
		}
	}
	return "", false
}

// ScanPivotExcel scans the drill-down table
func ScanPivotExcel(filePath string) (*PivotScanResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetList()[0] // the drill-down table has a single sheet
	rows, _ := f.GetRows(sheetName)

	result := &PivotScanResult{
		FeeItems:  pivotFeeItems,
		BlockRows: map[string]int{},
	}

	var months []string
	cells := []PivotCell{}
	blockRowStart := map[string]int{} // block -> header row 1

	// First pass: locate each block's header row
	// (signature: column A = "Fee Item" and column B is a block label)
	for rowIdx := 0; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		var aVal, bVal string
		if len(row) > 0 {
			aVal = strings.TrimSpace(row[0])
		}
		if len(row) > 1 {
			bVal = strings.TrimSpace(row[1])
		}
		if aVal == "Fee Item" {
			if blk, ok := isPivotBlockLabel(bVal); ok {
				blockRowStart[blk] = rowIdx // 0-based
			}
		}
	}

	// Determine block order (by appearance order)
	blockOrder := []string{}
	for rowIdx := 0; rowIdx < len(rows); rowIdx++ {
		var aVal, bVal string
		if len(rows[rowIdx]) > 0 {
			aVal = strings.TrimSpace(rows[rowIdx][0])
		}
		if len(rows[rowIdx]) > 1 {
			bVal = strings.TrimSpace(rows[rowIdx][1])
		}
		if aVal == "Fee Item" {
			if blk, ok := isPivotBlockLabel(bVal); ok {
				if !contains(blockOrder, blk) {
					blockOrder = append(blockOrder, blk)
				}
			}
		}
	}
	result.Blocks = blockOrder

	// Read the months from header row 2 (R+1) of the first block
	if len(blockOrder) > 0 {
		startIdx := blockRowStart[blockOrder[0]]
		monthRowIdx := startIdx + 1 // header row 2
		if monthRowIdx < len(rows) {
			row := rows[monthRowIdx]
			// From column D (index 3), 2 columns per month
			for c := 3; c+1 <= len(row); c += 2 {
				v := strings.TrimSpace(row[c])
				if v == "" || v == "·" {
					continue
				}
				ym := monthLabelToYM(v)
				if ym != "" {
					months = append(months, ym)
				}
			}
		}
	}
	result.Months = months

	// Scan each block's fee-item data
	for _, blk := range blockOrder {
		startIdx := blockRowStart[blk]
		// Fee items start after header row 3 (R+2)
		feeStartRow := startIdx + 3
		result.BlockRows[blk] = feeStartRow + 1 // 1-based
		for i, fee := range pivotFeeItems {
			r := feeStartRow + i
			if r >= len(rows) {
				break
			}
			row := rows[r]
			// From column D (index 3), 2 columns per month (amount, rate)
			for mi, ym := range months {
				amountCol := 3 + mi*2
				rateCol := amountCol + 1
				if amountCol >= len(row) {
					continue
				}
				amountStr := strings.TrimSpace(row[amountCol])
				amountStr = strings.ReplaceAll(amountStr, ",", "")
				if amountStr == "" || amountStr == "·" {
					continue
				}
				var wan float64
				fmt.Sscanf(amountStr, "%f", &wan)
				if wan == 0 {
					continue
				}
				cell := PivotCell{
					Block: blk, YearMonth: ym, FeeItem: fee,
					Wan: wan, ExcelRow: r + 1,
					AmountCol: amountCol, RateCol: -1,
				}
				if rateCol < len(row) {
					rateStr := strings.TrimSpace(row[rateCol])
					if rateStr != "" && rateStr != "·" {
						rateStr = strings.TrimSuffix(rateStr, "%")
						var rate float64
						fmt.Sscanf(rateStr, "%f", &rate)
						if rate != 0 {
							cell.HasRate = true
							cell.Rate = rate / 100
							cell.RateCol = rateCol
						}
					}
				}
				cells = append(cells, cell)
			}
		}
	}
	result.Cells = cells
	return result, nil
}

// PivotFillItem is a single fill item (overwrites a channel x month x fee item)
type PivotFillItem struct {
	Block     string  `json:"block"`
	YearMonth string  `json:"year_month"`
	FeeItem   string  `json:"fee_item"`
	Wan       float64 `json:"wan"`      // new value in 10k CNY (if <= 0, do not overwrite)
	Rate      float64 `json:"rate"`     // new rate (decimal)
	HasRate   bool    `json:"has_rate"`
}

// FillPivotExcel fills new data into the drill-down table (overwrite mode)
func FillPivotExcel(filePath string, items []PivotFillItem) (string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sheetName := f.GetSheetList()[0]
	// Drill-down table styles (Microsoft YaHei 11, centered)
	bdStyle := pivotAmountStyle(f) // integer amount style
	rateSt := pivotRateStyle(f)    // 1-decimal rate style

	// Scan the structure first to locate positions
	scan, err := ScanPivotExcel(filePath)
	if err != nil {
		return "", err
	}
	// Build the lookup index: block|ym|fee -> cell
	cellMap := map[string]PivotCell{}
	for _, c := range scan.Cells {
		cellMap[c.Block+"|"+c.YearMonth+"|"+c.FeeItem] = c
	}
	// Even if the original cell is empty, the row can be located via BlockRows
	// block|ym -> column number + row number (fee-item relative offset)

	monthIdx := map[string]int{}
	for i, m := range scan.Months {
		monthIdx[m] = i
	}
	feeIdx := map[string]int{}
	for i, fee := range pivotFeeItems {
		feeIdx[fee] = i
	}

	written := 0
	for _, it := range items {
		key := it.Block + "|" + it.YearMonth + "|" + it.FeeItem
		baseRow, ok1 := scan.BlockRows[it.Block]
		fi, ok2 := feeIdx[it.FeeItem]
		mi, ok3 := monthIdx[it.YearMonth]
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		row := baseRow + fi
		amountCol := 3 + mi*2
		rateCol := amountCol + 1
		amountCell := fmt.Sprintf("%s%d", colLetter(amountCol), row)
		rateCell := fmt.Sprintf("%s%d", colLetter(rateCol), row)

		// Write the amount (skip if it has a formula, keep the original formula)
		if it.Wan != 0 && !hasFormula(f, sheetName, amountCell) {
			f.SetCellValue(sheetName, amountCell, it.Wan)
			f.SetCellStyle(sheetName, amountCell, amountCell, bdStyle)
			written++
		}
		// Write the rate (skip if it has a formula)
		if it.HasRate && !hasFormula(f, sheetName, rateCell) {
			f.SetCellValue(sheetName, rateCell, it.Rate)
			f.SetCellStyle(sheetName, rateCell, rateCell, rateSt)
			written++
		}
		_ = key
		_ = cellMap
	}

	ts := time.Now().Format("0102_1504")
	filename := fmt.Sprintf("PivotFillResult_%s.xlsx", ts)
	path := exportDir() + "/" + filename
	if err := saveExcelWithCalc(f, path); err != nil {
		return "", err
	}
	return path, nil
}

// ============================================================
// FillPivotAuto auto-fills the drill-down table (data comes entirely from the in-memory
// monthly config).
// Logic: aggregate by department -> for each block (Influencer Distribution = all /
// channel = that channel) x month compute the fee items -> fill in.
// Same algorithm as ExportPivotTable, but writes into an existing Excel file's structure.
// ============================================================

// FillPivotAutoResult is the drill-down auto-fill result
type FillPivotAutoResult struct {
	OutputPath string             `json:"output_path"`
	Department string             `json:"department"`
	Stats      PivotStats         `json:"stats"`
	Filled     []PivotFilledBlock `json:"filled"`
}

type PivotStats struct {
	Blocks int `json:"blocks"` // number of blocks processed
	Months int `json:"months"` // number of month columns
	Cells  int `json:"cells"`  // number of cells written
}

type PivotFilledBlock struct {
	Block     string `json:"block"`
	CellCount int    `json:"cell_count"`
}

// FillPivotAuto auto-fills the drill-down table
// department: which department's data; overwrite: whether to overwrite existing data in the Excel file
func FillPivotAuto(filePath, department string, overwrite bool) (*FillPivotAutoResult, error) {
	// Scan the structure first (locate block rows, month columns)
	scan, err := ScanPivotExcel(filePath)
	if err != nil {
		return nil, err
	}
	// Read all configs of the department
	allRows := QueryByDepartment(department)
	if len(allRows) == 0 {
		return nil, fmt.Errorf("Tab-2 monthly config has no data for department '%s'; fill in the monthly config page first", department)
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetList()[0]
	// Drill-down table styles (Microsoft YaHei 11; rates rely on sample formulas, not
	// hardcoded, so only the amount style is needed)
	bdStyle := pivotAmountStyle(f) // integer amount style

	// Month -> column number mapping (from the scan result)
	monthCol := map[string]int{} // ym -> amount column index (0-based)
	for mi, m := range scan.Months {
		monthCol[m] = 3 + mi*2
	}
	// Fee item -> row offset
	feeRowOff := map[string]int{}
	for i, fee := range pivotFeeItems {
		feeRowOff[fee] = i
	}

	result := &FillPivotAutoResult{
		Department: department,
		Filled:     []PivotFilledBlock{},
	}
	totalCells := 0

	for _, blk := range scan.Blocks {
		baseRow, ok := scan.BlockRows[blk]
		if !ok {
			continue
		}
		// Determine whether this is a top-level summary block (a block name not in the
		// channel list = top level, e.g. "Influencer Distribution" / "Channel Operations")
		isTopBlock := !contains(Channels, blk)
		var blockRows []ConfigRow
		if isTopBlock {
			blockRows = allRows // all channels
		} else {
			for _, r := range allRows {
				if r.Channel == blk {
					blockRows = append(blockRows, r)
				}
			}
		}
		if len(blockRows) == 0 {
			continue
		}

		// Group and aggregate by month (same as ExportPivotTable)
		byMonth := map[string][]ComputeResult{}
		for _, r := range blockRows {
			rec := computeFromRow(r)
			byMonth[r.YearMonth] = append(byMonth[r.YearMonth], rec)
		}
		monthAggs := map[string]map[string]float64{}
		for m, recs := range byMonth {
			monthAggs[m] = Aggregate(recs)
		}

		// Fee item value function: (fee, ym) -> (amount in 10k, rate, has rate)
		getFeeVal := func(fee, ym string) (float64, float64, bool) {
			agg, ok := monthAggs[ym]
			if !ok {
				return 0, 0, false
			}
			cr := agg["company_rev"]
			switch fee {
			case "Revenue":
				return agg["company_rev"], 0, false
			case "Gross Profit":
				return agg["gross_profit"], agg["gross_rate"], true
			case "AP":
				return agg["ap"], safeDivF(agg["ap"], cr), true
			case "TP":
				return agg["tp_total"], safeDivF(agg["tp_total"], cr), true
			case "CP":
				return agg["cp_total"], safeDivF(agg["cp_total"], cr), true
			case "PP":
				return agg["pp"], safeDivF(agg["pp"], cr), true
			case "MP":
				return agg["mp"], safeDivF(agg["mp"], cr), true
			case "FP":
				return agg["fp"], safeDivF(agg["fp"], cr), true
			case "Assessed Net Profit":
				return agg["assess_net"], safeDivF(agg["assess_net"], cr), true
			case "Non-HQ Support":
				return 0, 0, true
			case "Operating Net Profit":
				return agg["op_net"], safeDivF(agg["op_net"], cr), true
			}
			return 0, 0, false
		}

		blockCells := 0
		for _, fee := range pivotFeeItems {
			rowOff, ok := feeRowOff[fee]
			if !ok {
				continue
			}
			row := baseRow + rowOff
			for _, m := range scan.Months {
				amountCol, ok := monthCol[m]
				if !ok {
					continue
				}
				amountCell := fmt.Sprintf("%s%d", colLetter(amountCol), row)
				rateCell := fmt.Sprintf("%s%d", colLetter(amountCol+1), row)
				// Strategy: skip amount cells that have formulas (keep the sample formula);
				// never write rate cells (the sample formula D?/D$4 computes them automatically)
				amtHasFormula := hasFormula(f, sheetName, amountCell)
				rateHasFormula := hasFormula(f, sheetName, rateCell)
				// In no-overwrite mode, skip cells that already have data
				if !overwrite {
					existingAmt, _ := f.GetCellValue(sheetName, amountCell)
					existingRate, _ := f.GetCellValue(sheetName, rateCell)
					if existingAmt != "" || existingRate != "" || amtHasFormula || rateHasFormula {
						continue
					}
				}
				wan, _, _ := getFeeVal(fee, m)
				// Only write amounts (rates are computed by the sample formula, not hardcoded)
				if wan != 0 && !amtHasFormula {
					f.SetCellValue(sheetName, amountCell, yuanToWan(wan))
					f.SetCellStyle(sheetName, amountCell, amountCell, bdStyle)
					blockCells++
				}
			}
		}
		totalCells += blockCells
		result.Filled = append(result.Filled, PivotFilledBlock{
			Block: blk, CellCount: blockCells,
		})
	}

	result.Stats = PivotStats{
		Blocks: len(scan.Blocks), Months: len(scan.Months), Cells: totalCells,
	}
	ts := time.Now().Format("0102_1504")
	filename := fmt.Sprintf("PivotAutoFill_%s_%s.xlsx", department, ts)
	path := exportDir() + "/" + filename
	if err := saveExcelWithCalc(f, path); err != nil {
		return nil, err
	}
	result.OutputPath = path
	return result, nil
}

// safeDivF is the float64 version of safe division (returns float64)
func safeDivF(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
