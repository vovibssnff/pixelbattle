<template>
  <div>
    <app-header/>
    <form action="/api/faculty" method="POST" class="form-container">
      <div class="radio-container">
        <label class="radio-label">
          <input type="radio" name="selectedFaculty" value="KTU" required> КТУ
        </label>
        <label class="radio-label">
          <input type="radio" name="selectedFaculty" value="TINT"> ТИНТ
        </label>
        <label class="radio-label">
          <input type="radio" name="selectedFaculty" value="FTMF"> ФТМФ
        </label>
        <label class="radio-label">
          <input type="radio" name="selectedFaculty" value="FTMI"> ФТМИ
        </label>
        <label class="radio-label">
          <input type="radio" name="selectedFaculty" value="NOZH"> НОЖ
        </label>
      </div>
      <button type="submit" class="send-button">Подтвердить</button>
    </form>
  </div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";
import { mapGetters } from "vuex";
import axios from "axios";

export default {
  components: {AppHeader},
  computed: {
    ...mapGetters('UserModule', ['getID', 'getAuthorized']),
    currentID: {
      get() {
        return this.getID;
      },
      set(val) {
        this.$store.commit('UserModule/setID', val)
      }
    },
    currentAuthorized: {
      get() {
        return this.getAuthorized;
      },
      set(val) {
        this.$store.commit('UserModule/setAuthorized', val)
      }
    }
  },
  data() {
    return {
      selectedFaculty: 'КТУ',
    }
  },
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
}

.send-button:hover {
  background-color: #0056b3;
}
</style>
