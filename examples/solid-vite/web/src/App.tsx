import { createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { API, createSashSSESession, eventsURL } from "./bindings";

function isAbortError(e: unknown): boolean {
    return (
        (e instanceof DOMException && e.name === "AbortError") ||
        (e instanceof Error && e.name === "AbortError")
    );
}

export default function App() {
    const [message, setMessage] = createSignal("");
    const [error, setError] = createSignal("");
    const [loading, setLoading] = createSignal(false);
    const [messageRttMs, setMessageRttMs] = createSignal<number | null>(null);

    const [slowPhase, setSlowPhase] = createSignal<
        "idle" | "running" | "done" | "canceled" | "error"
    >("idle");
    const [slowDetail, setSlowDetail] = createSignal("");
    const [sseTicks, setSseTicks] = createSignal<string[]>([]);

    let slowController: AbortController | undefined;

    onMount(() => {
        const sse = createSashSSESession();
        const off = sse.listenJSON("tick", (data) => {
            setSseTicks((prev) => [...prev.slice(-11), `sec=${data.sec}`]);
        });
        onCleanup(() => {
            off();
            sse.close();
        });
    });

    async function fetchMessage() {
        setLoading(true);
        setError("");
        const start = performance.now();
        try {
            setMessage(await API.Message());
            setMessageRttMs(Math.round(performance.now() - start));
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
            setMessage("");
            setMessageRttMs(null);
        } finally {
            setLoading(false);
        }
    }

    async function startSlow() {
        slowController?.abort();
        const ac = new AbortController();
        slowController = ac;
        setSlowPhase("running");
        setSlowDetail("");
        try {
            await API.WaitFiveSeconds({ signal: ac.signal });
            setSlowPhase("done");
            setSlowDetail("Completed after 5 seconds on the Go side.");
        } catch (e) {
            if (isAbortError(e)) {
                setSlowPhase("canceled");
                setSlowDetail(
                    "Canceled in the browser: the fetch aborted and Go's request context was canceled.",
                );
            } else {
                setSlowPhase("error");
                setSlowDetail(e instanceof Error ? e.message : String(e));
            }
        } finally {
            slowController = undefined;
        }
    }

    function cancelSlow() {
        slowController?.abort();
    }

    return (
        <div class="min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center gap-6 p-8">
            <div class="max-w-md w-full rounded-2xl border border-slate-700 bg-slate-900/80 p-8 shadow-xl backdrop-blur-sm">
                <h1 class="text-xl font-semibold tracking-tight text-white mb-1">
                    Solid + Vite + Tailwind
                </h1>
                <p class="text-sm text-slate-400 mb-6">
                    RPC via generated bindings (same pattern as the counter
                    example).
                </p>
                <button
                    type="button"
                    class="flex min-h-11 w-full items-center justify-center rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-medium text-white shadow hover:bg-emerald-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 disabled:pointer-events-none disabled:opacity-55"
                    disabled={loading()}
                    onClick={() => void fetchMessage()}
                >
                    Call Go API.Message
                </button>
                <div class="mt-4 min-h-21 text-sm">
                    {error() ? (
                        <p class="text-rose-400">{error()}</p>
                    ) : message() ? (
                        <div class="flex flex-col gap-1">
                            <p class="min-h-5.5 text-emerald-300 leading-snug">
                                {message()}
                            </p>
                            <p
                                class="text-xs tabular-nums leading-snug text-slate-500"
                                style={{
                                    visibility:
                                        messageRttMs() != null
                                            ? "visible"
                                            : "hidden",
                                }}
                                aria-hidden={messageRttMs() == null}
                            >
                                Roundtrip{" "}
                                <span class="inline-block min-w-[3ch] text-right font-medium text-slate-400">
                                    {messageRttMs()}
                                </span>{" "}
                                ms (browser → Go → browser)
                            </p>
                        </div>
                    ) : (
                        <p class="text-slate-500">
                            Press the button to fetch from Go.
                        </p>
                    )}
                </div>

                <div class="mt-8 border-t border-slate-700 pt-6">
                    <p class="text-sm text-slate-400 mb-3">
                        Cancellation demo: the Go handler waits up to 5 seconds
                        unless the request context is canceled.
                    </p>
                    <div class="flex gap-2">
                        <button
                            type="button"
                            class="flex-1 rounded-lg bg-sky-600 px-3 py-2.5 text-sm font-medium text-white shadow hover:bg-sky-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-400 disabled:opacity-50"
                            disabled={slowPhase() === "running"}
                            onClick={() => void startSlow()}
                        >
                            {slowPhase() === "running"
                                ? "Waiting (~5s)…"
                                : "Request (~5 seconds)"}
                        </button>
                        <button
                            type="button"
                            class="rounded-lg border border-slate-600 bg-slate-800 px-3 py-2.5 text-sm font-medium text-slate-200 hover:bg-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 disabled:opacity-40 disabled:cursor-not-allowed"
                            disabled={slowPhase() !== "running"}
                            onClick={() => cancelSlow()}
                        >
                            Cancel
                        </button>
                    </div>
                    <p class="mt-3 text-xs text-slate-500 min-h-11">
                        {slowDetail()}
                    </p>
                    <p class="mt-4 text-xs font-medium text-slate-400">
                        SSE (Go → browser):{" "}
                        <code class="text-slate-500">{eventsURL()}</code>
                    </p>
                    <ul class="mt-2 max-h-28 overflow-y-auto rounded-md bg-slate-950/50 px-2 py-1.5 font-mono text-[11px] text-amber-200/90">
                        <Show
                            when={sseTicks().length > 0}
                            fallback={
                                <li class="text-slate-600 list-none">
                                    Run “Request (~5 seconds)” to see tick
                                    events.
                                </li>
                            }
                        >
                            <For each={sseTicks()}>
                                {(line) => <li>{line}</li>}
                            </For>
                        </Show>
                    </ul>
                </div>
            </div>
        </div>
    );
}
