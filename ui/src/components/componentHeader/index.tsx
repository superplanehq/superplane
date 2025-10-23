import React from "react";
import { resolveIcon } from "@/lib/utils";

export interface ComponentHeaderProps {
  iconSrc?: string;
  iconSlug?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor: string;
  title: string;
  description?: string;
}

export const ComponentHeader: React.FC<ComponentHeaderProps> = ({
  iconSrc,
  iconSlug,
  iconBackground,
  iconColor,
  headerColor,
  title,
  description,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  return (
    <div className={"w-full px-2 flex flex-col border-b p-2 gap-2 rounded-t-md items-center " + headerColor}>
      <div className="w-full flex items-center gap-2">
        <div className={`w-6 h-6 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
          {iconSrc ? <img src={iconSrc} alt={title} className="w-5 h-5 " /> : <Icon size={20} className={iconColor} />}
        </div>
        <h2 className="text-md font-semibold">{title}</h2>
      </div>
      {description && <p className="w-full text-md text-gray-500 pl-8">{description}</p>}
    </div>
  );
};