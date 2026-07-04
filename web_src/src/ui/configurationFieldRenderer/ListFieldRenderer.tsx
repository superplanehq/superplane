import React from "react";
import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ChevronDown, GripVertical, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Accordion, AccordionContent, AccordionItem } from "@/ui/accordion";
import { DayInYearFieldRenderer } from "./DayInYearFieldRenderer";
import { RepositoryFileFieldRenderer } from "./RepositoryFileFieldRenderer";
import type { FieldRendererProps, ValidationError } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { listFieldItemTitle } from "./listFieldItemTitle";
import { useListFieldDragReorder } from "./useListFieldDragReorder";
import { cn } from "@/lib/utils";
import { showErrorToast } from "@/lib/toast";

interface ExtendedFieldRendererProps extends FieldRendererProps {
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
}

function getApproverKey(item: Record<string, unknown>) {
  const type = typeof item.type === "string" ? item.type : "";
  if (!type) return undefined;

  if (type === "user" && typeof item.user === "string" && item.user.trim()) {
    return `user:${item.user}`;
  }
  if (type === "role" && typeof item.role === "string" && item.role.trim()) {
    return `role:${item.role}`;
  }
  if (type === "group" && typeof item.group === "string" && item.group.trim()) {
    return `group:${item.group}`;
  }
  return undefined;
}

