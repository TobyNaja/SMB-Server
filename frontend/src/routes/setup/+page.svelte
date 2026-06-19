<script lang="ts">
	import { goto } from '$app/navigation';
	import { post } from '$lib/api/client';
	import { toastError } from '$lib/stores/toast.svelte';

	let username = $state('');
	let password = $state('');
	let confirm = $state('');
	let loading = $state(false);
	let error = $state('');

	async function handleSetup(e: Event) {
		e.preventDefault();
		error = '';
		if (password !== confirm) { error = 'Passwords do not match'; return; }
		if (password.length < 8) { error = 'Password must be at least 8 characters'; return; }
		loading = true;
		try {
			await post('/auth/setup', { username, password });
			goto('/login');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Setup failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-900">
	<div class="w-full max-w-sm rounded-xl bg-gray-800 p-8 shadow-2xl">
		<div class="mb-6 text-center">
			<div class="mb-2 text-4xl">🗄️</div>
			<h1 class="text-xl font-bold text-white">SMB Permission Manager</h1>
			<p class="mt-1 text-sm text-gray-400">Create your admin account to get started</p>
		</div>

		<form onsubmit={handleSetup} class="space-y-4">
			<div>
				<label for="su-user" class="mb-1 block text-sm font-medium text-gray-300">Username</label>
				<input id="su-user" type="text" bind:value={username} required autocomplete="username"
					class="w-full rounded-lg border border-gray-600 bg-gray-700 px-3 py-2 text-white placeholder-gray-400 focus:border-blue-500 focus:outline-none"
					placeholder="admin" />
			</div>

			<div>
				<label for="su-pw" class="mb-1 block text-sm font-medium text-gray-300">Password</label>
				<input id="su-pw" type="password" bind:value={password} required minlength="8" autocomplete="new-password"
					class="w-full rounded-lg border border-gray-600 bg-gray-700 px-3 py-2 text-white placeholder-gray-400 focus:border-blue-500 focus:outline-none"
					placeholder="Min 8 characters" />
			</div>

			<div>
				<label for="su-confirm" class="mb-1 block text-sm font-medium text-gray-300">Confirm password</label>
				<input id="su-confirm" type="password" bind:value={confirm} required autocomplete="new-password"
					class="w-full rounded-lg border border-gray-600 bg-gray-700 px-3 py-2 text-white placeholder-gray-400 focus:border-blue-500 focus:outline-none"
					placeholder="••••••••" />
			</div>

			{#if error}
				<div class="rounded-lg border border-red-700 bg-red-900/50 px-3 py-2 text-sm text-red-300">
					{error}
				</div>
			{/if}

			<button type="submit" disabled={loading}
				class="w-full rounded-lg bg-blue-600 px-4 py-2.5 font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60">
				{loading ? 'Creating account…' : 'Create Admin Account'}
			</button>
		</form>
	</div>
</div>
