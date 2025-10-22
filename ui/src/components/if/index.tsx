import * as React from "react"
import { GitBranch } from "lucide-react"
import { Handle, Position } from "@xyflow/react"

import { cn } from "@/lib/utils"

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../card"

export interface IfData {
  label?: string
  component?: string
  channels?: string[]
  configuration?: Record<string, any>
}

export interface IfProps {
  data: IfData
  selected?: boolean
  collapsed?: boolean
  className?: string
  showHandles?: boolean
}

export const If: React.FC<IfProps> = ({
  data,
  selected = false,
  collapsed = false,
  className,
  showHandles = true,
}) => {
  const channels = (data.channels as string[]) || ['true', 'false']
  const expression = (data.configuration as Record<string, any>)?.expression

  if (collapsed) {
    return (
      <div className={cn("ui-theme flex w-fit flex-col items-center", className)}>
        <div
          className={cn(
            "flex h-20 w-20 items-center justify-center rounded-2xl text-gray-900 bg-blue-300",
            selected ? "border-[3px] border-black" : "border border-border",
          )}
        >
          <GitBranch className="size-10" />
        </div>
        <CardTitle className="text-base font-semibold text-neutral-900 pt-1">
          {data.label as string}
        </CardTitle>
        <div className="flex items-center gap-1 text-sm text-muted-foreground">
          {channels.map((channel, index) => (
            <React.Fragment key={channel}>
              {index > 0 && <span>/</span>}
              <span>{channel}</span>
            </React.Fragment>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className={cn("ui-theme w-fit min-w-[300px] max-w-md", className)}>
      <Card
        className={cn(
          "flex h-full w-full flex-col overflow-hidden p-0 rounded-3xl relative",
          selected
            ? "border-[3px] border-black shadow-none"
            : "border-none shadow-lg",
        )}
      >
        {showHandles && (
          <Handle
            type="target"
            position={Position.Left}
            className="!w-3 !h-3 !bg-slate-500 !border-2 !border-white dark:!border-zinc-800 !absolute !left-[-6px] !top-1/2 !transform !-translate-y-1/2"
          />
        )}

        <CardHeader
          className="space-y-1 rounded-t-3xl px-4 py-3 text-base text-neutral-900 bg-blue-300"
        >
          <CardTitle className="flex items-center gap-2 text-sm font-semibold">
            <GitBranch className="size-4" />
            {data.label as string}
          </CardTitle>
        </CardHeader>

        <CardContent className="flex flex-col gap-3 rounded-none px-4 py-3 bg-white">
          {expression && (
            <div className="text-xs font-mono text-neutral-600 bg-gray-50 px-3 py-2 rounded border">
              {expression}
            </div>
          )}

          <div className="flex flex-col gap-2">
            {channels.map((channel, index) => (
              <div
                key={channel}
                className="relative flex items-center justify-between px-3 py-2 bg-gray-50 rounded border hover:bg-gray-100 transition-colors"
              >
                <span className="text-xs font-medium text-neutral-700">
                  {channel}
                </span>
                {showHandles && (
                  <Handle
                    type="source"
                    position={Position.Right}
                    id={channel}
                    className="!w-3 !h-3 !bg-slate-500 !border-2 !border-white dark:!border-zinc-800 !absolute !right-[-6px] !top-1/2 !transform !-translate-y-1/2"
                  />
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}