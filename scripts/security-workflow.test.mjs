import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const workflow = readFileSync(
  new URL("../.github/workflows/security.yml", import.meta.url),
  "utf8",
).replaceAll("\r\n", "\n");

test("security audit runs periodically, manually and for dependency changes", () => {
  assert.match(workflow, /^  pull_request:/m);
  assert.match(workflow, /^  push:/m);
  assert.match(workflow, /^  schedule:\n    - cron: "23 2 \* \* 1"$/m);
  assert.match(workflow, /^  workflow_dispatch:$/m);
  assert.match(workflow, /"\.github\/workflows\/\*\*"/);
  assert.match(workflow, /"frontend\/package-lock\.json"/);
  assert.match(workflow, /"go\.sum"/);
});

test("security audit has read-only repository permissions", () => {
  assert.match(workflow, /^permissions:\n  contents: read$/m);
  assert.match(
    workflow,
    /^  audit:\n(?:.|\n)*?    permissions:\n      contents: read$/m,
  );
  assert.doesNotMatch(workflow, /^\s+contents:\s+write$/m);
  assert.doesNotMatch(workflow, /^\s+(?:attestations|id-token|packages):\s+write$/m);
});

test("security tools and actions are pinned and installation scripts are disabled", () => {
  assert.match(workflow, /actionlint@v1\.7\.12/);
  assert.match(workflow, /govulncheck@v1\.9\.0/);
  assert.match(workflow, /npm --prefix frontend audit --audit-level=low/);
  assert.match(workflow, /npm --prefix frontend ci --ignore-scripts/);
  for (const action of workflow.matchAll(/uses: ([^\s]+)/g)) {
    assert.match(
      action[1],
      /^actions\/[^@]+@[0-9a-f]{40}$/,
      `unreviewed action reference: ${action[1]}`,
    );
  }
});
