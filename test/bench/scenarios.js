// k6 scenarios of the comparison benchmark: one scenario per run, chosen by
// SCENARIO, against the base URL in BASE (smoothdb: http://host/api/<db>,
// PostgREST: http://host). TOKEN is the JWT of bench_user for the jwt_*
// scenarios. run.sh drives it; to run one by hand:
//
//   k6 run -e BASE=http://localhost:4102/api/k6bench -e SCENARIO=pk_lookup \
//          --vus 50 --duration 15s test/bench/scenarios.js

import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

// failed requests by status, and a sample of their bodies on stderr
const failed = {};
for (const s of ["0", "400", "401", "403", "404", "409", "416", "500", "503", "504", "other"]) {
    failed[s] = new Counter("failed_" + s);
}
let samples = 0;

const BASE = __ENV.BASE;
const TOKEN = __ENV.TOKEN || "";

const genres = ["fiction", "history", "science", "poetry", "travel", "crime", "fantasy", "biography"];

function rand(n) {
    return 1 + Math.floor(Math.random() * n);
}

function pick(list) {
    return list[Math.floor(Math.random() * list.length)];
}

// every request of a run is tagged with the scenario name, not its URL: the
// random ids would make a metric series per request
const tags = { name: __ENV.SCENARIO };

function get(path, headers) {
    return http.get(BASE + path, { headers: headers || {}, tags });
}

function send(method, path, body, headers) {
    const h = Object.assign({ "Content-Type": "application/json" }, headers || {});
    return http.request(method, BASE + path, JSON.stringify(body), { headers: h, tags });
}

const scenarios = {
    // reads
    small_all: () => get("/clients"),
    embed_to_one: () => get("/projects?select=id,name,clients(*)"),
    pk_lookup: () => get(`/books?id=eq.${rand(200000)}`),
    page_100: () => get(`/books?order=id&limit=100&offset=${rand(190000)}`),
    filter_multi: () => get(`/books?genre=eq.${pick(genres)}&year=gte.${1950 + rand(70)}&price=lt.50&limit=50`),
    in_list: () => {
        const ids = [];
        for (let i = 0; i < 20; i++) ids.push(rand(200000));
        return get(`/books?select=id,title&id=in.(${ids.join(",")})`);
    },
    or_filter: () => get(`/books?select=id,title&or=(genre.eq.${pick(genres)},year.eq.${1950 + rand(70)})&limit=50`),
    count_exact: () => get(`/books?select=id,title&genre=eq.${pick(genres)}&limit=20`, { Prefer: "count=exact" }),
    embed_to_many: () => get(`/authors?select=id,name,books(id,title,year)&order=id&limit=20&offset=${rand(9900)}`),
    aggregate: () => get(`/books?select=genre,count(),price.sum()&year=eq.${1950 + rand(70)}`),
    json_path: () => get(`/books?select=id,isbn:meta->>isbn&meta->>lang=eq.${pick(["en", "it", "fr", "de"])}&order=id&limit=50&offset=${rand(40000)}`),
    big_response: () => get("/books?order=id&limit=5000"),
    csv: () => get("/books?order=id&limit=1000", { Accept: "text/csv" }),
    rpc_get: () => get(`/rpc/books_by_genre?g=${pick(genres)}&select=id,title&limit=50`),
    rpc_count: () => get(`/rpc/books_by_genre?g=${pick(genres)}&select=id,title&limit=20`, { Prefer: "count=exact" }),
    rpc_scalar: () => get(`/rpc/add?a=${rand(1000)}&b=${rand(1000)}`),
    // writes
    insert: () => send("POST", "/events", { kind: "k6", payload: { n: rand(1000) } }, { Prefer: "return=minimal" }),
    insert_repr: () => send("POST", "/events", { kind: "k6", payload: { n: rand(1000) } }, { Prefer: "return=representation" }),
    bulk_insert_100: () => {
        const rows = [];
        for (let i = 0; i < 100; i++) rows.push({ kind: "k6bulk", payload: { n: i } });
        return send("POST", "/events", rows, { Prefer: "return=minimal" });
    },
    patch: () => send("PATCH", `/events?id=eq.${rand(10000)}`, { payload: { n: rand(1000) } }, { Prefer: "return=minimal" }),
    // authenticated
    jwt_pk_lookup: () => get(`/books?id=eq.${rand(200000)}`, { Authorization: "Bearer " + TOKEN }),
    jwt_page_100: () => get(`/books?order=id&limit=100&offset=${rand(190000)}`, { Authorization: "Bearer " + TOKEN }),
};

const run = scenarios[__ENV.SCENARIO];
if (!run) {
    throw new Error(`unknown SCENARIO ${__ENV.SCENARIO}; one of: ${Object.keys(scenarios).join(" ")}`);
}

export default function () {
    const res = run();
    const ok = check(res, { "status 2xx": (r) => r.status >= 200 && r.status < 300 });
    if (!ok) {
        (failed[String(res.status)] || failed.other).add(1);
        if (samples++ < 3) {
            console.warn(`${res.status} ${res.request.url}: ${String(res.body).slice(0, 300)} ${res.error || ""}`);
        }
    }
}

// The numbers the comparison reads, as JSON: to the file in SUMMARY (run.sh),
// or to stdout.
export function handleSummary(data) {
    const m = data.metrics;
    const out = {
        rps: m.http_reqs.values.rate,
        reqs: m.http_reqs.values.count,
        p50: m.http_req_duration.values.med,
        p95: m.http_req_duration.values["p(95)"],
        max: m.http_req_duration.values.max,
        ok: m.checks ? m.checks.values.rate : 0,
        failed: {},
    };
    for (const name of Object.keys(m)) {
        if (name.startsWith("failed_") && m[name].values.count > 0) {
            out.failed[name.slice(7)] = m[name].values.count;
        }
    }
    return { [__ENV.SUMMARY || "stdout"]: JSON.stringify(out) + "\n" };
}
