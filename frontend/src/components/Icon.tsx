export type IconName =
  | "studio"
  | "home"
  | "compress"
  | "queue"
  | "settings"
  | "plus"
  | "arrow"
  | "video"
  | "audio"
  | "subtitle"
  | "check"
  | "close"
  | "copy"
  | "refresh"
  | "search"
  | "sun"
  | "moon"
  | "folder"
  | "chevron"
  | "terminal"
  | "info"
  | "pause"
  | "minimise"
  | "maximise"
  | "restore";

const paths: Record<IconName, string> = {
  studio: "M6 5v14 M7 12l9-7 M7 12l9 7 M17 9l4 3-4 3z",
  home: "M3 10l9-7 9 7v10h-6v-6H9v6H3z",
  compress:
    "M4 3v6h6 M20 3v6h-6 M4 21v-6h6 M20 21v-6h-6 M4 9l5-5 M20 9l-5-5 M4 15l5 5 M20 15l-5 5",
  queue: "M8 6h13 M8 12h13 M8 18h13 M3 6h1 M3 12h1 M3 18h1",
  settings:
    "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M9 3h6l1 3 3 1 2 5-2 5-3 1-1 3H9l-1-3-3-1-2-5 2-5 3-1z",
  plus: "M12 5v14 M5 12h14",
  arrow: "M5 12h14 M14 7l5 5-5 5",
  video: "M4 5h16v14H4z M9 9l6 3-6 3z",
  audio:
    "M9 18V5l11-2v13 M9 7l11-2 M9 18a3 3 0 1 1-3-3 3 3 0 0 1 3 3 M20 16a3 3 0 1 1-3-3 3 3 0 0 1 3 3",
  subtitle: "M3 5h18v14H3z M7 10h3 M14 10h3 M7 14h5 M15 14h2",
  check: "M5 12l4 4L19 6",
  close: "M6 6l12 12 M6 18L18 6",
  copy: "M8 8h12v13H8z M16 8V3H3v13h5",
  refresh:
    "M20 7v5h-5 M4 17v-5h5 M6.1 9a7 7 0 0 1 11.5-2L20 12 M4 12l2.4 5a7 7 0 0 0 11.5-2",
  search: "M10.5 18a7.5 7.5 0 1 0 0-15 7.5 7.5 0 0 0 0 15 M16 16l5 5",
  sun: "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M12 2v2 M12 20v2 M2 12h2 M20 12h2 M5 5l1 1 M18 18l1 1 M5 19l1-1 M18 6l1-1",
  moon: "M20 15A9 9 0 0 1 9 4a9 9 0 1 0 11 11z",
  folder: "M3 7V4h7l3 3h8v13H3z",
  chevron: "M8 5l7 7-7 7",
  terminal: "M3 4h18v16H3z M7 9l3 3-3 3 M13 15h4",
  info: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 11v6 M12 7h.01",
  pause: "M8 5v14 M16 5v14",
  minimise: "M5 15h14",
  maximise: "M5 5h14v14H5z",
  restore: "M8 8h11v11H8z M5 15V5h10",
};

export function Icon({
  name,
  className = "",
}: {
  name: IconName;
  className?: string;
}) {
  return (
    <svg
      className={`icon ${className}`}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={paths[name]} />
    </svg>
  );
}
