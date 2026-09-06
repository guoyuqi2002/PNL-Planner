package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails application struct; every method exposed to the frontend is defined here
type App struct {
	ctx context.Context
}

// NewApp creates an application instance
func NewApp() *App {
	return &App{}
}

// startup is called when Wails starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	initStorage()
}

// ============================================================
// Dimension metadata
// ============================================================

func (a *App) GetDepartments() []string               { return Departments }
func (a *App) GetChannels() []string                  { return Channels }
func (a *App) GetModes(dept, channel string) []string { return GetDynamicModes(dept, channel) }
func (a *App) GetShops(channel string) []string       { return GetDynamicShops(channel) }
func (a *App) GetMonths() []string                    { return AllMonthsFY2026() }

// ============================================================
// Session state (new / import)
// ============================================================

// GetSessionState returns the current session mode (new/import) and imported file paths
func (a *App) GetSessionState() SessionState { return GetSession() }

// StartNew selects "new" mode: use the default shop/mode structure
func (a *App) StartNew() map[string]interface{} {
	StartNewSession()
	return map[string]interface{}{"ok": true, "mode": "new"}
}

// ImportSplitFile selects and imports a split table (obtains the dynamic shop structure)
func (a *App) ImportSplitFile() (map[string]interface{}, error) {
	filePath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select the interest e-commerce business plan split table",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Excel files", Pattern: "*.xlsx"},
		},
	})
	if err != nil || filePath == "" {
		return map[string]interface{}{"ok": false}, err
	}
	scan, err := ImportSplitSession(filePath)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ok":               true,
		"file_path":        filePath,
		"positions":        len(scan.Configs),
		"shops_by_channel": scan.ShopsByChannel,
	}, nil
}

// ImportPivotFile selects and imports a drill-down table (records the file path)
func (a *App) ImportPivotFile() (map[string]interface{}, error) {
	filePath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select the influencer budget drill-down table",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Excel files", Pattern: "*.xlsx"},
		},
	})
	if err != nil || filePath == "" {
		return map[string]interface{}{"ok": false}, err
	}
	if err := ImportPivotSession(filePath); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ok":        true,
		"file_path": filePath,
	}, nil
}

// ============================================================
// Settlement mode
// ============================================================

// PickOrderFiles lets the user pick multiple order Excel files
func (a *App) PickOrderFiles() ([]string, error) {
	return wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select order files",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Order files (Excel/CSV)", Pattern: "*.xlsx;*.xls;*.csv"},
		},
	})
}

// ProcessOrderFilesReq is a settlement processing request
type ProcessOrderFilesReq struct {
	Channel   string   `json:"channel"`
	FilePaths []string `json:"file_paths"`
	Threshold int      `json:"threshold"`
	Password  string   `json:"password"` // password for encrypted Kuaishou files
}

// ProcessOrderFiles parses orders + filters + aggregates, returning the SKU list
func (a *App) ProcessOrderFiles(req ProcessOrderFilesReq) (*JieAnResult, error) {
	if len(req.FilePaths) == 0 {
		return nil, fmt.Errorf("no files selected")
	}
	if req.Threshold <= 0 {
		req.Threshold = 1
	}
	return ProcessOrders(req.Channel, req.FilePaths, req.Threshold, req.Password)
}

// ExportJieAn exports the settlement table (generate a temp file first, then show Save As)
func (a *App) ExportJieAn(req JieAnExportRequest) (string, error) {
	tempPath, err := ExportJieAnTable(req)
	if err != nil {
		return "", err
	}
	// Show Save As dialog
	defaultName := fmt.Sprintf("Settlement_%s.xlsx", req.Channel)
	finalPath, err := a.saveToUserPath(tempPath, defaultName)
	if err != nil {
		return "", err
	}
	return finalPath, nil
}

// ExportZhouBao exports the weekly report (generate a temp file first, then show Save As)
func (a *App) ExportZhouBao(req ZhouBaoExportRequest) (string, error) {
	tempPath, err := ExportZhouBaoTable(req)
	if err != nil {
		return "", err
	}
	defaultName := fmt.Sprintf("WeeklyReport_%s.xlsx", req.Channel)
	finalPath, err := a.saveToUserPath(tempPath, defaultName)
	if err != nil {
		return "", err
	}
	return finalPath, nil
}

