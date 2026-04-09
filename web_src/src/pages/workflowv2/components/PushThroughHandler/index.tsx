import React from "react";
import { Button } from "@/ui/button";
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
      <Button variant="outline" className="h-7 py-1 px-2" onClick={handlePushThrough}>
        Push through
        <ArrowRight className="size-4" />
      </Button>
    </div>
  );
};
