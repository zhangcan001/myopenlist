import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../views/HomeView.vue'
import LogView from '../views/LogView.vue'
import MountView from '../views/MountView.vue'
import SettingsView from '../views/SettingsView.vue'
import UpdateView from '../views/UpdateView.vue'

const routes = [
  {
    path: '/',
    name: 'Dashboard',
    component: HomeView,
    meta: { title: 'Dashboard' },
  },
  {
    path: '/mount',
    name: 'Mount',
    component: MountView,
    meta: { title: 'Mount' },
  },
  {
    path: '/update',
    name: 'Update',
    component: UpdateView,
    meta: { title: 'Update' },
  },
  {
    path: '/settings',
    name: 'Settings',
    component: SettingsView,
    meta: { title: 'Settings' },
  },
  {
    path: '/logs',
    name: 'Logs',
    component: LogView,
    meta: { title: 'Logs' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
