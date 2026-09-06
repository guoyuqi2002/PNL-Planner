<script setup>
import { ref, reactive, watch, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getDepartments, getModes, getShops, getMonths,
  listSkuLibrary, getConfig, saveConfig, compute,
  getDimensions, deleteConfig,
} from '../api'

// Dimension options
const departments = ref([])
const months = ref([])
const skuList = ref([])
const modes = ref([])
const shops = ref([])

// Currently selected dimension
const form = reactive({
  department: '',
  channel: '',
  mode: '',
  shop: '',
  year_month: '',
  monthly_revenue_target: 1000000,
  target_mode: 'company', // 'company' (company revenue excl. tax) or 'platform' (platform revenue gross GSV)
})

// Available channels (filter out ones with no modes)
const availChannels = ref([])
const allChannels = ['Douyin', 'Kuaishou', 'WeChat Channels', 'Xiaohongshu']

// SKU config rows (rates stored as [percentage numbers], e.g. 2.6 means 2.6%;
// divided by 100 on submit)
const rows = ref([])

// Live preview results
const previewMap = ref({})  // { row index: computation result }

const isChannelLoading = ref(false)

onMounted(async () => {
  try {
    departments.value = (await getDepartments()).data || []
    months.value = (await getMonths()).data || []
    skuList.value = (await listSkuLibrary()).data || []
    if (departments.value.length === 0) {
      ElMessage.warning('No department data received')
      return
    }
    form.department = departments.value[0]
    form.year_month = months.value.find(m => m === '2026-08') || months.value[0] || ''
    await refreshChannels()
    refreshSaved()
  } catch (e) {
    ElMessage.error('Failed to load data: ' + (e?.message || String(e)))
  }
})

// Department changed -> refresh available channels
watch(() => form.department, refreshChannels)

async function refreshChannels() {
  if (!form.department) return
  isChannelLoading.value = true
  try {
    const valid = []
    for (const ch of allChannels) {
      const ms = (await getModes(form.department, ch)).data || []
      if (ms.length > 0) valid.push({ channel: ch, modes: ms })
    }
    availChannels.value = valid
    if (valid.length > 0) {
      form.channel = valid[0].channel
      modes.value = valid[0].modes
      form.mode = modes.value[0]
      shops.value = (await getShops(form.channel)).data || []
      form.shop = shops.value[0] || ''
      await loadConfig()
    }
  } finally {
    isChannelLoading.value = false
  }
}

// Channel changed -> refresh modes/shops (loadConfig is not called again here; it is
// triggered by the mode/shop watchers)
watch(() => form.channel, async () => {
  if (!form.channel || isChannelLoading.value) return
  const item = availChannels.value.find(c => c.channel === form.channel)
  modes.value = item ? item.modes : []
  form.mode = modes.value[0] || ''
  shops.value = (await getShops(form.channel)).data || []
  form.shop = shops.value[0] || ''
})

// Mode/shop/month changed -> reload the config
watch([() => form.mode, () => form.shop, () => form.year_month], () => {
  loadConfig()
}, { flush: 'post' })

async function loadConfig() {
  if (!form.department || !form.channel || !form.mode || !form.shop || !form.year_month) return
  try {
    const data = (await getConfig({
      year_month: form.year_month,
      department: form.department,
      channel: form.channel,
      mode: form.mode,
      shop: form.shop,
    })).data || []
    rows.value = data.map(r => ({
      combo_name: r.combo_name,
      // Rates: stored as decimals in the db (0.026), displayed /100 as a percentage (2.6)
      revenue_share: (r.revenue_share || 0) * 100,
      platform_price: r.platform_price,
      ap_rate: (r.ap_rate || 0) * 100,
      mp_rate: (r.mp_rate || 0) * 100,
      traffic_cost: r.traffic_cost,
      tp12_rate: (r.tp12_rate ?? 0) * 100,
      cp52_rate: (r.cp52_rate || 0) * 100,
      cp52_slot: r.cp52_slot,
      cp53_rate: (r.cp53_rate || 0) * 100,
      logistics_cost: r.logistics_cost,
    }))
    if (data.length > 0) {
      form.monthly_revenue_target = data[0].monthly_revenue_target
      form.target_mode = data[0].target_mode || 'company'
    }
    else rows.value = []
    refreshPreview()
  } catch (e) {
    rows.value = []
    previewMap.value = {}
  }
}

