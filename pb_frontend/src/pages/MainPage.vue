<template>
  <div>
<!--    <app-header/>-->
    <g-l-canvas/>
<!--    <app-canvas/>-->
  </div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";
import GLCanvas from "@/components/GLCanvas.vue";
// import AppCanvas from "@/components/AppCanvas.vue";

export default {
  components: {GLCanvas, AppHeader}
}
</script>

<style scoped>

</style>