<script setup>
import FileRenaming from './components/FileRenaming.vue'
</script>

<template>
  <n-config-provider>
    <n-message-provider>
      <FileRenaming />
    </n-message-provider>
  </n-config-provider>
</template>