// ============================================================
// Material search
// ============================================================

type MaterialSearchResult struct {
	MatID           string   `json:"mat_id"`
	Name            string   `json:"name"`
	StdCost         *float64 `json:"std_cost"`
	EcAssessRevenue *float64 `json:"ec_assess_revenue"`
	RetailSpec      float64  `json:"retail_spec"`
}

func (a *App) SearchMaterials(q string, limit int) []MaterialSearchResult {
	if limit <= 0 {
		limit = 30
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return []MaterialSearchResult{}
	}
	m := LoadMaster()
	var results []MaterialSearchResult
	for _, mat := range m {
		if strings.Contains(strings.ToLower(mat.MatID), q) ||
			strings.Contains(strings.ToLower(mat.Name), q) {
			results = append(results, MaterialSearchResult{
				MatID: mat.MatID, Name: mat.Name,
				StdCost: mat.StdCost, EcAssessRevenue: mat.EcAssessRevenue,
				RetailSpec: mat.RetailSpec,
			})
			if len(results) >= limit {
				break
			}
		}
	}
	return results
}

// ============================================================
// SKU library management
// ============================================================

func (a *App) ListSkuLibrary() []SkuLibraryItem { return loadSkuLibrary() }

// SaveSku saves a SKU (with price completeness validation)
// Validation: every sub-material must have material ID / assess revenue / standard cost /
// spec complete, otherwise the save is blocked
func (a *App) SaveSku(skuName string, subItems []SubItem) map[string]interface{} {
	// Validate sub-material data completeness
	m := LoadMaster()
	var missing []map[string]string
	for _, item := range subItems {
		matID := strings.TrimSpace(item.MatID)
		mat, exists := m[matID]
		if !exists {
			missing = append(missing, map[string]string{
				"mat_id": matID, "name": item.Name,
				"issue": "material ID not found in master data",
			})
			continue
		}
		if mat.RetailSpec <= 0 {
			missing = append(missing, map[string]string{
				"mat_id": matID, "name": mat.Name,
				"issue": "missing spec (units per case)",
			})
		}
		if mat.EcAssessRevenue == nil || *mat.EcAssessRevenue <= 0 {
			missing = append(missing, map[string]string{
				"mat_id": matID, "name": mat.Name,
				"issue": "missing e-commerce assess revenue",
			})
		}
		if mat.StdCost == nil || *mat.StdCost <= 0 {
			missing = append(missing, map[string]string{
				"mat_id": matID, "name": mat.Name,
				"issue": "missing standard cost",
			})
		}
	}
	if len(missing) > 0 {
		// Block the save and return the missing details
		return map[string]interface{}{
			"ok":       false,
			"sku_name": skuName,
			"error":    "material data incomplete, cannot save SKU",
			"missing":  missing,
		}
	}
	// Validation passed, save
	err := SaveSkuToLibrary(skuName, subItems)
	return map[string]interface{}{
		"ok": err == nil, "sku_name": skuName,
		"missing": []map[string]string{},
	}
}

func (a *App) DeleteSku(skuName string) map[string]interface{} {
	err := DeleteSkuFromLibrary(skuName)
	return map[string]interface{}{"ok": err == nil}
}

// ============================================================
// Monthly config (temporary, in memory)
// ============================================================

type SaveConfigRequest struct {
	YearMonth            string      `json:"year_month"`
	Department           string      `json:"department"`
	Channel              string      `json:"channel"`
	Mode                 string      `json:"mode"`
	Shop                 string      `json:"shop"`
	MonthlyRevenueTarget float64     `json:"monthly_revenue_target"`
	TargetMode           string      `json:"target_mode"`
	Rows                 []ConfigRow `json:"rows"`
}

func (a *App) GetConfig(yearMonth, dept, channel, mode, shop string) []ConfigRow {
	return QueryByDimension(yearMonth, dept, channel, mode, shop)
}

func (a *App) GetDimensions(yearMonth string) []DimensionInfo {
	return GetSavedDimensions(yearMonth)
}

