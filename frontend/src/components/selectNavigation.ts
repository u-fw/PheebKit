export interface SelectOption {
  value: string;
  label: string;
  description?: string;
  disabled?: boolean;
  translate?: boolean;
}

export function nextEnabled(
  options: readonly SelectOption[],
  current: number,
  direction: 1 | -1,
): number {
  if (!options.length) return -1;
  for (let step = 1; step <= options.length; step++) {
    const index =
      (((current + direction * step) % options.length) + options.length) %
      options.length;
    if (!options[index].disabled) return index;
  }
  return -1;
}

export function numericOption(options: readonly SelectOption[], text: string) {
  return /^\d+$/.test(text)
    ? options.find(
        (option) => !option.disabled && Number(option.value) === Number(text),
      )
    : undefined;
}

export function findOptionByPrefix(
  options: readonly SelectOption[],
  prefix: string,
  current: number,
): number {
  const normalized = prefix.toLocaleLowerCase();
  for (let step = 1; step <= options.length; step++) {
    const index = (Math.max(current, -1) + step) % options.length;
    if (
      !options[index].disabled &&
      options[index].label.toLocaleLowerCase().startsWith(normalized)
    )
      return index;
  }
  return -1;
}
