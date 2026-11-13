<script lang="ts">
	import { authStore } from "../auth.svelte";
	import { router } from "../routes";
	import { forgetAll } from "../data.svelte";
	import Icon from "./Icon.svelte";

	let showUserMenu = $state(false);

	function handleLogout() {
		showUserMenu = false;
		authStore.clearAuth();
		forgetAll();
		router.navigate("/login");
	}

	function handleClickOutside(event: MouseEvent) {
		if (showUserMenu && !(event.target as Element).closest(".user-menu-container")) {
			showUserMenu = false;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<header class="app-header">
	<a class="brand" href="/ui/databases">
		<span class="mark"><Icon name="database-2" size={14} /></span>
		smoothdb
	</a>
	{#if authStore.isAuthenticated && authStore.user}
		{@const name = authStore.user.username || authStore.user.email || authStore.user.role}
		<div class="user-menu-container">
			<button class="user-button" aria-haspopup="menu" aria-expanded={showUserMenu} onclick={() => (showUserMenu = !showUserMenu)}>
				<span class="avatar">{name.slice(0, 1).toUpperCase()}</span>
				<span>{name}</span>
				<Icon name="arrow-down-s" size={14} />
			</button>
			{#if showUserMenu}
				<div class="user-menu" role="menu">
					<div class="who">
						<span>{name}</span>
						<span class="muted">Role <span class="mono">{authStore.user.role}</span></span>
					</div>
					<button class="menu-item logout-button" role="menuitem" onclick={handleLogout}>
						<Icon name="logout-box-r" size={14} /> Sign out
					</button>
				</div>
			{/if}
		</div>
	{/if}
</header>

<style>
	.app-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 44px;
		padding: 0 12px 0 14px;
		border-bottom: 1px solid var(--border);
		background: var(--bg);
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 14px;
		font-weight: 600;
		letter-spacing: -0.01em;
	}
	.mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: 6px;
		background: var(--accent);
		color: var(--accent-fg);
	}
	.user-menu-container {
		position: relative;
	}
	.user-button {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 28px;
		padding: 0 6px 0 4px;
		border: none;
		border-radius: var(--radius);
		background: transparent;
		color: var(--text-2);
		cursor: pointer;
	}
	.user-button:hover {
		background: var(--bg-hover);
		color: var(--text);
	}
	.avatar {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: var(--accent-soft);
		color: var(--accent-text);
		font-size: 11px;
		font-weight: 600;
	}
	.user-menu {
		position: absolute;
		top: calc(100% + 4px);
		right: 0;
		z-index: 80;
		min-width: 200px;
		padding: 4px;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--bg);
		box-shadow: var(--shadow-pop);
	}
	.who {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 8px 8px 10px;
		margin-bottom: 4px;
		border-bottom: 1px solid var(--border);
	}
	.menu-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		height: 28px;
		padding: 0 8px;
		border: none;
		border-radius: var(--radius-sm);
		background: none;
		text-align: left;
		cursor: pointer;
	}
	.menu-item:hover {
		background: var(--bg-hover);
	}
	.logout-button {
		color: var(--danger);
	}
</style>
