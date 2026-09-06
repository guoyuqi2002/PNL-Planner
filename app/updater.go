package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// Database updates: material master data / SKU combo mapping / alias merging.
// Update results are written to same-named JSON files under %APPDATA%\pnl_planner\;
// when loading, the external file takes priority over the embedded copy.
// ============================================================

var dbMutex sync.Mutex

func masterOverridePath() string {
	initStorage()
	return filepath.Join(appDataDir(), "master_data.json")
}

func mappingOverridePath() string {
	initStorage()
	return filepath.Join(appDataDir(), "code_mapping.json")
}

// mappingFull is the full structure of code_mapping.json
type mappingFull struct {
	BToG        map[string][]MatItem `json:"b_to_g"`
	DisplayName map[string]string    `json:"display_name"`
	GToB        map[string][]string  `json:"g_to_b"`
	MasterCodes map[string]string    `json:"master_codes"` // display name -> explicitly set master code (defaults to empty, lexicographic fallback)
}

// sanitizeCode cleans a code: strips Excel formula prefixes ="..." / ='...' plus quotes/whitespace
// (mapping source files often use ="0121..." to keep leading zeros; storing them as-is
// would break master-data lookups)
func sanitizeCode(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "=") {
		s = strings.TrimSpace(s[1:])
	}
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

// loadMappingFull reads the full mapping (external override first, embedded fallback).
// All codes are cleaned via sanitizeCode and duplicate entries merged; g_to_b is rebuilt
// from the cleaned b_to_g.
func loadMappingFull() *mappingFull {
	raw := &mappingFull{}
	data, err := os.ReadFile(mappingOverridePath())
	if err != nil {
		data = codeMappingBytes
	}
	json.Unmarshal(data, raw)

	full := &mappingFull{
		BToG:        map[string][]MatItem{},
		DisplayName: map[string]string{},
		GToB:        map[string][]string{},
		MasterCodes: map[string]string{},
	}
	// b_to_g: clean codes + merge (for the same combo and material, keep the latter's bag count)
	for combo, items := range raw.BToG {
		cc := sanitizeCode(combo)
		if cc == "" {
			continue
		}
		dst := full.BToG[cc]
		for _, it := range items {
			mid := sanitizeCode(it.MatID)
			if mid == "" {
				continue
			}
			found := false
			for k := range dst {
				if dst[k].MatID == mid {
					dst[k].Bags = it.Bags
					found = true
					break
				}
			}
			if !found {
				dst = append(dst, MatItem{MatID: mid, Bags: it.Bags})
			}
		}
		full.BToG[cc] = dst
	}
	// display_name: clean keys and values
	for code, dn := range raw.DisplayName {
		cc := sanitizeCode(code)
		if cc == "" {
			continue
		}
		full.DisplayName[cc] = sanitizeCode(dn)
	}
	// master_codes: clean and keep only valid entries (display name exists and the master
	// code is still a member of that group)
	for dn, mc := range raw.MasterCodes {
		dn2 := sanitizeCode(dn)
		mc2 := sanitizeCode(mc)
		if dn2 == "" || mc2 == "" {
			continue
		}
		if full.DisplayName[mc2] == dn2 {
			full.MasterCodes[dn2] = mc2
		}
	}
	// g_to_b: always rebuilt (derived data)
	for combo, items := range full.BToG {
		for _, it := range items {
			full.GToB[it.MatID] = append(full.GToB[it.MatID], combo)
		}
	}
	return full
}

// saveMappingFull persists the mapping and clears caches (takes effect immediately)
func saveMappingFull(full *mappingFull) error {
	data, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(mappingOverridePath(), data, 0644); err != nil {
		return err
	}
	codeMappingCache = nil
	displayNamesCache = nil
	return nil
}

// persistMaster persists the master data and clears the cache
func persistMaster(m map[string]Material) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(masterOverridePath(), data, 0644); err != nil {
		return err
	}
	masterCache = nil
	return nil
}

// ============================================================
// 1. Material master-data update (xlsx/xlsm, sheet names:
//    Product List / Standard Cost / E-commerce Assess Revenue / Distribution Landing Price)
// ============================================================

// MasterUpdateRequest is a master-data update request
type MasterUpdateRequest struct {
	FilePath    string `json:"file_path"`
	ProductList bool   `json:"product_list"` // Product List: name + retail spec (can add new materials)
	StdCost     bool   `json:"std_cost"`     // Standard Cost
	AssessRev   bool   `json:"assess_rev"`   // E-commerce Assess Revenue
	DistPrice   bool   `json:"dist_price"`   // Distribution Landing Price
}

