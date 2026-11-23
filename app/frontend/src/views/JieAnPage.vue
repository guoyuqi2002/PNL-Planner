<script setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { pickOrderFiles, processOrderFiles, exportJieAn, exportZhouBao, openExportFolder } from '../api'

const emit = defineEmits(['back'])

// Step control
const step = ref(1) // 1=upload 2=threshold 3=config 4=done

// Channel files
const channels = [
  { key: 'WeChat Channels', label: 'WeChat Channels', active: true },
  { key: 'Douyin', label: 'Douyin', active: true },
  { key: 'Kuaishou', label: 'Kuaishou', active: true },
]
const channelFiles = ref({ 'WeChat Channels': [], 'Douyin': [], 'Kuaishou': [] })
const selectedChannel = ref('WeChat Channels')
const ksPassword = ref('') // Kuaishou file password (not preset; entered by the operator each time)

// Threshold
const threshold = ref(5)
const mode = ref('jiean') // jiean | zhoubao

// Processing result
const jieAnResult = ref(null)
const processing = ref(false)

// Config rows (edited by the user)
const configRows = ref([])

// === Step 1: file upload ===
async function handlePickFiles(chKey) {
  try {
    const res = await pickOrderFiles()
    if (!res || !res.data || !Array.isArray(res.data)) return
    channelFiles.value[chKey] = res.data
    ElMessage.success(`${chKey}: ${res.data.length} file(s) selected`)
  } catch (e) {
    ElMessage.error('File selection failed: ' + (e?.message || String(e)))
  }
}

function removeFile(chKey, idx) {
  channelFiles.value[chKey].splice(idx, 1)
}

// === Step 2: process ===
async function handleProcess() {
  const allFiles = channelFiles.value[selectedChannel.value]
  if (!allFiles || allFiles.length === 0) {
    ElMessage.warning(`Select ${selectedChannel.value} order files first`)
    return
  }
  if (selectedChannel.value === 'Kuaishou' && !ksPassword.value) {
    ElMessage.warning('Enter the Kuaishou file password')
    return
  }
  processing.value = true
  try {
    const res = await processOrderFiles(selectedChannel.value, allFiles, threshold.value, ksPassword.value)
    if (!res.ok || !res.data) {
      throw new Error(res.error?.message || String(res.error || 'No results were returned.'))
    }
    if (!res.data.total_rows) {
      ElMessage.warning('No order rows matched the channel filters. Check the format and order status.')
      return
    }
    if (!res.data.skus?.length) {
      ElMessage.warning('No SKU reached the quantity threshold. Lower the threshold and retry.')
      return
    }
    jieAnResult.value = res.data
    // Build the config rows
    configRows.value = (res.data.skus || []).map(sku => ({
      ax_code: sku.ax_code,
      display_name: sku.display_name || sku.ax_code,
      revenue: sku.revenue,
      creator_comm: sku.creator_comm || 0,
      own_revenue: sku.own_revenue ?? sku.revenue,
      merged_revenue: sku.merged_revenue || 0,
      count: sku.count,
      unit_price: sku.unit_price || 0,
      creator_rate: (sku.creator_rate || 0) * 100,
      platform_price: sku.platform_price || 0,
      mat_ids: sku.mat_ids || [],
      mat_names: sku.mat_names || [],
      sub_items: sku.sub_items || [],
      is_merged: sku.is_merged,
      merged_from: sku.merged_from,
      shop: 'Demo Mini Store',
      creator_slot: 0,
      leader_rate: 0,
      logistics: 5,
      ap_rate: 2.6,
      mp_rate: 7.9,
    }))
    step.value = 3
    ElMessage.success(`Processing done: ${configRows.value.length} SKU(s), ${res.data.total_rows || 0} rows filtered`)
  } catch (e) {
    ElMessage.error('Processing failed: ' + (e?.message || String(e)))
  } finally {
    processing.value = false
  }
}

