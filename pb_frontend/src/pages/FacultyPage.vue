<template>
  <div>
    <app-header/>
    <form action="/api/faculty" class="form-container">
      <div class="radio-container">
        <label class="radio-label">
          <input type="radio" name="faculty" value="KTU" required> КТУ
        </label>
        <label class="radio-label">
          <input type="radio" name="faculty" value="TINT"> ТИНТ
        </label>
        <label class="radio-label">
          <input type="radio" name="faculty" value="FTMF"> ФТМФ
        </label>
        <label class="radio-label">
          <input type="radio" name="faculty" value="FTMI"> ФТМИ
        </label>
        <label class="radio-label">
          <input type="radio" name="faculty" value="NOZH"> НОЖ
        </label>
      </div>
      <button type="submit" class="send-button">Подтвердить</button>
    </form>
  </div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";

export default {
  components: {AppHeader},
  mounted() {
    document.title='login'
  }
}
</script>

<style scoped>
.form-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  /* justify-content: center; */
  margin-top: 20px; /* Adjust this value as needed */
  transition: background 1s;
  filter: drop-shadow(1px 1px grey);
  font-family: Arial, sans-serif;
}

.radio-container {
  display: flex;
  align-items: center;
  flex-direction: column;
}

.radio-label {
  margin-bottom: 10px;
  font-size: 16px;
}

.send-button {
  padding: 10px 20px;
  /* margin-top: 20px; */
  background-color: #007bff;
  color: #fff;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  transition: background-color 0.3s ease;
  filter: drop-shadow(1px 1px grey);
  font-family: Arial, sans-serif;
}

.send-button:hover {
  background-color: #0056b3;
}
</style>