// MasterUpdateResult is the master-data update result
type MasterUpdateResult struct {
	Ok      bool           `json:"ok"`
	Message string         `json:"message"`
	Counts  map[string]int `json:"counts"`  // rows applied per sheet
	Created int            `json:"created"` // number of new materials
	Updated int            `json:"updated"` // number of updated materials
}

// UpdateMasterData reads the sheets of the known layout per selected category and updates the master data
func UpdateMasterData(req MasterUpdateRequest) (*MasterUpdateResult, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	ext := strings.ToLower(filepath.Ext(req.FilePath))
	if ext != ".xlsx" && ext != ".xlsm" {
		return nil, fmt.Errorf("master data update only supports .xlsx/.xlsm (with the same-named sheets), got: %s", ext)
	}
	if !req.ProductList && !req.StdCost && !req.AssessRev && !req.DistPrice {
		return nil, fmt.Errorf("select at least one update category")
	}

	f, err := excelize.OpenFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	m := LoadMaster()
	res := &MasterUpdateResult{Counts: map[string]int{}}
	created, updated := 0, 0

	// upsert callback: returns whether it was applied
	upsert := func(matID string, apply func(*Material)) bool {
		matID = strings.TrimSpace(matID)
		if matID == "" {
			return false
		}
		mat, exists := m[matID]
		if !exists {
			mat = Material{MatID: matID}
		}
		apply(&mat)
		m[matID] = mat
		if exists {
			updated++
		} else {
			created++
		}
		return true
	}

	type job struct {
		sheet  string
		enable bool
	}
	jobs := []job{
		{"Product List", req.ProductList},
		{"Standard Cost", req.StdCost},
		{"E-commerce Assess Revenue", req.AssessRev},
		{"Distribution Landing Price", req.DistPrice},
	}
	var missing []string
	for _, j := range jobs {
		if !j.enable {
			continue
		}
		idx, err := f.GetSheetIndex(j.sheet)
		if err != nil || idx < 0 {
			missing = append(missing, j.sheet)
			continue
		}
		rows, err := f.GetRows(j.sheet)
		if err != nil {
			return nil, fmt.Errorf("failed to read sheet '%s': %v", j.sheet, err)
		}
		n := 0
		for i := 1; i < len(rows); i++ { // skip the header; for multiple rows of the same material, later rows override earlier ones
			r := rows[i]
			switch j.sheet {
			case "Product List": // A: material code, B: description, C: retail spec
				name := strings.TrimSpace(getCell(r, 1))
				spec := parseNum(getCell(r, 2))
				if upsert(getCell(r, 0), func(mat *Material) {
					if name != "" {
						mat.Name = name
					}
					if spec > 0 {
						mat.RetailSpec = spec
					}
				}) {
					n++
				}
			case "Standard Cost": // A: material ID, E: standard cost
				val := parseNum(getCell(r, 4))
				if val > 0 {
					v := val
					if upsert(getCell(r, 0), func(mat *Material) { mat.StdCost = &v }) {
						n++
					}
				}
			case "E-commerce Assess Revenue": // C: material, G: price
				val := parseNum(getCell(r, 6))
				if val > 0 {
					v := val
					if upsert(getCell(r, 2), func(mat *Material) { mat.EcAssessRevenue = &v }) {
						n++
					}
				}
			case "Distribution Landing Price": // A: material ID, C: distribution landing price
				val := parseNum(getCell(r, 2))
				if val > 0 {
					v := val
					if upsert(getCell(r, 0), func(mat *Material) { mat.DistLandingPrice = &v }) {
						n++
					}
				}
			}
		}
		res.Counts[j.sheet] = n
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("file is missing sheet(s): %s (update cancelled)", strings.Join(missing, ", "))
	}
	if err := persistMaster(m); err != nil {
		return nil, fmt.Errorf("save failed: %v", err)
	}
	res.Ok = true
	res.Created = created
	res.Updated = updated
	parts := []string{}
	for _, j := range jobs {
		if j.enable {
			parts = append(parts, fmt.Sprintf("%s=%d rows", j.sheet, res.Counts[j.sheet]))
		}
	}
	res.Message = fmt.Sprintf("Update complete: %s; %d materials created, %d materials overwritten",
		strings.Join(parts, ", "), created, updated)
	return res, nil
}

// ============================================================
// 2. SKU combo <-> material mapping update (csv: B=combo code, V=single-item code,
//    AA=bag count; the first sheet of xlsx/xlsm is also supported)
// ============================================================

