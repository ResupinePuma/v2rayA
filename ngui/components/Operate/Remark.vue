<script lang="ts" setup>
const { data: row } = defineProps<{ data: any }>()

let input = $ref('')
let selectedOutbounds = $ref<string[]>(['proxy'])
let isVisible = $ref(false)

const openDialog = () => {
  input = row.remarks || ''
  selectedOutbounds = Array.isArray(row.outbounds) && row.outbounds.length ? [...row.outbounds] : ['proxy']
  isVisible = true
}

const remarkSubscription = async() => {
  const { data } = await useV2Fetch('subscription').patch({
    subscription: {
      ...row,
      remarks: input,
      outbounds: selectedOutbounds.length ? selectedOutbounds : ['proxy']
    }
  }).json()

  proxies.value.subs = data.value.data.touch.subscriptions
  isVisible = false
}
</script>

<template>
  <ElButton size="small" class="mr-3" @click="openDialog">
    <UnoIcon class="ri:edit-2-line mr-1" />{{ $t('operations.modify') }}
  </ElButton>

  <ElDialog v-model="isVisible" :title="$t('operations.import')">
    {{ $t("configureSubscription.title") }}
    <ElInput v-model="input" :placeholder="$t('subscription.remarks')" />
    <ElSelect v-model="selectedOutbounds" multiple class="mt-2 w-full" placeholder="Outbound groups">
      <ElOption v-for="outbound in proxies.outbounds" :key="outbound" :label="outbound" :value="outbound" />
    </ElSelect>
    <template #footer>
      <span class="dialog-footer">
        <ElButton @click="isVisible = false">{{ $t('operations.cancel') }}</ElButton>
        <ElButton type="primary" @click="remarkSubscription">
          {{ $t("operations.confirm") }}
        </ElButton>
      </span>
    </template>
  </ElDialog>
</template>
