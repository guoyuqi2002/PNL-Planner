import {
  GetDepartments, GetChannels, GetModes, GetShops, GetMonths,
  SearchMaterials, ListSkuLibrary, SaveSku, DeleteSku,
  GetConfig, GetDimensions, SaveConfig, DeleteConfig,
  Compute, ExportSplit, ExportPivot,
  GetSessionState, StartNew, ImportSplitFile, ImportPivotFile,
  FillSplitAuto, FillPivotAuto,
  OpenExportFolder,
  PickOrderFiles, ProcessOrderFiles, ExportJieAn, ExportZhouBao,
  PickDbFile, UpdateMasterData, UpdateB2GMapping, SetAlias, RemoveAlias, GetAliases, SetMaster, GetMasterInfo,
  SearchComboCodes, SearchAliasCodes, SearchDisplayNames,
} from '../../wailsjs/go/main/App'

const wrap = (p) => Promise.resolve(p).then(data => ({ data, ok: true })).catch(e => ({ data: null, ok: false, error: e }))

// Dimension metadata (shops/modes are dynamic: new = defaults, import = scanned from Excel)
export const getDepartments = () => wrap(GetDepartments())
export const getChannels = () => wrap(GetChannels())
export const getModes = (department, channel) => wrap(GetModes(department, channel))
export const getShops = (channel) => wrap(GetShops(channel))
export const getMonths = () => wrap(GetMonths())

// Materials / SKU library
export const searchMaterials = (q, limit = 30) => wrap(SearchMaterials(q, limit))
export const listSkuLibrary = () => wrap(ListSkuLibrary())
export const saveSku = (sku_name, sub_items) => wrap(SaveSku(sku_name, sub_items))
export const deleteSku = (sku_name) => wrap(DeleteSku(sku_name))

// Monthly config
export const getConfig = (params) => wrap(GetConfig(params.year_month, params.department, params.channel, params.mode, params.shop))
export const getDimensions = (year_month) => wrap(GetDimensions(year_month))
export const saveConfig = (data) => wrap(SaveConfig(data))
export const deleteConfig = (params) => wrap(DeleteConfig(params.year_month, params.department, params.channel, params.mode, params.shop))

// Computation
export const compute = (data) => wrap(Compute(data))

// Session management (new / import)
export const getSessionState = () => wrap(GetSessionState())
export const startNew = () => wrap(StartNew())
export const importSplitFile = () => wrap(ImportSplitFile())
export const importPivotFile = () => wrap(ImportPivotFile())

// Direct generation
export const exportSplit = (year_month) => wrap(ExportSplit(year_month))
export const exportPivot = (department) => wrap(ExportPivot(department))

// Auto fill (data comes from the monthly config; computed automatically and filled into
// the imported Excel)
export const fillSplitAuto = (year_month, overwrite) => wrap(FillSplitAuto({ year_month, overwrite }))
export const fillPivotAuto = (department, overwrite) => wrap(FillPivotAuto({ department, overwrite }))

// Export folder operations
export const openExportFolder = (filePath) => wrap(OpenExportFolder(filePath))

// Settlement mode
export const pickOrderFiles = () => wrap(PickOrderFiles())
export const processOrderFiles = (channel, filePaths, threshold, password = '') =>
  wrap(ProcessOrderFiles({ channel, file_paths: filePaths, threshold, password }))
export const exportJieAn = (data) => wrap(ExportJieAn(data))
export const exportZhouBao = (data) => wrap(ExportZhouBao(data))

// Database updates
export const pickDbFile = () => wrap(PickDbFile())
export const updateMasterData = (data) => wrap(UpdateMasterData(data))
export const updateB2GMapping = (filePath) => wrap(UpdateB2GMapping(filePath))
export const setAlias = (comboCode, alias) => wrap(SetAlias(comboCode, alias))
export const removeAlias = (alias) => wrap(RemoveAlias(alias))
export const getAliases = (comboCode) => wrap(GetAliases(comboCode))
export const searchComboCodes = (q, limit = 20) => wrap(SearchComboCodes(q, limit))
export const searchAliasCodes = (q, limit = 20) => wrap(SearchAliasCodes(q, limit))

export const searchDisplayNames = (q, limit = 20) => wrap(SearchDisplayNames(q, limit))

export const setMaster = (displayName, skuCode) => wrap(SetMaster(displayName, skuCode))
export const getMasterInfo = (displayName) => wrap(GetMasterInfo(displayName))
