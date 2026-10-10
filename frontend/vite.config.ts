import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { readFileSync } from "node:fs";

export default defineConfig({
  plugins: [react()],
  define: {
    __KOMPEG_VERSION__: JSON.stringify(
      readFileSync(new URL("../VERSION", import.meta.url), "utf8").trim(),
    ),
  },
  base: "./",
  server: { host: "127.0.0.1", port: 9245, strictPort: true },
  build: { outDir: "dist", emptyOutDir: true },
});
