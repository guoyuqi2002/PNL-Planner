<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  pickDbFile, updateMasterData, updateB2GMapping,
  setAlias, removeAlias, getAliases, searchComboCodes, searchDisplayNames,
  setMaster, getMasterInfo,
} from '../api'

// === 1. Material master data ===
const masterFile = ref('')
const masterLoading = ref(false)
const masterCats = ref({ product_list: true, std_cost: false, assess_rev: false, dist_price: false })
const masterResult = ref('')

async function pickMaster() {
  const p = await pickDbFile()
  if (p && p.data) masterFile.value = p.data
}
async function doUpdateMaster() {
  if (!masterFile.value) { ElMessage.warning('Select an xlsx/xlsm file first') ; return }
  if (!masterCats.value.product_list && !masterCats.value.std_cost &&
      !masterCats.value.assess_rev && !masterCats.value.dist_price) {
    ElMessage.warning('Select at least one category') ; return
  }
  masterLoading.value = true
  try {
    const res = await updateMasterData({ file_path: masterFile.value, ...masterCats.value })
    if (res.ok) { masterResult.value = res.data.message ; ElMessage.success(res.data.message) }
    else ElMessage.error(res.error || 'Update failed')
  } finally { masterLoading.value = false }
}

// === 2. SKU combo <-> material mapping ===
const b2gFile = ref('')
const b2gLoading = ref(false)
const b2gResult = ref('')

async function pickB2G() {
  const p = await pickDbFile()
  if (p && p.data) b2gFile.value = p.data
}
async function doUpdateB2G() {
  if (!b2gFile.value) { ElMessage.warning('Select a csv file first') ; return }
  b2gLoading.value = true
  try {
    const res = await updateB2GMapping(b2gFile.value)
    if (res.ok) { b2gResult.value = res.data.message ; ElMessage.success(res.data.message) }
    else ElMessage.error(res.error || 'Update failed')
  } finally { b2gLoading.value = false }
}

// === 3. Alias merging ===
const aliasCombo = ref('')
const aliasValue = ref('')
const aliasLoading = ref(false)
const aliasInfo = ref(null) // {display_name, aliases, exists}

async function comboSearch(q, cb) {
  const res = await searchComboCodes(q, 20)
  cb((res.data || []).map(s => ({ value: s })))
}
async function dnSearch(q, cb) {
  const res = await searchDisplayNames(q, 20)
  cb((res.data || []).map(s => ({ value: s })))
}

async function aliasSearch(q, cb) {
  const res = await searchDisplayNames(q, 20)
  cb((res.data || []).map(s => ({ value: s })))
}
async function loadAliasInfo() {
  if (!aliasCombo.value) { aliasInfo.value = null ; return }
  const res = await getAliases(aliasCombo.value)
  aliasInfo.value = res.data
  if (res.data && !res.data.exists) ElMessage.warning('This code is not in the mapping library')
}
async function doAddAlias() {
  if (!aliasCombo.value || !aliasValue.value) { ElMessage.warning('Fill in both the combo code and the alias') ; return }
  aliasLoading.value = true
  try {
    const res = await setAlias(aliasCombo.value, aliasValue.value)
    if (res.ok) { ElMessage.success(res.data.message) ; aliasValue.value = '' ; await loadAliasInfo() }
    else ElMessage.error(res.error || 'Add failed')
  } finally { aliasLoading.value = false }
}
// === Master code setting ===
const masterGroup = ref('')
const masterInfo = ref(null) // {master_code, explicit, members:[{code,has_bom,is_master}]}
const setMasterLoading = ref(false)
const masterPicked = ref('')
async function loadMasterInfo() {
  if (!masterGroup.value) { masterInfo.value = null; return }
  const res = await getMasterInfo(masterGroup.value)
  masterInfo.value = res.data
  masterPicked.value = res.data && res.data.master_code || ''
}
async function doSetMaster() {
  if (!masterGroup.value || !masterPicked.value) { ElMessage.warning('Select both a display name and a master code') ; return }
  setMasterLoading.value = true
  try {
    const res = await setMaster(masterGroup.value, masterPicked.value)
    if (res.ok) { ElMessage.success(res.data.message) ; await loadMasterInfo() }
    else ElMessage.error(res.error || 'Set failed')
  } finally { setMasterLoading.value = false }
}

async function doRemoveAlias(a) {
  const res = await removeAlias(a)
  if (res.ok) { ElMessage.success(res.data.message) ; await loadAliasInfo() }
  else ElMessage.error(res.error || 'Delete failed')
}
</script>

