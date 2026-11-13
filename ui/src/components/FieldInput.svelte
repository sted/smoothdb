<script lang="ts">
    import type { EditKind } from "../rowvalues";
    import { propsOnce } from "../forms/formdata.svelte";
    import { zoneLabel } from "../rowvalues";

    // The editor of one value, as compact as a row of the record panel. A
    // multi-line value (JSON, long text) takes one line until it has the focus.
    interface Props {
        id: string;
        kind: EditKind;
        value: string;
        isNull: boolean;
        nullable: boolean;
        disabled?: boolean;
        placeholder?: string;
        invalid?: boolean;
        autofocus?: boolean;
        // the time zone of a timestamp or a time, shown next to the picker
        zone?: string;
        onblur?: () => void;
    }
    let {
        id,
        kind,
        value = $bindable(),
        isNull = $bindable(),
        nullable,
        disabled = false,
        placeholder = "",
        invalid = false,
        autofocus = false,
        zone = "",
        onblur,
    }: Props = $props();

    let focused = $state(false);
    let el: HTMLInputElement | HTMLTextAreaElement | undefined = $state();

    // text with line breaks, or long text, is edited in a text area; decided
    // once, so that typing never swaps the element under the cursor
    const multiline = propsOnce(() => kind === "json" || (value ?? "").includes("\n") || (value ?? "").length > 60);
    // the native picker of a date, a timestamp or a time, when it can show the
    // value ("infinity" stays text); seconds, or milliseconds when the value has them
    const picker = propsOnce(() => {
        const v = value ?? "";
        if (kind === "date") return /^$|^\d{4}-\d{2}-\d{2}$/.test(v) ? "date" : null;
        if (kind === "timestamp") return /^$|^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(v) ? "datetime-local" : null;
        if (kind === "time") return /^$|^\d{2}:\d{2}/.test(v) ? "time" : null;
        return null;
    });
    const step = propsOnce(() => ((value ?? "").includes(".") ? "0.001" : "1"));

    let rows = $derived(focused ? Math.min(12, Math.max(3, (value ?? "").split("\n").length)) : 1);

    $effect(() => {
        if (autofocus && el) {
            el.focus();
            if ("select" in el && kind !== "json") el.select();
        }
    });

    function setNull(on: boolean) {
        isNull = on;
        if (!on) queueMicrotask(() => el?.focus());
    }
</script>

<div class="field-input" class:is-null={isNull} class:invalid>
    {#if kind === "bool"}
        <div class="segmented" role="radiogroup">
            {#each nullable ? ["true", "false", "null"] : ["true", "false"] as opt}
                {@const on = opt === "null" ? isNull : !isNull && value === opt}
                <button
                    type="button"
                    role="radio"
                    aria-checked={on}
                    class:on
                    class:mono={opt === "null"}
                    {disabled}
                    onclick={() => {
                        isNull = opt === "null";
                        if (opt !== "null") value = opt;
                    }}>{opt === "null" ? "NULL" : opt}</button
                >
            {/each}
        </div>
    {:else}
        <!-- the field, then the zone and the NULL button beside it: nothing lies
             over the field, because a native date picker ignores its padding -->
        {#if multiline}
            <textarea
                {id}
                bind:this={el}
                class="input"
                class:mono={kind === "json"}
                {rows}
                bind:value
                placeholder={isNull ? "NULL" : placeholder}
                disabled={disabled || isNull}
                spellcheck={kind !== "json"}
                onfocus={() => (focused = true)}
                onblur={() => {
                    focused = false;
                    onblur?.();
                }}
            ></textarea>
        {:else}
            <input
                {id}
                bind:this={el}
                class="input"
                class:mono={kind !== "text"}
                type={picker ?? "text"}
                step={picker && picker !== "date" ? step : undefined}
                inputmode={kind === "int" ? "numeric" : kind === "float" || kind === "numeric" ? "decimal" : undefined}
                bind:value
                placeholder={isNull ? "NULL" : placeholder}
                disabled={disabled || isNull}
                autocomplete="off"
                {onblur}
            />
        {/if}
        {#if zone}<span class="zone mono" title="Time zone of the stored value">{zoneLabel(zone)}</span>{/if}
        {#if nullable}
            <button
                type="button"
                class="null-toggle mono"
                class:on={isNull}
                aria-pressed={isNull}
                title={isNull ? "Set a value" : "Set to NULL"}
                {disabled}
                onclick={() => setNull(!isNull)}>NULL</button
            >
        {/if}
    {/if}
</div>

<style>
    .field-input {
        display: flex;
        align-items: flex-start;
        gap: 4px;
        width: 100%;
    }
    .input {
        flex: 1;
        min-width: 0;
        height: 26px;
        padding: 0 6px;
        border-radius: var(--radius-sm);
        font-size: 12.5px;
    }
    textarea.input {
        min-height: 26px;
        height: auto;
        padding: 4px 6px;
        resize: none;
        line-height: 1.45;
    }
    textarea.input:not(:focus) {
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }
    textarea.input:focus {
        white-space: pre-wrap;
        overflow-wrap: anywhere;
    }
    /* beside the field, never over its text: a date and time picker is wide */
    .zone {
        flex-shrink: 0;
        margin-top: 4px;
        padding: 0 5px;
        border-radius: 3px;
        background: var(--bg-muted);
        color: var(--text-3);
        font-size: 10px;
        line-height: 18px;
    }
    .is-null .input {
        background: var(--bg-subtle);
        font-style: italic;
    }
    .invalid .input {
        border-color: var(--danger);
    }
    .null-toggle {
        flex-shrink: 0;
        height: 26px;
        padding: 0 5px;
        border: 1px solid transparent;
        border-radius: var(--radius-sm);
        background: transparent;
        color: var(--text-3);
        font-size: 10px;
        cursor: pointer;
    }
    .null-toggle:hover {
        border-color: var(--border-strong);
        color: var(--text-2);
    }
    .null-toggle.on {
        border-color: var(--accent);
        background: var(--accent-soft);
        color: var(--accent-text);
    }
    .segmented {
        display: inline-flex;
        height: 26px;
        padding: 2px;
        border: 1px solid var(--border-strong);
        border-radius: var(--radius-sm);
        background: var(--bg-subtle);
    }
    .segmented button {
        min-width: 48px;
        padding: 0 8px;
        border: none;
        border-radius: 3px;
        background: transparent;
        color: var(--text-2);
        font-size: 12px;
        cursor: pointer;
    }
    .segmented button.mono {
        font-size: 10.5px;
    }
    .segmented button.on {
        background: var(--bg);
        color: var(--text);
        font-weight: 500;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
    }
    .segmented button:disabled {
        cursor: default;
        opacity: 0.6;
    }
</style>
