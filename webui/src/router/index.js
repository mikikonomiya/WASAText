import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'

const router = createRouter({
	history: createWebHashHistory(import.meta.env.BASE_URL),
	routes: [
		{ path: '/', component: HomeView },
		{ path: '/login', component: LoginView },
		{ path: '/conversations/:conversationId', component: () => import('../views/ChatView.vue') },
		{ path: '/profile', component: () => import('../views/ProfileView.vue') },
	]
})

router.beforeEach((to, from, next) => {
	const publicPages = ['/login']; // pages that do not require login
	const authRequired = !publicPages.includes(to.path); // if the page requires login
	const loggedIn = localStorage.getItem('token'); // if the user is logged in

	if (authRequired && !loggedIn) {
		next('/login'); // redirect
	} else {
		next();
	}
});

export default router