// === Step 3: export ===
async function handleExport() {
  try {
    const payload = {
      channel: selectedChannel.value,
      rows: configRows.value.map(r => ({
        ax_code: r.display_name || r.ax_code,
        revenue: r.revenue,
        creator_comm: r.creator_comm || 0,
        own_revenue: r.own_revenue,
        merged_revenue: r.merged_revenue,
        count: r.count,
        unit_price: r.unit_price,
        is_merged: r.is_merged,
        shop: r.shop,
        platform_price: r.platform_price,
        creator_rate: r.creator_rate / 100,
        creator_slot: r.creator_slot,
        leader_rate: r.leader_rate / 100,
        logistics: r.logistics,
        ap_rate: r.ap_rate / 100,
        mp_rate: r.mp_rate / 100,
        mat_ids: r.mat_ids,
        sub_items: r.sub_items,
      })),
    }
    const res = mode.value === 'zhoubao'
      ? await exportZhouBao(payload)
      : await exportJieAn(payload)
    if (res && res.data) {
      const title = mode.value === 'zhoubao' ? 'Weekly report generated' : 'Settlement table generated'
      ElMessageBox.confirm(
        `<b>${title}</b><br><code style="word-break:break-all;font-size:12px">${res.data}</code>`,
        'Done',
        {
          dangerouslyUseHTMLString: true,
          type: 'success',
          confirmButtonText: 'Open folder',
          cancelButtonText: 'Close',
        }
      ).then(() => openExportFolder(res.data)).catch(() => {})
    }
  } catch (e) {
    ElMessage.error('Export failed: ' + (e?.message || String(e)))
  }
}

function fmt(n) {
  if (n == null || isNaN(n)) return '-'
  return Number(n).toLocaleString('en-US', { maximumFractionDigits: 2 })
}
</script>

