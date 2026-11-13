<script lang="ts">
    // Remix Icon (Apache-2.0) SVGs from assets/icons, inlined so that they take
    // the text color. The files are our own assets, not user input.
    const icons = import.meta.glob("/assets/icons/*.svg", { eager: true, query: "?raw", import: "default" }) as Record<string, string>;

    interface Props {
        name: string;
        size?: number;
    }
    let { name, size = 16 }: Props = $props();

    let svg = $derived(icons[`/assets/icons/${name}.svg`] ?? "");
</script>

<span class="icon" style:width="{size}px" style:height="{size}px" aria-hidden="true">{@html svg}</span>

<style>
    .icon {
        display: inline-flex;
        flex-shrink: 0;
        vertical-align: -0.2em;
    }
    .icon :global(svg) {
        width: 100%;
        height: 100%;
    }
</style>