const showTp12 = computed(() => form.department === 'Channel Operations')

function addRow() {
  rows.value.push({
    combo_name: skuList.value[0]?.sku_name || '',
    revenue_share: 0, platform_price: 39.9,
    ap_rate: 2.6, mp_rate: 7.9, traffic_cost: 0,
    tp12_rate: 0,
    cp52_rate: 15, cp52_slot: 0, cp53_rate: 5, logistics_cost: 12,
  })
}

function removeRow(idx) {
  rows.value.splice(idx, 1)
  delete previewMap.value[idx]
}

async function handleSave() {
  if (rows.value.length === 0) { ElMessage.warning('Add a SKU first'); return }
  try {
    // On submit, divide the percentages by 100 to convert back to decimals
    const payloadRows = rows.value.map(r => ({
      combo_name: r.combo_name,
      revenue_share: r.revenue_share / 100,
      platform_price: r.platform_price,
      ap_rate: r.ap_rate / 100,
      mp_rate: r.mp_rate / 100,
      traffic_cost: r.traffic_cost,
      tp12_rate: r.tp12_rate / 100,
      cp52_rate: r.cp52_rate / 100,
      cp52_slot: r.cp52_slot,
      cp53_rate: r.cp53_rate / 100,
      logistics_cost: r.logistics_cost,
    }))
    const res = (await saveConfig({
      year_month: form.year_month,
      department: form.department,
      channel: form.channel,
      mode: form.mode,
      shop: form.shop,
      monthly_revenue_target: form.monthly_revenue_target,
      target_mode: form.target_mode,
      rows: payloadRows,
    })).data
    ElMessage.success(`Saved ${res.saved} SKU(s)`)
    if (res.saved === 0) ElMessage.warning('Saved 0 rows: SKU name not found in the library, check the name in "SKU Library"')
  } catch (e) {
    ElMessage.error('Save failed: ' + (e.response?.data?.detail || e.message))
  }
}

// Material detail hint
function skuDetail(name) {
  const sku = skuList.value.find(s => s.sku_name === name)
  if (!sku) return ''
  return sku.sub_items.map(it => `${(it.name || '').slice(0, 8)}x${it.bags}`).join(' + ')
}

// Live computation preview (debounced)
let previewTimer = null
function refreshPreview() {
  clearTimeout(previewTimer)
  previewTimer = setTimeout(doPreview, 400)
}

async function doPreview() {
  for (let i = 0; i < rows.value.length; i++) {
    const r = rows.value[i]
    const sku = skuList.value.find(s => s.sku_name === r.combo_name)
    if (!sku) { delete previewMap.value[i]; continue }
    try {
      const res = (await compute({
        sub_items: sku.sub_items.map(it => ({ mat_id: it.mat_id, bags: it.bags })),
        platform_price: r.platform_price,
        monthly_revenue_target: form.monthly_revenue_target,
        target_mode: form.target_mode,
        revenue_share: r.revenue_share / 100,
        ap_rate: r.ap_rate / 100,
        mp_rate: r.mp_rate / 100,
        traffic_cost: r.traffic_cost,
        tp12_rate: showTp12.value ? r.tp12_rate / 100 : null,
        cp52_rate: r.cp52_rate / 100,
        cp52_slot: r.cp52_slot,
        cp53_rate: r.cp53_rate / 100,
        logistics_cost: r.logistics_cost,
      })).data
      previewMap.value[i] = res
    } catch (e) {
      delete previewMap.value[i]
    }
  }
}

