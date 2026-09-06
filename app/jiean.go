package main

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// Settlement mode: order parsing + filtering + aggregation
// ============================================================

//go:embed code_mapping.json
var codeMappingBytes []byte

// MatItem is one material of a mapping (with bag count)
type MatItem struct {
	MatID string `json:"mat_id"`
	Bags  int    `json:"bags"`
}

// codeMappingCache is the embedded code mapping (SKU code -> material list with bag counts)
var codeMappingCache map[string][]MatItem
var displayNamesCache map[string]string // SKU code -> display name (e.g. "150 wonton*10")

func loadCodeMapping() map[string][]MatItem {
	if codeMappingCache != nil {
		return codeMappingCache
	}
	// Unified path via loadMappingFull: external override first + code cleaning
	// (strips the ="..." formula prefix)
	full := loadMappingFull()
	codeMappingCache = full.BToG
	displayNamesCache = full.DisplayName
	return codeMappingCache
}

// getDisplayName returns the display name of a SKU code (display_name preferred,
// otherwise the original code)
func getDisplayName(axCode string) string {
	if displayNamesCache == nil {
		loadCodeMapping()
	}
	if name, ok := displayNamesCache[axCode]; ok {
		return name
	}
	return axCode
}

// OrderRecord is one parsed order record (after filtering)
type OrderRecord struct {
	AXCode  string  `json:"ax_code"`  // AX column, custom SKU code
	Revenue float64 `json:"revenue"`  // BB-BL platform revenue
}

// JieAnSKU is one SKU item of the settlement
type JieAnSKU struct {
	AXCode        string    `json:"ax_code"`
	DisplayName   string    `json:"display_name"`
	Count         int       `json:"count"`          // total order qty (all rows, real value)
	Revenue       float64   `json:"revenue"`        // BB-BL platform revenue (sum of real net amounts, never discounted)
	CreatorComm   float64   `json:"creator_comm"`   // exact commission = Σ((BB-BL) x row AN rate)
	OwnRevenue    float64   `json:"own_revenue"`    // own platform revenue (excluding merged, real value)
	MergedRevenue float64   `json:"merged_revenue"` // below-threshold revenue merged in (real value)
	CreatorRate   float64   `json:"creator_rate"`   // influencer commission rate (AN column)
	PlatformPrice float64   `json:"platform_price"` // platform price (revenue / qty)
	UnitPrice     float64   `json:"unit_price"`     // modal unit price actually paid across the group (ideal discount rate)
	MatIDs        []string  `json:"mat_ids"`
	MatNames      []string  `json:"mat_names"`
	SubItems      []SubItem `json:"sub_items"`
	IsMerged      bool      `json:"is_merged"`
	MergedFrom    []string  `json:"merged_from"`
}

// JieAnResult is the settlement processing result
type JieAnResult struct {
	SKUs      []JieAnSKU `json:"skus"`       // SKU list at or above the threshold (including merged)
	Threshold int        `json:"threshold"`  // threshold used
	TotalRows int        `json:"total_rows"` // total rows after filtering
	Channel   string     `json:"channel"`    // channel
}

