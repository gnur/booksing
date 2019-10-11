import './../node_modules/bulma/css/bulma.css'
import 'vue-material-design-icons/styles.css'
import Vue from 'vue'
import App from './App.vue'
import Buefy from 'buefy'
import router from './router'
import store from './store'

// Firebase App (the core Firebase SDK) is always required and must be listed first
import * as firebase from 'firebase/app'
import 'firebase/auth'
import 'firebase/firestore'

Vue.use(Buefy)

Vue.config.productionTip = false

// Your web app's Firebase configuration
var firebaseConfig = {
  apiKey: 'AIzaSyAGQUy7C5rGdv4GflJPtxqa_ggBxBTEclI',
  authDomain: 'booksing-erwin-land.firebaseapp.com',
  databaseURL: 'https://booksing-erwin-land.firebaseio.com',
  projectId: 'booksing-erwin-land',
  storageBucket: 'booksing-erwin-land.appspot.com',
  messagingSenderId: '1046086232379',
  appId: '1:1046086232379:web:1f1ac5b3e9732796642f1e'
}

// Initialize Firebase
firebase.initializeApp(firebaseConfig)

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
