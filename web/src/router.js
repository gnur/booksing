import Vue from 'vue'
import Router from 'vue-router'
import Admin from './components/Admin.vue'
import Login from './components/Login.vue'
import Dashboard from './components/Dashboard.vue'
import AdvancedSearch from './components/AdvancedSearch.vue'

Vue.use(Router)

export default new Router({
  mode: 'history',
  routes: [
    {
      path: '/',
      name: 'home',
      component: AdvancedSearch
    },
    {
      path: '/login',
      name: 'login',
      component: Login
    },
    {
      path: '/admin',
      name: 'admin',
      component: Admin
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: Dashboard
    },
  ]
})
