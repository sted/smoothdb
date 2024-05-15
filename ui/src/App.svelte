<script lang="ts">
	import { setContext } from "svelte";
	import { router } from "./routes";
	import Sidebar from "./components/Sidebar.svelte";
	import Breadcrumb from "./components/Breadcrumb.svelte";

	setContext("router", router);
	router.navigate(window.location.pathname);
</script>

<div class="app-grid">
	<div class="sidebar">
		<Sidebar />
	</div>
	<div class="breadcrumb">
		<Breadcrumb />
	</div>
	<div class="router-content">
		<svelte:component this={$router.component} />
	</div>
</div>

<style>
	.app-grid {
		display: grid;
		grid-template-columns: 200px auto;
		grid-template-rows: 70px auto;
		grid-template-areas:
			"sidebar breadcrumb"
			"sidebar router-content";
		height: 100vh;
	}

	.sidebar {
		grid-area: sidebar;
		overflow-y: scroll;
		border-right: 1px solid lightgrey;
	}

	.breadcrumb {
		grid-area: breadcrumb;
	}

	.router-content {
		grid-area: router-content;
		overflow: hidden;
	}
</style>
