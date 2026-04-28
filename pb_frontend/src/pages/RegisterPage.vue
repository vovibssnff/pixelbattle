<template>
  <div class="register-page">
    <app-header/>
    <h2>Регистрация</h2>
    <div
      v-if="errorMessage"
      class="auth-flash auth-flash--error"
      role="alert"
    >
      {{ errorMessage }}
    </div>
    <form class="form-container" @submit.prevent="onSubmit">
      <label class="field">Логин (3–32 символа: буквы, цифры, _)
        <input
          v-model.trim="username"
          type="text"
          required
          minlength="3"
          maxlength="32"
          pattern="[a-zA-Z0-9_]+"
          autocomplete="username"
        />
      </label>
      <label class="field">Пароль (минимум 8 символов)
        <input
          v-model="password"
          type="password"
          required
          minlength="8"
          autocomplete="new-password"
        />
      </label>
      <div class="radio-container">
        <span class="label-title">Факультет:</span>
        <label class="radio-label">
          <input v-model="faculty" type="radio" value="KTU" required> КТУ
        </label>
        <label class="radio-label">
          <input v-model="faculty" type="radio" value="TINT"> ТИНТ
        </label>
        <label class="radio-label">
          <input v-model="faculty" type="radio" value="FTMF"> ФТМФ
        </label>
        <label class="radio-label">
          <input v-model="faculty" type="radio" value="FTMI"> ФТМИ
        </label>
        <label class="radio-label">
          <input v-model="faculty" type="radio" value="NOZH"> НОЖ
        </label>
      </div>
      <button type="submit" class="send-button" :disabled="submitting">
        {{ submitting ? 'Создание…' : 'Создать аккаунт' }}
      </button>
      <router-link to="/login" class="back-link">Уже есть аккаунт? Войти</router-link>
    </form>
  </div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";
import { postPasswordAuth } from "@/utils/authHttp";

export default {
  components: { AppHeader },
  data() {
    return {
      username: "",
      password: "",
      faculty: "",
      errorMessage: "",
      submitting: false,
    };
  },
  mounted() {
    document.title = 'Register — Pixelbattle'
  },
  methods: {
    async onSubmit() {
      this.errorMessage = "";
      if (!this.faculty) {
        this.errorMessage = "Выберите факультет.";
        return;
      }
      this.submitting = true;
      try {
        const r = await postPasswordAuth("/api/register", {
          username: this.username,
          password: this.password,
          faculty: this.faculty,
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
.register-page {
  font-size: 18px;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
  padding-top: 50px;
  box-sizing: border-box;
  position: absolute;
  width: 100dvw;
  min-height: 100dvh;
  left: 0;
  top: 0;
  background-image: url("@/img/paper-bg.jpg");
  background-size: cover;
  text-align: center;
}

h2 {
  color: #134293;
  margin-top: 30px;
}

.auth-flash {
  max-width: 400px;
  width: 100%;
  margin-bottom: 1rem;
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

.form-container {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  max-width: 400px;
  width: 100%;
  margin-top: 20px;
  gap: 12px;
}

.field {
  display: flex;
  flex-direction: column;
  text-align: left;
  color: #134293;
  font-size: 15px;
}

.field input {
  margin-top: 4px;
  padding: 8px;
}

.radio-container {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  margin-top: 8px;
}

.label-title {
  color: #134293;
  margin-bottom: 6px;
}

.radio-label {
  margin-bottom: 6px;
  color: #134293;
}

.send-button {
  border: 0;
  margin-top: 12px;
  background-image: url("@/img/button.png");
  cursor: pointer;
  transition: all .1s ease-out;
  background-repeat: no-repeat;
  background-size: contain;
  background-position: center;
  height: 40px;
  width: 100%;
  max-width: 400px;
  background-color: transparent;
  color: #134293;
}

.send-button:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.back-link {
  margin-top: 16px;
  color: #134293;
}
</style>
