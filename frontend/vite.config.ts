import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { readFileSync } from "node:fs";

export default defineConfig({
  plugins: [react()],
  // Match production Vite resolution for MCU 0.4's extensionless internal ESM
  // imports instead of passing its entire barrel directly to Node's loader.
  test: {
    server: { deps: { inline: ["@material/material-color-utilities"] } },
  },
  define: {
    __PHEEBKIT_VERSION__: JSON.stringify(
      readFileSync(new URL("../VERSION", import.meta.url), "utf8").trim(),
    ),
  },
  base: "./",
  server: { host: "127.0.0.1", port: 9245, strictPort: true },
  build: { outDir: "dist", emptyOutDir: true },
});
