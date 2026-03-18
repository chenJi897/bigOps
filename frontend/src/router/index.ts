import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/Login.vue'),
    },
    {
      path: '/',
      component: () => import('../views/Layout.vue'),
      meta: { requiresAuth: true },
      redirect: '/system/users',
      children: [
        { path: 'system/users', name: 'Users', component: () => import('../views/Users.vue') },
        { path: 'system/roles', name: 'Roles', component: () => import('../views/Roles.vue') },
        { path: 'system/menus', name: 'Menus', component: () => import('../views/Menus.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) return '/login'
  if (to.path === '/login' && token) return '/'
})

export default router
