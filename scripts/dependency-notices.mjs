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
  if (name && name !== "pheebkit") modules.set(name, { version, directory });
}
for (const name of ["react", "react-dom", "scheduler"]) {
  const directory = resolve("frontend/node_modules", name);
  const metadata = JSON.parse(
    readFileSync(join(directory, "package.json"), "utf8"),
  );
  modules.set(`npm:${name}`, { version: metadata.version, directory });
}
let result =
  "PheebKit third-party distribution notices\n\nFFmpeg and WebView2 are external; their distribution terms remain separate.\n";
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
// Wails v2 generates its JS runtime directly from the pinned Go module.
const wails = modules.get("github.com/wailsapp/wails/v2");
if (!wails || /-(?:alpha|beta|rc)/i.test(wails.version))
  throw new Error("A stable Wails v2 module is required");
append("generated Wails JavaScript runtime", wails.version, wails.directory);
if (existsSync(resolve(destination)))
  throw new Error(
    "Notice output already exists; use a fresh release staging directory",
  );
writeFileSync(destination, result, "utf8");
