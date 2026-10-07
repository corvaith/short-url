import { createApp, nextTick } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import { routes } from './router'
import { useAuth } from './composables/useAuth'
import { setUnauthorizedHandler } from './services/api'
import './assets/tokens.css'
import './assets/main.css'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

const auth = useAuth()

router.beforeEach(async (to) => {
  await auth.hydrate()
  if (to.meta.requiresAuth && !auth.isLoggedIn.value) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && auth.isLoggedIn.value) {
    return { path: '/dashboard' }
  }
})

// After navigation, move focus to the page h1 and update the document title.
router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} — ShortINK` : 'ShortINK'
  void nextTick(() => {
    const h1 = document.querySelector<HTMLElement>('h1[tabindex="-1"]')
    h1?.focus({ preventScroll: false })
  })
})

setUnauthorizedHandler(() => {
  auth.handleUnauthorized()
  const current = router.currentRoute.value.fullPath
  if (!current.startsWith('/login')) {
    void router.push({ path: '/login', query: { redirect: current } })
  }
})

createApp(App).use(router).mount('#app')