<template>
  <div style="max-width:900px;padding:0 8px">
    <el-alert type="info" :closable="false" style="margin-bottom:14px"
      title="Updates are written to same-named JSON files under %APPDATA%\pnl_planner\, take priority over the built-in data, apply immediately (no rebuild needed); delete the file to restore the built-in version" />

    <!-- 1 Material master data -->
    <el-card shadow="never" style="margin-bottom:16px">
      <template #header><b>1. Material master data update</b> (xlsx/xlsm; sheet names must be: Product List / Standard Cost / E-commerce Assess Revenue / Distribution Landing Price)</template>
      <el-row :gutter="12" align="middle">
        <el-col :span="18">
          <el-input v-model="masterFile" placeholder="Select a file" readonly>
            <template #append><el-button @click="pickMaster">Select file</el-button></template>
          </el-input>
        </el-col>
        <el-col :span="6"><el-button type="primary" :loading="masterLoading" @click="doUpdateMaster">Update</el-button></el-col>
      </el-row>
      <div style="margin:10px 0 6px;color:#606266">Update categories (multi-select):</div>
      <el-checkbox v-model="masterCats.product_list">Product List (material name + retail spec)</el-checkbox>
      <el-checkbox v-model="masterCats.std_cost">Standard Cost</el-checkbox>
      <el-checkbox v-model="masterCats.assess_rev">E-commerce Assess Revenue</el-checkbox>
      <el-checkbox v-model="masterCats.dist_price">Distribution Landing Price</el-checkbox>
      <div v-if="masterResult" style="margin-top:10px;color:#67c23a">{{ masterResult }}</div>
    </el-card>

    <!-- 2 Combo mapping -->
    <el-card shadow="never" style="margin-bottom:16px">
      <template #header><b>2. SKU combo <-> material mapping update</b> (csv: column B combo code, column V single-item code, column AA bag count; first sheet of xlsx also supported)</template>
      <el-row :gutter="12" align="middle">
        <el-col :span="18">
          <el-input v-model="b2gFile" placeholder="Select a csv file" readonly>
            <template #append><el-button @click="pickB2G">Select file</el-button></template>
          </el-input>
        </el-col>
        <el-col :span="6"><el-button type="primary" :loading="b2gLoading" @click="doUpdateB2G">Update</el-button></el-col>
      </el-row>
      <div v-if="b2gResult" style="margin-top:10px;color:#67c23a">{{ b2gResult }}</div>
    </el-card>

    <!-- 3 Alias merging -->
    <el-card shadow="never">
      <template #header><b>3. Alias merging</b> (the combo code must already exist in the mapping library; the alias can be a new name)</template>
      <el-form label-width="90px">
        <el-form-item label="SKU code">
          <el-autocomplete v-model="aliasCombo" :fetch-suggestions="comboSearch"
            placeholder="Enter a SKU code, autocomplete (must already be in the mapping library)" style="width:100%"
            @select="loadAliasInfo" @change="loadAliasInfo" />
        </el-form-item>
        <el-form-item v-if="aliasInfo && aliasInfo.exists" label="Status">
          <div>
            <div>Display name (alias): <b>{{ aliasInfo.display_name }}</b></div>
            <div v-if="aliasInfo.materials" style="margin-top:4px;color:#409eff;font-size:12px;word-break:break-all">
              Materials: {{ aliasInfo.materials }}
            </div>
            <div v-else style="color:#f56c6c;font-size:12px;margin-top:4px">This code has no material config (b_to_g)</div>
            <div v-if="aliasInfo.aliases && aliasInfo.aliases.length" style="margin-top:4px">
              Other SKU codes in this group:
              <el-tag v-for="a in aliasInfo.aliases" :key="a" closable style="margin-right:6px"
                @close="doRemoveAlias(a)">{{ a }}</el-tag>
            </div>
            <div v-else style="color:#909399;margin-top:4px">No aliases yet</div>
          </div>
        </el-form-item>
        <el-form-item label="New alias">
          <el-autocomplete v-model="aliasValue" :fetch-suggestions="aliasSearch"
            placeholder="Enter/select a display name (e.g. Demo Wonton Twin Pack), or type a brand-new name" style="width:calc(100% - 90px)" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="aliasLoading" @click="doAddAlias">Add alias</el-button>
        </el-form-item>
      </el-form>

      <el-divider content-position="left">Master code setting (first add the SKU code to the display-name group above)</el-divider>
      <el-form label-width="90px">
        <el-form-item label="Display name">
          <el-autocomplete v-model="masterGroup" :fetch-suggestions="dnSearch"
            placeholder="Enter a display name to search (e.g. Demo Wonton)" style="width:100%"
            @select="loadMasterInfo" @change="loadMasterInfo" />
        </el-form-item>
        <el-form-item v-if="masterInfo && masterInfo.exists" label="Group codes">
          <div style="width:100%">
            <el-radio-group v-model="masterPicked" style="width:100%;display:block">
              <div v-for="m in masterInfo.members" :key="m.code"
                :style="{
                  border: m.is_master ? '1px solid #67c23a' : '1px solid #e4e7ed',
                  background: m.is_master ? '#f0f9eb' : (m.has_bom ? '#fff' : '#fafafa'),
                  'border-radius': '4px', padding: '6px 10px', margin: '6px 0',
                }">
                <el-radio :value="m.code" :disabled="!m.has_bom" style="margin-right:0;height:auto">
                  <span style="font-weight:600">{{ m.code }}</span>
                  <el-tag v-if="m.is_master" size="small" type="success" style="margin-left:8px">
                    current {{ masterInfo.explicit ? '(explicit)' : '(fallback)' }}
                  </el-tag>
                  <el-tag v-if="!m.has_bom" size="small" type="danger" style="margin-left:8px">no material config</el-tag>
                </el-radio>
                <div v-if="m.bom" style="color:#409eff;font-size:12px;margin:2px 0 2px 26px;word-break:break-all;line-height:1.5">
                  {{ m.bom }}
                </div>
              </div>
            </el-radio-group>
            <div style="color:#909399;font-size:12px;margin:6px 0">
              The master code determines the material config used for all calculations of the group
              (Σassess/Σcost); when unset, the lexicographically smallest member is the fallback
            </div>
            <el-button type="primary" size="small" :loading="setMasterLoading" @click="doSetMaster">Set as master code</el-button>
          </div>
        </el-form-item>
        <el-form-item v-else-if="masterGroup && masterInfo && !masterInfo.exists">
          <span style="color:#f56c6c">This display name is not an existing alias group</span>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>
