package main

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestSyntheticSettlement(t *testing.T) {
	t.Setenv("PNL_PLANNER_DATA_DIR", t.TempDir())
	result, err := ProcessOrders("Douyin", []string{filepath.Join("..", "examples", "demo-douyin-orders.csv")}, 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalRows != 3 || len(result.SKUs) != 1 {
		t.Fatalf("unexpected settlement result: %+v", result)
	}
	sku := result.SKUs[0]
	if sku.DisplayName != "Demo Wonton Twin Pack" || sku.Revenue != 320 ||
		sku.OwnRevenue != 275 || sku.MergedRevenue != 45 || !sku.IsMerged {
		t.Fatalf("unexpected settlement SKU: %+v", sku)
	}
	if len(sku.SubItems) == 0 || len(sku.MergedFrom) != 1 {
		t.Fatalf("expected materials and a merged SKU: %+v", sku)
	}
	useTestExportDir(t)
	output, err := ExportJieAnTable(JieAnExportRequest{Channel: "Douyin", Rows: []JieAnExportRow{{
		AXCode: sku.DisplayName, Revenue: sku.Revenue, OwnRevenue: sku.OwnRevenue,
		MergedRevenue: sku.MergedRevenue, Count: sku.Count, UnitPrice: sku.UnitPrice,
		IsMerged: sku.IsMerged, SubItems: sku.SubItems, PlatformPrice: sku.PlatformPrice,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	workbook, err := excelize.OpenFile(output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { workbook.Close() })
	if sheets := workbook.GetSheetList(); len(sheets) != 1 || sheets[0] != "Douyin" {
		t.Fatalf("unexpected settlement sheets: %v", sheets)
	}
}

func TestUnknownSKUDoesNotReplaceSavedPlan(t *testing.T) {
	t.Setenv("PNL_PLANNER_DATA_DIR", t.TempDir())
	initStorage()
	const month, department, channel, mode, shop = "2026-08", "Influencer Distribution", "Douyin", "Influencer Live", "Demo Fresh Flagship Store"
	t.Cleanup(func() { DeleteByDimension(month, department, channel, mode, shop) })
	const sku = "Demo planner test SKU"
	if err := SaveSkuToLibrary(sku, []SubItem{{MatID: "1000000001", Bags: 2}}); err != nil {
		t.Fatal(err)
	}
	req := SaveConfigRequest{
		YearMonth: month, Department: department, Channel: channel, Mode: mode, Shop: shop,
		MonthlyRevenueTarget: 1000, TargetMode: "platform",
		Rows: []ConfigRow{{ComboName: sku, PlatformPrice: 40, RevenueShare: 1}},
	}
	app := NewApp()
	if result := app.SaveConfig(req); result["ok"] != true {
		t.Fatalf("save failed: %v", result)
	}
	calculated := ComputeSKU(ComputeInput{
		SubItems:      []SubItem{{MatID: "1000000001", Bags: 2}},
		PlatformPrice: 40, MonthlyRevenueTarget: 1000, RevenueShare: 1, TargetMode: "platform",
	})
	if calculated.EstQty != 25 || calculated.StdCostTotal != 275 {
		t.Fatalf("unexpected planning result: %+v", calculated)
	}
	req.Rows = append(req.Rows, ConfigRow{ComboName: "unknown SKU"})
	if result := app.SaveConfig(req); result["ok"] != false {
		t.Fatalf("invalid plan was accepted: %v", result)
	}
	saved := app.GetConfig(month, department, channel, mode, shop)
	if len(saved) != 1 || saved[0].ComboName != sku {
		t.Fatalf("original plan was changed: %+v", saved)
	}
	useTestExportDir(t)
	output, err := ExportSplitTable(month)
	if err != nil {
		t.Fatal(err)
	}
	workbook, err := excelize.OpenFile(output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { workbook.Close() })
	if sheets := workbook.GetSheetList(); len(sheets) != 5 || sheets[0] != "Summary" {
		t.Fatalf("unexpected planning sheets: %v", sheets)
	}
}

func useTestExportDir(t *testing.T) {
	t.Helper()
	previous := exportDirCache
	exportDirCache = t.TempDir()
	t.Cleanup(func() { exportDirCache = previous })
}
