// The plan of a request: the EXPLAIN that smoothdb answers for
// Accept: application/vnd.pgrst.plan+json, read into a tree with the
// measures that make it readable. The time and the cost of a node include
// the ones of its children: the exclusive values subtract them, so that
// the node that does the work stands out.

export type PlanOption = 'analyze' | 'buffers' | 'verbose' | 'settings';

export interface PlanNode {
    id: number;
    depth: number;
    type: string;
    // what the node works on: a relation, an index, a function, a CTE
    target: string;
    raw: Record<string, any>;
    children: PlanNode[];
    // the relation to its parent (Outer, Inner, InitPlan, SubPlan)
    relationship?: string;
    startupCost: number;
    totalCost: number;
    // the cost of the node without its children
    selfCost: number;
    planRows: number;
    // the processes that run the node at the same time: above 1 under a
    // Gather, where the loops are the workers and the leader
    parallel: number;
    // with analyze
    actualRows?: number;
    loops?: number;
    // milliseconds of all the loops, with and without the children
    time?: number;
    selfTime?: number;
    // actual rows against estimated rows, per loop: above 1 the planner
    // expected fewer rows, below 1 more
    estimateFactor?: number;
    removedRows: number;
    sharedHit?: number;
    sharedRead?: number;
    warnings: PlanWarning[];
}

export interface PlanWarning {
    kind: 'estimate' | 'filter' | 'disk' | 'seqscan';
    text: string;
}

export interface Plan {
    root: PlanNode;
    nodes: PlanNode[];
    analyzed: boolean;
    planningTime?: number;
    executionTime?: number;
    settings?: Record<string, string>;
    // the largest exclusive time (analyze) or cost: the scale of the bars
    maxSelf: number;
}

const num = (v: unknown): number | undefined => (typeof v === 'number' ? v : typeof v === 'string' && v !== '' && !isNaN(Number(v)) ? Number(v) : undefined);

// target names what a node reads or computes
function targetOf(n: Record<string, any>): string {
    const parts: string[] = [];
    if (n['Relation Name']) {
        const rel = n['Schema'] ? `${n['Schema']}.${n['Relation Name']}` : n['Relation Name'];
        parts.push(n['Alias'] && n['Alias'] !== n['Relation Name'] ? `${rel} ${n['Alias']}` : rel);
    }
    if (n['Index Name']) parts.push(`using ${n['Index Name']}`);
    if (n['Function Name']) parts.push(`${n['Function Name']}()`);
    if (n['CTE Name']) parts.push(`CTE ${n['CTE Name']}`);
    if (n['Subplan Name']) parts.push(n['Subplan Name']);
    return parts.join(' ');
}

// the rows a filter or a recheck threw away, in all the loops
function removedOf(n: Record<string, any>): number {
    const loops = num(n['Actual Loops']) ?? 1;
    const keys = ['Rows Removed by Filter', 'Rows Removed by Index Recheck', 'Rows Removed by Join Filter'];
    return keys.reduce((s, k) => s + (num(n[k]) ?? 0), 0) * loops;
}

