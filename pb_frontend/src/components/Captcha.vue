<template>
<div id="captcha-container"
     class="smart-captcha"
     data-sitekey="ysc1_iVmyqum34bmdy36kcLn2V4oJYqkdCogifEuoWBxP7fb9e386"
    ></div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";

export default {
  components: {AppHeader},
  mounted() {
    let captcha = document.createElement('script');
    captcha.setAttribute('src', 'https://smartcaptcha.yandexcloud.net/captcha.js');
    document.head.appendChild(captcha);
  }
}
</script>

<style scoped>
.smart-captcha {
  margin-top: 15px;
  margin-bottom: 20px;
}
</style>

