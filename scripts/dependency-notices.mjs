import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";

const destination = process.argv[2];
if (!destination)
  throw new Error("Usage: node scripts/dependency-notices.mjs <output-file>");
const lines = execFileSync(
  "go",
  [
    "list",
    "-tags",
    "production",
    "-deps",
    "-f",
    "{{if .Module}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}",
    ".",
  ],
  { encoding: "utf8" },
).split(/\r?\n/);
const modules = new Map();
for (const line of lines) {
  const [name, version, directory] = line.split("|");
  if (name && name !== "kompeg") modules.set(name, { version, directory });
}
for (const name of ["react", "react-dom", "scheduler"]) {
  const directory = resolve("frontend/node_modules", name);
  const metadata = JSON.parse(
    readFileSync(join(directory, "package.json"), "utf8"),
  );
  modules.set(`npm:${name}`, { version: metadata.version, directory });
}
let result =
  "kompeg third-party distribution notices\n\nFFmpeg and WebView2 are external; their distribution terms remain separate.\n";
function append(name, version, directory) {
  const files = readdirSync(directory).filter((file) =>
    /^(LICENSE|COPYING|NOTICE)(\.|$)/i.test(file),
  );
  if (!files.length) throw new Error(`Missing license notice for ${name}`);
  result += `\n${name} ${version}\n${"=".repeat(72)}\n`;
  for (const file of files)
    result += `\n${file}\n${readFileSync(join(directory, file), "utf8")}\n`;
}
for (const [name, info] of [...modules].sort(([a], [b]) => a.localeCompare(b)))
  append(name, info.version, info.directory);
const goRoot = execFileSync("go", ["env", "GOROOT"], {
  encoding: "utf8",
}).trim();
append(
  "Go standard library/runtime",
  execFileSync("go", ["version"], { encoding: "utf8" }).trim(),
  goRoot,
);
// npm's Wails runtime omits its license file; it shares the pinned Go module's MIT license.
const runtime = JSON.parse(
  readFileSync("frontend/node_modules/@wailsio/runtime/package.json", "utf8"),
);
const wails = modules.get("github.com/wailsapp/wails/v3");
if (!wails || runtime.version !== wails.version.slice(1))
  throw new Error("Wails Go/npm runtime version mismatch");
append("npm:@wailsio/runtime", runtime.version, wails.directory);
if (existsSync(resolve(destination)))
  throw new Error(
    "Notice output already exists; use a fresh release staging directory",
  );
writeFileSync(destination, result, "utf8");
