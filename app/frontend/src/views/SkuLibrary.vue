<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listSkuLibrary, saveSku, deleteSku, searchMaterials } from '../api'

const skuList = ref([])
const loading = ref(false)

// New SKU form
const newSkuName = ref('')
const builderItems = ref([])

// Material search (Element Plus el-select remote mode)
const selectedMat = ref('')
const searchResults = ref([])
const newBags = ref(1)

async function refresh() {
  loading.value = true
  try {
    const res = await listSkuLibrary()
    skuList.value = (res && res.data) ? res.data : []
  } catch (e) {
    skuList.value = []
    ElMessage.error('Failed to load the SKU library')
  } finally {
    loading.value = false
  }
}
// Use onMounted to make sure the DOM is ready before loading (the Wails bindings may not
// be ready yet at startup)
onMounted(refresh)

// remote-method callback: triggered as the user types
async function handleSearch(q) {
  if (!q || !q.trim()) {
    searchResults.value = []
    return
  }
  try {
    searchResults.value = (await searchMaterials(q.trim(), 30)).data
  } catch (e) {
    searchResults.value = []
  }
}

// After selecting a material, reset the bag count to 1 for easy input
function onMatChange() {
  if (selectedMat.value) {
    newBags.value = 1
  }
}

function addMaterial() {
  if (!selectedMat.value) {
    ElMessage.warning('Search and select a material first')
    return
  }
  const mat = searchResults.value.find(m => m.mat_id === selectedMat.value)
  if (!mat) {
    ElMessage.warning('Material info lost, please select again')
    return
  }
  // Price completeness validation
  const issues = []
  if (!mat.retail_spec || mat.retail_spec <= 0) issues.push('spec')
  if (!mat.ec_assess_revenue || mat.ec_assess_revenue <= 0) issues.push('assess revenue')
  if (!mat.std_cost || mat.std_cost <= 0) issues.push('standard cost')
  if (issues.length > 0) {
    ElMessage.warning(`This material is missing ${issues.join(' / ')} - not recommended, please pick another one`)
    return
  }
  // If it already exists, accumulate the bag count
  const exist = builderItems.value.find(it => it.mat_id === selectedMat.value)
  if (exist) {
    exist.bags += newBags.value
    ElMessage.success(`${mat.name.slice(0, 12)} bags +${newBags.value} (total ${exist.bags})`)
  } else {
    builderItems.value.push({
      mat_id: mat.mat_id,
      name: mat.name,
      bags: newBags.value,
    })
    ElMessage.success(`Added: ${mat.name.slice(0, 12)} x ${newBags.value}`)
  }
  // Clear the selection, ready to add the next one
  selectedMat.value = ''
  searchResults.value = []
  newBags.value = 1
}

function removeBuilderItem(idx) {
  builderItems.value.splice(idx, 1)
}

async function handleSaveSku() {
  if (!newSkuName.value.trim()) {
    ElMessage.warning('Please enter a SKU name')
    return
  }
  if (builderItems.value.length === 0) {
    ElMessage.warning('Please add at least one material')
    return
  }
  try {
    const res = await saveSku(newSkuName.value.trim(), builderItems.value)
    const d = res?.data || {}
    if (d.ok === false) {
      // Validation failed, show the missing details
      const missing = d.missing || []
      let detail = missing.map(m => `• ${m.mat_id} ${m.name}: ${m.issue}`).join('<br>')
      ElMessageBox.alert(
        `<b>Cannot save SKU</b><br>The following materials have incomplete data:<br><br>${detail}<br><br>` +
        `<span style="color:#909399;font-size:12px">Ask an administrator to fill in the missing prices in the master data (P&L workbook)</span>`,
        'Incomplete material data', { dangerouslyUseHTMLString: true, type: 'error' }
      )
      return
    }
    ElMessage.success(`SKU "${newSkuName.value}" saved`)
    newSkuName.value = ''
    builderItems.value = []
    selectedMat.value = ''
    searchResults.value = []
    refresh()
  } catch (e) {
    ElMessage.error('Save failed: ' + (e?.message || String(e)))
  }
}

// Check whether a material's prices are complete (spec>0 + has assess revenue + has standard cost)
function isComplete(m) {
  return m.retail_spec > 0 && m.ec_assess_revenue != null && m.ec_assess_revenue > 0
    && m.std_cost != null && m.std_cost > 0
}
// Dropdown option label (annotated when incomplete)
function optionLabel(m) {
  const base = `${m.mat_id} | ${m.name}`
  return isComplete(m) ? base : base + ' [missing price]'
}

