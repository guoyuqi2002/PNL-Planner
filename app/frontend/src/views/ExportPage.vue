<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getMonths, getDepartments, getDimensions,
  exportSplit, exportPivot, openExportFolder,
} from '../api'

const months = ref([])
const departments = ref([])

// Export type: split | pivot
const exportType = ref('split')
const selMonth = ref('')
const selDept = ref('')
const lastResult = ref(null)

// Config count of the month
const monthConfigCount = ref(0)

onMounted(async () => {
  months.value = (await getMonths()).data || []
  departments.value = (await getDepartments()).data || []
  selMonth.value = months.value.find(m => m === '2026-08') || months.value[0] || ''
  selDept.value = departments.value[0] || ''
  await checkMonthConfig()
})

async function checkMonthConfig() {
  if (!selMonth.value) return
  const res = await getDimensions(selMonth.value)
  monthConfigCount.value = (res && res.data) ? res.data.length : 0
}

// ========== Split table ==========
async function handleGenerateSplit() {
  try {
    const res = await exportSplit(selMonth.value)
    if (res && res.data) {
      showResult('Split table generated', res.data)
    }
  } catch (e) {
    ElMessage.error('Generation failed: ' + (e?.message || String(e)))
  }
}

// ========== Drill-down table ==========
async function handleGeneratePivot() {
  try {
    const res = await exportPivot(selDept.value)
    if (res && res.data) {
      showResult('Drill-down table generated', res.data)
    }
  } catch (e) {
    ElMessage.error('Generation failed: ' + (e?.message || String(e)))
  }
}

// Generic result dialog
function showResult(title, path) {
  if (!path) return
  lastResult.value = { title, path }
  ElMessageBox.confirm(
    `<b>${title}</b><br><code style="word-break:break-all;font-size:12px">${path}</code>`,
    'Done',
    {
      dangerouslyUseHTMLString: true,
      type: 'success',
      confirmButtonText: 'Open folder',
      cancelButtonText: 'Close',
    }
  ).then(() => {
    if (path) openExportFolder(path)
  }).catch(() => {})
}
</script>

<template>
  <div>
    <!-- Export type selection -->
    <div class="section-card">
      <div class="section-title">Select export type</div>
      <el-radio-group v-model="exportType">
        <el-radio-button value="split">Interest e-commerce split table</el-radio-button>
        <el-radio-button value="pivot">Influencer distribution drill-down table</el-radio-button>
      </el-radio-group>
    </div>

    <!-- Split table -->
    <div class="section-card" v-if="exportType === 'split'">
      <div class="section-title">Interest e-commerce split table</div>
      <el-form label-position="right" label-width="100px" inline style="margin-bottom:16px">
        <el-form-item label="Month">
          <el-select v-model="selMonth" style="width:160px" @change="checkMonthConfig">
            <el-option v-for="m in months" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
      </el-form>

      <div class="action-card">
        <div class="action-icon">New</div>
        <div class="action-info">
          <div class="action-title">Generate split table</div>
          <div class="action-desc">Reads all configs of the month and generates a summary sheet + 4 channel split sheets</div>
        </div>
        <el-button type="primary" @click="handleGenerateSplit">Generate</el-button>
      </div>
      <div class="config-hint" v-if="monthConfigCount === 0">
        Warning: {{ selMonth }} has no config data yet; fill in the Monthly Config page first
      </div>
      <div class="config-hint ok" v-else>
        Info: {{ selMonth }} has {{ monthConfigCount }} dimension(s) configured
      </div>
    </div>

    <!-- Drill-down table -->
    <div class="section-card" v-else>
      <div class="section-title">Influencer distribution drill-down table</div>
      <el-alert type="info" :closable="false" style="margin-bottom:16px">
        The drill-down table aggregates by <b>department + channel</b> only. It reads all
        months' configs of the department and generates a fee matrix by channel x month.
      </el-alert>
      <el-form label-position="right" label-width="100px" inline style="margin-bottom:16px">
        <el-form-item label="Department">
          <el-select v-model="selDept" style="width:160px">
            <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
      </el-form>

      <div class="action-card">
        <div class="action-icon">New</div>
        <div class="action-info">
          <div class="action-title">Generate drill-down table</div>
          <div class="action-desc">Reads all months of the department and generates a 4-block x 12-month matrix</div>
        </div>
        <el-button type="primary" @click="handleGeneratePivot">Generate</el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.action-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fafbfc;
}
.action-icon {
  font-size: 28px;
}
.action-info {
  flex: 1;
}
.action-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}
.action-desc {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}
.config-hint {
  margin-top: 12px;
  padding: 8px 12px;
  background: #fdf6ec;
  border-radius: 4px;
  font-size: 13px;
  color: #e6a23c;
}
.config-hint.ok {
  background: #f0f9eb;
  color: #67c23a;
}
</style>
