import path from "node:path";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// Dev server proxies API + redirect + legacy routes to the Go backend, which
// defaults to :8080. If that port is taken (or you run the backend elsewhere),
// set VITE_PROXY_TARGET in web/frontend/.env.local — e.g.
// VITE_PROXY_TARGET=http://localhost:8090 — and point the backend there too via
// the repo-root .env (PORT). In production the Go binary serves the built
// assets directly via embed.FS, so this proxy is dev-only.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const proxyTarget = env.VITE_PROXY_TARGET || "http://localhost:8080";

  return {
    plugins: [react()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: 5173,
      proxy: {
        "/api": proxyTarget,
        "/query": proxyTarget,
        "/auth": proxyTarget,
      },
    },
    build: {
      outDir: "dist",
      emptyOutDir: true,
    },
  };
});
