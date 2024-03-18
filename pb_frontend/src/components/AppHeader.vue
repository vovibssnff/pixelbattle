<template>
  <div class="header">
    <img href="#default" class="banner" src="@/img/vova.jpg">
  </div>
</template>

<script>
export default {

}
</script>

<style scoped>
.header {
  width: 100%;
}

.banner {
  width: 100%;
  height: auto; /* This ensures the image maintains its aspect ratio */
  display: block; /* Ensures that the image behaves as a block element */
}
</style>