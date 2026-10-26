import { appendFileSync, readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parseVersion, readVersion } from "./version.mjs";

export function planWindowsCI({ eventName, ref, inputs = {}, version }) {
  const { prerelease } = parseVersion(version);
  if (!["pull_request", "push", "workflow_dispatch"].includes(eventName))
    throw new Error("Unsupported Windows CI event");
  const dispatched = eventName === "workflow_dispatch";
  const checksOnly = dispatched && [true, "true"].includes(inputs.checks_only);
  const releaseTag = dispatched
    ? (inputs.release_tag ?? "")
    : eventName === "push" && ref.startsWith("refs/tags/")
      ? ref.slice("refs/tags/".length)
      : "";
  if (
    typeof releaseTag !== "string" ||
    (releaseTag && releaseTag !== `v${version}`)
  )
    throw new Error("Release tag must exactly match VERSION");

  let packageArtifacts = !checksOnly && (dispatched || Boolean(releaseTag));
  let reason = "Verify source, real media and the production binary";
  if (packageArtifacts && !prerelease && !inputs.stable_evidence?.trim()) {
    if (dispatched)
      throw new Error(
        "Final packaging requires reviewed stable_evidence for this revision",
      );
    packageArtifacts = false;
    reason =
      "Final tag: verification only; dispatch with stable_evidence to package and publish";
  } else if (packageArtifacts) {
    reason = releaseTag
      ? "Verify, package and publish the selected tag"
      : "Verify and package a reviewable candidate";
  }
  return {
    packageArtifacts,
    publish: packageArtifacts && Boolean(releaseTag),
    releaseTag,
    reason,
  };
}

if (
  process.argv[1] &&
  resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  const root = fileURLToPath(new URL("../", import.meta.url));
  const event = JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, "utf8"));
  const plan = planWindowsCI({
    eventName: process.env.GITHUB_EVENT_NAME,
    ref: process.env.GITHUB_REF ?? "",
    inputs: event.inputs ?? {},
    version: readVersion().version,
  });
  const git = (args) =>
    execFileSync("git", ["-C", root, ...args], { encoding: "utf8" }).trim();
  const revision = git(["rev-parse", "HEAD"]);
  if (!/^[a-f0-9]{40}$/.test(revision))
    throw new Error("Invalid CI source revision");
  if (
    plan.releaseTag &&
    git(["rev-parse", "--verify", `refs/tags/${plan.releaseTag}^{commit}`]) !==
      revision
  )
    throw new Error(
      "Selected tag does not identify the checked-out source revision",
    );
  appendFileSync(
    process.env.GITHUB_OUTPUT,
    [
      `source_revision=${revision}`,
      `release_tag=${plan.releaseTag}`,
      `package=${plan.packageArtifacts}`,
      `publish=${plan.publish}`,
      "",
    ].join("\n"),
  );
  if (process.env.GITHUB_STEP_SUMMARY)
    appendFileSync(
      process.env.GITHUB_STEP_SUMMARY,
      `${plan.reason}. Source: \`${revision}\`.\n`,
    );
  console.log(plan.reason);
}