<template>
  <div>
    <!-- Back button -->
    <div style="margin-bottom:16px">
      <el-button @click="emit('back')">Back to Monthly Config</el-button>
      <span style="margin-left:12px;color:#909399;font-size:13px">Settlement mode</span>
    </div>

    <!-- Step 1: file upload -->
    <div class="section-card" v-if="step === 1">
      <div class="section-title">Step 1: Upload order files</div>
      <el-row :gutter="16">
        <el-col :span="8" v-for="ch in channels" :key="ch.key">
          <div class="upload-card" :class="{ disabled: !ch.active, selected: selectedChannel === ch.key && ch.active }"
               @click="ch.active && (selectedChannel = ch.key)">
            <div class="upload-title">{{ ch.label }}</div>
            <div v-if="!ch.active" class="upload-placeholder">{{ ch.placeholder }}</div>
            <template v-else>
              <div v-if="selectedChannel === ch.key" style="font-size:11px;color:#409eff;margin-bottom:4px">Selected</div>
              <el-button type="primary" size="small" @click.stop="handlePickFiles(ch.key)" style="margin:8px 0">
                Select files
              </el-button>
              <!-- Kuaishou: password input -->
              <div v-if="ch.key === 'Kuaishou'" style="margin:6px 0">
                <el-input v-model="ksPassword" size="small" placeholder="File password" type="password" show-password style="width:80%" />
              </div>
              <div v-if="channelFiles[ch.key].length > 0" class="file-list">
                <div v-for="(f, i) in channelFiles[ch.key]" :key="i" class="file-item">
                  <span class="file-name">{{ f.split(/[\\\/]/).pop() }}</span>
                  <el-button size="small" type="danger" link @click.stop="removeFile(ch.key, i)">X</el-button>
                </div>
              </div>
              <div v-else class="upload-placeholder">Click to select order Excel files</div>
            </template>
          </div>
        </el-col>
      </el-row>

      <el-divider></el-divider>

      <div style="display:flex;align-items:center;gap:16px;margin-bottom:16px">
        <span>Mode:</span>
        <el-radio-group v-model="mode">
          <el-radio-button value="jiean">Settlement</el-radio-button>
          <el-radio-button value="zhoubao">Weekly report</el-radio-button>
        </el-radio-group>
      </div>

      <div style="display:flex;align-items:center;gap:12px">
        <span>Threshold (SKU occurrences >=):</span>
        <el-input-number v-model="threshold" :min="1" :step="1" style="width:120px" />
      </div>

      <el-button type="primary" size="large" style="margin-top:16px;width:100%"
        :loading="processing" :disabled="!channelFiles[selectedChannel].length"
        @click="handleProcess">
        Start processing ({{ selectedChannel }})
      </el-button>
    </div>

    <!-- Step 2: SKU config -->
    <div class="section-card" v-if="step === 3">
      <div class="section-title">
        Step 2: SKU config ({{ configRows.length }} SKUs, {{ jieAnResult?.total_rows || 0 }} rows filtered)
        <el-button size="small" link @click="step = 1" style="margin-left:8px">Re-upload</el-button>
      </div>

      <el-alert v-if="configRows.filter(r => r.is_merged).length > 0" type="info" :closable="false" style="margin-bottom:12px">
        The first SKU "{{ configRows[0]?.ax_code }}" merged the revenue of
        {{ configRows.filter(r => r.is_merged)[0]?.merged_from?.length || 0 }} low-frequency SKU(s)
      </el-alert>

      <div style="overflow-x:auto">
        <el-table :data="configRows" border size="small" style="min-width:1100px">
          <el-table-column label="SKU" min-width="140" fixed>
            <template #default="{ row }">{{ row.display_name || row.ax_code }}</template>
          </el-table-column>
          <el-table-column label="Count" prop="count" width="55" align="center" />
          <el-table-column label="Platform Revenue" width="100">
            <template #default="{ row }">{{ fmt(row.revenue) }}</template>
          </el-table-column>
          <el-table-column label="Materials" min-width="180">
            <template #default="{ row }">
              <span v-if="row.mat_names.length" style="font-size:12px">{{ row.mat_names.join(', ') }}</span>
              <span v-else style="color:#F56C6C">unmatched</span>
            </template>
          </el-table-column>
          <el-table-column label="Shop" width="120">
            <template #default="{ row }"><el-input v-model="row.shop" size="small" /></template>
          </el-table-column>
          <el-table-column label="Platform Price" width="80">
            <template #default="{ row }">
              <span style="font-weight:600">¥{{ row.platform_price.toFixed(2) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="Influ. Comm. %" width="80">
            <template #default="{ row }">
              <span v-if="selectedChannel === 'WeChat Channels'" style="font-weight:600">{{ row.creator_rate.toFixed(1) }}%</span>
              <el-input-number v-else v-model="row.creator_rate" :controls="false" :precision="1" size="small" style="width:100%" />
            </template>
          </el-table-column>
          <el-table-column label="Influ. Slot" width="80">
            <template #default="{ row }"><el-input-number v-model="row.creator_slot" :controls="false" :min="0" :precision="2" size="small" style="width:100%" /></template>
          </el-table-column>
          <el-table-column label="Agency Comm. %" width="80">
            <template #default="{ row }"><el-input-number v-model="row.leader_rate" :controls="false" :precision="1" size="small" style="width:100%" /></template>
          </el-table-column>
          <el-table-column label="Logistics/order" width="75">
            <template #default="{ row }"><el-input-number v-model="row.logistics" :controls="false" :precision="1" size="small" style="width:100%" /></template>
          </el-table-column>
          <el-table-column label="AP %" width="65">
            <template #default="{ row }"><el-input-number v-model="row.ap_rate" :controls="false" :precision="1" size="small" style="width:100%" /></template>
          </el-table-column>
          <el-table-column label="MP %" width="65">
            <template #default="{ row }"><el-input-number v-model="row.mp_rate" :controls="false" :precision="1" size="small" style="width:100%" /></template>
          </el-table-column>
        </el-table>
      </div>

      <el-button type="primary" size="large" style="margin-top:16px;width:100%" @click="handleExport">
        {{ mode === 'zhoubao' ? 'Export weekly report' : 'Export settlement table' }}
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.upload-card {
  border: 2px dashed #dcdfe6;
  border-radius: 10px;
  padding: 20px;
  text-align: center;
  min-height: 180px;
  transition: all 0.2s;
}
.upload-card:not(.disabled):hover {
  border-color: #409eff;
  background: #f0f7ff;
}
.upload-card.selected {
  border-color: #409eff;
  border-style: solid;
  background: #f0f7ff;
}
.upload-card.disabled {
  opacity: 0.5;
}
.upload-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}
.upload-placeholder {
  font-size: 12px;
  color: #909399;
}
.file-list {
  margin-top: 8px;
  max-height: 80px;
  overflow-y: auto;
}
.file-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2px 6px;
  background: #f0f9eb;
  border-radius: 3px;
  margin-bottom: 2px;
  font-size: 12px;
}
.file-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}
</style>
