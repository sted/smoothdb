<script lang="ts">
    import { untrack } from "svelte";
    import { makeRequest } from "../api";
    import { forget, refreshDatabases } from "../data.svelte";
    import { confirmDialog } from "../dialog.svelte";
    import SidePanel from "../components/SidePanel.svelte";
    import Icon from "../components/Icon.svelte";
    import AdminField from "./AdminField.svelte";
    import { forgetRefs, forgetTables } from "./refs.svelte";
    import type { AdminContext, EntityDef, FieldDef, Item, Request } from "./entities";

    // What the panel shows: an object of the list, or a new object.
    export type EntityTarget = { kind: "item"; item: Item } | { kind: "new" };

    interface Props {
        def: EntityDef;
        ctx: AdminContext;
        target: EntityTarget;
        editing: boolean;
        // the field to focus when the edit starts; the first editable field if none
        focus?: string;
        onEditing: (editing: boolean, focus?: string) => void;
        // after a create or an update: the list reloads and selects the object by
        // its key, or by its name when the server writes the key differently
        // (a function argument int4 comes back as integer)
        onSaved: (key: string, name: string) => void;
        onRemoved: () => void;
        onClose: () => void;
    }
    let { def, ctx, target, editing, focus = "", onEditing, onSaved, onRemoved, onClose }: Props = $props();

    let isNew = $derived(target.kind === "new");
    let item: Item = $derived(target.kind === "item" ? target.item : {});

    let values: Item = $state({});
    let initial: Item = {};
    let errors: Record<string, string> = $state({});
    let error = $state("");
    let busy = $state(false);
    let ready = $state(false);

    const same = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

    // a field can change: on a new object every field that is not read-only;
    // on an existing one, the fields that are not create-only
    const editable = (f: FieldDef) => !f.readonly && (isNew || !(f.createOnly || f.createOnlyWhen?.(item)));
    // the value of a field for the object on screen
    const valueOf = (f: FieldDef) => (f.read ? f.read(item) : item[f.key]);
    // a field that depends on another (a check expression on the type CHECK)
    const visible = (f: FieldDef) => !f.when || f.when(values);
    const empty = (v: unknown) => v === "" || v === undefined || v === null || (Array.isArray(v) && v.length === 0);
    const blank = (f: FieldDef) => (f.kind === "bool" ? false : f.kind === "roles" || f.kind === "columns" || f.kind === "refcolumns" ? [] : "");

    function beginEdit() {
        const start: Item = isNew ? { ...(def.initial?.(ctx) ?? {}) } : {};
        for (const f of def.fields) {
            if (!isNew) start[f.key] = valueOf(f) ?? blank(f);
            else start[f.key] ??= blank(f);
            // the type field edits the full type, with its modifier
            if (!isNew && f.kind === "type" && item.typedef) start[f.key] = item.typedef;
        }
        values = start;
        initial = JSON.parse(JSON.stringify(start));
        errors = {};
        error = "";
        ready = true;
    }

    // before the render, so that no field shows without its value; it depends
    // on editing and on the target only
    $effect.pre(() => {
        if (editing) {
            target;
            untrack(beginEdit);
        } else ready = false;
    });

    let changed: string[] = $derived(
        editing && ready ? def.fields.filter((f) => editable(f) && !same(values[f.key], initial[f.key])).map((f) => f.key) : [],
    );

    export async function confirmLeave(): Promise<boolean> {
        if (!editing || changed.length === 0) return true;
        return confirmDialog({
            title: "Discard your changes?",
            message: `${changed.length} ${changed.length === 1 ? "field has" : "fields have"} changes that are not saved.`,
            confirmLabel: "Discard",
            cancelLabel: "Keep editing",
            danger: true,
        });
    }

    async function cancel() {
        if (!(await confirmLeave())) return;
        if (isNew) onClose();
        else onEditing(false);
    }

    async function close() {
        if (await confirmLeave()) onClose();
    }

    async function send(method: string, req: Request) {
        return makeRequest(req.url, {
            method,
            headers: { ...(req.body !== undefined ? { "Content-Type": "application/json" } : {}), ...req.headers },
            body: req.body !== undefined ? JSON.stringify(req.body) : undefined,
            quiet: true,
        });
    }

    // a DDL change makes the cached definitions and lists stale
    function invalidate() {
        if (def.kind === "databases") refreshDatabases();
        if (def.kind === "roles") forgetRefs("roles");
        if (def.kind === "schemas") forgetRefs("schemas", ctx.db);
        if (ctx.db) {
            forget(ctx.db);
            forgetTables(ctx.db);
        }
    }

    async function save() {
        if (busy) return;
        errors = {};
        for (const f of def.fields) {
            if (f.required && editable(f) && visible(f) && empty(values[f.key])) errors[f.key] = "Required";
        }
        if (Object.keys(errors).length) {
            error = "Some fields are not valid.";
            return;
        }
        let req: Request | Request[] | undefined;
        if (isNew) {
            const body: Item = {};
            for (const f of def.fields) if (!f.readonly && visible(f) && values[f.key] !== "") body[f.key] = values[f.key];
            req = def.create?.(ctx, body);
        } else {
            if (changed.length === 0) return onEditing(false);
            const diff: Item = Object.fromEntries(changed.map((k) => [k, values[k]]));
            req = def.update?.(ctx, item, diff);
        }
        if (!req) return;
        busy = true;
        error = "";
        // a change in two steps stops at the first refusal
        for (const r of Array.isArray(req) ? req : [req]) {
            const method = isNew || (r.body as Item | undefined)?.replace ? "POST" : "PATCH";
            const res = await send(method, r);
            if (!res.success) {
                busy = false;
                error = res.error || "The server refused the change.";
                return;
            }
        }
        busy = false;
        invalidate();
        onSaved(def.key({ ...item, ...values }), values.name ?? item.name ?? "");
    }

    async function remove() {
        if (!def.remove) return;
        const ok = await confirmDialog({
            title: `${def.removeVerb ?? "Drop"} ${def.singular} ${def.title(item)}?`,
            message: def.removeWarning?.(item) ?? "You cannot undo this.",
            confirmLabel: `${def.removeVerb ?? "Drop"} ${def.singular}`,
            danger: true,
        });
        if (!ok) return;
        busy = true;
        const res = await send("DELETE", def.remove(ctx, item));
        busy = false;
        if (!res.success) {
            error = res.error || "The server refused to drop it.";
            return;
        }
        invalidate();
        onRemoved();
    }

    // the text of a field in reading mode
    function shown(f: FieldDef, v: unknown): string {
        if (f.kind === "bool") return v ? "Yes" : "No";
        // an empty role list of a policy means every role; of a membership, none
        if (Array.isArray(v)) return v.length ? v.join(", ") : f.kind === "roles" && !f.readonly ? "PUBLIC" : "";
        if (f.kind === "role" && f.emptyLabel && !v) return "PUBLIC";
        if (f.kind === "type" && item.typedef) return item.typedef;
        return v === null || v === undefined ? "" : String(v);
    }

    // in reading mode, the fields the object has; in editing mode, every field
    let fields = $derived(
        def.fields.filter((f) =>
            editing ? (isNew ? !f.readonly && visible(f) : !f.createOnly || f.key in item) : f.key in item || (f.readonly && f.key in item),
        ),
    );
    let canEdit = $derived(!!def.update && def.fields.some((f) => !f.readonly && !f.createOnly));
