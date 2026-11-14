<script lang="ts">
    import { makeRequest } from "../api";
    import { tableUrl } from "../config.svelte";
    import { showToast } from "../toast.svelte";
    import {
        describe,
        detailsOf,
        fmtCost,
        fmtCount,
        fmtFactor,
        fmtMs,
        isMisestimate,
        planAccept,
        readPlan,
        type Plan,
        type PlanNode,
        type PlanOption,
    } from "../plan";
    import Icon from "./Icon.svelte";

    // The plan of the request of the grid: the same table, select, filters,
    // order and page, with Accept: application/vnd.pgrst.plan. It runs again
    // when the request or the options change.
    interface Props {
        db: string;
        schema: string;
        table: string;
        // the query string of the grid's request
        params: string;
        onClose: () => void;
    }
    let { db, schema, table, params, onClose }: Props = $props();

    type View = "tree" | "text" | "json";
    const optionList: { key: PlanOption; label: string; help: string }[] = [
        { key: "analyze", label: "Analyze", help: "Run the query and measure the real time and rows of each node" },
        { key: "buffers", label: "Buffers", help: "Count the pages read from the cache and from disk (with Analyze)" },
        { key: "verbose", label: "Verbose", help: "Show the output columns of each node" },
        { key: "settings", label: "Settings", help: "Show the settings that differ from the defaults" },
    ];

    let options = $state<Record<PlanOption, boolean>>({ analyze: true, buffers: true, verbose: false, settings: false });
    let view = $state<View>("tree");
    let loading = $state(false);
    let error = $state("");
    let disabled = $state(false);
    let plan = $state<Plan | null>(null);
    let json = $state("");
    let text = $state("");
    let open = $state<Set<number>>(new Set());
    let collapsed = $state<Set<number>>(new Set());

    let chosen = $derived(optionList.filter((o) => options[o.key]).map((o) => o.key));
    let url = $derived(`${tableUrl(db, table)}?${params}`);

    let seq = 0;
    async function run(format: "json" | "text", opts: PlanOption[]) {
        const mine = ++seq;
        loading = true;
        error = "";
        const r = await makeRequest(url, { quiet: true, headers: { Accept: planAccept(format, opts), "Accept-Profile": schema } });
        if (mine !== seq) return;
        loading = false;
        disabled = r.status === 406;
        if (!r.success) {
            error = r.error ?? "The plan failed";
            plan = null;
            return;
        }
        if (format === "text") {
            text = typeof r.data === "string" ? r.data : "";
            return;
        }
        try {
            plan = readPlan(r.data);
            json = JSON.stringify(r.data, null, 2);
            open = new Set();
            collapsed = new Set();
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
            plan = null;
        }
    }

    $effect(() => {
        // the text view asks for the text format; the others share the JSON one
        run(view === "text" ? "text" : "json", chosen);
    });

    // the rows of the tree, without the children of a collapsed node
    let visible = $derived.by(() => {
        const out: PlanNode[] = [];
        const walk = (n: PlanNode) => {
            out.push(n);
            if (!collapsed.has(n.id)) n.children.forEach(walk);
        };
        if (plan) walk(plan.root);
        return out;
    });

    let slowest = $derived(
        plan && plan.maxSelf > 0 ? plan.nodes.find((n) => (plan!.analyzed ? n.selfTime : n.selfCost) === plan!.maxSelf) : undefined,
    );
    let warnings = $derived(plan ? plan.nodes.reduce((s, n) => s + n.warnings.length, 0) : 0);

    const selfOf = (n: PlanNode) => (plan?.analyzed ? (n.selfTime ?? 0) : n.selfCost);
    const share = (n: PlanNode) => (plan && plan.maxSelf > 0 ? selfOf(n) / plan.maxSelf : 0);
    const total = (p: Plan) => (p.analyzed ? (p.executionTime ?? p.root.time ?? 0) : p.root.totalCost);
    const percent = (n: PlanNode) => (plan && total(plan) > 0 ? Math.round((100 * selfOf(n)) / total(plan)) : 0);

    function toggle(set: Set<number>, id: number): Set<number> {
        const next = new Set(set);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
    }

    function focusNode(n: PlanNode) {
        open = new Set([...open, n.id]);
        document.getElementById(`plan-node-${n.id}`)?.scrollIntoView({ block: "nearest" });
    }

    async function copy(what: string, label: string) {
        try {
            await navigator.clipboard.writeText(what);
            showToast("info", `${label} copied`);
        } catch {
            showToast("error", "Cannot write to the clipboard");
        }
    }

    let curl = $derived(
        `curl '${new URL(url, location.origin).href}' \\\n  -H 'Accept: ${planAccept(view === "text" ? "text" : "json", chosen)}' \\\n  -H 'Accept-Profile: ${schema}' \\\n  -H "Authorization: Bearer $TOKEN"`,
    );
