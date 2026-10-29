import assert from "node:assert/strict";
import {
  copyFileSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { join, sep } from "node:path";
import { test } from "node:test";
import { planWindowsCI } from "./ci-plan.mjs";

const workflow = readFileSync(
  new URL("../.github/workflows/windows.yml", import.meta.url),
  "utf8",
).replaceAll("\r\n", "\n");

test("verification is read-only and publication has isolated write permission", () => {
  assert.match(workflow, /^permissions:\n  contents: read$/m);
  const verification = workflow
    .split("\n  windows:\n")[1]
    .split("\n  attest:\n")[0];
  assert.doesNotMatch(
    verification,
    /^\s+(?:attestations|id-token|packages):\s+write$/m,
  );
  assert.doesNotMatch(verification, /^\s+contents:\s+write$/m);
  assert.match(
    workflow,
    /  publish:\n(?:.|\n)*?    permissions:\n      contents: write/,
  );
});

for (const version of ["0.2.0-rc.7", "0.2.0"]) {
  for (const eventName of ["pull_request", "push"]) {
    test(`${eventName} ${version} verifies without requiring release evidence`, () => {
      const plan = planWindowsCI({
        eventName,
        ref: "refs/heads/main",
        version,
      });
      assert.equal(plan.packageArtifacts, false);
      assert.equal(plan.publish, false);
      assert.equal(plan.releaseTag, "");
    });
  }
}

test("prerelease tags publish while final tag pushes require a reviewed dispatch", () => {
  for (const version of ["0.2.0-rc.7", "0.2.0"]) {
    const plan = planWindowsCI({
      eventName: "push",
      ref: `refs/tags/v${version}`,
      version,
    });
    assert.equal(plan.packageArtifacts, version.includes("-"));
    assert.equal(plan.publish, version.includes("-"));
    assert.equal(plan.releaseTag, `v${version}`);
  }
});

test("manual candidates package without publishing, and final candidates still require evidence", () => {
  const input = {
    eventName: "workflow_dispatch",
    ref: "refs/heads/main",
    version: "0.2.0-rc.7",
  };
  assert.equal(planWindowsCI(input).packageArtifacts, true);
  assert.equal(planWindowsCI(input).publish, false);
  assert.throws(
    () => planWindowsCI({ ...input, version: "0.2.0" }),
    /stable_evidence/,
  );
  const reviewed = planWindowsCI({
    ...input,
    version: "0.2.0",
    inputs: { stable_evidence: "reviewed record" },
  });
  assert.equal(reviewed.packageArtifacts, true);
  assert.equal(reviewed.publish, false);
});

test("explicit manual tags publish only after final evidence and matching VERSION", () => {
  for (const version of ["0.2.0-rc.7", "0.2.0"]) {
    const inputs = {
      release_tag: `v${version}`,
      stable_evidence: "reviewed record",
    };
    const plan = planWindowsCI({
      eventName: "workflow_dispatch",
      ref: "refs/heads/main",
      version,
      inputs,
    });
    assert.equal(plan.packageArtifacts, true);
    assert.equal(plan.publish, true);
  }
});

test("checks-only dispatch never packages or publishes, including final tags without evidence", () => {
  for (const checks_only of [true, "true"]) {
    for (const version of ["0.2.0-rc.7", "0.2.0"]) {
      const plan = planWindowsCI({
        eventName: "workflow_dispatch",
        ref: "refs/heads/main",
        version,
        inputs: { checks_only, release_tag: `v${version}` },
      });
      assert.equal(plan.packageArtifacts, false);
      assert.equal(plan.publish, false);
    }
  }
  assert.equal(
    planWindowsCI({
      eventName: "workflow_dispatch",
      version: "0.2.0-rc.7",
      inputs: { checks_only: "false" },
    }).packageArtifacts,
    true,
  );
});

test("invalid or mismatched tags cannot enter output records, and PR inputs cannot request publication", () => {
  for (const release_tag of [
    "v0.3.0",
    "main",
    "v0.2.0\npublish=true",
    "--help",
  ]) {
    assert.throws(
      () =>
        planWindowsCI({
          eventName: "workflow_dispatch",
          version: "0.2.0",
          inputs: { release_tag },
        }),
      /VERSION/,
    );
  }
  assert.throws(
    () =>
      planWindowsCI({
        eventName: "push",
        ref: "refs/tags/v0.3.0",
        version: "0.2.0",
      }),
    /VERSION/,
  );
  assert.throws(
    () => planWindowsCI({ eventName: "pull_request_target", version: "0.2.0" }),
    /Unsupported/,
  );
  assert.equal(
    planWindowsCI({
      eventName: "pull_request",
      version: "0.2.0",
      inputs: {
        release_tag: "v0.2.0",
        stable_evidence: "not an authorized dispatch",
      },
    }).publish,
    false,
  );
});

test("workflow executes the tested plan and all verification paths use the same quality checks", () => {
  assert.match(
    workflow,
    /id: plan\n\s+shell: pwsh\n\s+run: \|\n\s+node scripts\/ci-plan\.mjs/,
  );
  assert.match(
    workflow,
    /ref: \$\{\{ inputs\.release_tag \|\| github\.sha \}\}/,
  );
  const checks = workflow
    .split("      - name: Verify source and frontend\n")[1]
    .split("      - name:")[0];
  assert.doesNotMatch(checks, /\n\s+if:/);
  assert.match(checks, /release-workflow\.test\.mjs/);
  assert.match(
    checks,
    /go vet @packages[\s\S]*staticcheck@v0\.8\.2 @packages[\s\S]*govulncheck@v1\.9\.0/,
  );
  assert.match(
    workflow,
    /Verify production binary without publishing\n\s+if: steps\.plan\.outputs\.package != 'true'/,
  );
  assert.match(
    workflow,
    /Verify and package Windows x64\n\s+if: steps\.plan\.outputs\.package == 'true'/,
  );
  assert.match(
    workflow,
    /name: Run Go race detector[\s\S]*CGO_ENABLED: "1"[\s\S]*go test -race @packages -count=1/,
  );
});

test("publication binds the tested revision and digest and refuses asset replacement", () => {
  const publish = workflow.split("\n  publish:\n")[1];
  assert.match(publish, /needs: \[windows, attest\]/);
  assert.match(publish, /needs\.windows\.outputs\.publish == 'true'/);
  assert.match(publish, /github\.event_name == 'push'/);
  assert.match(publish, /github\.event_name == 'workflow_dispatch'/);
  assert.match(
    publish,
    /ref: \$\{\{ needs\.windows\.outputs\.source_revision \}\}/,
  );
  assert.match(publish, /needs\.windows\.outputs\.checksums_sha256/);
  assert.match(publish, /Get-VerifiedReleaseAssets/);
  assert.equal(publish.match(/Assert-ReleaseTag -Tag/g).length, 2);
  assert.match(publish, /--verify-tag --draft/);
  assert.doesNotMatch(publish, /--clobber/);
  assert.doesNotMatch(publish, /--build-info|Start-Process/);
});

test("packaged artifacts are attested in an isolated OIDC job before publication", () => {
  assert.match(
    workflow,
    /  windows:\n(?:.|\n)*?      version: \$\{\{ steps\.plan\.outputs\.version \}\}\n      package: \$\{\{ steps\.plan\.outputs\.package \}\}/,
  );
  const attest = workflow.split("\n  attest:\n")[1].split("\n  publish:\n")[0];
  assert.match(attest, /needs: windows/);
  assert.match(attest, /needs\.windows\.outputs\.package == 'true'/);
  assert.match(
    attest,
    /permissions:\n      attestations: write\n      contents: read\n      id-token: write/,
  );
  assert.match(
    attest,
    /name: Generate release artifact provenance\n        uses: actions\/attest@1e69f48acb82d1966a394da916b4c1698aa569d6 # v4\.2\.2\n        with:\n          subject-path: release-upload\/\$\{\{ needs\.windows\.outputs\.version \}\}\/\*/,
  );
  assert.doesNotMatch(attest, /--build-info|Start-Process|release-windows/);
});

test("CI upgrades from verified published packages without bootstrapping historical Wails beta tooling", () => {
  const upgrade = workflow
    .split("      - name: Download and verify published upgrade baseline\n")[1]
    .split("      - name:")[0];
  assert.match(upgrade, /GH_TOKEN: \$\{\{ github\.token \}\}/);
  assert.match(
    upgrade,
    /prepare-upgrade-baseline\.ps1 -Ref \$env:PHEEBKIT_UPGRADE_FROM_TAG -Published/,
  );
  assert.doesNotMatch(upgrade, /wails3|go install|worktree/);
});

test("planner CLI emits the actual checked-out revision and refuses a retargeted tag before writing outputs", () => {
  const tools = fileURLToPath(new URL("../.tools/", import.meta.url));
  mkdirSync(tools, { recursive: true });
  const root = mkdtempSync(join(tools, "ci-plan-test-"));
  try {
    mkdirSync(join(root, "scripts"));
    for (const name of ["ci-plan.mjs", "version.mjs"]) {
      copyFileSync(
        fileURLToPath(new URL(name, import.meta.url)),
        join(root, "scripts", name),
      );
    }
    writeFileSync(join(root, "VERSION"), "0.2.0\n");
    const git = (args) => {
      const result = spawnSync(
        "git",
        [
          "-C",
          root,
          "-c",
          "user.name=PheebKit tests",
          "-c",
          "user.email=tests@example.invalid",
          "-c",
          "commit.gpgsign=false",
          "-c",
          "tag.gpgsign=false",
          ...args,
        ],
        { encoding: "utf8", windowsHide: true },
      );
      assert.equal(result.status, 0, result.stderr);
      return result.stdout.trim();
    };
    git(["init", "-q"]);
    git(["add", "."]);
    git(["commit", "-qm", "planner fixture"]);
    const revision = git(["rev-parse", "HEAD"]);
    git(["tag", "v0.2.0"]);
    const eventPath = join(root, "event.json");
    const outputPath = join(root, "outputs.txt");
    const run = (eventName, ref, inputs = {}) => {
      writeFileSync(eventPath, JSON.stringify({ inputs }));
      writeFileSync(outputPath, "");
      return spawnSync(
        process.execPath,
        [join(root, "scripts", "ci-plan.mjs")],
        {
          encoding: "utf8",
          windowsHide: true,
          env: {
            ...process.env,
            GITHUB_EVENT_NAME: eventName,
            GITHUB_REF: ref,
            GITHUB_EVENT_PATH: eventPath,
            GITHUB_OUTPUT: outputPath,
            GITHUB_STEP_SUMMARY: join(root, "summary.txt"),
          },
        },
      );
    };
    let result = run("pull_request", "refs/pull/1/merge");
    assert.equal(result.status, 0, result.stderr);
    assert.equal(
      readFileSync(outputPath, "utf8"),
      `source_revision=${revision}\nrelease_tag=\nversion=0.2.0\npackage=false\npublish=false\n`,
    );
    result = run("workflow_dispatch", "refs/heads/main", {
      release_tag: "v0.2.0",
      stable_evidence: "reviewed record",
    });
    assert.equal(result.status, 0, result.stderr);
    assert.match(
      readFileSync(outputPath, "utf8"),
      /release_tag=v0\.2\.0\nversion=0\.2\.0\npackage=true\npublish=true/,
    );
    result = run("push", "refs/tags/v0.2.0");
    assert.equal(result.status, 0, result.stderr);
    assert.match(
      readFileSync(outputPath, "utf8"),
      /package=false\npublish=false/,
    );
    const otherRevision = git([
      "commit-tree",
      git(["rev-parse", "HEAD^{tree}"]),
      "-p",
      revision,
      "-m",
      "retarget fixture",
    ]);
    git(["tag", "-f", "v0.2.0", otherRevision]);
    result = run("workflow_dispatch", "refs/heads/main", {
      release_tag: "v0.2.0",
      stable_evidence: "reviewed record",
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /does not identify the checked-out source/);
    assert.equal(readFileSync(outputPath, "utf8"), "");
  } finally {
    assert.ok(root.startsWith(tools.endsWith(sep) ? tools : tools + sep));
    rmSync(root, { recursive: true, force: true });
  }
});

test(
  "upload verification rejects altered assets, provenance and lightweight or annotated tag movement",
  { timeout: 60_000 },
  (t) => {
    const result = spawnSync(
      "pwsh",
      [
        "-NoProfile",
        "-NonInteractive",
        "-File",
        fileURLToPath(new URL("./release-upload.test.ps1", import.meta.url)),
      ],
      { encoding: "utf8", timeout: 55_000, windowsHide: true },
    );
    assert.equal(result.error, undefined, result.error?.message);
    assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`);
    t.diagnostic(result.stdout.trim());
  },
);
