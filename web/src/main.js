import './../node_modules/bulma/css/bulma.css'
import 'vue-material-design-icons/styles.css'
import Vue from 'vue'
import App from './App.vue'
import Buefy from 'buefy'
import router from './router'
import store from './store'
import config from './firebase-config'

// Firebase App (the core Firebase SDK) is always required and must be listed first
import * as firebase from 'firebase/app'
import 'firebase/auth'
import 'firebase/firestore'

Vue.use(Buefy)

Vue.config.productionTip = false

// Initialize Firebase
firebase.initializeApp(config)

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