// ProcessOrders parses order files and returns the filtered, aggregated SKU list.
// channel: "WeChat Channels" / "Douyin" / "Kuaishou"
// filePaths: list of user-uploaded Excel file paths
// threshold: SKU occurrence-count threshold
func ProcessOrders(channel string, filePaths []string, threshold int, password string) (*JieAnResult, error) {
	cm := loadCodeMapping()
	m := LoadMaster()

	// Build the display_name -> master code reverse mapping (used to look up materials
	// after normalization).
	// Priority: explicitly set master_codes > lexicographically smallest fallback
	// (guarantees deterministic results)
	nameToMaster := map[string]string{} // "150 wonton*10" -> "sph20240730003"
	for axCode := range cm {
		dn := getDisplayName(axCode)
		if dn != axCode {
			if cur, exists := nameToMaster[dn]; !exists || axCode < cur {
				nameToMaster[dn] = axCode
			}
		}
	}
	// Explicitly set master codes override the fallback (loadMappingFull already filters
	// stale entries; validated again here defensively)
	for dn, mc := range loadMappingFull().MasterCodes {
		if _, ok := cm[mc]; ok && getDisplayName(mc) == dn {
			nameToMaster[dn] = mc
		}
	}

	// SKU code (normalized) -> record
	axMap := map[string][]float64{}                  // AX -> []revenue
	axExactCommMap := map[string]float64{}           // AX -> exact commission Σ((BB-BL) x row rate)
	axRateRevMap := map[string]map[float64]float64{} // AX -> {commission rate: total qty at that rate}
	axQtyMap := map[string]float64{}                 // AX -> total qty (sum of BD/E)
	axUnitPriceMap := map[string][]float64{}         // AX -> []unit price actually paid (group-wide)

	if channel == "Kuaishou" {
		// Kuaishou xlsx format (encrypted, password required): read via excelize + Password
		for _, fp := range filePaths {
			f, err := excelize.OpenFile(fp, excelize.Options{Password: password})
			if err != nil {
				return nil, fmt.Errorf("failed to open Kuaishou file (check the password): %v", err)
			}
			sheet := f.GetSheetList()[0]
			rows, _ := f.GetRows(sheet)

			for i := 1; i < len(rows); i++ {
				r := rows[i]
				// Column G (0-based=6): order status - drop "Closed"
				gVal := strings.TrimSpace(getCell(r, 6))
				if gVal == "Closed" {
					continue
				}
				// Column Y (0-based=24): order carrier - drop "Short Video"
				yVal := strings.TrimSpace(getCell(r, 24))
				if yVal == "Short Video" {
					continue
				}
				// Column V (0-based=21): after-sales status - drop "Refunded"
				vVal := strings.TrimSpace(getCell(r, 21))
				if vVal == "Refunded" {
					continue
				}
				// Column AF (0-based=31): channel - keep only "Distribution" (influencer sales)
				afVal := strings.TrimSpace(getCell(r, 31))
				if afVal != "Distribution" {
					continue
				}
				// Column AD (0-based=29): SKU code
				adVal := strings.TrimSpace(getCell(r, 29))
				if adVal == "" {
					continue
				}
				axVal := getDisplayName(adVal)

				// Column P (0-based=15): deal quantity
				pVal := parseNum(getCell(r, 15))
				if pVal <= 0 {
					pVal = 1
				}
				// Column H (0-based=7): order amount actually paid -> platform revenue
				hVal := parseNum(getCell(r, 7))

				// Platform revenue = column H amount paid
				revenue := hVal
				axMap[axVal] = append(axMap[axVal], revenue)
				if axRateRevMap[axVal] == nil {
					axRateRevMap[axVal] = map[float64]float64{}
				}
				// Kuaishou has no commission column AN; commission rate is 0 (the user
				// enters it manually later)
				axQtyMap[axVal] += pVal
				// Platform unit price for the merged discount rate: mode of column H
				axUnitPriceMap[axVal] = append(axUnitPriceMap[axVal], hVal)
			}
			f.Close()
		}
	} else if channel == "Douyin" {
		// Douyin CSV format: read with encoding/csv
		for _, fp := range filePaths {
			file, err := os.Open(fp)
			if err != nil {
				continue
			}
			reader := csv.NewReader(file)
			reader.LazyQuotes = true
			reader.FieldsPerRecord = -1
			rows, err := reader.ReadAll()
			file.Close()
			if err != nil || len(rows) < 2 {
				continue
			}

			for i := 1; i < len(rows); i++ {
				r := rows[i]
				// Column AK (0-based=36): order status - drop "Closed"
				akVal := strings.TrimSpace(getCell(r, 36))
				if akVal == "Closed" {
					continue
				}
				// Column AJ (0-based=35): traffic source - keep only "Selected Alliance"
				// (the influencer marketplace)
				ajVal := strings.TrimSpace(getCell(r, 35))
				if ajVal != "Selected Alliance" {
					continue
				}
				// Column AZ (0-based=51): traffic format - drop "Short Video"
				azVal := strings.TrimSpace(getCell(r, 51))
				if azVal == "Short Video" {
					continue
				}
				// Column AP (0-based=41): creator nickname - drop rows containing
				// "Official" (our own brand accounts)
				apVal := strings.TrimSpace(getCell(r, 41))
				if strings.Contains(apVal, "Official") {
					continue
				}
				// Column AR (0-based=43): after-sales status - drop "Refunded"
				arVal := strings.TrimSpace(getCell(r, 43))
				if arVal == "Refunded" {
					continue
				}
				// Column G (0-based=6): custom SKU code
				gVal := strings.TrimSpace(getCell(r, 6))
				if gVal == "" {
					continue
				}
				axVal := getDisplayName(gVal)

				// Column E (0-based=4): purchase quantity
				eVal := parseNum(getCell(r, 4))
				if eVal <= 0 {
					eVal = 1
				}
				// Column H (0-based=7): unit price
				hVal := parseNum(getCell(r, 7))
				// Column BG (0-based=58): discount amount borne by the merchant
				bgVal := parseNum(getCell(r, 58))

				// Platform revenue = unit price x qty - merchant discount
				revenue := hVal*eVal - bgVal
				axMap[axVal] = append(axMap[axVal], revenue)
				if axRateRevMap[axVal] == nil {
					axRateRevMap[axVal] = map[float64]float64{}
				}
				// Douyin has no commission column AN; commission rate is 0 (the user
				// enters it manually later)
				axQtyMap[axVal] += eVal
				axUnitPriceMap[axVal] = append(axUnitPriceMap[axVal], hVal)
			}
		}
	} else {
		// WeChat Channels / WeChat Mini Store xlsx format (original logic)
		for _, fp := range filePaths {
			f, err := excelize.OpenFile(fp)
			if err != nil {
				continue
			}
			sheet := f.GetSheetList()[0]
			rows, _ := f.GetRows(sheet)

			// Locate columns by header name (adapts automatically to the overall right
			// shift caused by a new column at V in the new export version; falls back to
			// the legacy column numbers when a header is not found)
			hdr := map[string]int{}
			if len(rows) > 0 {
				for i, h := range rows[0] {
					hh := strings.TrimSpace(h)
					if _, dup := hdr[hh]; !dup {
						hdr[hh] = i
					}
				}
			}
			col := func(name string, def int) int {
				if i, ok := hdr[name]; ok {
					return i
				}
				return def
			}
			cStatus := col("Order Status", 5)
			cPromoter := col("Promotion Type", 32)
			cSKU := col("SKU Code (Custom)", 49)
			cRate := col("Commission Rate", 39)
			cUnit := col("Actual Price (Unit)", 52)
			cTotal := col("Actual Price (Total)", 53)
			cQty := col("Quantity", 55)
			cRefund := col("Refunded Amount", 63)

			for i := 1; i < len(rows); i++ { // skip the header
				r := rows[i]
				if len(r) < 38 {
					continue
				}

				// Order status (legacy format: column F)
				fVal := strings.TrimSpace(getCell(r, cStatus))
				if fVal != "Pending Shipment" && fVal != "Shipped" && fVal != "Completed" {
					continue
				}

				// Promotion type / traffic source (legacy format: column AG)
				agVal := strings.TrimSpace(getCell(r, cPromoter))
				if agVal != "Influencer Promo" && agVal != "Agency Promo" {
					continue
				}

				// SKU code, custom (legacy format: column AX)
				rawSKU := strings.TrimSpace(getCell(r, cSKU))
				if rawSKU == "" {
					continue
				}

				// Normalization: alias codes -> uniformly use display_name as the grouping key
				axVal := getDisplayName(rawSKU)

				// Commission rate such as "15%"/"20%" (legacy format: column AN)
				anRaw := strings.TrimSpace(getCell(r, cRate))
				anRaw = strings.TrimSuffix(anRaw, "%")
				anVal := parseNum(anRaw)
				if anVal > 1 {
					anVal = anVal / 100
				}

				// Actual price per unit - the mode is the ideal unit price (legacy: column BA)
				baUnitVal := parseNum(getCell(r, cUnit))

				// Actual total price (legacy format: column BB)
				bbVal := parseNum(getCell(r, cTotal))
				// Refunded amount (legacy format: column BL)
				blVal := parseNum(getCell(r, cRefund))
				// Quantity (legacy format: column BD)
				bdVal := parseNum(getCell(r, cQty))
				if bdVal <= 0 {
					bdVal = 1
				}

				revenue := bbVal - blVal
				axMap[axVal] = append(axMap[axVal], revenue)
				axExactCommMap[axVal] += revenue * anVal // exact commission accumulated per row
				if axRateRevMap[axVal] == nil {
					axRateRevMap[axVal] = map[float64]float64{}
				}
				axRateRevMap[axVal][anVal] += bdVal
				axQtyMap[axVal] += bdVal
				axUnitPriceMap[axVal] = append(axUnitPriceMap[axVal], baUnitVal)
			}
			f.Close()
		}
	}

	// Per-SKU statistics
	type axStat struct {
		code          string
		count         int
		revenue       float64
		exactComm     float64 // exact commission Σ((BB-BL) x row rate)
		creatorRate   float64
		platformPrice float64
		unitPrice     float64 // modal unit price actually paid across the group (ideal discount rate)
	}
	var allAX []axStat
	totalRows := 0
	for ax, revs := range axMap {
		sum := 0.0
		for _, r := range revs {
			sum += r
		}
		// Influencer commission rate: weighted by quantity (Σ rate x qty / Σ qty)
		var totalQtyForRate, weightedSum float64
		for rate, qty := range axRateRevMap[ax] {
			totalQtyForRate += qty
			weightedSum += rate * qty
		}
		bestRate := 0.0
		if totalQtyForRate > 0 {
			bestRate = weightedSum / totalQtyForRate
		}
		// Platform price: total revenue / total qty
		bestPrice := 0.0
		qtyTotal := axQtyMap[ax]
		if qtyTotal > 0 {
			bestPrice = sum / qtyTotal
		}
		// Ideal-discount unit price: modal unit price actually paid across the group
		bestUnitPrice := modeValue(axUnitPriceMap[ax])
		// Occurrence count: total BD quantity, or the number of rows when BD is missing
		totalQty := int(axQtyMap[ax])
		if totalQty == 0 {
			totalQty = len(revs)
		}
		allAX = append(allAX, axStat{ax, totalQty, sum, axExactCommMap[ax], bestRate, bestPrice, bestUnitPrice})
		totalRows += len(revs)
	}

	// Sort by occurrence count, descending
	sort.Slice(allAX, func(i, j int) bool {
		return allAX[i].count > allAX[j].count
	})

	// Split: >= threshold vs < threshold
	var above, below []axStat
	for _, s := range allAX {
		if s.count >= threshold {
			above = append(above, s)
		} else {
			below = append(below, s)
		}
	}

	// Revenue of items below the threshold is merged into the most frequent SKU
	mergedRevenue := 0.0
	mergedExactComm := 0.0
	var mergedCodes []string
	if len(above) > 0 {
		for _, s := range below {
			mergedRevenue += s.revenue
			mergedExactComm += s.exactComm
			mergedCodes = append(mergedCodes, s.code)
		}
		above[0].revenue += mergedRevenue
		above[0].exactComm += mergedExactComm
	}

	// Build the result
	result := &JieAnResult{
		Threshold: threshold,
		TotalRows: totalRows,
		Channel:   channel,
		SKUs:      []JieAnSKU{}, // initialize to an empty slice so JSON does not serialize null
	}

	for idx, s := range above {
		sku := JieAnSKU{
			AXCode:        s.code,
			DisplayName:   getDisplayName(s.code),
			Count:         s.count,
			Revenue:       s.revenue,
			CreatorComm:   s.exactComm,
			CreatorRate:   s.creatorRate,
			PlatformPrice: s.platformPrice,
			UnitPrice:     s.unitPrice,
			MatIDs:        []string{},
			MatNames:      []string{},
			SubItems:      []SubItem{},
			MergedFrom:    []string{},
		}

		// Normalized code -> reverse-lookup the group's original master code -> look up
		// code_mapping to get the materials
		lookupCode := s.code // default to the normalized value (= display_name or the original code)
		if master, ok := nameToMaster[s.code]; ok {
			lookupCode = master // when a display_name exists, use the master code for the material lookup
		}

		if matItems, ok := cm[lookupCode]; ok {
			for _, mi := range matItems {
				sku.MatIDs = append(sku.MatIDs, mi.MatID)
				sku.SubItems = append(sku.SubItems, SubItem{MatID: mi.MatID, Bags: mi.Bags})
				if mat, exists := m[mi.MatID]; exists {
					name := mat.Name
					if mi.Bags > 1 {
						name = fmt.Sprintf("%s x%d bags", name, mi.Bags)
					}
					sku.MatNames = append(sku.MatNames, name)
				} else {
					sku.MatNames = append(sku.MatNames, mi.MatID+" (not in library)")
				}
			}
		}

		// Mark merge info on the first SKU (splitting own revenue from merged revenue)
		if idx == 0 && len(mergedCodes) > 0 {
			sku.IsMerged = true
			sku.MergedFrom = mergedCodes
			sku.MergedRevenue = mergedRevenue
			sku.OwnRevenue = s.revenue - mergedRevenue // own revenue = total - merged
		} else {
			sku.OwnRevenue = s.revenue // normal item: own revenue = total
		}

		result.SKUs = append(result.SKUs, sku)
	}

	return result, nil
}

