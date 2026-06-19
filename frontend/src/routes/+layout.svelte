<script lang="ts">
	import './layout.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';
	import Toast from '$lib/components/Toast.svelte';
	import { authApi } from '$lib/api/auth';
	import { getUser, setAuth, clearAuth, isAuthenticated } from '$lib/stores/auth.svelte';
	import {
		Server, Folder, User, Users, Building2,
		Shield, ClipboardList, Settings, LogOut
	} from 'lucide-svelte';

	let { children } = $props();

	const navItems = [
		{ href: '/shares',  label: 'Shares',          icon: Folder        },
		{ href: '/users',   label: 'Users',            icon: User          },
		{ href: '/groups',  label: 'Groups',           icon: Users         },
		{ href: '/ad',      label: 'Active Directory', icon: Building2     },
		{ href: '/builtin', label: 'Builtin Groups',   icon: Shield        },
		{ href: '/audit',   label: 'Audit Log',        icon: ClipboardList },
		{ href: '/settings',label: 'Settings',         icon: Settings      }
	];

	const isLoginPage = $derived($page.url.pathname === '/login');
	const isSetupPage = $derived($page.url.pathname === '/setup');
	const isPublicPage = $derived(isLoginPage || isSetupPage);

	onMount(async () => {
		try {
			const status = await fetch('/auth/status').then(r => r.json());
			if (status.setup_required && !isSetupPage) { goto('/setup'); return; }
			if (!status.setup_required && isSetupPage) { goto('/login'); return; }
		} catch { /* network error — let the page handle it */ }

		if (isPublicPage) return;
		if (!isAuthenticated()) { goto('/login'); return; }
		try {
			const me = await authApi.me();
			setAuth(me, localStorage.getItem('access_token') ?? '');
		} catch {
			clearAuth();
			goto('/login');
		}
	});

	async function handleLogout() {
		try { await authApi.logout(); } catch { /* ignore */ }
		finally { clearAuth(); goto('/login'); }
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>SMB Manager</title>
</svelte:head>

<Toast />

{#if isPublicPage}
	{@render children()}
{:else}
	<div class="flex h-screen bg-gray-100 dark:bg-gray-900">
		<!-- Sidebar -->
		<aside class="flex w-56 flex-col bg-gray-800 text-white">
			<div class="flex items-center gap-2 border-b border-gray-700 px-4 py-5">
				<Server size={20} class="text-blue-400 flex-none" />
				<span class="text-lg font-bold">SMB Manager</span>
			</div>

			<nav class="flex-1 overflow-y-auto py-2">
				{#each navItems as item}
					{@const Icon = item.icon}
					<a
						href={item.href}
						class="flex items-center gap-3 px-4 py-2.5 text-sm transition-colors
							{$page.url.pathname.startsWith(item.href)
							? 'bg-blue-600 text-white'
							: 'text-gray-300 hover:bg-gray-700'}"
					>
						<Icon size={16} class="flex-none" />
						<span>{item.label}</span>
					</a>
				{/each}
			</nav>

			<div class="border-t border-gray-700 p-4">
				{#if getUser()}
					<div class="mb-2 truncate text-xs text-gray-400">
						Logged in as <strong class="text-white">{getUser()?.username}</strong>
					</div>
				{/if}
				<button
					onclick={handleLogout}
					class="flex w-full items-center justify-center gap-2 rounded bg-gray-700 px-3 py-1.5 text-sm text-gray-200 transition-colors hover:bg-red-600 hover:text-white"
				>
					<LogOut size={14} />
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
