import { Approval, type ApprovalProps } from "@/ui/approval";
import { Composite, type CompositeProps } from "@/ui/composite";
import {
  SwitchComponent,
  type SwitchComponentProps,
} from "@/ui/switchComponent";
import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Handle, Position } from "@xyflow/react";
import { MoreVertical, PencilIcon, SparklesIcon } from "lucide-react";
import { Button } from "../button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../dropdownMenu";
import { Filter, FilterProps } from "../filter";
import { If, IfProps } from "../if";
import { Noop, NoopProps } from "../noop";

type BlockState = "pending" | "working" | "success" | "failed" | "running";
type BlockType =
  | "trigger"
  | "composite"
  | "approval"
  | "filter"
  | "if"
  | "noop"
  | "switch";

interface BlockAi {
  show: boolean;
  suggestion: string | null;
  onApply: () => void;
  onDismiss: () => void;
}

export interface BlockData {
  label: string;

  state: BlockState;
  type: BlockType;

  // last input event received by this block (for simulation display)
  lastEvent?: unknown;

  // trigger node specific props
  trigger?: TriggerProps;

  // composite node specific props
  composite?: CompositeProps;

  // approval node specific props
  approval?: ApprovalProps;

  // filter node specific props
  filter?: FilterProps;

  // if node specific props
  if?: IfProps;

  // noop node specific props
  noop?: NoopProps;

  // switch node specific props
  switch?: SwitchComponentProps;
}

interface BlockProps {
  data: BlockData;
  nodeId?: string;
  selected?: boolean;

  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  onClick?: () => void;
  onEdit?: (nodeId: string) => void;

  ai?: BlockAi;
}

export function Block(props: BlockProps) {
  const data = props.data;
  const ai = props.ai || {
    show: false,
    suggestion: null,
    onApply: () => {},
    onDismiss: () => {},
  };

  const handleEdit = () => {
    if (props.onEdit && props.nodeId) {
      props.onEdit(props.nodeId);
    }
  };

  return (
    <>
      <AiPopup {...ai} />

      <div className="relative w-fit" onClick={props.onClick}>
        <LeftHandle data={data} />
        <BlockContent {...props} onClick={props.onClick} />
        <RightHandle data={data} />

        {/* Three-dots menu at top right */}
        {props.onEdit && (
          <div className="absolute top-2 right-2 z-50">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 w-6 p-0 hover:bg-gray-100 dark:hover:bg-zinc-700"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreVertical size={16} />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
                <DropdownMenuItem onClick={handleEdit}>
                  <PencilIcon size={14} className="mr-2" />
                  Edit
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </div>
    </>
  );
}

//
// Handles are small connection points on the sides of blocks
//

const HANDLE_STYLE = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: "3px solid #C9D5E1",
  background: "transparent",
};

function LeftHandle({ data }: BlockProps) {
  if (data.type === "trigger") return null;

  const isCollapsed =
    (data.type === "composite" && data.composite?.collapsed) ||
    (data.type === "approval" && data.approval?.collapsed) ||
    (data.type === "filter" && data.filter?.collapsed) ||
    (data.type === "if" && data.if?.collapsed) ||
    (data.type === "noop" && data.noop?.collapsed) ||
    (data.type === "switch" && data.switch?.collapsed);

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{
        ...HANDLE_STYLE,
        left: -15,
        top: isCollapsed ? "50%" : 30,
        transform: isCollapsed ? "translateY(-50%)" : undefined,
      }}
    />
  );
}

function RightHandle({ data }: BlockProps) {
  const isCollapsed =
    (data.type === "composite" && data.composite?.collapsed) ||
    (data.type === "approval" && data.approval?.collapsed) ||
    (data.type === "trigger" && data.trigger?.collapsed) ||
    (data.type === "filter" && data.filter?.collapsed) ||
    (data.type === "if" && data.if?.collapsed) ||
    (data.type === "noop" && data.noop?.collapsed) ||
    (data.type === "switch" && data.switch?.collapsed);

  return (
    <Handle
      type="source"
      position={Position.Right}
      style={{
        ...HANDLE_STYLE,
        right: -15,
        top: isCollapsed ? "50%" : 30,
        transform: isCollapsed ? "translateY(-50%)" : undefined,
      }}
    />
  );
}

function AiPopup({ show, suggestion, onApply, onDismiss }: BlockAi) {
  if (!show) return null;
  if (!suggestion) return null;

  const handleApply = (e: React.MouseEvent) => {
    e.stopPropagation();
    onApply();
  };

  const handleDismiss = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDismiss();
  };

  return (
    <div className="absolute left-0 -translate-y-[100%] text-left text-base">
      <div className="bg-white rounded-lg shadow p-3 relative mb-2 border-blue-500 border-2">
        <div className="flex items-center gap-1 mb-2">
          <SparklesIcon className="inline-block text-blue-500" size={14} />
          <div className="text-gray-800 font-bold">Improvements</div>
        </div>

        <div className="text-sm">{suggestion}</div>

        <div className="flex gap-2 mt-2">
          <Button
            size="sm"
            variant="default"
            className="mt-2"
            onClick={handleApply}
          >
            Apply
          </Button>

          <Button
            size="sm"
            variant="secondary"
            className="mt-2"
            onClick={handleDismiss}
          >
            Dismiss
          </Button>
        </div>
      </div>
    </div>
  );
}

//
// Block content is the inner area of the block.
//

function BlockContent({
  data,
  onExpand,
  nodeId,
  selected = false,
}: BlockProps) {
  const handleExpand = () => {
    if (onExpand && nodeId) {
      onExpand(nodeId, data);
    }
  };

  switch (data.type) {
    case "trigger":
      return (
        <Trigger {...(data.trigger as TriggerProps)} selected={selected} />
      );
    case "composite":
      return (
        <Composite
          {...(data.composite as CompositeProps)}
          onExpandChildEvents={handleExpand}
          selected={selected}
        />
      );
    case "approval":
      return (
        <Approval {...(data.approval as ApprovalProps)} selected={selected} />
      );
    case "filter":
      return <Filter {...(data.filter as FilterProps)} selected={selected} />;
    case "if":
      return <If {...(data.if as IfProps)} selected={selected} />;
    case "noop":
      return <Noop {...(data.noop as NoopProps)} selected={selected} />;
    case "switch":
      return (
        <SwitchComponent
          {...(data.switch as SwitchComponentProps)}
          selected={selected}
        />
      );
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}