// getCell safely reads a column value from a row
func getCell(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

// parseNum parses a number (tolerant)
func parseNum(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "¥", "")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// modeValue returns the mode (most frequent value)
func modeValue(vals []float64) float64 {
	counts := map[float64]int{}
	best, bestCnt := 0.0, 0
	for _, v := range vals {
		counts[v]++
		if counts[v] > bestCnt {
			best = v
			bestCnt = counts[v]
		}
	}
	return best
}

// ============================================================
// Settlement table export (columns B-R, aligned with the July settlement format)
// ============================================================

// JieAnExportRow is one row config of the settlement table (user-entered parameters)
type JieAnExportRow struct {
	AXCode        string    `json:"ax_code"`
	Revenue       float64   `json:"revenue"`        // total platform revenue (sum of real net amounts, including merged)
	CreatorComm   float64   `json:"creator_comm"`   // exact commission (WeChat Channels AN accumulated per row; if 0, fall back to rate x revenue)
	OwnRevenue    float64   `json:"own_revenue"`    // own platform revenue (excluding merged, real value)
	MergedRevenue float64   `json:"merged_revenue"` // below-threshold revenue merged in (real value)
	Count         int       `json:"count"`          // actual total order qty
	UnitPrice     float64   `json:"unit_price"`     // modal unit price actually paid across the group (ideal discount for the merged part)
	IsMerged      bool      `json:"is_merged"`      // whether this is a merged row
	Shop          string    `json:"shop"`           // shop
	PlatformPrice float64   `json:"platform_price"` // platform price (display reference only)
	CreatorRate   float64   `json:"creator_rate"`   // influencer commission rate
	CreatorSlot   float64   `json:"creator_slot"`   // slot fee (fixed amount, CNY)
	LeaderRate    float64   `json:"leader_rate"`    // agency commission rate
	Logistics     float64   `json:"logistics"`      // logistics per order
	APRate        float64   `json:"ap_rate"`        // AP rate
	MPRate        float64   `json:"mp_rate"`        // MP rate
	MatIDs        []string  `json:"mat_ids"`
	SubItems      []SubItem `json:"sub_items"`
}

// JieAnExportRequest is the settlement export request
type JieAnExportRequest struct {
	Channel string           `json:"channel"`
	Rows    []JieAnExportRow `json:"rows"`
}

// ExportJieAnTable generates the settlement Excel (columns B-R)
func ExportJieAnTable(req JieAnExportRequest) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := req.Channel
	if sheet == "" {
		sheet = "WeChat Channels"
	}
	f.SetSheetName("Sheet1", sheet)

	// Styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Family: "Microsoft YaHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	moneySt := splitMoneyStyle(f)
	rateSt := splitRateStyle(f)
	textSt := splitTextStyle(f)

	// Headers (columns A-R)
	headers := []string{
		"Shop", "Attribute", "Product Name", "Platform Revenue (GSV)", "Company Revenue (zpr9 excl. tax)",
		"Standard Cost", "Discount Amount", "Discount Rate", "Traffic Spend", "Influencer Commission",
		"Agency Commission", "Influencer Slot Fee", "Service Fee", "Non-HQ Support",
		"AOV (zpr9 excl. tax)", "Order Qty", "Standard Cost Total", "Gross Margin",
	}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", colLetter(i))
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Data rows: one row per SKU (attribute = Actual)
	r := 2
	for _, row := range req.Rows {
		// Compute the fields
		platformRev := row.Revenue // D: platform revenue (gross GSV, from orders)

		// First compute Σassess and Σcost (from the materials)
		var sumAssess, sumCost float64
		for _, si := range row.SubItems {
			spec := getSpec(si.MatID)
			assess := getAssess(si.MatID)
			cost := getCost(si.MatID)
			if spec > 0 {
				sumAssess += assess / spec * float64(si.Bags)
				sumCost += cost / spec * float64(si.Bags)
			}
		}

		// Order qty: always the actual order quantity
		var estQty, companyRev, aov, stdCostTotal float64
		estQty = float64(row.Count) // actual total order qty

		if sumAssess > 0 {
			aov = sumAssess / 1.13 // AOV (excl. tax)

			if row.IsMerged && row.UnitPrice > 0 {
				// Merged row: split computation (group-wide unified config)
				// Own part: normal formula Σassess x actual qty / 1.13
				ownCompanyRev := sumAssess * estQty / 1.13
				// Merged part: ideal discount rate applied to merged revenue (real value)
				// x Σassess / (1.13 x modal unit price)
				mergedCompanyRev := row.MergedRevenue * sumAssess / (1.13 * row.UnitPrice)
				companyRev = ownCompanyRev + mergedCompanyRev
			} else {
				// Normal row: Σassess x actual qty / 1.13
				companyRev = sumAssess * estQty / 1.13
			}
		}
		discountAmt := companyRev - platformRev/1.13 // G: discount amount
		stdCostTotal = sumCost * estQty

		grossRate := 0.0
		if companyRev > 0 {
			grossRate = (companyRev - stdCostTotal) / companyRev
		}

		// Fees
		// Commission: use the exact value when available (WeChat Channels AN accumulated
		// per row), otherwise fall back to rate x revenue
		creatorComm := row.CreatorComm
		if creatorComm == 0 {
			creatorComm = platformRev * row.CreatorRate // J: influencer commission
		}
		leaderComm := platformRev * row.LeaderRate // K: agency commission
		creatorSlot := row.CreatorSlot             // L: slot fee (fixed amount)
		ap := companyRev * row.APRate              // AP
		pp := row.Logistics * estQty               // PP (logistics)
		mp := companyRev * row.MPRate              // MP

		// Write
		stdCostUnit := 0.0
		if estQty > 0 {
			stdCostUnit = stdCostTotal / estQty
		}
		setCellJieAn(f, sheet, r, 0, row.Shop, textSt)     // A Shop
		setCellJieAn(f, sheet, r, 1, "Actual", textSt)     // B Attribute
		setCellJieAn(f, sheet, r, 2, row.AXCode, textSt)   // C Product Name
		setCellJieAn(f, sheet, r, 3, platformRev, moneySt) // D Platform Revenue
		setCellJieAn(f, sheet, r, 4, companyRev, moneySt)  // E Company Revenue
		setCellJieAn(f, sheet, r, 5, stdCostUnit, moneySt) // F Standard Cost (unit, auto)
		setCellJieAn(f, sheet, r, 6, discountAmt, moneySt) // G Discount Amount
		discountRateAuto := 0.0
		if companyRev > 0 {
			discountRateAuto = discountAmt / companyRev
		}
		setCellJieAn(f, sheet, r, 7, discountRateAuto, rateSt) // H Discount Rate (auto)
		setCellJieAn(f, sheet, r, 8, 0, moneySt)               // I Traffic (left empty for now)
		setCellJieAn(f, sheet, r, 9, creatorComm, moneySt)     // J Influencer Commission
		setCellJieAn(f, sheet, r, 10, leaderComm, moneySt)     // K Agency Commission
		setCellJieAn(f, sheet, r, 11, creatorSlot, moneySt)    // L Slot Fee
		setCellJieAn(f, sheet, r, 12, 0, moneySt)              // M Service Fee (empty)
		setCellJieAn(f, sheet, r, 13, 0, moneySt)              // N Non-HQ Support
		setCellJieAn(f, sheet, r, 14, aov, moneySt)            // O AOV
		setCellJieAn(f, sheet, r, 15, estQty, textSt)          // P Order Qty
		setCellJieAn(f, sheet, r, 16, stdCostTotal, moneySt)   // Q Standard Cost Total
		setCellJieAn(f, sheet, r, 17, grossRate, rateSt)       // R Gross Margin

		_ = ap
		_ = pp
		_ = mp
		r++
	}

	// Save
	path := saveToTempDialog(f, "Settlement_"+sheet+".xlsx")
	return path, nil
}

