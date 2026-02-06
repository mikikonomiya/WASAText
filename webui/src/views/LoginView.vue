<template>
	<div class="container d-flex justify-content-center align-items-center vh-100">
		<div class="card shadow p-4" style="max-width: 400px; width: 100%;">
			<h2 class="text-center mb-4">WASAText Login</h2>
			<form @submit.prevent="login">
				<div class="mb-3">
					<label for="username" class="form-label">Username</label>
					<input type="text" id="username" v-model="username" class="form-control" placeholder="Enter your username" required />
				</div>
				<div v-if="error" class="alert alert-danger">
					{{ error }}
				</div>
				<button type="submit" class="btn btn-primary w-100" :disabled="loading">
					<span v-if="loading" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
					Login
				</button>
			</form>
		</div>
	</div>
</template>

<script>
export default {
	data() {
		return {
			username: '',
			loading: false,
			error: null,
		};
	},
	methods: {
		async login() {
			this.loading = true;
			this.error = null;
			try {
				let response = await this.$axios.post('/session', { name: this.username });
				localStorage.setItem('token', response.data.identifier); // Assuming identifier is the token
                localStorage.setItem('username', this.username);
				this.$router.push('/');
			} catch (e) {
				console.error(e);
				this.error = e.response ? e.response.data : 'Login failed';
			}
			this.loading = false;
		},
	},
};
</script>

<style scoped>
.form-control {
    background-color: var(--app-bg);
    border-color: transparent;
}
</style>
