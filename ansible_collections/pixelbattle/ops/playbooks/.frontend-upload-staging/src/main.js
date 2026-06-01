import App from "@/App.vue";
import router from "@/router/router";
import { createApp } from 'vue';
import store from "@/store";

const app = createApp(App)

app
    .use(store)
    .use(router)
    .mount('#app')