// Watch rows changes to trigger the preview
watch(rows, refreshPreview, { deep: true })
watch(() => form.monthly_revenue_target, refreshPreview)
watch(() => form.target_mode, refreshPreview)

// Totals preview
const totalPreview = computed(() => {
  const vals = Object.values(previewMap.value)
  if (vals.length === 0) return null
  const companyRev = vals.reduce((s, v) => s + (v.company_rev || 0), 0)
  const grossProfit = vals.reduce((s, v) => s + (v.gross_profit || 0), 0)
  const taxNet = vals.reduce((s, v) => s + (v.op_net || 0), 0) // after-tax net = operating net
  return {
    company_rev: companyRev,
    gross_profit: grossProfit,
    gross_rate: companyRev ? grossProfit / companyRev : 0,
    tax_net: taxNet,
    tax_rate: companyRev ? taxNet / companyRev : 0,
  }
})

// After-tax net margin (single row): op_net / company_rev
function taxRate(row) {
  if (!row || !row.company_rev) return '-'
  return pct(row.op_net / row.company_rev)
}

function fmt(n) {
  if (n == null || isNaN(n)) return '-'
  return Number(n).toLocaleString('en-US', { maximumFractionDigits: 0 })
}
function pct(n) {
  if (n == null || isNaN(n)) return '-'
  return (Number(n) * 100).toFixed(1) + '%'
}

// ============ Saved configs (all dimensions of the month; view/edit/delete) ============
const savedDimensions = ref([])       // all saved dimensions of the month
const savedDetail = ref({})            // { dimension key: [config rows] }
const detailLoading = ref({})          // { dimension key: bool }

async function refreshSaved() {
  if (!form.year_month) return
  try {
    savedDimensions.value = (await getDimensions(form.year_month)).data || []
    savedDetail.value = {}
  } catch (e) {
    savedDimensions.value = []
  }
}

// Refresh saved configs when the month changes
watch(() => form.year_month, refreshSaved)
// Also refresh after saving
async function handleSaveAndRefresh() {
  await handleSave()
  refreshSaved()
}

function dimKey(d) {
  return `${d.department}|${d.channel}|${d.mode}|${d.shop}`
}

// Load details when a panel expands (el-collapse @change passes the activeNames array)
async function onCollapseChange(activeNames) {
  // activeNames is the list of currently expanded panel names
  const names = Array.isArray(activeNames) ? activeNames : [activeNames]
  for (const k of names) {
    if (savedDetail.value[k]) continue  // already loaded
    // Find the corresponding dimension in savedDimensions
    const d = savedDimensions.value.find(x => dimKey(x) === k)
    if (!d) continue
    detailLoading.value[k] = true
    try {
      const data = (await getConfig({
        year_month: form.year_month,
        department: d.department, channel: d.channel, mode: d.mode, shop: d.shop,
      })).data || []
      savedDetail.value[k] = data
    } catch (e) {
      savedDetail.value[k] = []
    } finally {
      detailLoading.value[k] = false
    }
  }
}

