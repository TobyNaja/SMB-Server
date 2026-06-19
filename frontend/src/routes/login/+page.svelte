<script lang="ts">
	import { goto } from '$app/navigation';
	import { authApi } from '$lib/api/auth';
	import { setAuth } from '$lib/stores/auth.svelte';
	import { toastError } from '$lib/stores/toast.svelte';
	import { Server } from 'lucide-svelte';

	let username = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');

	async function handleLogin(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';
		try {
			const res = await authApi.login(username, password);
			const me = await authApi.me();
			setAuth(me, res.access_token);
			goto('/shares');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-900">
	<div class="w-full max-w-sm rounded-xl bg-gray-800 p-8 shadow-2xl">
		<div class="mb-6 text-center">
			<div class="mb-3 flex justify-center">
				<div class="rounded-xl bg-blue-600/20 p-3">
					<Server size={32} class="text-blue-400" />
				</div>
			</div>
			<h1 class="text-xl font-bold text-white">SMB Permission Manager</h1>
			<p class="text-sm text-gray-400 mt-1">Sign in to continue</p>
		</div>

		<form onsubmit={handleLogin} class="space-y-4">
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-300" for="username">Username</label>
				<input
					id="username"
					type="text"
					bind:value={username}
					autocomplete="username"
					required
					class="w-full rounded-lg bg-gray-700 px-3 py-2 text-white placeholder-gray-400
						border border-gray-600 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
					placeholder="admin"
				/>
			</div>

			<div>
				<label class="mb-1 block text-sm font-medium text-gray-300" for="password">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					autocomplete="current-password"
					required
					class="w-full rounded-lg bg-gray-700 px-3 py-2 text-white placeholder-gray-400
						border border-gray-600 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
					placeholder="••••••••"
				/>
			</div>

			{#if error}
				<div class="rounded-lg bg-red-900/50 border border-red-700 px-3 py-2 text-sm text-red-300">
					{error}
				</div>
			{/if}

			<button
				type="submit"
				disabled={loading}
				class="w-full rounded-lg bg-blue-600 px-4 py-2.5 font-medium text-white
					hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500
					disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
			>
				{loading ? 'Signing in…' : 'Sign in'}
			</button>
		</form>
	</div>
</div>
