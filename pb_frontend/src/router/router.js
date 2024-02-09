import LoginPage from "@/pages/LoginPage.vue";
import MainPage from "@/pages/MainPage.vue";
import { createRouter, createWebHistory } from "vue-router";
import store from "@/store"

const routes = [
  {
    path: '/',
    name: 'home',
    component: MainPage,
  },
  {
    path: '/main',
    name: 'main',
    component: MainPage,
  },
  {
    path: '/login',
    name: 'login',
    component: LoginPage
  }
];

const router = createRouter({
  routes,
  history: createWebHistory(process.env.BASE_URL)
});

// router.beforeEach((to, from, next) => {
//   if (to.name !== 'login' && !store.getters["dotModule/getAuthorized"]) {
//     next('/login');
//   } else {
//     next();
//   }
// });

export default router;