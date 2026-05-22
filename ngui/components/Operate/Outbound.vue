<script lang="ts" setup>
const { t } = useI18n()

let isVisible = $ref(false)
let currentOutbound = $ref('')
let setting = $ref<{
  probeInterval: string
  probeURL: string
  type: string
}>()
let supportedTypes = $ref<string[]>(['leastping', 'leastload', 'roundrobin', 'random', 'health'])

const viewOutbound = async(outbound: string) => {
  isVisible = true
  currentOutbound = outbound
  const { data } = await useV2Fetch(`outbound?outbound=${outbound}`).json()

  setting = data.value.data.setting
  if (data.value?.data?.supportedTypes?.length)
    supportedTypes = data.value.data.supportedTypes
}

const deleteOutbound = async(outbound: string) => {
  const { data } = await useV2Fetch('outbound').delete({ outbound }).json()
  proxies.value.outbounds = data.value.data.outbounds
  isVisible = false
}

const editOutbound = async(outbound: string) => {
  const { data } = await useV2Fetch('outbound').put({ outbound, setting }).json()
  if (data.value.code === 'SUCCESS')
    ElMessage.success(t('common.success'))
}
</script>

<template>
  <ElDropdown class="ml-2">
    <ElButton size="small">{{ proxies.currentOutbound.toUpperCase() }}</ElButton>
    <template #dropdown>
      <ElDropdownMenu class="w-28">
        <ElDropdownItem v-for="i in proxies.outbounds" :key="i" class="flex justify-between">
          <div class="w-full" @click="proxies.currentOutbound = i">{{ i }}</div>
          <UnoIcon class="ri:settings-fill ml-2" @click="viewOutbound(i)" />
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>

  <ElDialog v-model="isVisible" :title="`${currentOutbound}- ${$t('common.outboundSetting')}`">
    <ElForm>
      <ElFormItem label="probeURL">
        <ElInput v-model="setting!.probeURL" />
      </ElFormItem>
      <ElFormItem label="probeInterval">
        <ElInput v-model="setting!.probeInterval" />
      </ElFormItem>
      <ElFormItem label="type">
        <ElSelect v-model="setting!.type">
          <ElOption
            v-for="t in supportedTypes"
            :key="t"
            :label="t === 'health' ? 'health (alias of leastping)' : t"
            :value="t"
          />
        </ElSelect>
      </ElFormItem>
    </ElForm>
    <template #footer>
      <span class="dialog-footer">
        <ElButton @click="deleteOutbound(currentOutbound)">{{ $t('operations.delete') }}</ElButton>
        <ElButton @click="isVisible = false">{{ $t('operations.cancel') }}</ElButton>
        <ElButton type="primary" @click="editOutbound(currentOutbound)">
          {{ $t("operations.confirm") }}
        </ElButton>
      </span>
    </template>
  </ElDialog>
</template>
