export type TimeUnit = "hours" | "minutes" | "seconds";

export function timeParts(value: number) {
  const total = Number.isFinite(value)
    ? Math.max(0, Math.round(value * 1000))
    : 0;
  return {
    hours: Math.floor(total / 3600000),
    minutes: Math.floor(total / 60000) % 60,
    seconds: Math.floor(total / 1000) % 60,
    milliseconds: total % 1000,
  };
}

export function chooseTimePart(
  value: number,
  unit: TimeUnit,
  part: number,
  max: number,
): number {
  const parts = timeParts(value);
  parts[unit] = part;
  return (
    Math.min(
      Math.floor(max * 1000),
      ((parts.hours * 60 + parts.minutes) * 60 + parts.seconds) * 1000 +
        parts.milliseconds,
    ) / 1000
  );
}

export function timeOptions(value: number, unit: TimeUnit, max: number) {
  const parts = timeParts(value);
  const limit =
    unit === "hours"
      ? Math.floor(max / 3600)
      : unit === "minutes"
        ? Math.min(59, Math.floor((max - parts.hours * 3600) / 60))
        : Math.min(
            59,
            Math.floor(max - parts.hours * 3600 - parts.minutes * 60),
          );
  return Array.from({ length: Math.max(0, limit) + 1 }, (_, part) => {
    const next = { ...parts, [unit]: part, milliseconds: 0 };
    const seconds = (next.hours * 60 + next.minutes) * 60 + next.seconds;
    // Smaller units are clamped when moving into the last hour/minute.
    const minimum =
      unit === "hours"
        ? part * 3600
        : unit === "minutes"
          ? parts.hours * 3600 + part * 60
          : seconds;
    return {
      value: String(part),
      label: String(part).padStart(2, "0"),
      disabled: minimum > max,
    };
  });
}
