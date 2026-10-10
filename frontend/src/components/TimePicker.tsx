import { MaterialSelect } from "./MaterialSelect";
import { formatTimecode } from "../trim";
import { chooseTimePart, timeOptions, timeParts } from "../timePicker";

export function TimePicker({
  label,
  value,
  max,
  onChange,
  invalid,
  describedBy,
}: {
  label: string;
  value: number;
  max: number;
  onChange: (value: number) => void;
  invalid: boolean;
  describedBy: string;
}) {
  const parts = timeParts(value);
  return (
    <fieldset className="time-picker">
      <legend>
        {label}
        <span>{formatTimecode(value)}</span>
      </legend>
      <div className="time-picker-units">
        {(["hours", "minutes", "seconds"] as const).map((unit, index) => (
          <MaterialSelect
            key={unit}
            label={["时", "分", "秒"][index]}
            accessibleLabel={`${label} · ${["时", "分", "秒"][index]}`}
            value={String(parts[unit])}
            options={timeOptions(value, unit, max)}
            onChange={(part) =>
              onChange(chooseTimePart(value, unit, Number(part), max))
            }
            invalid={invalid}
            describedBy={describedBy}
            disabled={unit === "hours" && max < 3600}
          />
        ))}
      </div>
      <div className="time-picker-fine">
        <button
          type="button"
          className="button outlined"
          disabled={value <= 0}
          onClick={() =>
            onChange(Math.max(0, Math.round(value * 1000) - 100) / 1000)
          }
          aria-label={`${label}减少 0.1 秒`}
        >
          − 0.1 秒
        </button>
        <span className="field-hint">
          毫秒 {String(parts.milliseconds).padStart(3, "0")}
        </span>
        <button
          type="button"
          className="button outlined"
          disabled={value >= Math.floor(max * 1000) / 1000}
          onClick={() =>
            onChange(
              Math.min(Math.floor(max * 1000), Math.round(value * 1000) + 100) /
                1000,
            )
          }
          aria-label={`${label}增加 0.1 秒`}
        >
          + 0.1 秒
        </button>
      </div>
    </fieldset>
  );
}
