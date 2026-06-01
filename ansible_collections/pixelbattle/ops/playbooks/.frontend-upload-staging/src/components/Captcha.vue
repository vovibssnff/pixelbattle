<template>
  <div
    id="captcha-container"
    class="smart-captcha"
    :data-sitekey="siteKey"
  ></div>
</template>

<script>
export default {
  data() {
    return {
      siteKey: process.env.VUE_APP_YANDEX_CAPTCHA_SITEKEY || 'replace_me',
    };
  },
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

