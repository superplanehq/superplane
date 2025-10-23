import * as React from "react"

import { cn } from "@/lib/utils"

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../card"
import { ApprovalItem, type ApprovalItemProps } from "../approvalItem"
import { ItemGroup, ItemSeparator } from "../item"
import { Button } from "../button"
import { Hand } from "lucide-react"

export interface ApprovalProps {
  title: string
  status?: string
  version?: string
  approvals?: ApprovalItemProps[]
  footerContent?: React.ReactNode
  className?: string
  selected?: boolean
  collapsed?: boolean
}

export const Approval: React.FC<ApprovalProps> = ({
  title,
  status,
  version,
  approvals,
  className,
  selected = false,
  collapsed = false,
}) => {
  if (collapsed) {
    const pendingCount = approvals?.filter(approval => !approval.approved).length || 0
    const totalCount = approvals?.length || 0

    return (
      <div className={cn("flex w-fit flex-col items-center", className)}>
        <div
          className={cn(
            "flex h-20 w-20 items-center justify-center rounded-2xl text-gray-900 bg-yellow-300",
            selected ? "border-[3px] border-black" : "border border-border",
          )}
        >
          <Hand className="size-10" />
        </div>
        <CardTitle className="text-base font-semibold text-neutral-900 pt-1">
          {title}
        </CardTitle>
        <Button
          variant="linkSubdued"
          className="justify-center text-sm"
          asChild
        >
          <a
            href="#"
            className="flex items-center"
          >
            {pendingCount}/{totalCount} pending
          </a>
        </Button>
      </div>
    )
  }

  return (
    <div className={cn("w-full min-w-[500px] max-w-6xl", className)}>
      <Card
        className={cn(
          "flex h-full w-full flex-col overflow-hidden p-0 rounded-3xl",
          selected
            ? "border-[3px] border-black shadow-none"
            : "border-none shadow-lg",
        )}
      >
        <CardHeader
          className="space-y-2 rounded-t-3xl px-8 py-6 text-base text-neutral-900 bg-yellow-300"
        >
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6 rounded-none px-8 py-6 bg-white">
          {status && (
            <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {status}
            </div>
          )}
          {version && (
            <div className="text-2xl font-bold text-neutral-900">
              {version}
            </div>
          )}
          {approvals && approvals.length > 0 ? (
            <ItemGroup className="w-full">
              {approvals.map((approval, index) => (
                <React.Fragment key={`${approval.title}-${index}`}>
                  <ApprovalItem {...approval} />
                  {index < approvals.length - 1 && <ItemSeparator />}
                </React.Fragment>
              ))}
            </ItemGroup>
          ) : (
            <p className="text-sm text-muted-foreground">No approvals yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
