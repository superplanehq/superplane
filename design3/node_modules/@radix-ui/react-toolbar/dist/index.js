"use strict";
"use client";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// src/index.ts
var index_exports = {};
__export(index_exports, {
  Button: () => Button,
  Link: () => Link,
  Root: () => Root4,
  Separator: () => Separator,
  ToggleGroup: () => ToggleGroup,
  ToggleItem: () => ToggleItem,
  Toolbar: () => Toolbar,
  ToolbarButton: () => ToolbarButton,
  ToolbarLink: () => ToolbarLink,
  ToolbarSeparator: () => ToolbarSeparator,
  ToolbarToggleGroup: () => ToolbarToggleGroup,
  ToolbarToggleItem: () => ToolbarToggleItem,
  createToolbarScope: () => createToolbarScope
});
module.exports = __toCommonJS(index_exports);

// src/toolbar.tsx
var React = __toESM(require("react"));
var import_primitive = require("@radix-ui/primitive");
var import_react_context = require("@radix-ui/react-context");
var RovingFocusGroup = __toESM(require("@radix-ui/react-roving-focus"));
var import_react_roving_focus = require("@radix-ui/react-roving-focus");
var import_react_primitive = require("@radix-ui/react-primitive");
var SeparatorPrimitive = __toESM(require("@radix-ui/react-separator"));
var ToggleGroupPrimitive = __toESM(require("@radix-ui/react-toggle-group"));
var import_react_toggle_group = require("@radix-ui/react-toggle-group");
var import_react_direction = require("@radix-ui/react-direction");
var import_jsx_runtime = require("react/jsx-runtime");
var TOOLBAR_NAME = "Toolbar";
var [createToolbarContext, createToolbarScope] = (0, import_react_context.createContextScope)(TOOLBAR_NAME, [
  import_react_roving_focus.createRovingFocusGroupScope,
  import_react_toggle_group.createToggleGroupScope
]);
var useRovingFocusGroupScope = (0, import_react_roving_focus.createRovingFocusGroupScope)();
var useToggleGroupScope = (0, import_react_toggle_group.createToggleGroupScope)();
var [ToolbarProvider, useToolbarContext] = createToolbarContext(TOOLBAR_NAME);
var Toolbar = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, orientation = "horizontal", dir, loop = true, ...toolbarProps } = props;
    const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeToolbar);
    const direction = (0, import_react_direction.useDirection)(dir);
    return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(ToolbarProvider, { scope: __scopeToolbar, orientation, dir: direction, children: /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
      RovingFocusGroup.Root,
      {
        asChild: true,
        ...rovingFocusGroupScope,
        orientation,
        dir: direction,
        loop,
        children: /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
          import_react_primitive.Primitive.div,
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
    return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
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
    return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(RovingFocusGroup.Item, { asChild: true, ...rovingFocusGroupScope, focusable: !props.disabled, children: /* @__PURE__ */ (0, import_jsx_runtime.jsx)(import_react_primitive.Primitive.button, { type: "button", ...buttonProps, ref: forwardedRef }) });
  }
);
ToolbarButton.displayName = BUTTON_NAME;
var LINK_NAME = "ToolbarLink";
var ToolbarLink = React.forwardRef(
  (props, forwardedRef) => {
    const { __scopeToolbar, ...linkProps } = props;
    const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeToolbar);
    return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(RovingFocusGroup.Item, { asChild: true, ...rovingFocusGroupScope, focusable: true, children: /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
      import_react_primitive.Primitive.a,
      {
        ...linkProps,
        ref: forwardedRef,
        onKeyDown: (0, import_primitive.composeEventHandlers)(props.onKeyDown, (event) => {
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
    return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
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
    return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(ToolbarButton, { asChild: true, ...scope, children: /* @__PURE__ */ (0, import_jsx_runtime.jsx)(ToggleGroupPrimitive.Item, { ...toggleGroupScope, ...toggleItemProps, ref: forwardedRef }) });
  }
);
ToolbarToggleItem.displayName = TOGGLE_ITEM_NAME;
var Root4 = Toolbar;
var Separator = ToolbarSeparator;
var Button = ToolbarButton;
var Link = ToolbarLink;
var ToggleGroup = ToolbarToggleGroup;
var ToggleItem = ToolbarToggleItem;
//# sourceMappingURL=index.js.map
