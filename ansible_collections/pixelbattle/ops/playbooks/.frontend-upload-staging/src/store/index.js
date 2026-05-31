import { createStore } from 'vuex';
import UserModule from './UserModule';

export default createStore({
  modules: {
    UserModule,
  }
});
