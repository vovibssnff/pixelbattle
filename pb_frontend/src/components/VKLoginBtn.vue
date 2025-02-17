<template>
  <div ref="vkidContainer" class="vkid-widget"></div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";
import { mapMutations } from "vuex";

export default {
  components: { AppHeader },
  name: 'VKIDLogin',
  data() {
    return {
      vkRedirectUrl: 'https://' + window.location.hostname +  '/api/login',
      redirectUrl: '/api/login',
      userID: null,
      deviceID: null,
      refreshToken: null,
      accessToken: null,
      idToken: null,
    }
  },
  mounted() {
    const vkRedirectUrl = this.vkRedirectUrl;
    let vkscript = document.createElement('script');
    vkscript.setAttribute('src', 'https://unpkg.com/@vkid/sdk@<3.0.0/dist-sdk/umd/index.js');
    document.head.appendChild(vkscript);

    vkscript.onload = () => {
        // Wait until VKIDSDK is available on window
      const VKID = window.VKIDSDK;

      // Initialize VKID config
      VKID.Config.init({
        app: 51845999,
        redirectUrl: vkRedirectUrl,
        responseMode: VKID.ConfigResponseMode.Callback,
        source: VKID.ConfigSource.LOWCODE,
      });

      // Create the OneTap widget
      const oneTap = new VKID.OneTap();

      // Render the widget inside the container div
      this.auth('in_progress');
      oneTap.render({
        container: this.$refs.vkidContainer,
        showAlternativeLogin: true,
        contentId: 8
      })
      .on(VKID.WidgetEvents.ERROR, this.vkidOnError)
      .on(VKID.OneTapInternalEvents.LOGIN_SUCCESS, (payload) => {
        // console.log(payload);
        this.deviceID = payload.device_id;
        const code = payload.code;
        const deviceId = payload.device_id;
        // Exchange the code for authentication
        VKID.Auth.exchangeCode(code, deviceId)
          .then(this.vkidOnSuccess)
          .catch(this.vkidOnError);
      });
    }
  },
  methods: {
    ...mapMutations('UserModule', ['setAuthorized']),
    auth(val) {
      this.setAuthorized(val);
    },
    vkidOnSuccess(data) {
      // Handle successful login
      console.log(data);
      const usr = {
        userID: data.user_id,
        deviceID: this.deviceID,
        refreshToken: data.refresh_token,
        accessToken: data.access_token,
        idToken: data.id_token,
      };
      this.auth('in_progress');
      fetch(this.vkRedirectUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(usr)
      })
      .then((response) => response.json())
      .then((data) => {
        if (data.status === 'redirect_to_faculty') {
          this.$router.push('/faculty');
        } else if (data.status === 'redirect_to_main') {
          this.$router.push('/main');
        }
      })
      .catch((error) => {
        console.error('Error during the request:', error);
      });
    },
    vkidOnError(error) {
      console.error('VKID Error:', error);
    }
  }
}
</script>

<style scoped>
.vkid-widget {
  display: grid;
  place-content: center;
  margin-top: 20px;
}
</style>
