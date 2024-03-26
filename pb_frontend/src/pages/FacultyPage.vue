<template>
  <div>
    <app-header/>
  </div>
  <form action="/api/faculty" method="POST">
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
        // sendData() {
        //   console.log(this.selectedFaculty);
        //   axios.post('/api/faculty', {
        //     faculty: this.selectedFaculty,
        //   }).then(
        //     // this.currentAuthorized(true),
        //     this.$router.push("/")
        //   )
        //     .catch(error => {
        //         console.error(error);
        //     });
        // }
        sendData() {
          // Ensure a faculty is selected
          if (!this.selectedFaculty) {
            alert('Please select a faculty.');
            return;
          }

          axios.post('/api/faculty', {
            faculty: this.selectedFaculty
          });
        }
    }
  }
</script>
  
<style scoped>
.radio-container {
  display: flex;
  align-items: center;
  flex-direction: column;
  height: 100vh;
  margin-top: 5%; /* Add margin to push the radio container down */
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