async function handleDelete(name) {
  try {
    await deleteSku(name)
    ElMessage.success(`Deleted "${name}"`)
    refresh()
  } catch (e) {
    ElMessage.error('Delete failed')
  }
}

function loadEdit(sku) {
  newSkuName.value = sku.sku_name
  builderItems.value = sku.sub_items.map(it => ({ ...it }))
  // Scroll to the top create area
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
</script>

<template>
  <div>
    <!-- Create SKU -->
    <div class="section-card">
      <div class="section-title">Create SKU</div>
      <el-form label-position="top">
        <el-form-item label="SKU name">
          <el-input v-model="newSkuName" placeholder="e.g. Demo Wonton 300g 4+4" style="max-width:400px" />
        </el-form-item>
      </el-form>

      <div style="margin-bottom:12px">
        <div class="rate-input-tip" style="margin-bottom:6px">Add materials - type a material ID or name keyword, the dropdown autocompletes</div>
        <el-row :gutter="8">
          <el-col :span="11">
            <el-select
              v-model="selectedMat"
              filterable
              remote
              clearable
              :remote-method="handleSearch"
              :loading="loading"
              placeholder="Enter material ID or name (e.g. Wonton / 1000001)"
              style="width:100%"
              @change="onMatChange"
            >
              <el-option
                v-for="m in searchResults"
                :key="m.mat_id"
                :label="optionLabel(m)"
                :value="m.mat_id"
              >
                <span>{{ m.mat_id }} | {{ m.name }}</span>
                <span v-if="!isComplete(m)" style="color:#F56C6C;font-size:12px;margin-left:6px">[missing price]</span>
              </el-option>
              <template #empty>
                <div style="padding:12px;color:#909399;font-size:13px">
                  {{ loading ? 'Searching...' : 'Type a keyword to search materials' }}
                </div>
              </template>
            </el-select>
          </el-col>
          <el-col :span="4">
            <el-input-number v-model="newBags" :min="1" :controls="false" style="width:100%" />
          </el-col>
          <el-col :span="3">
            <el-button type="primary" @click="addMaterial">Add</el-button>
          </el-col>
        </el-row>
      </div>

      <!-- Added materials -->
      <div v-if="builderItems.length > 0">
        <div class="rate-input-tip" style="margin:12px 0 6px">Materials in this SKU:</div>
        <el-table :data="builderItems" border size="small" style="margin-bottom:12px">
          <el-table-column label="Material ID" prop="mat_id" width="140" />
          <el-table-column label="Material name" prop="name" min-width="200" />
          <el-table-column label="Bags" width="100">
            <template #default="{ row }">
              <el-input-number v-model="row.bags" :min="1" :controls="false" size="small" style="width:80px" />
            </template>
          </el-table-column>
          <el-table-column label="Actions" width="70">
            <template #default="{ $index }">
              <el-button type="danger" size="small" link @click="removeBuilderItem($index)">X</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div style="display:flex;gap:12px">
          <el-button type="primary" @click="handleSaveSku">Save as SKU</el-button>
          <el-button @click="builderItems = []; newSkuName = ''">Clear</el-button>
        </div>
      </div>
      <el-empty v-else description="No materials added yet - search above and add them" :image-size="60" />
    </div>

    <!-- Existing SKU library -->
    <div class="section-card">
      <div class="section-title">SKU Library ({{ skuList.length }})</div>
      <el-empty v-if="skuList.length === 0" description="SKU library is empty" :image-size="60" />
      <el-collapse v-else>
        <el-collapse-item
          v-for="sku in skuList"
          :key="sku.sku_name"
          :title="`${sku.sku_name} (${sku.sub_items.length} materials)`"
        >
          <el-table :data="sku.sub_items" border size="small">
            <el-table-column label="Material ID" prop="mat_id" width="140" />
            <el-table-column label="Material name" prop="name" min-width="200" />
            <el-table-column label="Bags" prop="bags" width="80" />
          </el-table>
          <div style="margin-top:8px;display:flex;gap:8px">
            <el-button size="small" @click="loadEdit(sku)">Load for editing</el-button>
            <el-button size="small" type="danger" @click="handleDelete(sku.sku_name)">Delete</el-button>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>
