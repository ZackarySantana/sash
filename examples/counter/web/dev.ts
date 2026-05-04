import homepage from "./index.html";

const port = Number(process.env.PORT);
if (!Number.isInteger(port) || port <= 0 || port > 65535) {
    console.error(
        "Missing or invalid PORT; run the dev server via `sash dev` (or set PORT manually).",
    );
    process.exit(1);
}

const server = Bun.serve({
    hostname: "127.0.0.1",
    port,
    routes: {
        "/": homepage,
    },
});

console.error(`Serving ${server.url}`);
