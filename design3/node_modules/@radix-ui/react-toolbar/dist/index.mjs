"use client";

// src/toolbar.tsx
import * as React from "react";
import { composeEventHandlers } from "@radix-ui/primitive";
import { createContextScope } from "@radix-ui/react-context";
import * as RovingFocusGroup from "@radix-ui/react-roving-focus";
import { createRovingFocusGroupScope } from "@radix-ui/react-roving-focus";
import { Primitive } from "@radix-ui/react-primitive";
import * as SeparatorPrimitive from "@radix-ui/react-separator";
import * as ToggleGroupPrimitive from "@radix-ui/react-toggle-group";
import { createToggleGroupScope } from "@radix-ui/react-toggle-group";
import { useDirection } from "@radix-ui/react-direction";
import { jsx } from "react/jsx-runtime";
var TOOLBAR_NAME = "Toolbar";
var [createToolbarContext, createToolbarScope] = createContextScope(TOOLBAR_NAME, [
  createRovingFocusGroupScope,
  createToggleGroupScope
]);
var useRovingFocusGroupScope = createRovingFocusGroupScope();
var useToggleGroupScope = createToggleGroupScope();
var [ToolbarProvider, useToolbarContext] = createToolbarContext(TOOLBAR_NAME);
var Toolbar = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, orientation = "horizontal", dir, loop = true, ...toolbarProps } = props;
    const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeToolbar);
    const direction = useDirection(dir);
    return /* @__PURE__ */ jsx(ToolbarProvider, { scope: __scopeToolbar, orientation, dir: direction, children: /* @__PURE__ */ jsx(
      RovingFocusGroup.Root,
      {
        asChild: true,
        ...rovingFocusGroupScope,
        orientation,
        dir: direction,
        loop,
        children: /* @__PURE__ */ jsx(
          Primitive.div,
          {
            role: "toolbar",
            "aria-orientation": orientation,
            dir: direction,
            ...toolbarProps,
            ref: forwardedRef
          }
        )
      }
    ) });
  }
);
Toolbar.displayName = TOOLBAR_NAME;
var SEPARATOR_NAME = "ToolbarSeparator";
var ToolbarSeparator = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, ...separatorProps } = props;
    const context = useToolbarContext(SEPARATOR_NAME, __scopeToolbar);
    return /* @__PURE__ */ jsx(
      SeparatorPrimitive.Root,
      {
        orientation: context.orientation === "horizontal" ? "vertical" : "horizontal",
        ...separatorProps,
        ref: forwardedRef
      }
    );
  }
);
ToolbarSeparator.displayName = SEPARATOR_NAME;
var BUTTON_NAME = "ToolbarButton";
var ToolbarButton = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, ...buttonProps } = props;
    const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeToolbar);
    return /* @__PURE__ */ jsx(RovingFocusGroup.Item, { asChild: true, ...rovingFocusGroupScope, focusable: !props.disabled, children: /* @__PURE__ */ jsx(Primitive.button, { type: "button", ...buttonProps, ref: forwardedRef }) });
  }
);
ToolbarButton.displayName = BUTTON_NAME;
var LINK_NAME = "ToolbarLink";
var ToolbarLink = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, ...linkProps } = props;
    const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeToolbar);
    return /* @__PURE__ */ jsx(RovingFocusGroup.Item, { asChild: true, ...rovingFocusGroupScope, focusable: true, children: /* @__PURE__ */ jsx(
      Primitive.a,
      {
        ...linkProps,
        ref: forwardedRef,
        onKeyDown: composeEventHandlers(props.onKeyDown, (event) => {
          if (event.key === " ") event.currentTarget.click();
        })
      }
    ) });
  }
);
ToolbarLink.displayName = LINK_NAME;
var TOGGLE_GROUP_NAME = "ToolbarToggleGroup";
var ToolbarToggleGroup = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, ...toggleGroupProps } = props;
    const context = useToolbarContext(TOGGLE_GROUP_NAME, __scopeToolbar);
    const toggleGroupScope = useToggleGroupScope(__scopeToolbar);
    return /* @__PURE__ */ jsx(
      ToggleGroupPrimitive.Root,
      {
        "data-orientation": context.orientation,
        dir: context.dir,
        ...toggleGroupScope,
        ...toggleGroupProps,
        ref: forwardedRef,
        rovingFocus: false
      }
    );
  }
);
ToolbarToggleGroup.displayName = TOGGLE_GROUP_NAME;
var TOGGLE_ITEM_NAME = "ToolbarToggleItem";
var ToolbarToggleItem = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, ...toggleItemProps } = props;
    const toggleGroupScope = useToggleGroupScope(__scopeToolbar);
    const scope = { __scopeToolbar: props.__scopeToolbar };
    return /* @__PURE__ */ jsx(ToolbarButton, { asChild: true, ...scope, children: /* @__PURE__ */ jsx(ToggleGroupPrimitive.Item, { ...toggleGroupScope, ...toggleItemProps, ref: forwardedRef }) });
  }
);
ToolbarToggleItem.displayName = TOGGLE_ITEM_NAME;
var Root4 = Toolbar;
var Separator = ToolbarSeparator;
var Button = ToolbarButton;
var Link = ToolbarLink;
var ToggleGroup = ToolbarToggleGroup;
var ToggleItem = ToolbarToggleItem;
export {
  Button,
  Link,
  Root4 as Root,
  Separator,
  ToggleGroup,
  ToggleItem,
  Toolbar,
  ToolbarButton,
  ToolbarLink,
  ToolbarSeparator,
  ToolbarToggleGroup,
  ToolbarToggleItem,
  createToolbarScope
};
//# sourceMappingURL=index.mjs.map
