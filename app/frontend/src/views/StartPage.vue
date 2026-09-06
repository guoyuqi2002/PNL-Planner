<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSessionState, startNew } from '../api'

const emit = defineEmits(['session-changed'])

const session = ref({ mode: '' })
const loading = ref(false)

onMounted(async () => {
  const res = await getSessionState()
  session.value = (res && res.data) ? res.data : { mode: '' }
  // Enter "new" mode automatically on startup
  if (!session.value.mode || session.value.mode === '') {
    await handleStartNew()
  }
})

async function handleStartNew() {
  loading.value = true
  try {
    await startNew()
    const res = await getSessionState()
    session.value = (res && res.data) ? res.data : {}
    ElMessage.success('Ready, using the default shop configuration')
    emit('session-changed')
  } catch (e) {
    ElMessage.error('Operation failed: ' + (e?.message || String(e)))
  } finally {
    loading.value = false
  }
}

defineExpose({ refreshSession: async () => {
  const res = await getSessionState()
  session.value = (res && res.data) ? res.data : {}
}})
</script>

<template>
  <div>
    <el-alert type="success" :closable="false" style="margin-bottom:20px">
      <b>Work mode: create a new Excel file</b><br />
      All data is entered via the SKU Library + Monthly Config, and brand-new split /
      drill-down tables are generated at the end.
    </el-alert>

    <div class="section-card">
      <div class="section-title">Next steps</div>
      <el-steps :active="0" simple>
        <el-step title="1. Start (this page)" description="Confirm new mode" />
        <el-step title="2. SKU Library" description="Build combo SKUs" />
        <el-step title="3. Monthly Config" description="Enter data per dimension" />
        <el-step title="4. Export" description="Generate a new Excel file" />
      </el-steps>
      <div class="rate-input-tip" style="margin-top:16px">
        Tip: shops/modes follow the default configuration (the template structure). After
        finishing the monthly config, go to the Export page to generate the Excel file.
      </div>
    </div>
  </div>
</template>
