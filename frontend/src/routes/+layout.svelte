<script lang="ts">
	import './layout.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';
	import Toast from '$lib/components/Toast.svelte';
	import { authApi } from '$lib/api/auth';
	import { getUser, setAuth, clearAuth, isAuthenticated } from '$lib/stores/auth.svelte';
	import { toastError } from '$lib/stores/toast.svelte';

	let { children } = $props();

	const navItems = [
		{ href: '/shares', label: 'Shares', icon: '📁' },
		{ href: '/users', label: 'Users', icon: '👤' },
		{ href: '/groups', label: 'Groups', icon: '👥' },
		{ href: '/ad', label: 'Active Directory', icon: '🏢' },
		{ href: '/builtin', label: 'Builtin Groups', icon: '🛡️' },
		{ href: '/audit', label: 'Audit Log', icon: '📋' },
		{ href: '/settings', label: 'Settings', icon: '⚙️' }
	];

	const isLoginPage = $derived($page.url.pathname === '/login');

	onMount(async () => {
		if (isLoginPage) return;
		if (!isAuthenticated()) {
			goto('/login');
			return;
		}
		// Refresh current user info
		try {
			const me = await authApi.me();
			setAuth(me, localStorage.getItem('access_token') ?? '');
		} catch {
			clearAuth();
			goto('/login');
		}
	});

	async function handleLogout() {
		try {
			await authApi.logout();
		} catch {
			// ignore
		} finally {
			clearAuth();
			goto('/login');
		}
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>SMB Manager</title>
</svelte:head>

<Toast />

{#if isLoginPage}
	{@render children()}
{:else}
	<div class="flex h-screen bg-gray-100 dark:bg-gray-900">
		<!-- Sidebar -->
		<aside class="flex w-56 flex-col bg-gray-800 text-white">
			<div class="flex items-center gap-2 px-4 py-5 font-bold text-lg border-b border-gray-700">
				<span>🗄️</span>
				<span>SMB Manager</span>
			</div>

			<nav class="flex-1 overflow-y-auto py-2">
				{#each navItems as item}
					<a
						href={item.href}
						class="flex items-center gap-3 px-4 py-2.5 text-sm transition-colors
							{$page.url.pathname.startsWith(item.href)
							? 'bg-blue-600 text-white'
							: 'text-gray-300 hover:bg-gray-700'}"
					>
						<span>{item.icon}</span>
						<span>{item.label}</span>
					</a>
				{/each}
			</nav>

			<div class="border-t border-gray-700 p-4">
				{#if getUser()}
					<div class="mb-2 text-xs text-gray-400 truncate">
						Logged in as <strong class="text-white">{getUser()?.username}</strong>
					</div>
				{/if}
				<button
					onclick={handleLogout}
					class="w-full rounded bg-gray-700 px-3 py-1.5 text-sm text-gray-200 hover:bg-red-600 hover:text-white transition-colors"
				>
					Logout
				</button>
			</div>
		</aside>

		<!-- Main content -->
		<main class="flex-1 overflow-auto">
			<div class="p-6">
				{@render children()}
			</div>
		</main>
	</div>
{/if}
