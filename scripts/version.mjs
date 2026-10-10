import { readFileSync } from "node:fs";

export function parseVersion(value) {
  const text = value.trim();
  const match =
    /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(alpha|beta|rc)\.(0|[1-9]\d*))?$/.exec(
      text,
    );
  if (!match) throw new Error(`Invalid release version: ${text}`);
  const numbers = match.slice(1, 4).map(Number);
  // Ordered Windows version resources: alpha < beta < rc < final.
  const sequence = match[5] === undefined ? 0 : Number(match[5]);
  if (sequence > 9999 || numbers.some((number) => number > 65535))
    throw new Error("Release version exceeds Windows version-resource limits");
  const revision = match[4]
    ? { alpha: 0, beta: 10000, rc: 20000 }[match[4]] + sequence
    : 65535;
  return {
    version: text,
    windowsVersion: [...numbers, revision].join("."),
    prerelease: Boolean(match[4]),
  };
}

export function readVersion() {
  return parseVersion(
    readFileSync(new URL("../VERSION", import.meta.url), "utf8"),
  );
}
