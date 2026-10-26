import type { IconName } from "./components/Icon";

export interface CommandAction {
  id: string;
  label: string;
  description: string;
  icon: IconName;
  keywords?: readonly string[];
  shortcut?: string;
  disabled?: boolean;
  run: () => void | Promise<void>;
}

export function filterCommands(
  commands: readonly CommandAction[],
  query: string,
  translate: (value: string) => string = (value) => value,
): CommandAction[] {
  const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (!terms.length) return [...commands];
  return commands.filter((command) => {
    const haystack = [
      command.label,
      translate(command.label),
      command.description,
      translate(command.description),
      command.shortcut ?? "",
      ...(command.keywords ?? []).flatMap((keyword) => [
        keyword,
        translate(keyword),
      ]),
    ]
      .join(" ")
      .toLowerCase();
    return terms.every((term) => haystack.includes(term));
  });
}

export function nextCommandIndex(
  commands: readonly CommandAction[],
  current: number,
  direction: 1 | -1,
): number {
  if (!commands.length) return -1;
  let index = current;
  for (let count = 0; count < commands.length; count++) {
    index = (index + direction + commands.length) % commands.length;
    if (!commands[index]?.disabled) return index;
  }
  return -1;
}
