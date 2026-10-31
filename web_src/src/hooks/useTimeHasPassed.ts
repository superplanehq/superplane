import { useEffect, useState } from "react";

export function useTimeHasPassed(time?: string) {
  const [now, setNow] = useState(Date.now);
  const deadline = time ? new Date(time).getTime() : NaN;

  useEffect(() => {
    if (!Number.isFinite(deadline)) return;
    let timer: ReturnType<typeof setTimeout>;
    const scheduleUpdate = () => {
      const current = Date.now();
      setNow(current);
      if (deadline > current) {
        // Monthly resets can exceed the browser's maximum timeout delay.
        timer = setTimeout(scheduleUpdate, Math.min(deadline - current, 2147483647));
      }
    };
    scheduleUpdate();
    return () => clearTimeout(timer);
  }, [deadline]);

  return deadline <= Math.max(now, Date.now());
}