export const ListFieldRenderer: React.FC<ExtendedFieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  integrationId,
  organizationId,
  hasError: _,
  validationErrors,
  fieldPath = field.name || "",
  autocompleteExampleObj,
  allowExpressions = false,
}) => {
  const listOptions = field.typeOptions?.list;
  const itemDefinition = listOptions?.itemDefinition;
  const maxItems = listOptions?.maxItems;
  const useAccordion = listOptions?.accordion === true;
  const allowReorder = listOptions?.reorderable === true;
  const items = Array.isArray(value)
    ? itemDefinition?.type === "day-in-year"
      ? value.filter((item) => typeof item === "string" && item.trim().length > 0)
      : value
    : [];
  const itemsRef = React.useRef(items);
  itemsRef.current = items;
  const itemLabel = listOptions?.itemLabel || "Item";
  const canAddMore = maxItems === undefined || items.length < maxItems;
  const isApprovalItemsList =
    itemDefinition?.type === "object" &&
    Array.isArray(itemDefinition.schema) &&
    itemDefinition.schema.some((schemaField) => schemaField.name === "type") &&
    itemDefinition.schema.some((schemaField) => ["user", "role", "group"].includes(schemaField.name || ""));

  // Keep Accordion controlled for its full lifetime: use "" as the "closed" value.
  const [openItem, setOpenItem] = React.useState<string>("");
  const rowRefs = React.useRef<Array<HTMLDivElement | null>>([]);
  const {
    dragState,
    renderedItems,
    startDrag: beginDrag,
  } = useListFieldDragReorder({
    items,
    allowReorder,
    useAccordion,
    onChange,
    setOpenItem,
    rowRefs,
  });

  React.useEffect(() => {
    if (items.length === 0) {
      setOpenItem("");
    } else if (openItem !== "" && Number(openItem) >= items.length) {
      setOpenItem("");
    }
  }, [items.length, openItem]);

  const addItem = () => {
    const newItem =
      itemDefinition?.type === "object"
        ? {}
        : itemDefinition?.type === "number"
          ? 0
          : itemDefinition?.type === "day-in-year"
            ? "01/01"
            : "";
    const newItems = [...itemsRef.current, newItem];
    itemsRef.current = newItems;
    onChange(newItems);
    if (useAccordion) {
      setOpenItem(String(newItems.length - 1));
    }
  };

  const removeItem = (index: number) => {
    const newItems = itemsRef.current.filter((_, i) => i !== index);
    itemsRef.current = newItems;
    onChange(newItems.length > 0 ? newItems : undefined);
    if (!useAccordion) return;

    setOpenItem((current) => {
      if (newItems.length === 0 || current === "") return "";
      const openIndex = Number(current);
      if (openIndex === index) return "";
      if (openIndex > index) return String(openIndex - 1);
      return current;
    });
  };

  const updateItem = (index: number, newValue: unknown) => {
    const newItems = [...itemsRef.current];
    newItems[index] = newValue;
    if (isApprovalItemsList) {
      const newKey =
        newValue && typeof newValue === "object" ? getApproverKey(newValue as Record<string, unknown>) : undefined;
      if (newKey) {
        const hasDuplicate = newItems.some((item, itemIndex) => {
          if (itemIndex === index || !item || typeof item !== "object") return false;
          return getApproverKey(item as Record<string, unknown>) === newKey;
        });
        if (hasDuplicate) {
          showErrorToast("Approver already added.");
          return;
        }
      }
    }
    itemsRef.current = newItems;
    onChange(newItems);
  };

  const renderObjectItemFields = (item: unknown, index: number) => {
    if (!itemDefinition?.schema) return null;

    const itemValues =
      item && typeof item === "object" ? (item as Record<string, unknown>) : ({} as Record<string, unknown>);
    const nestedValues = isApprovalItemsList
      ? {
          ...itemValues,
          __listItems: items,
          __itemIndex: index,
          __isApprovalList: true,
        }
      : itemValues;

    return itemDefinition.schema.map((schemaField, schemaIndex) => {
      const nestedFieldPath = `${fieldPath}[${index}].${schemaField.name}`;
      const hasNestedError = (() => {
        if (!validationErrors) return false;

        if (validationErrors instanceof Set) {
          return validationErrors.has(nestedFieldPath);
        }
        return validationErrors.some((error) => error.field === nestedFieldPath);
      })();

      return (
        <ConfigurationFieldRenderer
          allowExpressions={allowExpressions}
          key={schemaField.name ?? `field-${schemaIndex}`}
          field={schemaField}
          value={itemValues[schemaField.name!]}
          onChange={(val) => {
            const newItem = { ...itemValues, [schemaField.name!]: val };
            updateItem(index, newItem);
          }}
          allValues={nestedValues}
          domainId={domainId}
          domainType={domainType}
          integrationId={integrationId}
          organizationId={organizationId}
          hasError={hasNestedError}
          autocompleteExampleObj={autocompleteExampleObj}
        />
      );
    });
  };

  const renderListItemBody = (item: unknown, index: number) => {
    if (itemDefinition?.type === "object" && itemDefinition.schema) {
      return <div className="space-y-4">{renderObjectItemFields(item, index)}</div>;
    }

    if (itemDefinition?.type === "day-in-year") {
      return (
        <DayInYearFieldRenderer
          field={{ name: `${field.name || "item"}-${index}`, label: itemLabel, type: "day-in-year" }}
          value={item}
          onChange={(val) => updateItem(index, val)}
        />
      );
    }

    if (itemDefinition?.type === "repository-file") {
      return (
        <RepositoryFileFieldRenderer
          field={{ name: `${field.name || "item"}-${index}`, label: itemLabel, type: "repository-file" }}
          value={item}
          onChange={(val) => updateItem(index, val)}
        />
      );
    }

    return (
      <Input
        type={itemDefinition?.type === "number" ? "number" : "text"}
        value={(item as string | number) ?? ""}
        onChange={(e) => {
          const val =
            itemDefinition?.type === "number"
              ? e.target.value === ""
                ? undefined
                : Number(e.target.value)
              : e.target.value;
          updateItem(index, val);
        }}
      />
    );
  };

  const renderDragHandle = (index: number, className?: string) => {
    if (!allowReorder || items.length < 2) {
      return null;
    }

    const title = listFieldItemTitle(renderedItems[index], index, itemLabel);
    const isActive = dragState?.activeIndex === index;

    return (
      <button
        type="button"
        aria-label={`Drag to reorder ${title}`}
        className={cn(
          "mt-1 flex h-9 w-8 shrink-0 items-center justify-center rounded-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800",
          dragState ? "cursor-grabbing" : "cursor-grab",
          className,
        )}
        onClick={(event) => event.stopPropagation()}
        onMouseDown={(event) => beginDrag(event, index, openItem)}
      >
        <GripVertical className={cn("h-4 w-4", isActive && "text-gray-900 dark:text-gray-100")} aria-hidden />
      </button>
    );
  };

  const renderRemoveButton = (index: number, className?: string) => (
    <Button
      variant="ghost"
      size="icon-sm"
      onClick={(event) => {
        event.stopPropagation();
        removeItem(index);
      }}
      className={cn("group mt-1 text-gray-500 hover:bg-red-50 hover:text-red-500", className)}
      aria-label={`Remove ${listFieldItemTitle(renderedItems[index], index, itemLabel)}`}
    >
      <Trash2 className="size-4" />
    </Button>
  );

  if (useAccordion && itemDefinition?.type === "object" && itemDefinition.schema) {
    return (
      <div className="space-y-3">
        <Accordion type="single" collapsible value={openItem} onValueChange={setOpenItem} className="space-y-2">
          {renderedItems.map((item, index) => {
            const isItemOpen = openItem === String(index);
            return (
              <AccordionItem key={index} value={String(index)} className="border-0">
                <div
                  ref={(el) => {
                    rowRefs.current[index] = el;
                  }}
                  data-testid="list-item-row"
                  className="flex items-start gap-2"
                >
                  {renderDragHandle(index)}
                  <div className="relative min-w-0 flex-1 rounded-md border border-gray-300 dark:border-gray-700">
                    <AccordionPrimitive.Header
                      className={cn(
                        "relative z-10 flex h-11 shrink-0",
                        isItemOpen && "pointer-events-none absolute inset-x-0 top-0",
                      )}
                    >
                      <AccordionPrimitive.Trigger
                        className={cn(
                          "group relative flex h-11 w-full items-center pl-4 text-left text-sm font-medium hover:no-underline focus-visible:outline-none focus-visible:ring-0",
                          isItemOpen && "pointer-events-auto",
                        )}
                      >
                        <span
                          className={cn(
                            "min-w-0 flex-1 truncate pr-2 font-medium text-gray-800",
                            isItemOpen && "sr-only",
                          )}
                        >
                          {listFieldItemTitle(item, index, itemLabel)}
                        </span>
                        <span className="absolute top-1/2 right-2 flex size-8 -translate-y-1/2 items-center justify-center rounded-sm group-focus-visible:ring-2 group-focus-visible:ring-ring/50">
                          <ChevronDown className={cn("size-4 text-gray-500", isItemOpen && "rotate-180")} />
                        </span>
                      </AccordionPrimitive.Trigger>
                    </AccordionPrimitive.Header>
                    <AccordionContent className={cn("px-4 pb-4 pr-12", isItemOpen ? "pt-4" : "pt-0")}>
                      {renderListItemBody(item, index)}
                    </AccordionContent>
                  </div>
                  {renderRemoveButton(index, "mt-1 shrink-0")}
                </div>
              </AccordionItem>
            );
          })}
        </Accordion>
        <Button variant="outline" size="sm" onClick={addItem} className="mt-3 w-full" disabled={!canAddMore}>
          <Plus />
          Add {itemLabel}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {renderedItems.map((item, index) => {
        return (
          <div
            key={index}
            ref={(el) => {
              rowRefs.current[index] = el;
            }}
            data-testid="list-item-row"
            className="flex items-center gap-2"
          >
            {renderDragHandle(index)}
            <div className="flex-1">
              {itemDefinition?.type === "object" && itemDefinition.schema ? (
                <div className="rounded-md bg-slate-100 p-4 space-y-4">{renderObjectItemFields(item, index)}</div>
              ) : (
                renderListItemBody(item, index)
              )}
            </div>
            {renderRemoveButton(index, "mt-1")}
          </div>
        );
      })}
      <Button variant="outline" size="sm" onClick={addItem} className="mt-3 w-full" disabled={!canAddMore}>
        <Plus />
        Add {itemLabel}
      </Button>
    </div>
  );
};