// setCellJieAn writes a cell + style
func setCellJieAn(f *excelize.File, sheet string, row, col int, val interface{}, style int) {
	cell := fmt.Sprintf("%s%d", colLetter(col), row)
	f.SetCellValue(sheet, cell, val)
	f.SetCellStyle(sheet, cell, cell, style)
}

// ============================================================
// Weekly report export (per the weekly report template)
// ============================================================

// ZhouBaoExportRequest is the weekly report export request (reuses settlement data)
type ZhouBaoExportRequest struct {
	Channel string           `json:"channel"`
	Rows    []JieAnExportRow `json:"rows"`
}

// ExportZhouBaoTable generates the weekly report Excel
func ExportZhouBaoTable(req ZhouBaoExportRequest) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := req.Channel
	if sheet == "" {
		sheet = "WeChat Channels"
	}
	f.SetSheetName("Sheet1", sheet)

	moneySt := splitMoneyStyle(f)
	rateSt := splitRateStyle(f)
	textSt := splitTextStyle(f)

	// First aggregate all row data
	var totalRev, totalCompanyRev, totalStdCost, totalCreatorComm float64
	var totalAP, totalTP22, totalPP, totalMP, totalFP, totalAE, totalSlot float64
	var totalQty int
	for _, row := range req.Rows {
		platformRev := row.Revenue
		estQty := float64(row.Count)

		var sumAssess, sumCost float64
		for _, si := range row.SubItems {
			spec := getSpec(si.MatID)
			assess := getAssess(si.MatID)
			cost := getCost(si.MatID)
			if spec > 0 {
				sumAssess += assess / spec * float64(si.Bags)
				sumCost += cost / spec * float64(si.Bags)
			}
		}

		// Company revenue (same basis as the settlement table: group-wide unified config,
		// merged part via the ideal discount rate)
		var companyRev float64
		if sumAssess > 0 {
			if row.IsMerged && row.UnitPrice > 0 {
				ownCR := sumAssess * estQty / 1.13
				mergedCR := row.MergedRevenue * sumAssess / (1.13 * row.UnitPrice)
				companyRev = ownCR + mergedCR
			} else {
				companyRev = sumAssess * estQty / 1.13
			}
		}
		rowStdCost := sumCost * estQty

		grossProfit := companyRev - rowStdCost

		totalRev += platformRev
		totalCompanyRev += companyRev
		totalStdCost += rowStdCost
		// Commission: exact value first (WeChat Channels AN accumulated per row), fall
		// back to rate x revenue
		if row.CreatorComm != 0 {
			totalCreatorComm += row.CreatorComm
		} else {
			totalCreatorComm += platformRev * row.CreatorRate
		}
		totalAP += companyRev * row.APRate
		totalTP22 += platformRev * 0.02 // TP2.2 platform deduction = platform revenue x 2%
		totalPP += estQty * row.Logistics
		totalMP += companyRev * row.MPRate
		totalFP += grossProfit * 0.13 * 0.12
		totalAE += companyRev * 0.0108
		totalSlot += row.CreatorSlot
		totalQty += row.Count
	}

	// Derived metrics
	grossProfitTotal := totalCompanyRev - totalStdCost
	discountAmt := totalCompanyRev - totalRev/1.13                     // CP1.2 sales discount
	marketingFee := totalAP + totalTP22 + totalCreatorComm + totalSlot + discountAmt // total marketing fee (incl. AP)

	// Assessed net profit (incl. AP) = gross - marketing (incl. AP) - PP - MP - FP - AE
	assessNet := grossProfitTotal - marketingFee - totalPP - totalMP - totalFP - totalAE
	// P&L amount = (gross - marketing (excl. AP) - PP - MP - FP - AE) x 0.75 --
	// AP does not participate in the P&L deduction
	lossAmt := (assessNet + totalAP) * 0.75 // R10 P&L amount
	lossRate := 0.0
	if totalCompanyRev > 0 {
		lossRate = lossAmt / totalCompanyRev // R9 = P&L amount / company revenue
	}
	discountRate := 0.0
	if totalCompanyRev > 0 {
		discountRate = discountAmt / totalCompanyRev
	}

	// Write the weekly report (R1-R14)
	labels := []struct {
		label  string
		val    interface{}
		isRate bool
	}{
		{"Live Subtotal", "Check merchant backend", false}, // R1 left empty; must be filled from the merchant backend
		{"Influencer Live - Top", "", false},
		{"Influencer Live - Mid", "", false},
		{"Influencer Live - Tail", "", false},
		{"Slot Fees", totalSlot, false},
		{"Avg Commission", totalCreatorComm, false},
		{"Refund GMV", "Check merchant backend", false}, // R7 left empty
		{"Net GMV", "Check merchant backend", false},    // R8 left empty
		{"P&L Rate", lossRate, true},                    // R9 = P&L amount / company revenue
		{"P&L Amount", lossAmt, false},                  // R10 = assessed net (excl. AP) x 0.75
		{"Discount Rate", discountRate, true},           // R11
		{"Non-HQ", 0, false},                            // R12
		{"Actual P&L Amount", lossAmt, false},           // R13 = R10 (Non-HQ = 0)
		{"Actual P&L Rate", lossRate, true},             // R14 = R9 (Non-HQ = 0)
	}
	for i, item := range labels {
		setCellJieAn(f, sheet, i+1, 0, item.label, textSt)
		if item.val != "" {
			st := moneySt
			if item.isRate {
				st = rateSt
			}
			setCellJieAn(f, sheet, i+1, 1, item.val, st)
		}
	}

	// R17 detail header
	r17Headers := []string{"Net GMV", "P&L Amount", "zpr9 Revenue", "P&L Rate", "CP1.2 Sales Discount", "Platform Fee", "Commission"}
	for i, h := range r17Headers {
		setCellJieAn(f, sheet, 17, i, h, textSt)
	}
	// R18 aggregate data
	setCellJieAn(f, sheet, 18, 0, totalRev, moneySt)         // A Net GMV
	setCellJieAn(f, sheet, 18, 1, lossAmt, moneySt)          // B P&L Amount
	setCellJieAn(f, sheet, 18, 2, totalCompanyRev, moneySt)  // C zpr9 Revenue
	setCellJieAn(f, sheet, 18, 3, lossRate, rateSt)          // D P&L Rate
	setCellJieAn(f, sheet, 18, 4, discountAmt, moneySt)      // E CP1.2 Sales Discount
	setCellJieAn(f, sheet, 18, 5, totalTP22, moneySt)        // F Platform Fee = total platform revenue x 2%
	setCellJieAn(f, sheet, 18, 6, totalCreatorComm, moneySt) // G Commission

	path := saveToTempDialog(f, "WeeklyReport_"+sheet+".xlsx")
	return path, nil
}

// saveToTempDialog saves to a temp location (the actual Save As is handled by app.go's
// saveToUserPath)
func saveToTempDialog(f *excelize.File, defaultName string) string {
	enableFullCalc(f)
	dir := exportDir()
	path := dir + "/" + defaultName
	f.SaveAs(path)
	return path
}
