<template>
  <div class="app">
    <router-view></router-view>
  </div>
</template>

<style>

</style>
<script setup>
  import { useMeta } from 'vue-meta'
  useMeta({
    title: 'pixelbattle',
  })
</script>