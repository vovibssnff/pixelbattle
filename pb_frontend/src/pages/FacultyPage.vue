<template>
  <div>
    <app-header/>
  </div>
  <div class="radio-container">
    <div class="radio-container-container">
      <label class="radio-label">
        <input type="radio" v-model="selectedFaculty" value="KTU"> КТУ
      </label>
      <label class="radio-label">
        <input type="radio" v-model="selectedFaculty" value="TINT"> ТИНТ
      </label>
      <label class="radio-label">
        <input type="radio" v-model="selectedFaculty" value="FTMF"> ФТМФ
      </label>
      <label class="radio-label">
        <input type="radio" v-model="selectedFaculty" value="FTMI"> ФТМИ
      </label>
      <label class="radio-label">
        <input type="radio" v-model="selectedFaculty" value="NOZH"> НОЖ
      </label>
      <button class="send-button" @click="sendData">Подтвердить</button>
    </div>
  </div>
</template>
  
<script>
  import AppHeader from "@/components/AppHeader.vue";
  import { mapGetters, mapMutations } from "vuex";
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
    },
    methods: {
        sendData() {
          console.log(this.selectedFaculty);
          axios.post('/api/faculty', {
            faculty: this.selectedFaculty,
          }).then(
            // this.currentAuthorized(true),
            this.$router.push("/")
          )
            .catch(error => {
                console.error(error);
            });
        }
    }
  }
</script>
  
<style scoped>
.radio-container {
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
  height: 100vh;
  margin-top: 20px; /* Add margin to push the radio container down */
}

.radio-container-container {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center; /* Center horizontally */
}

.radio-label {
  margin-bottom: 10px;
  font-size: 16px;
}

.send-button {
  padding: 10px 20px;
  margin-top: 20px;
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