import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import ts from "../frontend/node_modules/typescript/lib/typescript.js";

const root = fileURLToPath(new URL("../", import.meta.url));
const catalog = JSON.parse(readFileSync(new URL("../locales/en.json", import.meta.url), "utf8"));
const files = execFileSync("git", ["ls-files", "--cached", "--others", "--exclude-standard"], { cwd: root, encoding: "utf8" }).trim().split("\n");
const han = /[\u3400-\u9fff]/u;
const failures = [];
const requireTranslation = (source, file) => {
  if (han.test(source) && !Object.hasOwn(catalog, source)) failures.push(`${file}: missing translation: ${source}`);
};
for (const [source, translated] of Object.entries(catalog)) {
  const markers = (value) => [...value.matchAll(/\{\{\d+\}\}/g)].map((match) => match[0]).sort().join(",");
  if (typeof translated !== "string" || !translated.trim() || markers(source) !== markers(translated)) failures.push(`Invalid translation or placeholders: ${source}`);
}
for (const file of files) {
  if (file.startsWith("frontend/src/") && /\.tsx?$/.test(file) && !file.includes(".test.") && !file.includes("/i18n/")) {
    const text = readFileSync(new URL(`../${file}`, import.meta.url), "utf8");
    const ast = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, file.endsWith("tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
    function visit(node) {
      if (ts.isJsxText(node)) requireTranslation(node.text.replace(/\s+/g, " ").trim(), file);
      else if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) requireTranslation(node.text, file);
      else if (ts.isTemplateExpression(node)) requireTranslation(node.head.text + node.templateSpans.map((span, index) => `{{${index}}}` + span.literal.text).join(""), file);
      ts.forEachChild(node, visit);
    }
    visit(ast);
  }
  if (file.endsWith(".go") && !file.endsWith("_test.go")) {
    const text = readFileSync(new URL(`../${file}`, import.meta.url), "utf8");
    for (const match of text.matchAll(/`[^`]*`|"(?:[^"\\]|\\.)*"/g)) {
      let source;
      try {
        source = match[0].startsWith('`') ? match[0].slice(1, -1) : JSON.parse(match[0]);
      } catch {
        if (han.test(match[0])) failures.push(`${file}: unsupported Go string escape in translatable text`);
        continue;
      }
      if (!han.test(source)) continue;
      let index = 0;
      source = source.replace(/%(?:[-+ #0]*\d*(?:\.\d+)?[a-zA-Z]|%)/g, (token) => token === "%%" ? "%" : `{{${index++}}}`);
      requireTranslation(source, file);
    }
  }
  if (/\.(?:go|tsx?|mjs|ps1|ya?ml|json|iss)$/.test(file) && !file.startsWith("internal/compat/") && !["frontend/src/legacy.ts", "scripts/legacy.ps1", "build/windows/legacy.iss"].includes(file)) {
    const text = readFileSync(new URL(`../${file}`, import.meta.url), "utf8");
    if (/\bKOMPEG_[A-Z_*]+/.test(text)) failures.push(`${file}: old environment namespace outside the compatibility boundary`);
  }
}
if (failures.length) {
  process.stderr.write(failures.join("\n") + "\n");
  process.exitCode = 1;
} else {
  console.log(`PASS: ${Object.keys(catalog).length} translations, source coverage, placeholders and public environment namespace.`);
}
