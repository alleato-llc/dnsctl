import { defineConfig } from "vite";

// Wails serves the built assets via the Go asset server; this config keeps the
// dev/build pipeline minimal (vanilla TypeScript, no framework).
//
// emptyOutDir is disabled so the committed `dist/.gitkeep` (which keeps the
// `//go:embed all:frontend/dist` directive compilable on a fresh clone)
// survives `vite build`.
export default defineConfig({
  build: {
    emptyOutDir: false,
  },
});
