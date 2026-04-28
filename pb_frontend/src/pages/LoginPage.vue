<template>
  <div class="login-page">
    <app-header/>
    <div class="local-auth">
      <h3>Вход по логину</h3>
      <div
        v-if="errorMessage"
        class="auth-flash auth-flash--error"
        role="alert"
      >
        {{ errorMessage }}
      </div>
      <form class="auth-form" @submit.prevent="onSubmit">
        <label>Логин <input v-model.trim="username" type="text" required autocomplete="username" /></label>
        <label>Пароль <input v-model="password" type="password" required autocomplete="current-password" /></label>
        <button type="submit" class="auth-submit" :disabled="submitting">
          {{ submitting ? 'Вход…' : 'Войти' }}
        </button>
      </form>
      <router-link to="/register" class="reg-link">Регистрация</router-link>
    </div>
  </div>

</template>

<script>
import AppHeader from "@/components/AppHeader.vue";
import { postPasswordAuth } from "@/utils/authHttp";

export default {
  components: {AppHeader},
  data() {
    return {
      username: "",
      password: "",
      errorMessage: "",
      submitting: false,
    };
  },
  beforeCreate() {
    document.title='Login to Pixelbattle'
  },
  methods: {
    async onSubmit() {
      this.errorMessage = "";
      this.submitting = true;
      try {
        const r = await postPasswordAuth("/api/login", {
          username: this.username,
          password: this.password,
        });
        if (!r.ok) {
          this.errorMessage = r.message;
        }
      } finally {
        this.submitting = false;
      }
    },
  },
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

.local-auth {
  margin-top: 2rem;
  color: #134293;
}

.local-auth h3 {
  margin-bottom: 0.75rem;
}

.auth-flash {
  max-width: 320px;
  margin: 0 auto 1rem;
  padding: 12px 14px;
  border-radius: 8px;
  text-align: left;
  font-size: 14px;
  line-height: 1.45;
  border: 1px solid transparent;
  box-sizing: border-box;
}

.auth-flash--error {
  background: rgba(139, 28, 28, 0.09);
  border-color: rgba(139, 28, 28, 0.35);
  color: #6b1515;
}

.auth-form {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  max-width: 320px;
  margin: 0 auto;
}

.auth-form label {
  display: flex;
  flex-direction: column;
  width: 100%;
  text-align: left;
  font-size: 14px;
}

.auth-form input {
  margin-top: 4px;
  padding: 6px 8px;
}

.auth-submit {
  margin-top: 0.5rem;
  padding: 8px 16px;
  cursor: pointer;
  background: transparent;
  border: 2px solid #134293;
  color: #134293;
  border-radius: 6px;
}

.auth-submit:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.reg-link {
  display: inline-block;
  margin-top: 1rem;
  color: #134293;
}
</style>
