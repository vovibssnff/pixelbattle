<template>
  <div id="VKIDAuthContainer">
    <button id="VKIDSDKAuthButton" class="VkIdWebSdk__button VkIdWebSdk__button_reset" @click="handleClick">
      <div class="VkIdWebSdk__button_container">
        <div class="VkIdWebSdk__button_icon">
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M4.54648 4.54648C3 6.09295 3 8.58197 3 13.56V14.44C3 19.418 3 21.907 4.54648 23.4535C6.09295 25 8.58197 25 13.56 25H14.44C19.418 25 21.907 25 23.4535 23.4535C25 21.907 25
           19.418 25 14.44V13.56C25 8.58197 25 6.09295 23.4535 4.54648C21.907 3 19.418 3 14.44 3H13.56C8.58197 3 6.09295 3 4.54648 4.54648ZM6.79999 10.15C6.91798 15.8728 9.92951 19.31 14.8932 19.31H15.1812V16.05C16.989 16.2332 18.3371
           17.5682 18.8875 19.31H21.4939C20.7869 16.7044 18.9535 15.2604 17.8141 14.71C18.9526 14.0293 20.5641 12.3893 20.9436 10.15H18.5722C18.0747 11.971 16.5945 13.6233 15.1803 13.78V10.15H12.7711V16.5C11.305 16.1337 9.39237 14.3538 9.314 10.15H6.79999Z" fill="white"/>
          </svg>
        </div>
        <div class="VkIdWebSdk__button_text">
          Войти через VK ID
        </div>
      </div>
    </button>
  </div>
</template>

<script>
import * as VKID from "@vkid/sdk";
import AppHeader from "@/components/AppHeader.vue";
// import { Connect } from "@vkontakte/superappkit";
import { mapMutations } from "vuex";
export default {
  components: {AppHeader},
  methods: {
    ...mapMutations('UserModule', ['setAuthorized']),
    auth(val) {
      this.setAuthorized(val);
    },
    handleClick() {
      this.auth('in_progress')
      VKID.Auth.login();
    },
  },
  beforeMount() {
    VKID.Config.set({
      app: ,
      redirectUrl: 'https://megapixelbattle/api/login'
    });
    const button = document.getElementById('VKIDSDKAuthButton');
    if (button) {
      button.onclick = this.handleClick;
    }
  },
}
</script>

<style scoped>
.VkIdWebSdk__button_reset {
  border: none;
  margin: 0;
  padding: 0;
  width: auto;
  overflow: visible;
  background: transparent;
  color: inherit;
  font: inherit;
  line-height: normal;
  -webkit-font-smoothing: inherit;
  -moz-osx-font-smoothing: inherit;
  -webkit-appearance: none;
}

.VkIdWebSdk__button {
  background: #0077ff;
  cursor: pointer;
  transition: all .1s ease-out;
}

.VkIdWebSdk__button:hover{
  opacity: 0.8;
}

.VkIdWebSdk__button:active {
  opacity: .7;
  transform: scale(.97);
}

.VkIdWebSdk__button {
  border-radius: 8px;
  width: 100%;
  min-height: 44px;
}

.VkIdWebSdk__button_container {
  display: flex;
  align-items: center;
  padding: 8px 10px;
}

.VkIdWebSdk__button_icon + .VkIdWebSdk__button_text {
  margin-left: -28px;
}

.VkIdWebSdk__button_text {
  display: flex;
  font-family: -apple-system, system-ui, "Helvetica Neue", Roboto, sans-serif;
  flex: 1;
  justify-content: center;
  color: #ffffff;
}

#VKIDAuthContainer {
  margin-top: 10%;
}
</style>