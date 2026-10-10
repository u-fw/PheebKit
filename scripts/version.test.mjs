import assert from "node:assert/strict";
import { test } from "node:test";
import { parseVersion } from "./version.mjs";

test("Windows versions order prerelease stages before final", () => {
  const versions = [
    "0.1.0-alpha.1",
    "0.1.0-beta.1",
    "0.1.0-beta.2",
    "0.1.0-rc.1",
    "0.1.0",
  ];
  const revisions = versions.map((version) =>
    Number(parseVersion(version).windowsVersion.split(".")[3]),
  );
  assert.deepEqual(
    [...revisions].sort((a, b) => a - b),
    revisions,
  );
  assert.equal(parseVersion("0.1.0-beta.1").windowsVersion, "0.1.0.10001");
  assert.equal(parseVersion("1.0.0").prerelease, false);
});
test("unsafe, ambiguous and overflowing release identifiers are rejected", () => {
  for (const version of [
    "v0.1.0",
    "01.0.0",
    "1.0.0-beta.01",
    "1.0.0-beta.10000",
    "65536.0.0",
    "1.0.0/../escape",
    "1.0.0;cmd",
    "1.0.0-beta",
    "1.0.0\n2.0.0",
  ])
    assert.throws(() => parseVersion(version));
});