func (a *App) SaveConfig(req SaveConfigRequest) map[string]interface{} {
	var rows []ConfigRow
	for _, r := range req.Rows {
		sku := GetSkuFromLibrary(r.ComboName)
		if sku == nil {
			continue
		}
		r.SubItems = sku.SubItems
		rows = append(rows, r)
	}
	saved := SaveConfigRows(req.YearMonth, req.Department, req.Channel,
		req.Mode, req.Shop, req.MonthlyRevenueTarget, req.TargetMode, rows)
	return map[string]interface{}{"saved": saved, "ok": true}
}

func (a *App) DeleteConfig(yearMonth, dept, channel, mode, shop string) map[string]interface{} {
	DeleteByDimension(yearMonth, dept, channel, mode, shop)
	return map[string]interface{}{"ok": true}
}

// ============================================================
// Real-time computation
// ============================================================

func (a *App) Compute(in ComputeInput) ComputeResult {
	return ComputeSKU(in)
}

// ============================================================
// Export
// ============================================================

// saveToUserPath shows a "Save As" dialog so the user picks a location, then copies
// the generated temp file there.
// tempPath: path of the generated temp file; defaultName: default file name (no path)
// Returns the final path the user saved to (empty string if the user cancelled)
func (a *App) saveToUserPath(tempPath, defaultName string) (string, error) {
	// Show the Save As dialog
	target, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Choose save location",
		DefaultFilename: defaultName,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Excel files", Pattern: "*.xlsx"},
		},
	})
	if err != nil {
		os.Remove(tempPath) // clean up the temp file
		return "", err
	}
	if target == "" {
		// User cancelled; keep the temp file and return empty without error
		return "", nil
	}
	// Ensure the extension is .xlsx
	if !strings.HasSuffix(strings.ToLower(target), ".xlsx") {
		target += ".xlsx"
	}
	// Copy the temp file to the location chosen by the user
	srcData, err := os.ReadFile(tempPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, srcData, 0644); err != nil {
		return "", err
	}
	// Delete the temp file
	os.Remove(tempPath)
	return target, nil
}

// ExportSplit generates the split table -> shows a Save As dialog for the user to pick a location
func (a *App) ExportSplit(yearMonth string) (string, error) {
	// Generate to a temp file first
	tempPath, err := ExportSplitTable(yearMonth)
	if err != nil {
		return "", err
	}
	defaultName := fmt.Sprintf("Split_%s.xlsx", yearMonth)
	return a.saveToUserPath(tempPath, defaultName)
}

// ExportPivot generates the drill-down table -> shows a Save As dialog for the user to pick a location
func (a *App) ExportPivot(department string) (string, error) {
	tempPath, err := ExportPivotTable(department)
	if err != nil {
		return "", err
	}
	defaultName := fmt.Sprintf("Pivot_%s.xlsx", department)
	return a.saveToUserPath(tempPath, defaultName)
}

// OpenExportFolder opens the directory containing the given path in the system file explorer
// and selects the file
func (a *App) OpenExportFolder(filePath string) error {
	dir := filepath.Dir(filePath)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: explorer /select selects the file
		cmd = exec.Command("explorer", "/select,", filePath)
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", "-R", filePath)
	} else {
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

// ============================================================
// Database updates (master data / combo mapping / aliases)
// ============================================================

// PickDbFile selects a file for database updates (xlsx/xlsm/csv)
func (a *App) PickDbFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select a data file",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Data files", Pattern: "*.xlsx;*.xlsm;*.csv"},
		},
	})
}

func (a *App) UpdateMasterData(req MasterUpdateRequest) (*MasterUpdateResult, error) {
	return UpdateMasterData(req)
}

func (a *App) UpdateB2GMapping(filePath string) (*UpdateB2GResult, error) {
	return UpdateB2G(filePath)
}

func (a *App) SetAlias(comboCode, alias string) (map[string]interface{}, error) {
	return SetAlias(comboCode, alias)
}

func (a *App) RemoveAlias(alias string) (map[string]interface{}, error) {
	return DeleteAlias(alias)
}

func (a *App) GetAliases(comboCode string) map[string]interface{} {
	return GetAliasesForCombo(comboCode)
}

// SetMaster sets the master code of a display-name group
func (a *App) SetMaster(displayName, skuCode string) (map[string]interface{}, error) {
	return SetMasterCode(displayName, skuCode)
}

