<template>
  <div class="header">              
    <h1>Добро пожаловать на Pixelbattle!</h1>
<!-- <h1>О мудром владении цветами и пергаментом на веб-сайте</h1><br/><br/> -->

<!-- Слушай же, о путник, и ведай, как управляться с волшебной палитрой:
<br/><br/>
ПКМ — держи перо крепко, дабы узоры чертить.<br/>
ЛКМ — ведай путь свой верно, дабы по холсту странствовать.<br/>
CTRL + ПКМ — познай цвет истинный, да занеси его в палитру свою.<br/><br/>
<span>В помощь тебе — <a href="https://color-hex.com">цветная книга мудрецов</a></span><br/>
<br/>
Время течет, как река, и ставить пиксели дозволено лишь раз в три удара сердца.
<br/> -->
<br/>
<br/>
Пусть же рука твоя будет тверда, а дух — неколебим. Веди перо свое, дабы создать творение, коему позавидуют сами небеса!

  </div>
  
</template>

<script>
export default {

}
</script>

<style scoped>
.header {
  width: 100%;
}

@font-face {
  font-family: 'Calligrapher';
  src: url('@/fonts/calligrapher.ttf') format('truetype');
}

* {
  padding: 0;
  margin: 0;
  font-family: 'Calligrapher', Times, serif;
}

h1, h2, h3 {
  color: #134293;
  padding-inline: 10%;
}

@media (max-width: 600px) {
  .header {
    font-size: 18px;
  }
}

a {
  color: #134293;
}

.banner {
  width: 100%;
  height: auto;
  display: block;
}
</style>