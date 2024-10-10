<script lang="ts">
	import { router } from "./routes";
	import Sidebar from "./components/Sidebar.svelte";
	import Breadcrumb from "./components/Breadcrumb.svelte";
	import BasicPage from "./pages/BasicPage.svelte";

	let page: BasicPage;

	router.navigate(window.location.pathname);

	function rowAdd(ev: Event) {
		page.rowAdd();
	}
</script>

<div class="app-grid">
	<div class="sidebar">
		<Sidebar />
	</div>
	<div class="breadcrumb">
		<Breadcrumb {rowAdd} />
	</div>
	<div class="router-content">
		<BasicPage bind:this={page}></BasicPage>
	</div>
</div>

<style>
	.app-grid {
		display: grid;
		grid-template-columns: 150px auto;
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
