import React from "react";
import { calcRelativeTimeFromDiff } from "@/lib/utils";

export function TimeGateCountdown({
  nextValidTime,
  timeAgo,
}: {
  nextValidTime: string;
  timeAgo?: string | React.ReactNode;
}) {
  const nextRunTime = React.useMemo(() => new Date(nextValidTime), [nextValidTime]);
  const [timeLeft, setTimeLeft] = React.useState<number>(() => nextRunTime.getTime() - Date.now());

  React.useEffect(() => {
    if (Number.isNaN(nextRunTime.getTime())) {
      return;
    }

    const update = () => {
      setTimeLeft(nextRunTime.getTime() - Date.now());
    };

    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [nextRunTime]);

  if (Number.isNaN(nextRunTime.getTime())) {
    return <span>{timeAgo || ""}</span>;
  }

  const timeLeftText = timeLeft > 0 ? calcRelativeTimeFromDiff(timeLeft) : "Ready to run";
  return (
    <span>
      Runs in {timeLeftText}
      {timeAgo ? (
        <>
          {" · "}
          {timeAgo}
        </>
      ) : (
        ""
      )}
    </span>
  );
}
