<template>
  <div class="login-page">
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
    document.title='Login to Pixelbattle'
  }
}


</script>

<style scoped>

.login-page {
  
  font-size: 18px;  
  display: flex;
  flex-direction: column;
  justify-content: center;
  box-sizing: border-box;
  position: fixed;
  background-image: url("@/img/paper-bg.jpg");
  background-size: cover;
  transition: all 0.5s ease-in-out;
  text-align: center;
  position: absolute;
  width: 100dvw;
  height: 100dvh;
  left: 0;
  top: 0;
  padding-inline: 10%;
}
</style>