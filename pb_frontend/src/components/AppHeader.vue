<template>
  <div class="header">
    <img href="#default" class="logo" src="@/img/ava.png">
    <div class="header-right">
      nuff said<br/>
    </div>
  </div>
</template>

<script>
export default {

}
</script>

<style scoped>
.header {
  overflow: hidden;
  background-color: #f1f1f1;
  background-size: cover;
  padding: 20px 10px;
  background-image: url("@/img/SL-093020-35920-01.jpg");
  border-radius: 10px;
  margin: 0.5%;
  position: relative;
}

.logo {
  width: 5%;
  margin: 1%;
  border-radius: 50%;
}

.header a:hover {
  background-color: #ddd;
  color: black;
}

.header-right {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  right: 0;
  padding-right: 5%;
  font-size: 140%;
  font-family: "DejaVu Sans Mono", sans-serif;
  color: #ff5258;
}

@media screen and (max-width: 500px) {
  .header a {
    float: none;
    display: block;
    text-align: left;
  }

  .header-right {
    float: none;
  }
}
</style>