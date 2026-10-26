import { useEffect } from "react";

export function useMaterialRipple() {
  useEffect(() => {
    const timers = new Map<HTMLElement, number>();
    function showRipple(event: PointerEvent) {
      if (event.pointerType === "mouse" && event.button !== 0) return;
      if (!(event.target instanceof Element)) return;
      const control = event.target.closest<HTMLElement>(
        "button:not(:disabled), a[href]",
      );
      if (!control || control.dataset.noRipple === "true") return;
      const rect = control.getBoundingClientRect();
      control.style.setProperty("--ripple-x", `${event.clientX - rect.left}px`);
      control.style.setProperty("--ripple-y", `${event.clientY - rect.top}px`);
      control.classList.remove("ripple-active");
      void control.offsetWidth;
      control.classList.add("ripple-active");
      const previous = timers.get(control);
      if (previous !== undefined) window.clearTimeout(previous);
      const timer = window.setTimeout(() => {
        control.classList.remove("ripple-active");
        timers.delete(control);
      }, 520);
      timers.set(control, timer);
    }
    document.addEventListener("pointerdown", showRipple, {
      capture: true,
      passive: true,
    });
    return () => {
      document.removeEventListener("pointerdown", showRipple, true);
      timers.forEach((timer) => window.clearTimeout(timer));
    };
  }, []);
}
