import React from "react";
import { calcRelativeTimeFromDiff } from "@/lib/utils";

interface TimeLeftCountdownProps {
  createdAt: Date;
  expectedDuration: number;
}

export const TimeLeftCountdown: React.FC<TimeLeftCountdownProps> = ({ createdAt, expectedDuration }) => {
  const [timeLeft, setTimeLeft] = React.useState<number>(0);

  React.useEffect(() => {
    const updateTimeLeft = () => {
      const elapsed = Date.now() - createdAt.getTime();
      const remaining = Math.max(0, expectedDuration - elapsed);
      setTimeLeft(remaining);
    };

    updateTimeLeft();

    const interval = setInterval(updateTimeLeft, 1000);

    return () => clearInterval(interval);
  }, [createdAt, expectedDuration]);

  const timeLeftText = timeLeft >= 0 ? `Time left: ${calcRelativeTimeFromDiff(timeLeft)}` : "Ready to run";

  return <span>{timeLeftText}</span>;
};