// GetMasterInfo queries the master-code status and member list of a display-name group
func (a *App) GetMasterInfo(displayName string) map[string]interface{} {
	return GetMasterInfo(displayName)
}

func (a *App) SearchComboCodes(q string, limit int) []string { return SearchComboCodes(q, limit) }

func (a *App) SearchAliasCodes(q string, limit int) []string { return SearchAliasCodes(q, limit) }

// SearchDisplayNames autocompletes by display name (combo selection with alias merging)
func (a *App) SearchDisplayNames(q string, limit int) []string { return SearchDisplayNames(q, limit) }

// ============================================================
// Import (scan + fill)
// ============================================================

// PickExcelFile opens a file selection dialog and returns the selected Excel path
func (a *App) PickExcelFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select an Excel file to import",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Excel files", Pattern: "*.xlsx"},
		},
	})
}

func (a *App) ScanSplit(filePath, yearMonth string) (*ScanResult, error) {
	return ScanSplitExcel(filePath, yearMonth)
}

// ScanPivot scans the drill-down table (influencer distribution budget), returning a
// channel x month x fee-item data map
func (a *App) ScanPivot(filePath string) (*PivotScanResult, error) {
	return ScanPivotExcel(filePath)
}

// FillPivotRequest is a drill-down fill request
type FillPivotRequest struct {
	FilePath string          `json:"file_path"`
	Items    []PivotFillItem `json:"items"`
}

// FillPivot fills the edited data into the drill-down table (overwrite mode),
// returning the new file path
func (a *App) FillPivot(req FillPivotRequest) (string, error) {
	return FillPivotExcel(req.FilePath, req.Items)
}

// FillSplitAutoRequest is an auto-fill request for the split table (data comes from monthly config)
type FillSplitAutoRequest struct {
	YearMonth string `json:"year_month"`
	Overwrite bool   `json:"overwrite"`
}

// FillSplitAuto auto-fills the split table: scan Excel positions -> read monthly config ->
// compute -> fill -> show Save As
func (a *App) FillSplitAuto(req FillSplitAutoRequest) (*FillSplitAutoResult, error) {
	s := GetSession()
	if s.SplitFilePath == "" {
		return nil, fmt.Errorf("no split table imported; import one on the Start page first, or choose direct generation")
	}
	result, err := FillSplitAuto(s.SplitFilePath, req.YearMonth, req.Overwrite)
	if err != nil {
		return nil, err
	}
	// Show a Save As dialog so the user can pick a location
	defaultName := fmt.Sprintf("SplitFilled_%s.xlsx", req.YearMonth)
	finalPath, err := a.saveToUserPath(result.OutputPath, defaultName)
	if err != nil {
		return nil, err
	}
	if finalPath != "" {
		result.OutputPath = finalPath
	}
	return result, nil
}

// FillPivotAutoRequest is an auto-fill request for the drill-down table
type FillPivotAutoRequest struct {
	Department string `json:"department"`
	Overwrite  bool   `json:"overwrite"`
}

// FillPivotAuto auto-fills the drill-down table: aggregate by department + channel ->
// compute -> fill -> show Save As
func (a *App) FillPivotAuto(req FillPivotAutoRequest) (*FillPivotAutoResult, error) {
	s := GetSession()
	if s.PivotFilePath == "" {
		return nil, fmt.Errorf("no drill-down table imported; import one on the Start page first, or choose direct generation")
	}
	result, err := FillPivotAuto(s.PivotFilePath, req.Department, req.Overwrite)
	if err != nil {
		return nil, err
	}
	// Show a Save As dialog so the user can pick a location
	defaultName := fmt.Sprintf("PivotFilled_%s.xlsx", req.Department)
	finalPath, err := a.saveToUserPath(result.OutputPath, defaultName)
	if err != nil {
		return nil, err
	}
	if finalPath != "" {
		result.OutputPath = finalPath
	}
	return result, nil
}

type FillRequest struct {
	FilePath     string          `json:"file_path"`
	YearMonth    string          `json:"year_month"`
	NewData      []ConfigRow     `json:"new_data"`
	OverwriteMap map[string]bool `json:"overwrite_map"`
}

func (a *App) FillSplit(req FillRequest) (string, error) {
	return FillExcel(req.FilePath, req.YearMonth, req.NewData, req.OverwriteMap)
}
