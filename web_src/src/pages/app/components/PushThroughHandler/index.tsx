import React from "react";
import { Button } from "@/components/ui/button";
import { ArrowRight } from "lucide-react";

export interface PushThroughHandlerProps {
  onPushThrough?: () => void | Promise<void>;
}

export const PushThroughHandler: React.FC<PushThroughHandlerProps> = ({ onPushThrough }) => {
  if (!onPushThrough) {
    return null;
  }

  const handlePushThrough = async (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    await onPushThrough?.();
  };

  return (
    <div className="w-full p-3">
      <Button variant="outline" size="sm" onClick={handlePushThrough}>
        Push through
        <ArrowRight />
      </Button>
    </div>
  );
};
