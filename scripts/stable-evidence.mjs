import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import { readVersion } from "./version.mjs";

export const stableChecks = ["windows10-clean", "windows11-clean", "webview2-missing", "native-media-workflows", "native-recovery-exit", "native-download-cancel-retry", "cpu-only", "gpu", "installer-upgrade", "hosted-ci", "source-and-assets-review"];

export function validateStableEvidence(value, version, revision) {
  if (value?.schemaVersion !== 1 || value.version !== version || value.sourceRevision !== revision)
    throw new Error("Stable evidence must match the exact version and clean source revision");
  for (const name of stableChecks) {
    const check = value.checks?.[name];
    if (check?.passed !== true || typeof check.evidence !== "string" || check.evidence.trim().length < 12 || typeof check.verifiedBy !== "string" || !check.verifiedBy.trim() || !/^\d{4}-\d{2}-\d{2}$/.test(check.verifiedOn ?? ""))
      throw new Error(`Missing stable acceptance evidence: ${name}`);
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (!process.argv[2]) throw new Error("Stable packaging requires -StableEvidence with reviewed acceptance records; no bypass is provided");
  const version = readVersion().version;
  const revision = execFileSync("git", ["rev-parse", "HEAD"], {encoding:"utf8"}).trim();
  validateStableEvidence(JSON.parse(readFileSync(process.argv[2], "utf8")), version, revision);
  console.log(`Stable acceptance evidence matches ${version} at ${revision}`);
}