export function readPlan(json: unknown): Plan {
    const top = (Array.isArray(json) ? json[0] : json) as Record<string, any>;
    if (!top || typeof top !== 'object' || !top['Plan']) throw new Error('The answer is not a plan');
    const nodes: PlanNode[] = [];
    const analyzed = num(top['Plan']['Actual Loops']) !== undefined;

    // par is the number of processes that run a node at the same time: 1,
    // or under a Gather its workers and the leader. Their loops add up but
    // their time runs in parallel, so the time of a loop counts once per
    // process.
    function walk(raw: Record<string, any>, depth: number, par: number): PlanNode {
        const node: PlanNode = {
            id: nodes.length,
            depth,
            type: raw['Node Type'] ?? '?',
            target: targetOf(raw),
            raw,
            children: [],
            relationship: raw['Parent Relationship'],
            startupCost: num(raw['Startup Cost']) ?? 0,
            totalCost: num(raw['Total Cost']) ?? 0,
            selfCost: 0,
            planRows: num(raw['Plan Rows']) ?? 0,
            removedRows: removedOf(raw),
            parallel: par,
            warnings: [],
        };
        if (raw['Join Type']) node.type = `${raw['Join Type']} ${node.type}`;
        if (raw['Strategy'] && node.type === 'Aggregate') node.type = `${raw['Strategy']} Aggregate`;
        nodes.push(node);
        const loops = num(raw['Actual Loops']);
        if (loops !== undefined) {
            node.loops = loops;
            node.actualRows = num(raw['Actual Rows']);
            node.time = ((num(raw['Actual Total Time']) ?? 0) * loops) / Math.min(par, Math.max(loops, 1));
        }
        node.sharedHit = num(raw['Shared Hit Blocks']);
        node.sharedRead = num(raw['Shared Read Blocks']);
        const gather = raw['Node Type'] === 'Gather' || raw['Node Type'] === 'Gather Merge';
        const childPar = gather ? (num(raw['Workers Launched']) ?? num(raw['Workers Planned']) ?? 0) + 1 : par;
        node.children = ((raw['Plans'] as Record<string, any>[] | undefined) ?? []).map((c) => walk(c, depth + 1, childPar));

        // an InitPlan runs once on its own: its cost is not part of the
        // parent's, so only the other children are subtracted
        const inner = node.children.filter((c) => c.relationship !== 'InitPlan');
        node.selfCost = Math.max(0, node.totalCost - inner.reduce((s, c) => s + c.totalCost, 0));
        if (node.time !== undefined) {
            node.selfTime = Math.max(0, node.time - inner.reduce((s, c) => s + (c.time ?? 0), 0));
        }
        if (node.actualRows !== undefined && loops) {
            const est = Math.max(node.planRows, 1);
            const act = Math.max(node.actualRows, 1);
            node.estimateFactor = act / est;
        }
        node.warnings = warningsOf(node);
        return node;
    }

    const root = walk(top['Plan'], 0, 1);
    const maxSelf = Math.max(...nodes.map((n) => (analyzed ? (n.selfTime ?? 0) : n.selfCost)), 0);
    return {
        root,
        nodes,
        analyzed,
        planningTime: num(top['Planning Time']),
        executionTime: num(top['Execution Time']),
        settings: top['Settings'],
        maxSelf,
    };
}

// isMisestimate says whether the rows of a node are far from the planner's
// estimate: ten times or more, on at least 100 rows, and not on a Limit
export function isMisestimate(n: PlanNode): boolean {
    const f = n.estimateFactor;
    if (f === undefined || n.raw['Node Type'] === 'Limit') return false;
    return (f >= 10 || f <= 0.1) && Math.max(n.planRows, n.actualRows ?? 0) >= 100;
}

function warningsOf(n: PlanNode): PlanWarning[] {
    const w: PlanWarning[] = [];
    const f = n.estimateFactor;
    // a Limit estimates the rows it asks for, not the rows that exist; a
    // few rows off on a small table cost nothing
    if (f !== undefined && isMisestimate(n)) {
        w.push({
            kind: 'estimate',
            text: `The planner expected ${fmtCount(n.planRows)} rows per loop and got ${fmtCount(n.actualRows ?? 0)} (${fmtFactor(f)}). The statistics can be old (run ANALYZE on the table), or the conditions are correlated.`,
        });
    }
    const kept = (n.actualRows ?? 0) * (n.loops ?? 1);
    if (n.removedRows > 1000 && n.removedRows > 9 * kept) {
        const share = Math.round((100 * n.removedRows) / (n.removedRows + kept));
        w.push({
            kind: n.type === 'Seq Scan' ? 'seqscan' : 'filter',
            text:
                n.type === 'Seq Scan'
                    ? `The scan reads every row and the filter drops ${share}% of them. An index that matches the filter can avoid the full read.`
                    : `The filter drops ${share}% of the rows that the node reads.`,
        });
    }
    if (n.raw['Sort Space Type'] === 'Disk') {
        w.push({ kind: 'disk', text: `The sort spilled to disk (${n.raw['Sort Space Used']} kB). A larger work_mem keeps it in memory.` });
    }
    if ((num(n.raw['Hash Batches']) ?? 1) > 1) {
        w.push({ kind: 'disk', text: `The hash split into ${n.raw['Hash Batches']} batches on disk. A larger work_mem keeps it in memory.` });
    }
    return w;
}