</script>

<SidePanel
    icon={def.icon}
    title={isNew ? `New ${def.singular}` : def.title(item)}
    subtitle={isNew ? "" : def.singular}
    {editing}
    {isNew}
    {canEdit}
    canDelete={!!def.remove}
    {busy}
    {error}
    changed={changed.length}
    onEdit={() => onEditing(true)}
    onCancel={cancel}
    onSave={save}
    onDelete={def.remove ? remove : undefined}
    onClose={close}
>
    {#snippet actions()}
        {#if def.data && !isNew}
            <a class="btn btn-sm btn-ghost" href={def.data(ctx, item)}><Icon name="table" size={13} /> Data</a>
        {/if}
        {#if def.open && !isNew}
            <a class="btn btn-sm btn-ghost" href={def.open(ctx, item)}><Icon name="arrow-right-s" size={13} /> Open</a>
        {/if}
        {#if !canEdit && def.remove && !isNew}
            <button class="btn btn-sm btn-ghost btn-danger" onclick={remove}><Icon name="delete-bin-6" size={13} /> {def.removeVerb ?? "Drop"}</button>
        {/if}
    {/snippet}

    <dl class="kv" class:editing>
        {#each fields as f}
            {@const isChanged = changed.includes(f.key)}
            <div class="kv-row" class:changed={isChanged}>
                <dt>
                    <label for={`af-${f.key}`}>
                        {f.label}{#if editing && f.required && editable(f)}<span class="req">*</span>{/if}
                    </label>
                </dt>
                <dd
                    class:null={!editing && shown(f, valueOf(f)) === ""}
                    ondblclick={() => {
                        if (!editing && canEdit && !isNew && editable(f)) onEditing(true, f.key);
                    }}
                >
                    {#if editing && ready && editable(f)}
                        <AdminField
                            id={`af-${f.key}`}
                            field={f}
                            {ctx}
                            {values}
                            bind:value={values[f.key]}
                            disabled={busy}
                            invalid={!!errors[f.key]}
                            autofocus={f === (fields.find((x) => x.key === focus && editable(x)) ?? fields.find(editable))}
                        />
                        {#if errors[f.key]}<span class="kv-error">{errors[f.key]}</span>
                        {:else if f.help}<span class="kv-help">{f.help}</span>{/if}
                    {:else}
                        {@const text = shown(f, isNew ? values[f.key] : valueOf(f))}
                        <span class:mono={f.kind !== "text" && f.kind !== "bool"} class:pre={f.multiline}>{text || "—"}</span>
                        {#if editing && !f.readonly}<span class="kv-help">Set at creation</span>{/if}
                    {/if}
                </dd>
            </div>
        {/each}
    </dl>
</SidePanel>

<style>
    .req {
        margin-left: 2px;
        color: var(--danger);
        font-weight: 600;
    }
    .pre {
        white-space: pre-wrap;
        font-size: 11.5px;
    }
    label {
        color: inherit;
    }
</style>
