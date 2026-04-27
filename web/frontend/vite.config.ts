import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev server proxies API + redirect + legacy static to the Go backend on :8080.
// In production the Go binary serves the built assets directly via embed.FS.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/query": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