// UpdateB2GResult is the mapping update result
type UpdateB2GResult struct {
	Ok       bool   `json:"ok"`
	Message  string `json:"message"`
	Rows     int    `json:"rows"`     // number of valid rows read
	Combos   int    `json:"combos"`   // number of combos involved
	Added    int    `json:"added"`    // new mapping entries
	Replaced int    `json:"replaced"` // existing entries whose bag count was updated
}

// UpdateB2G reads the mapping table, merges it into b_to_g, and rebuilds g_to_b
func UpdateB2G(filePath string) (*UpdateB2GResult, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	rows, err := readTabular(filePath)
	if err != nil {
		return nil, err
	}
	full := loadMappingFull()
	res := &UpdateB2GResult{}
	// First collect all CSV rows grouped by combo (multiple rows of the same combo = that
	// combo's full material list)
	imported := map[string][]MatItem{}
	order := []string{} // keep first-appearance order
	for i := 1; i < len(rows); i++ { // B(1)=combo code, V(21)=single-item code, AA(26)=bag count
		r := rows[i]
		combo := sanitizeCode(getCell(r, 1))
		mat := sanitizeCode(getCell(r, 21))
		if combo == "" || mat == "" {
			continue
		}
		bags := int(parseNum(getCell(r, 26)))
		if bags <= 0 {
			bags = 1
		}
		if _, seen := imported[combo]; !seen {
			order = append(order, combo)
		}
		items := imported[combo]
		found := false
		for k := range items {
			if items[k].MatID == mat {
				items[k].Bags = bags // duplicate material within a combo: later row wins
				found = true
				break
			}
		}
		if !found {
			items = append(items, MatItem{MatID: mat, Bags: bags})
		}
		imported[combo] = items
		res.Rows++
	}
	if res.Rows == 0 {
		return nil, fmt.Errorf("no valid mapping rows read (check column B combo code / column V single-item code / column AA bag count)")
	}
	for _, combo := range order {
		if _, existed := full.BToG[combo]; existed {
			res.Replaced++ // overwrite the existing combo's material list
		} else {
			res.Added++ // new combo
		}
		full.BToG[combo] = imported[combo]
	}
	res.Combos = len(order)
	// Rebuild the reverse index g_to_b: material -> list of combo codes
	full.GToB = map[string][]string{}
	for combo, items := range full.BToG {
		for _, it := range items {
			full.GToB[it.MatID] = append(full.GToB[it.MatID], combo)
		}
	}
	if err := saveMappingFull(full); err != nil {
		return nil, fmt.Errorf("save failed: %v", err)
	}
	res.Ok = true
	res.Message = fmt.Sprintf("Mapping update complete: %d valid rows, %d combos overwritten (material lists taken from this file), %d combos added",
		res.Rows, res.Replaced, res.Added)
	return res, nil
}

// readTabular converts csv (GBK auto-detected) or the first sheet of xlsx/xlsm to [][]string
func readTabular(filePath string) ([][]string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".csv" {
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		if !utf8.Valid(raw) { // GBK-encoded csv
			if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw); err == nil {
				raw = decoded
			}
		}
		reader := csv.NewReader(bytes.NewReader(raw))
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1
		return reader.ReadAll()
	}
	if ext == ".xlsx" || ext == ".xlsm" {
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %v", err)
		}
		defer f.Close()
		list := f.GetSheetList()
		if len(list) == 0 {
			return nil, fmt.Errorf("file contains no worksheets")
		}
		return f.GetRows(list[0])
	}
	return nil, fmt.Errorf("unsupported file type: %s (use csv/xlsx/xlsm)", ext)
}

// ============================================================
// 3. Alias merging (manual: the combo code must already exist + the alias can be anything)
// ============================================================

// resolveCombo resolves user input: combo code or display name -> (combo code, display name)
// On a display-name hit, the first code of that group in sorted order is used (deterministic)
func resolveCombo(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("input is empty")
	}
	full := loadMappingFull()
	if _, ok := full.BToG[input]; ok {
		dn := input
		if v, ok := full.DisplayName[input]; ok && v != "" {
			dn = v
		}
		return input, dn, nil
	}
	// Reverse-lookup one code of the group by display name
	var hits []string
	for code := range full.BToG {
		dn := code
		if v, ok := full.DisplayName[code]; ok && v != "" {
			dn = v
		}
		if dn == input {
			hits = append(hits, code)
		}
	}
	if len(hits) == 0 {
		return "", "", fmt.Errorf("'%s' is neither a combo code in the mapping library nor an existing display name", input)
	}
	sort.Strings(hits)
	return hits[0], input, nil
}