</script>

<aside class="explain" aria-label="Query plan">
    <header>
        <Icon name="pulse" size={14} />
        <span class="title">Explain</span>
        <span class="mono subtitle">{table}</span>
        <span class="tools">
            <button class="btn btn-sm btn-ghost btn-icon" title="Run again" aria-label="Run again" disabled={loading} onclick={() => run(view === "text" ? "text" : "json", chosen)}>
                <Icon name="refresh" size={14} />
            </button>
            <button class="btn btn-sm btn-ghost btn-icon" title="Copy as curl" aria-label="Copy as curl" onclick={() => copy(curl, "curl command")}>
                <Icon name="braces" size={14} />
            </button>
            <button class="btn btn-sm btn-ghost btn-icon" aria-label="Close" onclick={onClose}>
                <Icon name="close" size={15} />
            </button>
        </span>
    </header>

    <div class="controls">
        <div class="options" role="group" aria-label="Plan options">
            {#each optionList as o (o.key)}
                <label class="opt" title={o.help}>
                    <input type="checkbox" bind:checked={options[o.key]} />
                    {o.label}
                </label>
            {/each}
        </div>
        <div class="seg" role="tablist" aria-label="View">
            {#each [["tree", "Tree"], ["text", "Text"], ["json", "JSON"]] as [key, label] (key)}
                <button role="tab" class:on={view === key} aria-selected={view === key} onclick={() => (view = key as View)}>{label}</button>
            {/each}
        </div>
    </div>

    <div class="body">
        {#if disabled}
            <div class="banner banner-info">
                <Icon name="information" size={15} />
                <span>The server does not answer plans. Set <code>Database.PlanEnabled</code> to true in its configuration, or run it with <code>SMOOTHDB_DEBUG=true</code>.</span>
            </div>
        {:else if error}
            <div class="banner banner-danger" role="alert">
                <Icon name="error-warning" size={15} />
                {error}
            </div>
        {/if}

        {#if view === "text" && !disabled && !error}
            <pre class="raw">{text}</pre>
        {:else if view === "json" && plan}
            <div class="raw-tools">
                <button class="btn btn-sm" onclick={() => copy(json, "Plan")}>Copy JSON</button>
            </div>
            <pre class="raw">{json}</pre>
        {:else if plan}
            <div class="summary">
                {#if plan.analyzed}
                    <div class="stat"><span class="k">Execution</span><span class="v">{fmtMs(plan.executionTime)}</span></div>
                    <div class="stat"><span class="k">Planning</span><span class="v">{fmtMs(plan.planningTime)}</span></div>
                {/if}
                <div class="stat"><span class="k">Cost</span><span class="v">{fmtCost(plan.root.totalCost)}</span></div>
                <div class="stat"><span class="k">Rows</span><span class="v">{fmtCount(plan.root.actualRows ?? plan.root.planRows)}{plan.analyzed ? "" : " est."}</span></div>
                {#if slowest}
                    <button class="stat link" onclick={() => focusNode(slowest!)} title="Show the node">
                        <span class="k">{plan.analyzed ? "Slowest node" : "Costliest node"}</span>
                        <span class="v">{slowest.type} · {percent(slowest)}%</span>
                    </button>
                {/if}
                {#if warnings}
                    <div class="stat warn"><span class="k">Hints</span><span class="v">{warnings}</span></div>
                {/if}
            </div>
            <p class="note">
                {#if plan.analyzed}
                    Bars show the time each node spends itself, without its children.
                {:else}
                    Without Analyze the query does not run: the bars show the estimated cost of each node, without its children. Cost is in arbitrary units, where reading one page in sequence costs 1.
                {/if}
                The row count of the grid runs as a separate statement and is not in this plan.
            </p>

            <div class="tree-head" aria-hidden="true">
                <span class="name">Node</span>
                <span class="bar-head">{plan.analyzed ? "Time itself" : "Cost itself"}</span>
                <span class="num rows">{plan.analyzed ? "Rows" : "Rows est."}</span>
            </div>
            <ol class="tree">
                {#each visible as n (n.id)}
                    <li id={`plan-node-${n.id}`} class:open={open.has(n.id)}>
                        <div
                            class="row"
                            role="button"
                            tabindex="0"
                            onclick={() => (open = toggle(open, n.id))}
                            onkeydown={(e) => (e.key === "Enter" || e.key === " ") && (e.preventDefault(), (open = toggle(open, n.id)))}
                        >
                            <span class="indent" style:width="{n.depth * 14}px"></span>
                            {#if n.children.length}
                                <button
                                    class="twisty"
                                    aria-label={collapsed.has(n.id) ? "Expand children" : "Collapse children"}
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        collapsed = toggle(collapsed, n.id);
                                    }}
                                >
                                    <Icon name={collapsed.has(n.id) ? "arrow-right-s" : "arrow-down-s"} size={14} />
                                </button>
                            {:else}
                                <span class="twisty"></span>
                            {/if}
                            <span class="name">
                                <span class="type">{n.type}</span>
                                {#if n.target}<span class="mono target">{n.target}</span>{/if}
                                {#if n.relationship === "InitPlan" || n.relationship === "SubPlan"}<span class="badge">{n.relationship}</span>{/if}
                                {#if n.warnings.length}<span class="hint-dot" title={n.warnings.map((w) => w.text).join("\n")}><Icon name="error-warning" size={13} /></span>{/if}
                            </span>
                            <span class="bar" title={`${percent(n)}% of the ${plan.analyzed ? "time" : "cost"}`}>
                                <span class="fill" class:hot={percent(n) >= 50} style:width="{Math.max(share(n) * 100, selfOf(n) > 0 ? 2 : 0)}%"></span>
                            </span>
                            <span class="num self">{plan.analyzed ? fmtMs(n.selfTime) : fmtCost(n.selfCost)}</span>
                            <span class="num rows">
                                {#if n.actualRows !== undefined}
                                    {fmtCount(n.actualRows)}
                                    {#if n.estimateFactor !== undefined && isMisestimate(n)}
                                        <span class="est bad" title={`estimated ${fmtCount(n.planRows)}: ${fmtFactor(n.estimateFactor)}`}>est. {fmtCount(n.planRows)}</span>
                                    {/if}
                                {:else}
                                    <span class="est">{fmtCount(n.planRows)}</span>
                                {/if}
                            </span>
                        </div>
                        {#if open.has(n.id)}
                            <div class="details" style:margin-left="{n.depth * 14 + 20}px">
                                {#if describe(n)}<p class="what">{describe(n)}</p>{/if}
                                {#each n.warnings as w, i (i)}
                                    <div class="banner banner-warn"><Icon name="error-warning" size={14} /> {w.text}</div>
                                {/each}
                                <dl class="facts">
                                    {#if plan.analyzed}
                                        <dt>Time</dt>
                                        <dd>{fmtMs(n.time)} with children · {fmtMs(n.selfTime)} itself ({percent(n)}%)</dd>
                                        <dt>Rows</dt>
                                        <dd>
                                            {`${fmtCount(n.actualRows ?? 0)} per loop, estimated ${fmtCount(n.planRows)}`}{n.estimateFactor !== undefined &&
                                            n.estimateFactor !== 1
                                                ? ` (${fmtFactor(n.estimateFactor)})`
                                                : ""}
                                        </dd>
                                        {#if n.parallel > 1}
                                            <dt>Parallel</dt>
                                            <dd>{n.loops} processes ran it at the same time: {n.parallel - 1} workers and the leader</dd>
                                        {:else if (n.loops ?? 1) > 1}
                                            <dt>Loops</dt>
                                            <dd>{fmtCount(n.loops ?? 1)}: the node ran once per row of its outer side</dd>
                                        {/if}
                                    {:else}
                                        <dt>Rows</dt>
                                        <dd>{fmtCount(n.planRows)} estimated</dd>
                                    {/if}
                                    <dt>Cost</dt>
                                    <dd>{fmtCost(n.startupCost)} to start, {fmtCost(n.totalCost)} in all, {fmtCost(n.selfCost)} itself</dd>
                                    {#if n.removedRows}
                                        <dt>Removed</dt>
                                        <dd>{fmtCount(n.removedRows)} rows read and dropped by a filter or a recheck</dd>
                                    {/if}
                                    {#if n.sharedHit !== undefined}
                                        <dt>Buffers</dt>
                                        <dd>{fmtCount(n.sharedHit)} pages from the cache, {fmtCount(n.sharedRead ?? 0)} from disk</dd>
                                    {/if}
                                    {#each detailsOf(n) as [k, v] (k)}
                                        <dt>{k}</dt>
                                        <dd class="mono">{v}</dd>
                                    {/each}
                                </dl>
                            </div>
                        {/if}
                    </li>
                {/each}
            </ol>

            {#if plan.settings && Object.keys(plan.settings).length}
                <h3>Settings</h3>
                <dl class="facts">
                    {#each Object.entries(plan.settings) as [k, v] (k)}
                        <dt class="mono">{k}</dt>
                        <dd class="mono">{v}</dd>
                    {/each}
                </dl>
            {/if}
        {:else if loading}
            <p class="note">Asking for the plan…</p>
        {/if}
    </div>
</aside>

<style>
    .explain {
        display: flex;
        flex-direction: column;
        width: 560px;
        flex-shrink: 0;
        border-left: 1px solid var(--border);
        background: var(--bg);
        font-size: 12.5px;
    }
    header {
        display: flex;
        align-items: center;
        gap: 6px;
        height: 40px;
        padding: 0 6px 0 12px;
        border-bottom: 1px solid var(--border);
        color: var(--text-3);
    }
    .title {
        color: var(--text);
        font-weight: 600;
    }
    .subtitle {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .tools {
        margin-left: auto;
        display: flex;
        gap: 2px;
    }
    .controls {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 8px 12px;
        border-bottom: 1px solid var(--border);
        background: var(--bg-subtle);
    }
    .options {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
    }
    .opt {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        color: var(--text-2);
        cursor: pointer;
    }
    .seg {
        margin-left: auto;
        display: inline-flex;
        border: 1px solid var(--border);
        border-radius: var(--radius-sm);
        overflow: hidden;
    }
    .seg button {
        padding: 2px 9px;
        border: none;
        background: transparent;
        color: var(--text-2);
        cursor: pointer;
        font-size: 12px;
    }
    .seg button + button {
        border-left: 1px solid var(--border);
    }
    .seg button.on {
        background: var(--accent-soft);
        color: var(--accent-text);
    }
    .body {
        flex: 1;
        overflow: auto;
        padding: 10px 12px 16px;
    }
    .banner {
        margin-bottom: 10px;
    }
    .banner-warn {
        display: flex;
        gap: 6px;
        align-items: flex-start;
        padding: 6px 8px;
        border-radius: var(--radius);
        background: var(--warning-soft);
        color: var(--warning);
        margin: 4px 0;
    }
    code {
        font-family: var(--font-mono);
        font-size: 11.5px;
    }
    .summary {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin-bottom: 8px;
    }
    .stat {
        display: flex;
        flex-direction: column;
        gap: 1px;
        padding: 5px 9px;
        border: 1px solid var(--border);
        border-radius: var(--radius);
        background: var(--bg-subtle);
        text-align: left;
        font: inherit;
    }
    .stat .k {
        color: var(--text-3);
        font-size: 11px;
    }
    .stat .v {
        color: var(--text);
        font-weight: 600;
        font-variant-numeric: tabular-nums;
    }
    .stat.link {
        cursor: pointer;
    }
    .stat.link:hover {
        border-color: var(--accent);
    }
    .stat.warn .v {
        color: var(--warning);
    }
    .note {
        margin: 0 0 10px;
        color: var(--text-3);
        line-height: 1.45;
    }
    .tree-head {
        display: flex;
        gap: 6px;
        padding: 0 4px 3px;
        color: var(--text-3);
        font-size: 11px;
    }
    .bar-head {
        width: 140px;
        text-align: right;
    }
    .tree {
        list-style: none;
        margin: 0;
        padding: 0;
        border-top: 1px solid var(--border);
    }
    .tree li {
        border-bottom: 1px solid var(--border);
    }
    .row {
        display: flex;
        align-items: center;
        gap: 6px;
        min-height: 28px;
        padding: 2px 4px;
        cursor: pointer;
    }
    .row:hover {
        background: var(--bg-hover);
    }
    li.open > .row {
        background: var(--bg-active);
    }
    .indent {
        flex-shrink: 0;
        align-self: stretch;
        background: repeating-linear-gradient(to right, var(--border) 0 1px, transparent 1px 14px);
        background-position: 6px 0;
    }
    .twisty {
        display: inline-flex;
        width: 16px;
        flex-shrink: 0;
        padding: 0;
        border: none;
        background: transparent;
        color: var(--text-3);
        cursor: pointer;
    }
    .name {
        flex: 1;
        min-width: 0;
        display: flex;
        align-items: center;
        gap: 6px;
        overflow: hidden;
        white-space: nowrap;
    }
    .type {
        color: var(--text);
        font-weight: 600;
    }
    .target {
        overflow: hidden;
        text-overflow: ellipsis;
        color: var(--text-2);
    }
    .hint-dot {
        display: inline-flex;
        color: var(--warning);
    }
    .bar {
        width: 70px;
        height: 6px;
        flex-shrink: 0;
        border-radius: 3px;
        background: var(--bg-muted);
        overflow: hidden;
    }
    .fill {
        display: block;
        height: 100%;
        background: var(--accent);
    }
    .fill.hot {
        background: var(--warning);
    }
    .num {
        flex-shrink: 0;
        text-align: right;
        font-variant-numeric: tabular-nums;
        color: var(--text-2);
    }
    .self {
        width: 64px;
    }
    .rows {
        width: 92px;
    }
    .est {
        color: var(--text-3);
    }
    .est.bad {
        margin-left: 3px;
        padding: 0 3px;
        border-radius: var(--radius-sm);
        background: var(--warning-soft);
        color: var(--warning);
        font-size: 11px;
    }
    .details {
        padding: 6px 8px 10px;
    }
    .what {
        margin: 0 0 6px;
        color: var(--text-2);
    }
    .facts {
        display: grid;
        grid-template-columns: max-content 1fr;
        gap: 3px 10px;
        margin: 4px 0 0;
    }
    .facts dt {
        color: var(--text-3);
    }
    .facts dd {
        margin: 0;
        overflow-wrap: anywhere;
    }
    h3 {
        margin: 14px 0 4px;
        font-size: 12px;
        font-weight: 600;
    }
    .raw {
        margin: 0;
        padding: 8px;
        border-radius: var(--radius);
        background: var(--bg-subtle);
        font-family: var(--font-mono);
        font-size: 11.5px;
        line-height: 1.5;
        white-space: pre;
        overflow: auto;
    }
    .raw-tools {
        display: flex;
        justify-content: flex-end;
        margin-bottom: 6px;
    }
</style>
