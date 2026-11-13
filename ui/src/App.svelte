<script lang="ts">
	import { router } from "./routes";
	import NavigationSidebar from "./components/NavigationSidebar.svelte";
	import Breadcrumb from "./components/Breadcrumb.svelte";
	import AppHeader from "./components/AppHeader.svelte";
	import Toasts from "./components/Toasts.svelte";
	import ConfirmDialog from "./components/ConfirmDialog.svelte";
	import BasicPage from "./pages/BasicPage.svelte";
	import LoginPage from "./pages/LoginPage.svelte";
	import ErrorPage from "./pages/ErrorPage.svelte";
	import { authStore } from "./auth.svelte";
	import { ui } from "./config.svelte";

	const redirectKey = "smoothdb_redirectAfterLogin";

	router.navigate(window.location.pathname + window.location.search, false);

	let needsLogin = $derived(ui.authRequired && !authStore.isAuthenticated);
	let onLoginRoute = $derived(router.path === "/ui/login");

	// The login page is the only page without a token; after the login the
	// app goes back to the page that asked for it.
	$effect(() => {
		if (needsLogin && !onLoginRoute) {
			sessionStorage.setItem(redirectKey, router.path + (router.query ? "?" + router.query : ""));
			router.navigate("/login");
		} else if (!needsLogin && onLoginRoute) {
			const target = sessionStorage.getItem(redirectKey) || "/databases";
			sessionStorage.removeItem(redirectKey);
			router.navigate(target);
		}
	});
</script>

{#if ui.error}
	<ErrorPage message={ui.error} />
{:else if onLoginRoute}
	<LoginPage />
{:else if !needsLogin}
	<div class="app-layout">
		<AppHeader />
		<div class="app-grid">
			<div class="sidebar">
				<NavigationSidebar />
			</div>
			<div class="breadcrumb">
				<Breadcrumb />
			</div>
			<div class="router-content">
				<BasicPage />
			</div>
		</div>
	</div>
{/if}
<Toasts />
<ConfirmDialog />

<style>
	.app-layout {
		display: flex;
		flex-direction: column;
		height: 100vh;
		background: var(--bg);
	}

	.app-grid {
		display: grid;
		grid-template-columns: 220px minmax(0, 1fr);
		grid-template-rows: 40px minmax(0, 1fr);
		grid-template-areas:
			"sidebar breadcrumb"
			"sidebar router-content";
		flex: 1;
		min-height: 0;
	}

	.sidebar {
		grid-area: sidebar;
		min-height: 0;
		border-right: 1px solid var(--border);
		background: var(--bg-subtle);
	}

	.breadcrumb {
		grid-area: breadcrumb;
		border-bottom: 1px solid var(--border);
	}

	.router-content {
		grid-area: router-content;
		min-height: 0;
		overflow: hidden;
	}
</style>