// Load into the editor above
function loadToEditor(d) {
  form.department = d.department
  form.channel = d.channel
  form.mode = d.mode
  form.shop = d.shop
  form.monthly_revenue_target = d.monthly_revenue_target
  // refreshChannels/loadConfig are triggered by the watchers
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// Delete a dimension
async function handleDeleteDim(d) {
  try {
    await deleteConfig({
      year_month: form.year_month,
      department: d.department, channel: d.channel, mode: d.mode, shop: d.shop,
    })
    ElMessage.success(`Deleted ${d.department}|${d.channel}|${d.mode}|${d.shop}`)
    refreshSaved()
  } catch (e) {
    ElMessage.error('Delete failed')
  }
}
</script>

<template>
  <div>
    <!-- Dimension selection -->
    <div class="section-card">
      <div class="section-title">Select dimension</div>
      <el-form label-position="top" size="default">
        <el-row :gutter="12">
          <el-col :span="4">
            <el-form-item label="Department">
              <el-select v-model="form.department" style="width:100%">
                <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="Channel">
              <el-select v-model="form.channel" style="width:100%">
                <el-option v-for="c in availChannels" :key="c.channel" :label="c.channel" :value="c.channel" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="Mode">
              <el-select v-model="form.mode" style="width:100%">
                <el-option v-for="m in modes" :key="m" :label="m" :value="m" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="Shop">
              <el-select v-model="form.shop" style="width:100%">
                <el-option v-for="s in shops" :key="s" :label="s" :value="s" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="Month">
              <el-select v-model="form.year_month" style="width:100%">
                <el-option v-for="m in months" :key="m" :label="m" :value="m" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item :label="form.target_mode === 'platform' ? 'Platform revenue target (CNY gross)' : 'Company revenue target (CNY excl. tax)'">
              <div style="display:flex;flex-direction:column;gap:4px;width:100%">
                <el-radio-group v-model="form.target_mode" size="small" style="margin-bottom:2px">
                  <el-radio-button value="company">Company revenue (excl. tax)</el-radio-button>
                  <el-radio-button value="platform">Platform revenue (gross GSV)</el-radio-button>
                </el-radio-group>
                <el-input-number v-model="form.monthly_revenue_target" :min="0" :step="100000" :controls="false" style="width:100%" />
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <!-- SKU config table -->
    <div class="section-card">
      <div class="section-title">
        SKU Config
        <span style="font-weight:normal;color:#909399;font-size:13px;margin-left:8px">Enter rates as percentage numbers (e.g. AP 2.6 means 2.6%)</span>
      </div>
      <el-table :data="rows" border style="width:100%" size="small" :key="showTp12">
        <el-table-column label="SKU" min-width="200" fixed>
          <template #default="{ row }">
            <el-select v-model="row.combo_name" size="small" filterable style="width:100%" @change="refreshPreview">
              <el-option v-for="s in skuList" :key="s.sku_name" :label="s.sku_name" :value="s.sku_name" />
            </el-select>
            <div class="rate-input-tip">{{ skuDetail(row.combo_name) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="Share %" width="85">
          <template #default="{ row }"><el-input-number v-model="row.revenue_share" :controls="false" :precision="1" :step="5" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Platform Price" width="95">
          <template #default="{ row }"><el-input-number v-model="row.platform_price" :controls="false" :min="0" :precision="2" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="AP %" width="75">
          <template #default="{ row }"><el-input-number v-model="row.ap_rate" :controls="false" :precision="1" :step="0.1" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="MP %" width="75">
          <template #default="{ row }"><el-input-number v-model="row.mp_rate" :controls="false" :precision="1" :step="0.1" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Traffic/order" width="85">
          <template #default="{ row }"><el-input-number v-model="row.traffic_cost" :controls="false" :min="0" :precision="2" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column v-if="showTp12" label="TP1.2 %" width="80">
          <template #default="{ row }"><el-input-number v-model="row.tp12_rate" :controls="false" :min="0" :precision="1" :step="0.1" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Influ. Comm. %" width="90">
          <template #default="{ row }"><el-input-number v-model="row.cp52_rate" :controls="false" :min="0" :precision="1" :step="1" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Influ. Slot" width="95">
          <template #default="{ row }"><el-input-number v-model="row.cp52_slot" :controls="false" :min="0" :precision="2" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Agency Comm. %" width="90">
          <template #default="{ row }"><el-input-number v-model="row.cp53_rate" :controls="false" :min="0" :precision="1" :step="1" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Logistics/order" width="85">
          <template #default="{ row }"><el-input-number v-model="row.logistics_cost" :controls="false" :min="0" :precision="2" size="small" style="width:100%" /></template>
        </el-table-column>
        <el-table-column label="Actions" width="65" fixed="right">
          <template #default="{ $index }">
            <el-button type="danger" size="small" link @click="removeRow($index)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div style="margin-top:12px;display:flex;gap:12px">
        <el-button @click="addRow">Add SKU</el-button>
        <el-button type="primary" @click="handleSaveAndRefresh">Save config</el-button>
      </div>
    </div>

    <!-- Live preview -->
    <div class="section-card" v-if="rows.length > 0">
      <div class="section-title">Live computation preview <span style="font-weight:normal;color:#909399;font-size:13px">(refreshes automatically after edits)</span></div>
      <el-table :data="rows" border size="small" style="margin-bottom:12px">
        <el-table-column label="SKU" min-width="160">
          <template #default="{ row }">{{ row.combo_name }}</template>
        </el-table-column>
        <el-table-column label="Est. Qty" width="110">
          <template #default="{ $index }">{{ previewMap[$index] ? fmt(previewMap[$index].est_qty) : '-' }}</template>
        </el-table-column>
        <el-table-column label="Company Revenue" width="120">
          <template #default="{ $index }">{{ previewMap[$index] ? fmt(previewMap[$index].company_rev) : '-' }}</template>
        </el-table-column>
        <el-table-column label="Gross Profit" width="160">
          <template #default="{ $index }">
            <span v-if="previewMap[$index]">{{ fmt(previewMap[$index].gross_profit) }} ({{ pct(previewMap[$index].gross_rate) }})</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="After-tax Net" width="160">
          <template #default="{ $index }">
            <span v-if="previewMap[$index]">{{ fmt(previewMap[$index].op_net) }} ({{ taxRate(previewMap[$index]) }})</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
      <el-descriptions v-if="totalPreview" :column="3" border size="small" title="Totals">
        <el-descriptions-item label="Company Revenue">{{ fmt(totalPreview.company_rev) }}</el-descriptions-item>
        <el-descriptions-item label="Gross Profit">{{ fmt(totalPreview.gross_profit) }} ({{ pct(totalPreview.gross_rate) }})</el-descriptions-item>
        <el-descriptions-item label="After-tax Net">{{ fmt(totalPreview.tax_net) }} ({{ pct(totalPreview.tax_rate) }})</el-descriptions-item>
      </el-descriptions>
    </div>

    <!-- Saved configs (view/edit/delete) -->
    <div class="section-card">
      <div class="section-title">
        Saved configs - {{ form.year_month }}
        <el-button size="small" link @click="refreshSaved" style="margin-left:8px">Refresh</el-button>
      </div>
      <el-empty v-if="savedDimensions.length === 0" description="No saved configs for this month" :image-size="60" />
      <el-collapse v-else @change="onCollapseChange">
        <el-collapse-item
          v-for="d in savedDimensions"
          :key="dimKey(d)"
          :name="dimKey(d)"
          :title="`${d.department} | ${d.channel} | ${d.mode} | ${d.shop}  (target ${fmt(d.monthly_revenue_target)})`"
        >
          <div v-loading="detailLoading[dimKey(d)]">
            <el-table :data="savedDetail[dimKey(d)] || []" border size="small" style="margin-bottom:10px">
              <el-table-column label="SKU" prop="combo_name" min-width="150" />
              <el-table-column label="Share" width="75">
                <template #default="{ row }">{{ pct(row.revenue_share) }}</template>
              </el-table-column>
              <el-table-column label="Price" width="80">
                <template #default="{ row }">¥{{ row.platform_price }}</template>
              </el-table-column>
              <el-table-column label="Influ. Comm." width="85">
                <template #default="{ row }">{{ pct(row.cp52_rate) }}</template>
              </el-table-column>
              <el-table-column label="Logistics/order" width="80">
                <template #default="{ row }">¥{{ row.logistics_cost }}</template>
              </el-table-column>
            </el-table>
            <div style="display:flex;gap:8px">
              <el-button size="small" type="primary" @click="loadToEditor(d)">Load into editor above</el-button>
              <el-popconfirm title="Delete all configs of this dimension?" @confirm="handleDeleteDim(d)">
                <template #reference>
                  <el-button size="small" type="danger">Delete this dimension</el-button>
                </template>
              </el-popconfirm>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>