// the conditions and keys of a node, in the order a reader looks for them
const detailKeys = [
    'Index Cond',
    'Recheck Cond',
    'Filter',
    'Hash Cond',
    'Merge Cond',
    'Join Filter',
    'Sort Key',
    'Group Key',
    'Presorted Key',
    'One-Time Filter',
    'Function Call',
    'Output',
    'Sort Method',
    'Sort Space Used',
    'Sort Space Type',
    'Hash Buckets',
    'Hash Batches',
    'Peak Memory Usage',
    'Heap Fetches',
    'Workers Planned',
    'Workers Launched',
];

export function detailsOf(n: PlanNode): [string, string][] {
    const out: [string, string][] = [];
    for (const k of detailKeys) {
        const v = n.raw[k];
        if (v === undefined || v === null) continue;
        out.push([k, Array.isArray(v) ? v.join(', ') : String(v)]);
    }
    return out;
}

// what a node does, in one sentence, for the common node types
const descriptions: Record<string, string> = {
    'Seq Scan': 'Reads every row of the table and keeps the ones that pass the filter.',
    'Index Scan': 'Finds the rows through the index, then reads each row from the table.',
    'Index Only Scan': 'Answers from the index alone, without reading the table (for the visible pages).',
    'Bitmap Index Scan': 'Marks in a bitmap the table pages that the index condition matches.',
    'Bitmap Heap Scan': 'Reads the table pages that the bitmap marks, in physical order, and checks the rows again.',
    'Nested Loop': 'For each row of the outer side, runs the inner side once.',
    'Hash Join': 'Builds a hash table of the inner side, then probes it with each row of the outer side.',
    'Merge Join': 'Joins two inputs that are sorted on the join key.',
    Hash: 'Builds the hash table for the join above.',
    Sort: 'Sorts its input; with a LIMIT above it can keep only the top rows.',
    'Incremental Sort': 'Sorts an input that is already sorted on a prefix of the key.',
    Limit: 'Stops after the rows that the page needs.',
    Aggregate: 'Computes aggregates (count, sum, ...) over its input.',
    'Plain Aggregate': 'Computes aggregates over all its input, in one group.',
    'Sorted Aggregate': 'Computes aggregates over input that is sorted by group.',
    'Hashed Aggregate': 'Computes aggregates per group with a hash table.',
    'Function Scan': 'Runs a function and reads the rows it returns.',
    'CTE Scan': 'Reads the rows of a WITH query.',
    'Subquery Scan': 'Reads the rows of a subquery.',
    Result: 'Computes a value without reading a table, or checks a condition once.',
    Materialize: 'Keeps the rows of its input in memory to read them again.',
    Memoize: 'Caches the results of the inner side by the key of the outer row.',
    Append: 'Concatenates the rows of its inputs (UNION ALL, partitions).',
    Gather: 'Collects the rows of parallel workers.',
    'Gather Merge': 'Collects the sorted rows of parallel workers, keeping the order.',
    ModifyTable: 'Writes the rows: insert, update or delete.',
    'Values Scan': 'Reads the rows of a VALUES list.',
    Unique: 'Removes the duplicate rows of a sorted input.',
    WindowAgg: 'Computes window functions.',
    LockRows: 'Locks the rows (SELECT ... FOR UPDATE).',
};

export function describe(n: PlanNode): string | undefined {
    const base = n.raw['Node Type'] as string;
    return descriptions[n.type] ?? descriptions[base];
}

export function fmtMs(ms: number | undefined): string {
    if (ms === undefined) return '';
    if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`;
    if (ms >= 10) return `${ms.toFixed(0)} ms`;
    if (ms >= 1) return `${ms.toFixed(1)} ms`;
    return `${ms.toFixed(3)} ms`;
}

export function fmtCount(n: number): string {
    if (n >= 1e6) return `${(n / 1e6).toFixed(1)}M`;
    if (n >= 1e4) return `${(n / 1e3).toFixed(0)}k`;
    return n.toLocaleString();
}

export function fmtCost(c: number): string {
    return c >= 100 ? c.toFixed(0) : c.toFixed(2);
}

export function fmtFactor(f: number): string {
    return f >= 1 ? `${f >= 10 ? f.toFixed(0) : f.toFixed(1)}× more` : `${(1 / f >= 10 ? (1 / f).toFixed(0) : (1 / f).toFixed(1))}× fewer`;
}

// the Accept header of a plan, as PostgREST names it
export function planAccept(format: 'json' | 'text', options: PlanOption[]): string {
    const opts = options.length ? `; options=${options.join('|')}` : '';
    return `application/vnd.pgrst.plan+${format}${opts}`;
}
