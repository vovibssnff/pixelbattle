<template>
  <div>
    <app-header/>
    <v-k-login-btn/>
  </div>

</template>

<script>
import AppHeader from "@/components/AppHeader.vue";
import VKLoginBtn from "@/components/VKLoginBtn.vue";

export default {
  components: {AppHeader, VKLoginBtn},
  beforeCreate() {
    document.title='MEGA PixelBattle: Login'
  }
}


</script>

<style scoped>

</style>