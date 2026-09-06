<script setup>
import { ref, onMounted } from 'vue'
import SkuLibrary from './views/SkuLibrary.vue'
import ConfigInput from './views/ConfigInput.vue'
import ExportPage from './views/ExportPage.vue'
import JieAnPage from './views/JieAnPage.vue'
import DatabasePage from './views/DatabasePage.vue'
import { startNew } from './api'

const activeTab = ref('sku')
const jieAnMode = ref(false) // whether settlement mode is active

onMounted(async () => {
  try {
    await startNew()
  } catch (e) {}
})

function enterJieAn() {
  jieAnMode.value = true
}
function exitJieAn() {
  jieAnMode.value = false
}
</script>

<template>
  <div>
    <header class="app-header">
      <div>
        <h1>Channel P&L Planner</h1>
        <div class="sub">SKU Library -> Monthly Config -> Export</div>
      </div>
    </header>

    <div class="app-container">
      <!-- Settlement mode (independent full screen) -->
      <JieAnPage v-if="jieAnMode" @back="exitJieAn" />

      <!-- Planning mode -->
      <el-tabs v-model="activeTab" type="card" v-else>
        <el-tab-pane label="1. SKU Library" name="sku">
          <SkuLibrary v-if="activeTab === 'sku'" />
        </el-tab-pane>
        <el-tab-pane name="config">
          <template #label>
            <span>2. Monthly Config</span>
            <el-button size="small" type="warning" link style="margin-left:4px"
              @click="enterJieAn">Settlement Mode</el-button>
          </template>
          <ConfigInput v-if="activeTab === 'config'" />
        </el-tab-pane>
        <el-tab-pane label="3. Export" name="export">
          <ExportPage v-if="activeTab === 'export'" />
        </el-tab-pane>
        <el-tab-pane label="4. Data Update" name="database">
          <DatabasePage v-if="activeTab === 'database'" />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>