// SearchDisplayNames searches display names (autocomplete for adding aliases).
// Only returns display names already present in display_name (deduplicated);
// combo codes without a display name are not included.
func SearchDisplayNames(q string, limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	q = strings.ToLower(strings.TrimSpace(q))
	full := loadMappingFull()
	set := map[string]bool{}
	for _, dn := range full.DisplayName {
		if dn != "" {
			set[dn] = true
		}
	}
	out := []string{}
	for dn := range set {
		if q == "" || strings.Contains(strings.ToLower(dn), q) {
			out = append(out, dn)
		}
	}
	sort.Strings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// SetAlias sets an alias (display name) for a SKU code: display_name[sku code] = alias
// skuCode must already be in the mapping library (b_to_g); aliasName is the display name
// (can autocomplete an existing one, or be a brand-new name)
func SetAlias(skuCode, aliasName string) (map[string]interface{}, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	skuCode = strings.TrimSpace(skuCode)
	aliasName = strings.TrimSpace(aliasName)
	if skuCode == "" || aliasName == "" {
		return nil, fmt.Errorf("SKU code and alias (display name) cannot be empty")
	}
	full := loadMappingFull()
	if _, ok := full.BToG[skuCode]; !ok {
		return nil, fmt.Errorf("SKU code '%s' is not in the mapping library (add it via mapping update first, or check the code)", skuCode)
	}
	if aliasName == skuCode {
		return nil, fmt.Errorf("alias is identical to the SKU code, nothing to set")
	}
	prev := ""
	if old, ok := full.DisplayName[skuCode]; ok {
		prev = old
	}
	full.DisplayName[skuCode] = aliasName
	if err := saveMappingFull(full); err != nil {
		return nil, fmt.Errorf("save failed: %v", err)
	}
	msg := fmt.Sprintf("Set alias (display name) of SKU '%s' = '%s'", skuCode, aliasName)
	if prev != "" && prev != aliasName {
		msg += fmt.Sprintf(" (was '%s', updated; this SKU was removed from its previous group)", prev)
	}
	return map[string]interface{}{"ok": true, "message": msg, "display_name": aliasName, "sku_code": skuCode}, nil
}

// DeleteAlias deletes an alias: removes the display-name mapping of a SKU code
// (the SKU leaves the group and returns to being grouped independently under its own code)
func DeleteAlias(alias string) (map[string]interface{}, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	full := loadMappingFull()
	dn, ok := full.DisplayName[alias]
	if !ok {
		return nil, fmt.Errorf("SKU code '%s' has no alias set", alias)
	}
	delete(full.DisplayName, alias)
	if err := saveMappingFull(full); err != nil {
		return nil, fmt.Errorf("save failed: %v", err)
	}
	return map[string]interface{}{"ok": true,
		"message": fmt.Sprintf("Deleted alias of SKU '%s' (display name was '%s'); the SKU is grouped independently again", alias, dn)}, nil
}

// bomString returns the combo's material-list string (using master-data names):
// "Name x bags + ..."; falls back to the material ID when the material is not in the library
func bomString(code string, full *mappingFull) string {
	items := full.BToG[code]
	if len(items) == 0 {
		return ""
	}
	m := LoadMaster()
	parts := make([]string, 0, len(items))
	for _, it := range items {
		label := it.MatID
		if mat, ok := m[it.MatID]; ok && mat.Name != "" {
			label = mat.Name
		}
		parts = append(parts, fmt.Sprintf("%s x%d", label, it.Bags))
	}
	return strings.Join(parts, " + ")
}

// GetAliasesForCombo queries a SKU code's display name / material list / other group members
// (the input can be a SKU code or a display name)
func GetAliasesForCombo(comboInput string) map[string]interface{} {
	full := loadMappingFull()
	comboCode, displayName, err := resolveCombo(comboInput)
	if err != nil {
		return map[string]interface{}{"exists": false, "display_name": strings.TrimSpace(comboInput), "aliases": []string{}}
	}
	// Group members: other codes sharing the display name (including pure-alias codes not in b_to_g)
	members := []string{}
	for code, dn := range full.DisplayName {
		if dn == displayName && code != comboCode {
			members = append(members, code)
		}
	}
	sort.Strings(members)
	return map[string]interface{}{
		"display_name": displayName,
		"sku_code":     comboCode,
		"materials":    bomString(comboCode, full),
		"aliases":      members,
		"exists":       true,
	}
}

// SetMasterCode explicitly sets the master code of a display-name group
// (all calculations use the master code's b_to_g configuration).
// Validation: the display name is an existing group, the code is a member of that group,
// and the code has a material config in b_to_g
func SetMasterCode(displayName, skuCode string) (map[string]interface{}, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	displayName = strings.TrimSpace(displayName)
	skuCode = sanitizeCode(skuCode)
	if displayName == "" || skuCode == "" {
		return nil, fmt.Errorf("display name and master code cannot be empty")
	}
	full := loadMappingFull()
	// Group existence: at least one member points to this display name
	member := false
	for _, dn := range full.DisplayName {
		if dn == displayName {
			member = true
			break
		}
	}
	if !member {
		return nil, fmt.Errorf("display name '%s' is not an existing alias group (add an alias above first)", displayName)
	}
	// skuCode must be a member of the group
	if full.DisplayName[skuCode] != displayName {
		return nil, fmt.Errorf("SKU code '%s' is not a member of the '%s' group (add it to the group first)", skuCode, displayName)
	}
	// It must have a material config (the master code determines the group's Σassess/Σcost)
	if _, ok := full.BToG[skuCode]; !ok {
		return nil, fmt.Errorf("SKU code '%s' has no material config (b_to_g) in the mapping library, so it cannot be the master code", skuCode)
	}
	full.MasterCodes[displayName] = skuCode
	if err := saveMappingFull(full); err != nil {
		return nil, fmt.Errorf("save failed: %v", err)
	}
	return map[string]interface{}{
		"ok":      true,
		"message": fmt.Sprintf("Master code of group '%s' set to '%s' (the whole group's calculations will use its material config)", displayName, skuCode),
	}, nil
}

// GetMasterInfo queries a display-name group's master-code status: current master code
// (explicit first, otherwise the lexicographic fallback) + all members
func GetMasterInfo(displayName string) map[string]interface{} {
	full := loadMappingFull()
	displayName = strings.TrimSpace(displayName)
	members := []string{}
	for code, dn := range full.DisplayName {
		if dn == displayName {
			members = append(members, code)
		}
	}
	sort.Strings(members)
	if len(members) == 0 {
		return map[string]interface{}{"exists": false, "members": []string{}}
	}
	// Fallback: the lexicographically smallest member
	fallback := members[0]
	for _, m := range members {
		if _, ok := full.BToG[m]; ok { // prefer members that have a material config
			fallback = m
			break
		}
	}
	mc, explicit := full.MasterCodes[displayName]
	valid := explicit && full.DisplayName[mc] == displayName
	if _, ok := full.BToG[mc]; valid {
		valid = ok
	}
	current, isExplicit := fallback, false
	if valid {
		current, isExplicit = mc, true
	}
	// Per-member extra info: whether it has a material config
	type memberInfo struct {
		Code     string `json:"code"`
		HasBom   bool   `json:"has_bom"`
		IsMaster bool   `json:"is_master"`
		Bom      string `json:"bom"` // material list "Name x bags + ..."
	}
	infos := []memberInfo{}
	for _, m := range members {
		_, hasBom := full.BToG[m]
		infos = append(infos, memberInfo{Code: m, HasBom: hasBom, IsMaster: m == current, Bom: bomString(m, full)})
	}
	return map[string]interface{}{
		"exists":       true,
		"display_name": displayName,
		"master_code":  current,
		"explicit":     isExplicit,
		"members":      infos,
	}
}

// SearchComboCodes searches combo codes (for autocomplete)
func SearchComboCodes(q string, limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	q = strings.ToLower(strings.TrimSpace(q))
	full := loadMappingFull()
	out := []string{}
	if q == "" {
		return out
	}
	for code := range full.BToG {
		if strings.Contains(strings.ToLower(code), q) {
			out = append(out, code)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

// SearchAliasCodes searches alias candidates (combo codes + existing aliases, for autocomplete)
func SearchAliasCodes(q string, limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	q = strings.ToLower(strings.TrimSpace(q))
	full := loadMappingFull()
	seen := map[string]bool{}
	out := []string{}
	if q == "" {
		return out
	}
	add := func(s string) {
		if len(out) >= limit || seen[s] {
			return
		}
		if strings.Contains(strings.ToLower(s), q) {
			seen[s] = true
			out = append(out, s)
		}
	}
	for code := range full.BToG {
		add(code)
	}
	for code := range full.DisplayName {
		add(code)
	}
	return out
}
