import LoginPage from "@/pages/LoginPage.vue";
import RegisterPage from "@/pages/RegisterPage.vue";
import MainPage from "@/pages/MainPage.vue";
import FacultyPage from "@/pages/FacultyPage.vue"
import { createRouter, createWebHistory } from "vue-router";

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
  },
  {
    path: '/register',
    name: 'register',
    component: RegisterPage
  },
  {
    path: '/faculty',
    name: 'faculty',
    component: FacultyPage
  }
];

const router = createRouter({
  routes,
  history: createWebHistory(process.env.BASE_URL)
});

// router.beforeEach((to, from, next) => {

//   if (to.name == 'faculty' && store.geters["UserModule/getAuthorized"]=="in_progress") {
//     next('faculty');
//   } else if (to.name !== 'login' && !store.getters["UserModule/getAuthorized"]) {
//     next('/login');
//   } else {
//     next();
//   }
// });

export default router;