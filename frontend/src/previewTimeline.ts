// Preview positions never mutate a project's trim/segments. Frame stepping uses
// the declared average frame rate; variable-rate sources are an approximation.
export function boundedPosition(seconds: number, duration: number): number {
  if (!Number.isFinite(duration) || duration <= 0) return 0;
  return Math.min(
    duration,
    Math.max(0, Number.isFinite(seconds) ? seconds : 0),
  );
}

export function framePosition(
  seconds: number,
  direction: -1 | 1,
  frameRate: number,
  duration: number,
): number {
  const step =
    Number.isFinite(frameRate) && frameRate > 0 ? 1 / frameRate : 0.1;
  return boundedPosition(seconds + direction * step, duration);
}

export function filmstripPositions(duration: number): number[] {
  if (!Number.isFinite(duration) || duration <= 0) return [];
  // Eight independent, evenly spread source positions; no full-file decode.
  const count = Math.min(8, Math.max(1, Math.ceil(duration)));
  return Array.from(
    { length: count },
    (_, index) => (index * duration) / count,
  );
}

export function selectionPlayback(
  position: number,
  start: number,
  end: number,
  loop: boolean,
): { seek: number; pause: boolean } | null {
  if (
    !Number.isFinite(start) ||
    !Number.isFinite(end) ||
    start < 0 ||
    end <= start
  )
    return null;
  if (position < start) return { seek: start, pause: false };
  if (position < end) return null;
  return { seek: loop ? start : end, pause: !loop };
}
