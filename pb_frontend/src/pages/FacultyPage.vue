<template>
  <div class="faculty-page">
    <app-header/>
    <h2>Выберите факультет: </h2>
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
      <br/>
      <div class="tutorial">
      Слушай же, о путник, и ведай, как управляться с волшебной палитрой:
      <br/><br/>
      ПКМ (передвижение пальца на телефоне) — держи перо крепко, дабы узоры чертить.<br/>
      ЛКМ (касание экрана на телефоне) — ведай путь свой верно, дабы по холсту странствовать.<br/>
      CTRL + ПКМ (зажать 1 секунду на телефоне) — познай цвет истинный, да занеси его в палитру свою.<br/><br/>
      <span>В помощь тебе — <a target="_blank" href="https://color-hex.com">цветная книга мудрецов</a></span><br/>
      <br/>
      Время течет, как река, и ставить пиксели дозволено лишь раз в три удара сердца.
      <br/>
      <br/>
      </div>

      <button type="submit" class="send-button">Играть</button>
    </form>
  </div>
</template>

<script>
import AppHeader from "@/components/AppHeader.vue";

export default {
  components: {AppHeader},
  mounted() {
    document.title='Choose your faculty'
  }
}
</script>

<style scoped>
.faculty-page {
  
  font-size: 18px;  
  display: flex;
  flex-direction: column;
  justify-content: start;
  padding: 20px;
  padding-top: 50px;
  max-width: 100dvh;
  overflow: scroll;
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

input[type="radio"] {
  /* display: none; */
  /* appearance: none; */
  background-color: #134293;
}
/* 
input[type="radio"] + label {
  border-color: black;
  border-radius: 5px;
} */

/* label:before {
	content: "";
	display: inline-block;
	width: 17px;
	height: 18px;
	position: absolute;
	left: -40px;
	bottom: -2px;
	background-color: transparent;
  border: 2px solid #134293;
  border-radius: 20%;
} */

label {
  color: #134293;
  position: relative;
}

input[type="radio"]:checked ~ label {
  background-color: #bbb;
}

.form-container {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  /* justify-content: center; */
  margin-top: 20px;
  transition: background 1s;
  /* filter: drop-shadow(1px 1px grey); */
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
  border: 0;
  
  background-image: url("@/img/button.png");
  cursor: pointer;
  transition: all .1s ease-out;
  background-repeat: no-repeat;
  background-size: contain;
  background-position: center;
  height: 40px;
  width: 400px;
  background-color: transparent;
  color: #134293;
}

.send-button:hover {
  color: #134293;
}

* {
  font-family: 'Calligrapher';
  text-align: center;
}

h2 {
  margin: 0;
  margin-top: 30px;
  color: #134293;
}


a {
  color: #134293;
}
</style>
