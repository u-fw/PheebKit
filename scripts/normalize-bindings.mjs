import { readFileSync, writeFileSync } from "node:fs";

// Wails' supported generator emits whitespace-only lines and multiple terminal
// blank lines. Normalize presentation deterministically without editing types.
const path = new URL("../frontend/wailsjs/go/models.ts", import.meta.url);
const before = readFileSync(path, "utf8");
const after = before.replaceAll("\r\n", "\n").replace(/[\t ]+$/gm, "").trimEnd() + "\n";
if (before !== after) writeFileSync(path, after, "utf8");
