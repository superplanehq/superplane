var $cg2C9$babelruntimehelpersextends = require("@babel/runtime/helpers/extends");
var $cg2C9$react = require("react");
var $cg2C9$reactdom = require("react-dom");
var $cg2C9$radixuinumber = require("@radix-ui/number");
var $cg2C9$radixuiprimitive = require("@radix-ui/primitive");
var $cg2C9$radixuireactcollection = require("@radix-ui/react-collection");
var $cg2C9$radixuireactcomposerefs = require("@radix-ui/react-compose-refs");
var $cg2C9$radixuireactcontext = require("@radix-ui/react-context");
var $cg2C9$radixuireactdirection = require("@radix-ui/react-direction");
var $cg2C9$radixuireactdismissablelayer = require("@radix-ui/react-dismissable-layer");
var $cg2C9$radixuireactfocusguards = require("@radix-ui/react-focus-guards");
var $cg2C9$radixuireactfocusscope = require("@radix-ui/react-focus-scope");
var $cg2C9$radixuireactid = require("@radix-ui/react-id");
var $cg2C9$radixuireactpopper = require("@radix-ui/react-popper");
var $cg2C9$radixuireactportal = require("@radix-ui/react-portal");
var $cg2C9$radixuireactprimitive = require("@radix-ui/react-primitive");
var $cg2C9$radixuireactslot = require("@radix-ui/react-slot");
var $cg2C9$radixuireactusecallbackref = require("@radix-ui/react-use-callback-ref");
var $cg2C9$radixuireactusecontrollablestate = require("@radix-ui/react-use-controllable-state");
var $cg2C9$radixuireactuselayouteffect = require("@radix-ui/react-use-layout-effect");
var $cg2C9$radixuireactuseprevious = require("@radix-ui/react-use-previous");
var $cg2C9$radixuireactvisuallyhidden = require("@radix-ui/react-visually-hidden");
var $cg2C9$ariahidden = require("aria-hidden");
var $cg2C9$reactremovescroll = require("react-remove-scroll");

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}
function $parcel$interopDefault(a) {
  return a && a.__esModule ? a.default : a;
}

$parcel$export(module.exports, "createSelectScope", () => $1345bda09ffc1bc6$export$286727a75dc039bd);
$parcel$export(module.exports, "Select", () => $1345bda09ffc1bc6$export$ef9b1a59e592288f);
$parcel$export(module.exports, "SelectTrigger", () => $1345bda09ffc1bc6$export$3ac1e88a1c0b9f1);
$parcel$export(module.exports, "SelectValue", () => $1345bda09ffc1bc6$export$e288731fd71264f0);
$parcel$export(module.exports, "SelectIcon", () => $1345bda09ffc1bc6$export$99b400cabb58c515);
$parcel$export(module.exports, "SelectPortal", () => $1345bda09ffc1bc6$export$b2af6c9944296213);
$parcel$export(module.exports, "SelectContent", () => $1345bda09ffc1bc6$export$c973a4b3cb86a03d);
$parcel$export(module.exports, "SelectViewport", () => $1345bda09ffc1bc6$export$9ed6e7b40248d36d);
$parcel$export(module.exports, "SelectGroup", () => $1345bda09ffc1bc6$export$ee25a334c55de1f4);
$parcel$export(module.exports, "SelectLabel", () => $1345bda09ffc1bc6$export$f67338d29bd972f8);
$parcel$export(module.exports, "SelectItem", () => $1345bda09ffc1bc6$export$13ef48a934230896);
$parcel$export(module.exports, "SelectItemText", () => $1345bda09ffc1bc6$export$3572fb0fb821ff49);
$parcel$export(module.exports, "SelectItemIndicator", () => $1345bda09ffc1bc6$export$6b9198de19accfe6);
$parcel$export(module.exports, "SelectScrollUpButton", () => $1345bda09ffc1bc6$export$d8117927658af6d7);
$parcel$export(module.exports, "SelectScrollDownButton", () => $1345bda09ffc1bc6$export$ff951e476c12189);
$parcel$export(module.exports, "SelectSeparator", () => $1345bda09ffc1bc6$export$eba4b1df07cb1d3);
$parcel$export(module.exports, "SelectArrow", () => $1345bda09ffc1bc6$export$314f4cb8f8099628);
$parcel$export(module.exports, "Root", () => $1345bda09ffc1bc6$export$be92b6f5f03c0fe9);
$parcel$export(module.exports, "Trigger", () => $1345bda09ffc1bc6$export$41fb9f06171c75f4);
$parcel$export(module.exports, "Value", () => $1345bda09ffc1bc6$export$4c8d1a57a761ef94);
$parcel$export(module.exports, "Icon", () => $1345bda09ffc1bc6$export$f04a61298a47a40f);
$parcel$export(module.exports, "Portal", () => $1345bda09ffc1bc6$export$602eac185826482c);
$parcel$export(module.exports, "Content", () => $1345bda09ffc1bc6$export$7c6e2c02157bb7d2);
$parcel$export(module.exports, "Viewport", () => $1345bda09ffc1bc6$export$d5c6c08dc2d3ca7);
$parcel$export(module.exports, "Group", () => $1345bda09ffc1bc6$export$eb2fcfdbd7ba97d4);
$parcel$export(module.exports, "Label", () => $1345bda09ffc1bc6$export$b04be29aa201d4f5);
$parcel$export(module.exports, "Item", () => $1345bda09ffc1bc6$export$6d08773d2e66f8f2);
$parcel$export(module.exports, "ItemText", () => $1345bda09ffc1bc6$export$d6e5bf9c43ea9319);
$parcel$export(module.exports, "ItemIndicator", () => $1345bda09ffc1bc6$export$c3468e2714d175fa);
$parcel$export(module.exports, "ScrollUpButton", () => $1345bda09ffc1bc6$export$2f60d3ec9ad468f2);
$parcel$export(module.exports, "ScrollDownButton", () => $1345bda09ffc1bc6$export$bf1aedc3039c8d63);
$parcel$export(module.exports, "Separator", () => $1345bda09ffc1bc6$export$1ff3c3f08ae963c0);
$parcel$export(module.exports, "Arrow", () => $1345bda09ffc1bc6$export$21b07c8f274aebd5);

























const $1345bda09ffc1bc6$var$OPEN_KEYS = [
    ' ',
    'Enter',
    'ArrowUp',
    'ArrowDown'
];
const $1345bda09ffc1bc6$var$SELECTION_KEYS = [
    ' ',
    'Enter'
];
/* -------------------------------------------------------------------------------------------------
 * Select
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$SELECT_NAME = 'Select';
const [$1345bda09ffc1bc6$var$Collection, $1345bda09ffc1bc6$var$useCollection, $1345bda09ffc1bc6$var$createCollectionScope] = $cg2C9$radixuireactcollection.createCollection($1345bda09ffc1bc6$var$SELECT_NAME);
const [$1345bda09ffc1bc6$var$createSelectContext, $1345bda09ffc1bc6$export$286727a75dc039bd] = $cg2C9$radixuireactcontext.createContextScope($1345bda09ffc1bc6$var$SELECT_NAME, [
    $1345bda09ffc1bc6$var$createCollectionScope,
    $cg2C9$radixuireactpopper.createPopperScope
]);
const $1345bda09ffc1bc6$var$usePopperScope = $cg2C9$radixuireactpopper.createPopperScope();
const [$1345bda09ffc1bc6$var$SelectProvider, $1345bda09ffc1bc6$var$useSelectContext] = $1345bda09ffc1bc6$var$createSelectContext($1345bda09ffc1bc6$var$SELECT_NAME);
const [$1345bda09ffc1bc6$var$SelectNativeOptionsProvider, $1345bda09ffc1bc6$var$useSelectNativeOptionsContext] = $1345bda09ffc1bc6$var$createSelectContext($1345bda09ffc1bc6$var$SELECT_NAME);
const $1345bda09ffc1bc6$export$ef9b1a59e592288f = (props)=>{
    const { __scopeSelect: __scopeSelect , children: children , open: openProp , defaultOpen: defaultOpen , onOpenChange: onOpenChange , value: valueProp , defaultValue: defaultValue , onValueChange: onValueChange , dir: dir , name: name , autoComplete: autoComplete , disabled: disabled , required: required  } = props;
    const popperScope = $1345bda09ffc1bc6$var$usePopperScope(__scopeSelect);
    const [trigger, setTrigger] = $cg2C9$react.useState(null);
    const [valueNode, setValueNode] = $cg2C9$react.useState(null);
    const [valueNodeHasChildren, setValueNodeHasChildren] = $cg2C9$react.useState(false);
    const direction = $cg2C9$radixuireactdirection.useDirection(dir);
    const [open = false, setOpen] = $cg2C9$radixuireactusecontrollablestate.useControllableState({
        prop: openProp,
        defaultProp: defaultOpen,
        onChange: onOpenChange
    });
    const [value, setValue] = $cg2C9$radixuireactusecontrollablestate.useControllableState({
        prop: valueProp,
        defaultProp: defaultValue,
        onChange: onValueChange
    });
    const triggerPointerDownPosRef = $cg2C9$react.useRef(null); // We set this to true by default so that events bubble to forms without JS (SSR)
    const isFormControl = trigger ? Boolean(trigger.closest('form')) : true;
    const [nativeOptionsSet, setNativeOptionsSet] = $cg2C9$react.useState(new Set()); // The native `select` only associates the correct default value if the corresponding
    // `option` is rendered as a child **at the same time** as itself.
    // Because it might take a few renders for our items to gather the information to build
    // the native `option`(s), we generate a key on the `select` to make sure React re-builds it
    // each time the options change.
    const nativeSelectKey = Array.from(nativeOptionsSet).map((option)=>option.props.value
    ).join(';');
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactpopper.Root, popperScope, /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectProvider, {
        required: required,
        scope: __scopeSelect,
        trigger: trigger,
        onTriggerChange: setTrigger,
        valueNode: valueNode,
        onValueNodeChange: setValueNode,
        valueNodeHasChildren: valueNodeHasChildren,
        onValueNodeHasChildrenChange: setValueNodeHasChildren,
        contentId: $cg2C9$radixuireactid.useId(),
        value: value,
        onValueChange: setValue,
        open: open,
        onOpenChange: setOpen,
        dir: direction,
        triggerPointerDownPosRef: triggerPointerDownPosRef,
        disabled: disabled
    }, /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$Collection.Provider, {
        scope: __scopeSelect
    }, /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectNativeOptionsProvider, {
        scope: props.__scopeSelect,
        onNativeOptionAdd: $cg2C9$react.useCallback((option)=>{
            setNativeOptionsSet((prev)=>new Set(prev).add(option)
            );
        }, []),
        onNativeOptionRemove: $cg2C9$react.useCallback((option)=>{
            setNativeOptionsSet((prev)=>{
                const optionsSet = new Set(prev);
                optionsSet.delete(option);
                return optionsSet;
            });
        }, [])
    }, children)), isFormControl ? /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$BubbleSelect, {
        key: nativeSelectKey,
        "aria-hidden": true,
        required: required,
        tabIndex: -1,
        name: name,
        autoComplete: autoComplete,
        value: value // enable form autofill
        ,
        onChange: (event)=>setValue(event.target.value)
        ,
        disabled: disabled
    }, value === undefined ? /*#__PURE__*/ $cg2C9$react.createElement("option", {
        value: ""
    }) : null, Array.from(nativeOptionsSet)) : null));
};
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$ef9b1a59e592288f, {
    displayName: $1345bda09ffc1bc6$var$SELECT_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectTrigger
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$TRIGGER_NAME = 'SelectTrigger';
const $1345bda09ffc1bc6$export$3ac1e88a1c0b9f1 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , disabled: disabled = false , ...triggerProps } = props;
    const popperScope = $1345bda09ffc1bc6$var$usePopperScope(__scopeSelect);
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$TRIGGER_NAME, __scopeSelect);
    const isDisabled = context.disabled || disabled;
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, context.onTriggerChange);
    const getItems = $1345bda09ffc1bc6$var$useCollection(__scopeSelect);
    const [searchRef, handleTypeaheadSearch, resetTypeahead] = $1345bda09ffc1bc6$var$useTypeaheadSearch((search)=>{
        const enabledItems = getItems().filter((item)=>!item.disabled
        );
        const currentItem = enabledItems.find((item)=>item.value === context.value
        );
        const nextItem = $1345bda09ffc1bc6$var$findNextItem(enabledItems, search, currentItem);
        if (nextItem !== undefined) context.onValueChange(nextItem.value);
    });
    const handleOpen = ()=>{
        if (!isDisabled) {
            context.onOpenChange(true); // reset typeahead when we open
            resetTypeahead();
        }
    };
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactpopper.Anchor, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        asChild: true
    }, popperScope), /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.button, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        type: "button",
        role: "combobox",
        "aria-controls": context.contentId,
        "aria-expanded": context.open,
        "aria-required": context.required,
        "aria-autocomplete": "none",
        dir: context.dir,
        "data-state": context.open ? 'open' : 'closed',
        disabled: isDisabled,
        "data-disabled": isDisabled ? '' : undefined,
        "data-placeholder": context.value === undefined ? '' : undefined
    }, triggerProps, {
        ref: composedRefs // Enable compatibility with native label or custom `Label` "click" for Safari:
        ,
        onClick: $cg2C9$radixuiprimitive.composeEventHandlers(triggerProps.onClick, (event)=>{
            // Whilst browsers generally have no issue focusing the trigger when clicking
            // on a label, Safari seems to struggle with the fact that there's no `onClick`.
            // We force `focus` in this case. Note: this doesn't create any other side-effect
            // because we are preventing default in `onPointerDown` so effectively
            // this only runs for a label "click"
            event.currentTarget.focus();
        }),
        onPointerDown: $cg2C9$radixuiprimitive.composeEventHandlers(triggerProps.onPointerDown, (event)=>{
            // prevent implicit pointer capture
            // https://www.w3.org/TR/pointerevents3/#implicit-pointer-capture
            const target = event.target;
            if (target.hasPointerCapture(event.pointerId)) target.releasePointerCapture(event.pointerId);
             // only call handler if it's the left button (mousedown gets triggered by all mouse buttons)
            // but not when the control key is pressed (avoiding MacOS right click)
            if (event.button === 0 && event.ctrlKey === false) {
                handleOpen();
                context.triggerPointerDownPosRef.current = {
                    x: Math.round(event.pageX),
                    y: Math.round(event.pageY)
                }; // prevent trigger from stealing focus from the active item after opening.
                event.preventDefault();
            }
        }),
        onKeyDown: $cg2C9$radixuiprimitive.composeEventHandlers(triggerProps.onKeyDown, (event)=>{
            const isTypingAhead = searchRef.current !== '';
            const isModifierKey = event.ctrlKey || event.altKey || event.metaKey;
            if (!isModifierKey && event.key.length === 1) handleTypeaheadSearch(event.key);
            if (isTypingAhead && event.key === ' ') return;
            if ($1345bda09ffc1bc6$var$OPEN_KEYS.includes(event.key)) {
                handleOpen();
                event.preventDefault();
            }
        })
    })));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$3ac1e88a1c0b9f1, {
    displayName: $1345bda09ffc1bc6$var$TRIGGER_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectValue
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$VALUE_NAME = 'SelectValue';
const $1345bda09ffc1bc6$export$e288731fd71264f0 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    // We ignore `className` and `style` as this part shouldn't be styled.
    const { __scopeSelect: __scopeSelect , className: className , style: style , children: children , placeholder: placeholder , ...valueProps } = props;
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$VALUE_NAME, __scopeSelect);
    const { onValueNodeHasChildrenChange: onValueNodeHasChildrenChange  } = context;
    const hasChildren = children !== undefined;
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, context.onValueNodeChange);
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        onValueNodeHasChildrenChange(hasChildren);
    }, [
        onValueNodeHasChildrenChange,
        hasChildren
    ]);
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.span, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, valueProps, {
        ref: composedRefs // we don't want events from the portalled `SelectValue` children to bubble
        ,
        style: {
            pointerEvents: 'none'
        }
    }), context.value === undefined && placeholder !== undefined ? placeholder : children);
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$e288731fd71264f0, {
    displayName: $1345bda09ffc1bc6$var$VALUE_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectIcon
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$ICON_NAME = 'SelectIcon';
const $1345bda09ffc1bc6$export$99b400cabb58c515 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , children: children , ...iconProps } = props;
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.span, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        "aria-hidden": true
    }, iconProps, {
        ref: forwardedRef
    }), children || '▼');
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$99b400cabb58c515, {
    displayName: $1345bda09ffc1bc6$var$ICON_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectPortal
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$PORTAL_NAME = 'SelectPortal';
const $1345bda09ffc1bc6$export$b2af6c9944296213 = (props)=>{
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactportal.Portal, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        asChild: true
    }, props));
};
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$b2af6c9944296213, {
    displayName: $1345bda09ffc1bc6$var$PORTAL_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectContent
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$CONTENT_NAME = 'SelectContent';
const $1345bda09ffc1bc6$export$c973a4b3cb86a03d = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$CONTENT_NAME, props.__scopeSelect);
    const [fragment, setFragment] = $cg2C9$react.useState(); // setting the fragment in `useLayoutEffect` as `DocumentFragment` doesn't exist on the server
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        setFragment(new DocumentFragment());
    }, []);
    if (!context.open) {
        const frag = fragment;
        return frag ? /*#__PURE__*/ $cg2C9$reactdom.createPortal(/*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectContentProvider, {
            scope: props.__scopeSelect
        }, /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$Collection.Slot, {
            scope: props.__scopeSelect
        }, /*#__PURE__*/ $cg2C9$react.createElement("div", null, props.children))), frag) : null;
    }
    return /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectContentImpl, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, props, {
        ref: forwardedRef
    }));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$c973a4b3cb86a03d, {
    displayName: $1345bda09ffc1bc6$var$CONTENT_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectContentImpl
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$CONTENT_MARGIN = 10;
const [$1345bda09ffc1bc6$var$SelectContentProvider, $1345bda09ffc1bc6$var$useSelectContentContext] = $1345bda09ffc1bc6$var$createSelectContext($1345bda09ffc1bc6$var$CONTENT_NAME);
const $1345bda09ffc1bc6$var$CONTENT_IMPL_NAME = 'SelectContentImpl';
const $1345bda09ffc1bc6$var$SelectContentImpl = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , position: position = 'item-aligned' , onCloseAutoFocus: onCloseAutoFocus , onEscapeKeyDown: onEscapeKeyDown , onPointerDownOutside: onPointerDownOutside , side: //
    // PopperContent props
    side , sideOffset: sideOffset , align: align , alignOffset: alignOffset , arrowPadding: arrowPadding , collisionBoundary: collisionBoundary , collisionPadding: collisionPadding , sticky: sticky , hideWhenDetached: hideWhenDetached , avoidCollisions: avoidCollisions , //
    ...contentProps } = props;
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$CONTENT_NAME, __scopeSelect);
    const [content, setContent] = $cg2C9$react.useState(null);
    const [viewport, setViewport] = $cg2C9$react.useState(null);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, (node)=>setContent(node)
    );
    const [selectedItem, setSelectedItem] = $cg2C9$react.useState(null);
    const [selectedItemText, setSelectedItemText] = $cg2C9$react.useState(null);
    const getItems = $1345bda09ffc1bc6$var$useCollection(__scopeSelect);
    const [isPositioned, setIsPositioned] = $cg2C9$react.useState(false);
    const firstValidItemFoundRef = $cg2C9$react.useRef(false); // aria-hide everything except the content (better supported equivalent to setting aria-modal)
    $cg2C9$react.useEffect(()=>{
        if (content) return $cg2C9$ariahidden.hideOthers(content);
    }, [
        content
    ]); // Make sure the whole tree has focus guards as our `Select` may be
    // the last element in the DOM (because of the `Portal`)
    $cg2C9$radixuireactfocusguards.useFocusGuards();
    const focusFirst = $cg2C9$react.useCallback((candidates)=>{
        const [firstItem, ...restItems] = getItems().map((item)=>item.ref.current
        );
        const [lastItem] = restItems.slice(-1);
        const PREVIOUSLY_FOCUSED_ELEMENT = document.activeElement;
        for (const candidate of candidates){
            // if focus is already where we want to go, we don't want to keep going through the candidates
            if (candidate === PREVIOUSLY_FOCUSED_ELEMENT) return;
            candidate === null || candidate === void 0 || candidate.scrollIntoView({
                block: 'nearest'
            }); // viewport might have padding so scroll to its edges when focusing first/last items.
            if (candidate === firstItem && viewport) viewport.scrollTop = 0;
            if (candidate === lastItem && viewport) viewport.scrollTop = viewport.scrollHeight;
            candidate === null || candidate === void 0 || candidate.focus();
            if (document.activeElement !== PREVIOUSLY_FOCUSED_ELEMENT) return;
        }
    }, [
        getItems,
        viewport
    ]);
    const focusSelectedItem = $cg2C9$react.useCallback(()=>focusFirst([
            selectedItem,
            content
        ])
    , [
        focusFirst,
        selectedItem,
        content
    ]); // Since this is not dependent on layout, we want to ensure this runs at the same time as
    // other effects across components. Hence why we don't call `focusSelectedItem` inside `position`.
    $cg2C9$react.useEffect(()=>{
        if (isPositioned) focusSelectedItem();
    }, [
        isPositioned,
        focusSelectedItem
    ]); // prevent selecting items on `pointerup` in some cases after opening from `pointerdown`
    // and close on `pointerup` outside.
    const { onOpenChange: onOpenChange , triggerPointerDownPosRef: triggerPointerDownPosRef  } = context;
    $cg2C9$react.useEffect(()=>{
        if (content) {
            let pointerMoveDelta = {
                x: 0,
                y: 0
            };
            const handlePointerMove = (event)=>{
                var _triggerPointerDownPo, _triggerPointerDownPo2, _triggerPointerDownPo3, _triggerPointerDownPo4;
                pointerMoveDelta = {
                    x: Math.abs(Math.round(event.pageX) - ((_triggerPointerDownPo = (_triggerPointerDownPo2 = triggerPointerDownPosRef.current) === null || _triggerPointerDownPo2 === void 0 ? void 0 : _triggerPointerDownPo2.x) !== null && _triggerPointerDownPo !== void 0 ? _triggerPointerDownPo : 0)),
                    y: Math.abs(Math.round(event.pageY) - ((_triggerPointerDownPo3 = (_triggerPointerDownPo4 = triggerPointerDownPosRef.current) === null || _triggerPointerDownPo4 === void 0 ? void 0 : _triggerPointerDownPo4.y) !== null && _triggerPointerDownPo3 !== void 0 ? _triggerPointerDownPo3 : 0))
                };
            };
            const handlePointerUp = (event)=>{
                // If the pointer hasn't moved by a certain threshold then we prevent selecting item on `pointerup`.
                if (pointerMoveDelta.x <= 10 && pointerMoveDelta.y <= 10) event.preventDefault();
                else // otherwise, if the event was outside the content, close.
                if (!content.contains(event.target)) onOpenChange(false);
                document.removeEventListener('pointermove', handlePointerMove);
                triggerPointerDownPosRef.current = null;
            };
            if (triggerPointerDownPosRef.current !== null) {
                document.addEventListener('pointermove', handlePointerMove);
                document.addEventListener('pointerup', handlePointerUp, {
                    capture: true,
                    once: true
                });
            }
            return ()=>{
                document.removeEventListener('pointermove', handlePointerMove);
                document.removeEventListener('pointerup', handlePointerUp, {
                    capture: true
                });
            };
        }
    }, [
        content,
        onOpenChange,
        triggerPointerDownPosRef
    ]);
    $cg2C9$react.useEffect(()=>{
        const close = ()=>onOpenChange(false)
        ;
        window.addEventListener('blur', close);
        window.addEventListener('resize', close);
        return ()=>{
            window.removeEventListener('blur', close);
            window.removeEventListener('resize', close);
        };
    }, [
        onOpenChange
    ]);
    const [searchRef, handleTypeaheadSearch] = $1345bda09ffc1bc6$var$useTypeaheadSearch((search)=>{
        const enabledItems = getItems().filter((item)=>!item.disabled
        );
        const currentItem = enabledItems.find((item)=>item.ref.current === document.activeElement
        );
        const nextItem = $1345bda09ffc1bc6$var$findNextItem(enabledItems, search, currentItem);
        if (nextItem) /**
       * Imperative focus during keydown is risky so we prevent React's batching updates
       * to avoid potential bugs. See: https://github.com/facebook/react/issues/20332
       */ setTimeout(()=>nextItem.ref.current.focus()
        );
    });
    const itemRefCallback = $cg2C9$react.useCallback((node, value, disabled)=>{
        const isFirstValidItem = !firstValidItemFoundRef.current && !disabled;
        const isSelectedItem = context.value !== undefined && context.value === value;
        if (isSelectedItem || isFirstValidItem) {
            setSelectedItem(node);
            if (isFirstValidItem) firstValidItemFoundRef.current = true;
        }
    }, [
        context.value
    ]);
    const handleItemLeave = $cg2C9$react.useCallback(()=>content === null || content === void 0 ? void 0 : content.focus()
    , [
        content
    ]);
    const itemTextRefCallback = $cg2C9$react.useCallback((node, value, disabled)=>{
        const isFirstValidItem = !firstValidItemFoundRef.current && !disabled;
        const isSelectedItem = context.value !== undefined && context.value === value;
        if (isSelectedItem || isFirstValidItem) setSelectedItemText(node);
    }, [
        context.value
    ]);
    const SelectPosition = position === 'popper' ? $1345bda09ffc1bc6$var$SelectPopperPosition : $1345bda09ffc1bc6$var$SelectItemAlignedPosition; // Silently ignore props that are not supported by `SelectItemAlignedPosition`
    const popperContentProps = SelectPosition === $1345bda09ffc1bc6$var$SelectPopperPosition ? {
        side: side,
        sideOffset: sideOffset,
        align: align,
        alignOffset: alignOffset,
        arrowPadding: arrowPadding,
        collisionBoundary: collisionBoundary,
        collisionPadding: collisionPadding,
        sticky: sticky,
        hideWhenDetached: hideWhenDetached,
        avoidCollisions: avoidCollisions
    } : {};
    return /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectContentProvider, {
        scope: __scopeSelect,
        content: content,
        viewport: viewport,
        onViewportChange: setViewport,
        itemRefCallback: itemRefCallback,
        selectedItem: selectedItem,
        onItemLeave: handleItemLeave,
        itemTextRefCallback: itemTextRefCallback,
        focusSelectedItem: focusSelectedItem,
        selectedItemText: selectedItemText,
        position: position,
        isPositioned: isPositioned,
        searchRef: searchRef
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$reactremovescroll.RemoveScroll, {
        as: $cg2C9$radixuireactslot.Slot,
        allowPinchZoom: true
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactfocusscope.FocusScope, {
        asChild: true // we make sure we're not trapping once it's been closed
        ,
        trapped: context.open,
        onMountAutoFocus: (event)=>{
            // we prevent open autofocus because we manually focus the selected item
            event.preventDefault();
        },
        onUnmountAutoFocus: $cg2C9$radixuiprimitive.composeEventHandlers(onCloseAutoFocus, (event)=>{
            var _context$trigger;
            (_context$trigger = context.trigger) === null || _context$trigger === void 0 || _context$trigger.focus({
                preventScroll: true
            });
            event.preventDefault();
        })
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactdismissablelayer.DismissableLayer, {
        asChild: true,
        disableOutsidePointerEvents: true,
        onEscapeKeyDown: onEscapeKeyDown,
        onPointerDownOutside: onPointerDownOutside // When focus is trapped, a focusout event may still happen.
        ,
        onFocusOutside: (event)=>event.preventDefault()
        ,
        onDismiss: ()=>context.onOpenChange(false)
    }, /*#__PURE__*/ $cg2C9$react.createElement(SelectPosition, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        role: "listbox",
        id: context.contentId,
        "data-state": context.open ? 'open' : 'closed',
        dir: context.dir,
        onContextMenu: (event)=>event.preventDefault()
    }, contentProps, popperContentProps, {
        onPlaced: ()=>setIsPositioned(true)
        ,
        ref: composedRefs,
        style: {
            // flex layout so we can place the scroll buttons properly
            display: 'flex',
            flexDirection: 'column',
            // reset the outline by default as the content MAY get focused
            outline: 'none',
            ...contentProps.style
        },
        onKeyDown: $cg2C9$radixuiprimitive.composeEventHandlers(contentProps.onKeyDown, (event)=>{
            const isModifierKey = event.ctrlKey || event.altKey || event.metaKey; // select should not be navigated using tab key so we prevent it
            if (event.key === 'Tab') event.preventDefault();
            if (!isModifierKey && event.key.length === 1) handleTypeaheadSearch(event.key);
            if ([
                'ArrowUp',
                'ArrowDown',
                'Home',
                'End'
            ].includes(event.key)) {
                const items = getItems().filter((item)=>!item.disabled
                );
                let candidateNodes = items.map((item)=>item.ref.current
                );
                if ([
                    'ArrowUp',
                    'End'
                ].includes(event.key)) candidateNodes = candidateNodes.slice().reverse();
                if ([
                    'ArrowUp',
                    'ArrowDown'
                ].includes(event.key)) {
                    const currentElement = event.target;
                    const currentIndex = candidateNodes.indexOf(currentElement);
                    candidateNodes = candidateNodes.slice(currentIndex + 1);
                }
                /**
         * Imperative focus during keydown is risky so we prevent React's batching updates
         * to avoid potential bugs. See: https://github.com/facebook/react/issues/20332
         */ setTimeout(()=>focusFirst(candidateNodes)
                );
                event.preventDefault();
            }
        })
    }))))));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$var$SelectContentImpl, {
    displayName: $1345bda09ffc1bc6$var$CONTENT_IMPL_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectItemAlignedPosition
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$ITEM_ALIGNED_POSITION_NAME = 'SelectItemAlignedPosition';
const $1345bda09ffc1bc6$var$SelectItemAlignedPosition = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , onPlaced: onPlaced , ...popperProps } = props;
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$CONTENT_NAME, __scopeSelect);
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$CONTENT_NAME, __scopeSelect);
    const [contentWrapper, setContentWrapper] = $cg2C9$react.useState(null);
    const [content, setContent] = $cg2C9$react.useState(null);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, (node)=>setContent(node)
    );
    const getItems = $1345bda09ffc1bc6$var$useCollection(__scopeSelect);
    const shouldExpandOnScrollRef = $cg2C9$react.useRef(false);
    const shouldRepositionRef = $cg2C9$react.useRef(true);
    const { viewport: viewport , selectedItem: selectedItem , selectedItemText: selectedItemText , focusSelectedItem: focusSelectedItem  } = contentContext;
    const position = $cg2C9$react.useCallback(()=>{
        if (context.trigger && context.valueNode && contentWrapper && content && viewport && selectedItem && selectedItemText) {
            const triggerRect = context.trigger.getBoundingClientRect(); // -----------------------------------------------------------------------------------------
            //  Horizontal positioning
            // -----------------------------------------------------------------------------------------
            const contentRect = content.getBoundingClientRect();
            const valueNodeRect = context.valueNode.getBoundingClientRect();
            const itemTextRect = selectedItemText.getBoundingClientRect();
            if (context.dir !== 'rtl') {
                const itemTextOffset = itemTextRect.left - contentRect.left;
                const left = valueNodeRect.left - itemTextOffset;
                const leftDelta = triggerRect.left - left;
                const minContentWidth = triggerRect.width + leftDelta;
                const contentWidth = Math.max(minContentWidth, contentRect.width);
                const rightEdge = window.innerWidth - $1345bda09ffc1bc6$var$CONTENT_MARGIN;
                const clampedLeft = $cg2C9$radixuinumber.clamp(left, [
                    $1345bda09ffc1bc6$var$CONTENT_MARGIN,
                    rightEdge - contentWidth
                ]);
                contentWrapper.style.minWidth = minContentWidth + 'px';
                contentWrapper.style.left = clampedLeft + 'px';
            } else {
                const itemTextOffset = contentRect.right - itemTextRect.right;
                const right = window.innerWidth - valueNodeRect.right - itemTextOffset;
                const rightDelta = window.innerWidth - triggerRect.right - right;
                const minContentWidth = triggerRect.width + rightDelta;
                const contentWidth = Math.max(minContentWidth, contentRect.width);
                const leftEdge = window.innerWidth - $1345bda09ffc1bc6$var$CONTENT_MARGIN;
                const clampedRight = $cg2C9$radixuinumber.clamp(right, [
                    $1345bda09ffc1bc6$var$CONTENT_MARGIN,
                    leftEdge - contentWidth
                ]);
                contentWrapper.style.minWidth = minContentWidth + 'px';
                contentWrapper.style.right = clampedRight + 'px';
            } // -----------------------------------------------------------------------------------------
            // Vertical positioning
            // -----------------------------------------------------------------------------------------
            const items = getItems();
            const availableHeight = window.innerHeight - $1345bda09ffc1bc6$var$CONTENT_MARGIN * 2;
            const itemsHeight = viewport.scrollHeight;
            const contentStyles = window.getComputedStyle(content);
            const contentBorderTopWidth = parseInt(contentStyles.borderTopWidth, 10);
            const contentPaddingTop = parseInt(contentStyles.paddingTop, 10);
            const contentBorderBottomWidth = parseInt(contentStyles.borderBottomWidth, 10);
            const contentPaddingBottom = parseInt(contentStyles.paddingBottom, 10);
            const fullContentHeight = contentBorderTopWidth + contentPaddingTop + itemsHeight + contentPaddingBottom + contentBorderBottomWidth; // prettier-ignore
            const minContentHeight = Math.min(selectedItem.offsetHeight * 5, fullContentHeight);
            const viewportStyles = window.getComputedStyle(viewport);
            const viewportPaddingTop = parseInt(viewportStyles.paddingTop, 10);
            const viewportPaddingBottom = parseInt(viewportStyles.paddingBottom, 10);
            const topEdgeToTriggerMiddle = triggerRect.top + triggerRect.height / 2 - $1345bda09ffc1bc6$var$CONTENT_MARGIN;
            const triggerMiddleToBottomEdge = availableHeight - topEdgeToTriggerMiddle;
            const selectedItemHalfHeight = selectedItem.offsetHeight / 2;
            const itemOffsetMiddle = selectedItem.offsetTop + selectedItemHalfHeight;
            const contentTopToItemMiddle = contentBorderTopWidth + contentPaddingTop + itemOffsetMiddle;
            const itemMiddleToContentBottom = fullContentHeight - contentTopToItemMiddle;
            const willAlignWithoutTopOverflow = contentTopToItemMiddle <= topEdgeToTriggerMiddle;
            if (willAlignWithoutTopOverflow) {
                const isLastItem = selectedItem === items[items.length - 1].ref.current;
                contentWrapper.style.bottom = "0px";
                const viewportOffsetBottom = content.clientHeight - viewport.offsetTop - viewport.offsetHeight;
                const clampedTriggerMiddleToBottomEdge = Math.max(triggerMiddleToBottomEdge, selectedItemHalfHeight + (isLastItem ? viewportPaddingBottom : 0) + viewportOffsetBottom + contentBorderBottomWidth);
                const height = contentTopToItemMiddle + clampedTriggerMiddleToBottomEdge;
                contentWrapper.style.height = height + 'px';
            } else {
                const isFirstItem = selectedItem === items[0].ref.current;
                contentWrapper.style.top = "0px";
                const clampedTopEdgeToTriggerMiddle = Math.max(topEdgeToTriggerMiddle, contentBorderTopWidth + viewport.offsetTop + (isFirstItem ? viewportPaddingTop : 0) + selectedItemHalfHeight);
                const height = clampedTopEdgeToTriggerMiddle + itemMiddleToContentBottom;
                contentWrapper.style.height = height + 'px';
                viewport.scrollTop = contentTopToItemMiddle - topEdgeToTriggerMiddle + viewport.offsetTop;
            }
            contentWrapper.style.margin = `${$1345bda09ffc1bc6$var$CONTENT_MARGIN}px 0`;
            contentWrapper.style.minHeight = minContentHeight + 'px';
            contentWrapper.style.maxHeight = availableHeight + 'px'; // -----------------------------------------------------------------------------------------
            onPlaced === null || onPlaced === void 0 || onPlaced(); // we don't want the initial scroll position adjustment to trigger "expand on scroll"
            // so we explicitly turn it on only after they've registered.
            requestAnimationFrame(()=>shouldExpandOnScrollRef.current = true
            );
        }
    }, [
        getItems,
        context.trigger,
        context.valueNode,
        contentWrapper,
        content,
        viewport,
        selectedItem,
        selectedItemText,
        context.dir,
        onPlaced
    ]);
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>position()
    , [
        position
    ]); // copy z-index from content to wrapper
    const [contentZIndex, setContentZIndex] = $cg2C9$react.useState();
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        if (content) setContentZIndex(window.getComputedStyle(content).zIndex);
    }, [
        content
    ]); // When the viewport becomes scrollable at the top, the scroll up button will mount.
    // Because it is part of the normal flow, it will push down the viewport, thus throwing our
    // trigger => selectedItem alignment off by the amount the viewport was pushed down.
    // We wait for this to happen and then re-run the positining logic one more time to account for it.
    const handleScrollButtonChange = $cg2C9$react.useCallback((node)=>{
        if (node && shouldRepositionRef.current === true) {
            position();
            focusSelectedItem === null || focusSelectedItem === void 0 || focusSelectedItem();
            shouldRepositionRef.current = false;
        }
    }, [
        position,
        focusSelectedItem
    ]);
    return /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectViewportProvider, {
        scope: __scopeSelect,
        contentWrapper: contentWrapper,
        shouldExpandOnScrollRef: shouldExpandOnScrollRef,
        onScrollButtonChange: handleScrollButtonChange
    }, /*#__PURE__*/ $cg2C9$react.createElement("div", {
        ref: setContentWrapper,
        style: {
            display: 'flex',
            flexDirection: 'column',
            position: 'fixed',
            zIndex: contentZIndex
        }
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, popperProps, {
        ref: composedRefs,
        style: {
            // When we get the height of the content, it includes borders. If we were to set
            // the height without having `boxSizing: 'border-box'` it would be too big.
            boxSizing: 'border-box',
            // We need to ensure the content doesn't get taller than the wrapper
            maxHeight: '100%',
            ...popperProps.style
        }
    }))));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$var$SelectItemAlignedPosition, {
    displayName: $1345bda09ffc1bc6$var$ITEM_ALIGNED_POSITION_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectPopperPosition
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$POPPER_POSITION_NAME = 'SelectPopperPosition';
const $1345bda09ffc1bc6$var$SelectPopperPosition = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , align: align = 'start' , collisionPadding: collisionPadding = $1345bda09ffc1bc6$var$CONTENT_MARGIN , ...popperProps } = props;
    const popperScope = $1345bda09ffc1bc6$var$usePopperScope(__scopeSelect);
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactpopper.Content, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, popperScope, popperProps, {
        ref: forwardedRef,
        align: align,
        collisionPadding: collisionPadding,
        style: {
            // Ensure border-box for floating-ui calculations
            boxSizing: 'border-box',
            ...popperProps.style,
            '--radix-select-content-transform-origin': 'var(--radix-popper-transform-origin)',
            '--radix-select-content-available-width': 'var(--radix-popper-available-width)',
            '--radix-select-content-available-height': 'var(--radix-popper-available-height)',
            '--radix-select-trigger-width': 'var(--radix-popper-anchor-width)',
            '--radix-select-trigger-height': 'var(--radix-popper-anchor-height)'
        }
    }));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$var$SelectPopperPosition, {
    displayName: $1345bda09ffc1bc6$var$POPPER_POSITION_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectViewport
 * -----------------------------------------------------------------------------------------------*/ const [$1345bda09ffc1bc6$var$SelectViewportProvider, $1345bda09ffc1bc6$var$useSelectViewportContext] = $1345bda09ffc1bc6$var$createSelectContext($1345bda09ffc1bc6$var$CONTENT_NAME, {});
const $1345bda09ffc1bc6$var$VIEWPORT_NAME = 'SelectViewport';
const $1345bda09ffc1bc6$export$9ed6e7b40248d36d = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , ...viewportProps } = props;
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$VIEWPORT_NAME, __scopeSelect);
    const viewportContext = $1345bda09ffc1bc6$var$useSelectViewportContext($1345bda09ffc1bc6$var$VIEWPORT_NAME, __scopeSelect);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, contentContext.onViewportChange);
    const prevScrollTopRef = $cg2C9$react.useRef(0);
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$react.Fragment, null, /*#__PURE__*/ $cg2C9$react.createElement("style", {
        dangerouslySetInnerHTML: {
            __html: `[data-radix-select-viewport]{scrollbar-width:none;-ms-overflow-style:none;-webkit-overflow-scrolling:touch;}[data-radix-select-viewport]::-webkit-scrollbar{display:none}`
        }
    }), /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$Collection.Slot, {
        scope: __scopeSelect
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        "data-radix-select-viewport": "",
        role: "presentation"
    }, viewportProps, {
        ref: composedRefs,
        style: {
            // we use position: 'relative' here on the `viewport` so that when we call
            // `selectedItem.offsetTop` in calculations, the offset is relative to the viewport
            // (independent of the scrollUpButton).
            position: 'relative',
            flex: 1,
            overflow: 'auto',
            ...viewportProps.style
        },
        onScroll: $cg2C9$radixuiprimitive.composeEventHandlers(viewportProps.onScroll, (event)=>{
            const viewport = event.currentTarget;
            const { contentWrapper: contentWrapper , shouldExpandOnScrollRef: shouldExpandOnScrollRef  } = viewportContext;
            if (shouldExpandOnScrollRef !== null && shouldExpandOnScrollRef !== void 0 && shouldExpandOnScrollRef.current && contentWrapper) {
                const scrolledBy = Math.abs(prevScrollTopRef.current - viewport.scrollTop);
                if (scrolledBy > 0) {
                    const availableHeight = window.innerHeight - $1345bda09ffc1bc6$var$CONTENT_MARGIN * 2;
                    const cssMinHeight = parseFloat(contentWrapper.style.minHeight);
                    const cssHeight = parseFloat(contentWrapper.style.height);
                    const prevHeight = Math.max(cssMinHeight, cssHeight);
                    if (prevHeight < availableHeight) {
                        const nextHeight = prevHeight + scrolledBy;
                        const clampedNextHeight = Math.min(availableHeight, nextHeight);
                        const heightDiff = nextHeight - clampedNextHeight;
                        contentWrapper.style.height = clampedNextHeight + 'px';
                        if (contentWrapper.style.bottom === '0px') {
                            viewport.scrollTop = heightDiff > 0 ? heightDiff : 0; // ensure the content stays pinned to the bottom
                            contentWrapper.style.justifyContent = 'flex-end';
                        }
                    }
                }
            }
            prevScrollTopRef.current = viewport.scrollTop;
        })
    }))));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$9ed6e7b40248d36d, {
    displayName: $1345bda09ffc1bc6$var$VIEWPORT_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectGroup
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$GROUP_NAME = 'SelectGroup';
const [$1345bda09ffc1bc6$var$SelectGroupContextProvider, $1345bda09ffc1bc6$var$useSelectGroupContext] = $1345bda09ffc1bc6$var$createSelectContext($1345bda09ffc1bc6$var$GROUP_NAME);
const $1345bda09ffc1bc6$export$ee25a334c55de1f4 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , ...groupProps } = props;
    const groupId = $cg2C9$radixuireactid.useId();
    return /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectGroupContextProvider, {
        scope: __scopeSelect,
        id: groupId
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        role: "group",
        "aria-labelledby": groupId
    }, groupProps, {
        ref: forwardedRef
    })));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$ee25a334c55de1f4, {
    displayName: $1345bda09ffc1bc6$var$GROUP_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectLabel
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$LABEL_NAME = 'SelectLabel';
const $1345bda09ffc1bc6$export$f67338d29bd972f8 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , ...labelProps } = props;
    const groupContext = $1345bda09ffc1bc6$var$useSelectGroupContext($1345bda09ffc1bc6$var$LABEL_NAME, __scopeSelect);
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        id: groupContext.id
    }, labelProps, {
        ref: forwardedRef
    }));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$f67338d29bd972f8, {
    displayName: $1345bda09ffc1bc6$var$LABEL_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectItem
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$ITEM_NAME = 'SelectItem';
const [$1345bda09ffc1bc6$var$SelectItemContextProvider, $1345bda09ffc1bc6$var$useSelectItemContext] = $1345bda09ffc1bc6$var$createSelectContext($1345bda09ffc1bc6$var$ITEM_NAME);
const $1345bda09ffc1bc6$export$13ef48a934230896 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , value: value , disabled: disabled = false , textValue: textValueProp , ...itemProps } = props;
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$ITEM_NAME, __scopeSelect);
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$ITEM_NAME, __scopeSelect);
    const isSelected = context.value === value;
    const [textValue, setTextValue] = $cg2C9$react.useState(textValueProp !== null && textValueProp !== void 0 ? textValueProp : '');
    const [isFocused, setIsFocused] = $cg2C9$react.useState(false);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, (node)=>{
        var _contentContext$itemR;
        return (_contentContext$itemR = contentContext.itemRefCallback) === null || _contentContext$itemR === void 0 ? void 0 : _contentContext$itemR.call(contentContext, node, value, disabled);
    });
    const textId = $cg2C9$radixuireactid.useId();
    const handleSelect = ()=>{
        if (!disabled) {
            context.onValueChange(value);
            context.onOpenChange(false);
        }
    };
    return /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectItemContextProvider, {
        scope: __scopeSelect,
        value: value,
        disabled: disabled,
        textId: textId,
        isSelected: isSelected,
        onItemTextChange: $cg2C9$react.useCallback((node)=>{
            setTextValue((prevTextValue)=>{
                var _node$textContent;
                return prevTextValue || ((_node$textContent = node === null || node === void 0 ? void 0 : node.textContent) !== null && _node$textContent !== void 0 ? _node$textContent : '').trim();
            });
        }, [])
    }, /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$Collection.ItemSlot, {
        scope: __scopeSelect,
        value: value,
        disabled: disabled,
        textValue: textValue
    }, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        role: "option",
        "aria-labelledby": textId,
        "data-highlighted": isFocused ? '' : undefined // `isFocused` caveat fixes stuttering in VoiceOver
        ,
        "aria-selected": isSelected && isFocused,
        "data-state": isSelected ? 'checked' : 'unchecked',
        "aria-disabled": disabled || undefined,
        "data-disabled": disabled ? '' : undefined,
        tabIndex: disabled ? undefined : -1
    }, itemProps, {
        ref: composedRefs,
        onFocus: $cg2C9$radixuiprimitive.composeEventHandlers(itemProps.onFocus, ()=>setIsFocused(true)
        ),
        onBlur: $cg2C9$radixuiprimitive.composeEventHandlers(itemProps.onBlur, ()=>setIsFocused(false)
        ),
        onPointerUp: $cg2C9$radixuiprimitive.composeEventHandlers(itemProps.onPointerUp, handleSelect),
        onPointerMove: $cg2C9$radixuiprimitive.composeEventHandlers(itemProps.onPointerMove, (event)=>{
            if (disabled) {
                var _contentContext$onIte;
                (_contentContext$onIte = contentContext.onItemLeave) === null || _contentContext$onIte === void 0 || _contentContext$onIte.call(contentContext);
            } else // even though safari doesn't support this option, it's acceptable
            // as it only means it might scroll a few pixels when using the pointer.
            event.currentTarget.focus({
                preventScroll: true
            });
        }),
        onPointerLeave: $cg2C9$radixuiprimitive.composeEventHandlers(itemProps.onPointerLeave, (event)=>{
            if (event.currentTarget === document.activeElement) {
                var _contentContext$onIte2;
                (_contentContext$onIte2 = contentContext.onItemLeave) === null || _contentContext$onIte2 === void 0 || _contentContext$onIte2.call(contentContext);
            }
        }),
        onKeyDown: $cg2C9$radixuiprimitive.composeEventHandlers(itemProps.onKeyDown, (event)=>{
            var _contentContext$searc;
            const isTypingAhead = ((_contentContext$searc = contentContext.searchRef) === null || _contentContext$searc === void 0 ? void 0 : _contentContext$searc.current) !== '';
            if (isTypingAhead && event.key === ' ') return;
            if ($1345bda09ffc1bc6$var$SELECTION_KEYS.includes(event.key)) handleSelect(); // prevent page scroll if using the space key to select an item
            if (event.key === ' ') event.preventDefault();
        })
    }))));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$13ef48a934230896, {
    displayName: $1345bda09ffc1bc6$var$ITEM_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectItemText
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$ITEM_TEXT_NAME = 'SelectItemText';
const $1345bda09ffc1bc6$export$3572fb0fb821ff49 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    // We ignore `className` and `style` as this part shouldn't be styled.
    const { __scopeSelect: __scopeSelect , className: className , style: style , ...itemTextProps } = props;
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$ITEM_TEXT_NAME, __scopeSelect);
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$ITEM_TEXT_NAME, __scopeSelect);
    const itemContext = $1345bda09ffc1bc6$var$useSelectItemContext($1345bda09ffc1bc6$var$ITEM_TEXT_NAME, __scopeSelect);
    const nativeOptionsContext = $1345bda09ffc1bc6$var$useSelectNativeOptionsContext($1345bda09ffc1bc6$var$ITEM_TEXT_NAME, __scopeSelect);
    const [itemTextNode, setItemTextNode] = $cg2C9$react.useState(null);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, (node)=>setItemTextNode(node)
    , itemContext.onItemTextChange, (node)=>{
        var _contentContext$itemT;
        return (_contentContext$itemT = contentContext.itemTextRefCallback) === null || _contentContext$itemT === void 0 ? void 0 : _contentContext$itemT.call(contentContext, node, itemContext.value, itemContext.disabled);
    });
    const textContent = itemTextNode === null || itemTextNode === void 0 ? void 0 : itemTextNode.textContent;
    const nativeOption = $cg2C9$react.useMemo(()=>/*#__PURE__*/ $cg2C9$react.createElement("option", {
            key: itemContext.value,
            value: itemContext.value,
            disabled: itemContext.disabled
        }, textContent)
    , [
        itemContext.disabled,
        itemContext.value,
        textContent
    ]);
    const { onNativeOptionAdd: onNativeOptionAdd , onNativeOptionRemove: onNativeOptionRemove  } = nativeOptionsContext;
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        onNativeOptionAdd(nativeOption);
        return ()=>onNativeOptionRemove(nativeOption)
        ;
    }, [
        onNativeOptionAdd,
        onNativeOptionRemove,
        nativeOption
    ]);
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$react.Fragment, null, /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.span, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        id: itemContext.textId
    }, itemTextProps, {
        ref: composedRefs
    })), itemContext.isSelected && context.valueNode && !context.valueNodeHasChildren ? /*#__PURE__*/ $cg2C9$reactdom.createPortal(itemTextProps.children, context.valueNode) : null);
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$3572fb0fb821ff49, {
    displayName: $1345bda09ffc1bc6$var$ITEM_TEXT_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectItemIndicator
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$ITEM_INDICATOR_NAME = 'SelectItemIndicator';
const $1345bda09ffc1bc6$export$6b9198de19accfe6 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , ...itemIndicatorProps } = props;
    const itemContext = $1345bda09ffc1bc6$var$useSelectItemContext($1345bda09ffc1bc6$var$ITEM_INDICATOR_NAME, __scopeSelect);
    return itemContext.isSelected ? /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.span, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        "aria-hidden": true
    }, itemIndicatorProps, {
        ref: forwardedRef
    })) : null;
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$6b9198de19accfe6, {
    displayName: $1345bda09ffc1bc6$var$ITEM_INDICATOR_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectScrollUpButton
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$SCROLL_UP_BUTTON_NAME = 'SelectScrollUpButton';
const $1345bda09ffc1bc6$export$d8117927658af6d7 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$SCROLL_UP_BUTTON_NAME, props.__scopeSelect);
    const viewportContext = $1345bda09ffc1bc6$var$useSelectViewportContext($1345bda09ffc1bc6$var$SCROLL_UP_BUTTON_NAME, props.__scopeSelect);
    const [canScrollUp1, setCanScrollUp] = $cg2C9$react.useState(false);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, viewportContext.onScrollButtonChange);
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        if (contentContext.viewport && contentContext.isPositioned) {
            const viewport = contentContext.viewport;
            function handleScroll() {
                const canScrollUp = viewport.scrollTop > 0;
                setCanScrollUp(canScrollUp);
            }
            handleScroll();
            viewport.addEventListener('scroll', handleScroll);
            return ()=>viewport.removeEventListener('scroll', handleScroll)
            ;
        }
    }, [
        contentContext.viewport,
        contentContext.isPositioned
    ]);
    return canScrollUp1 ? /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectScrollButtonImpl, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, props, {
        ref: composedRefs,
        onAutoScroll: ()=>{
            const { viewport: viewport , selectedItem: selectedItem  } = contentContext;
            if (viewport && selectedItem) viewport.scrollTop = viewport.scrollTop - selectedItem.offsetHeight;
        }
    })) : null;
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$d8117927658af6d7, {
    displayName: $1345bda09ffc1bc6$var$SCROLL_UP_BUTTON_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectScrollDownButton
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$SCROLL_DOWN_BUTTON_NAME = 'SelectScrollDownButton';
const $1345bda09ffc1bc6$export$ff951e476c12189 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$SCROLL_DOWN_BUTTON_NAME, props.__scopeSelect);
    const viewportContext = $1345bda09ffc1bc6$var$useSelectViewportContext($1345bda09ffc1bc6$var$SCROLL_DOWN_BUTTON_NAME, props.__scopeSelect);
    const [canScrollDown1, setCanScrollDown] = $cg2C9$react.useState(false);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, viewportContext.onScrollButtonChange);
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        if (contentContext.viewport && contentContext.isPositioned) {
            const viewport = contentContext.viewport;
            function handleScroll() {
                const maxScroll = viewport.scrollHeight - viewport.clientHeight; // we use Math.ceil here because if the UI is zoomed-in
                // `scrollTop` is not always reported as an integer
                const canScrollDown = Math.ceil(viewport.scrollTop) < maxScroll;
                setCanScrollDown(canScrollDown);
            }
            handleScroll();
            viewport.addEventListener('scroll', handleScroll);
            return ()=>viewport.removeEventListener('scroll', handleScroll)
            ;
        }
    }, [
        contentContext.viewport,
        contentContext.isPositioned
    ]);
    return canScrollDown1 ? /*#__PURE__*/ $cg2C9$react.createElement($1345bda09ffc1bc6$var$SelectScrollButtonImpl, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, props, {
        ref: composedRefs,
        onAutoScroll: ()=>{
            const { viewport: viewport , selectedItem: selectedItem  } = contentContext;
            if (viewport && selectedItem) viewport.scrollTop = viewport.scrollTop + selectedItem.offsetHeight;
        }
    })) : null;
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$ff951e476c12189, {
    displayName: $1345bda09ffc1bc6$var$SCROLL_DOWN_BUTTON_NAME
});
const $1345bda09ffc1bc6$var$SelectScrollButtonImpl = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , onAutoScroll: onAutoScroll , ...scrollIndicatorProps } = props;
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext('SelectScrollButton', __scopeSelect);
    const autoScrollTimerRef = $cg2C9$react.useRef(null);
    const getItems = $1345bda09ffc1bc6$var$useCollection(__scopeSelect);
    const clearAutoScrollTimer = $cg2C9$react.useCallback(()=>{
        if (autoScrollTimerRef.current !== null) {
            window.clearInterval(autoScrollTimerRef.current);
            autoScrollTimerRef.current = null;
        }
    }, []);
    $cg2C9$react.useEffect(()=>{
        return ()=>clearAutoScrollTimer()
        ;
    }, [
        clearAutoScrollTimer
    ]); // When the viewport becomes scrollable on either side, the relevant scroll button will mount.
    // Because it is part of the normal flow, it will push down (top button) or shrink (bottom button)
    // the viewport, potentially causing the active item to now be partially out of view.
    // We re-run the `scrollIntoView` logic to make sure it stays within the viewport.
    $cg2C9$radixuireactuselayouteffect.useLayoutEffect(()=>{
        var _activeItem$ref$curre;
        const activeItem = getItems().find((item)=>item.ref.current === document.activeElement
        );
        activeItem === null || activeItem === void 0 || (_activeItem$ref$curre = activeItem.ref.current) === null || _activeItem$ref$curre === void 0 || _activeItem$ref$curre.scrollIntoView({
            block: 'nearest'
        });
    }, [
        getItems
    ]);
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        "aria-hidden": true
    }, scrollIndicatorProps, {
        ref: forwardedRef,
        style: {
            flexShrink: 0,
            ...scrollIndicatorProps.style
        },
        onPointerDown: $cg2C9$radixuiprimitive.composeEventHandlers(scrollIndicatorProps.onPointerDown, ()=>{
            if (autoScrollTimerRef.current === null) autoScrollTimerRef.current = window.setInterval(onAutoScroll, 50);
        }),
        onPointerMove: $cg2C9$radixuiprimitive.composeEventHandlers(scrollIndicatorProps.onPointerMove, ()=>{
            var _contentContext$onIte3;
            (_contentContext$onIte3 = contentContext.onItemLeave) === null || _contentContext$onIte3 === void 0 || _contentContext$onIte3.call(contentContext);
            if (autoScrollTimerRef.current === null) autoScrollTimerRef.current = window.setInterval(onAutoScroll, 50);
        }),
        onPointerLeave: $cg2C9$radixuiprimitive.composeEventHandlers(scrollIndicatorProps.onPointerLeave, ()=>{
            clearAutoScrollTimer();
        })
    }));
});
/* -------------------------------------------------------------------------------------------------
 * SelectSeparator
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$SEPARATOR_NAME = 'SelectSeparator';
const $1345bda09ffc1bc6$export$eba4b1df07cb1d3 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , ...separatorProps } = props;
    return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactprimitive.Primitive.div, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({
        "aria-hidden": true
    }, separatorProps, {
        ref: forwardedRef
    }));
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$eba4b1df07cb1d3, {
    displayName: $1345bda09ffc1bc6$var$SEPARATOR_NAME
});
/* -------------------------------------------------------------------------------------------------
 * SelectArrow
 * -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$ARROW_NAME = 'SelectArrow';
const $1345bda09ffc1bc6$export$314f4cb8f8099628 = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { __scopeSelect: __scopeSelect , ...arrowProps } = props;
    const popperScope = $1345bda09ffc1bc6$var$usePopperScope(__scopeSelect);
    const context = $1345bda09ffc1bc6$var$useSelectContext($1345bda09ffc1bc6$var$ARROW_NAME, __scopeSelect);
    const contentContext = $1345bda09ffc1bc6$var$useSelectContentContext($1345bda09ffc1bc6$var$ARROW_NAME, __scopeSelect);
    return context.open && contentContext.position === 'popper' ? /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactpopper.Arrow, ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, popperScope, arrowProps, {
        ref: forwardedRef
    })) : null;
});
/*#__PURE__*/ Object.assign($1345bda09ffc1bc6$export$314f4cb8f8099628, {
    displayName: $1345bda09ffc1bc6$var$ARROW_NAME
});
/* -----------------------------------------------------------------------------------------------*/ const $1345bda09ffc1bc6$var$BubbleSelect = /*#__PURE__*/ $cg2C9$react.forwardRef((props, forwardedRef)=>{
    const { value: value , ...selectProps } = props;
    const ref = $cg2C9$react.useRef(null);
    const composedRefs = $cg2C9$radixuireactcomposerefs.useComposedRefs(forwardedRef, ref);
    const prevValue = $cg2C9$radixuireactuseprevious.usePrevious(value); // Bubble value change to parents (e.g form change event)
    $cg2C9$react.useEffect(()=>{
        const select = ref.current;
        const selectProto = window.HTMLSelectElement.prototype;
        const descriptor = Object.getOwnPropertyDescriptor(selectProto, 'value');
        const setValue = descriptor.set;
        if (prevValue !== value && setValue) {
            const event = new Event('change', {
                bubbles: true
            });
            setValue.call(select, value);
            select.dispatchEvent(event);
        }
    }, [
        prevValue,
        value
    ]);
    /**
   * We purposefully use a `select` here to support form autofill as much
   * as possible.
   *
   * We purposefully do not add the `value` attribute here to allow the value
   * to be set programatically and bubble to any parent form `onChange` event.
   * Adding the `value` will cause React to consider the programatic
   * dispatch a duplicate and it will get swallowed.
   *
   * We use `VisuallyHidden` rather than `display: "none"` because Safari autofill
   * won't work otherwise.
   */ return /*#__PURE__*/ $cg2C9$react.createElement($cg2C9$radixuireactvisuallyhidden.VisuallyHidden, {
        asChild: true
    }, /*#__PURE__*/ $cg2C9$react.createElement("select", ($parcel$interopDefault($cg2C9$babelruntimehelpersextends))({}, selectProps, {
        ref: composedRefs,
        defaultValue: value
    })));
});
$1345bda09ffc1bc6$var$BubbleSelect.displayName = 'BubbleSelect';
function $1345bda09ffc1bc6$var$useTypeaheadSearch(onSearchChange) {
    const handleSearchChange = $cg2C9$radixuireactusecallbackref.useCallbackRef(onSearchChange);
    const searchRef = $cg2C9$react.useRef('');
    const timerRef = $cg2C9$react.useRef(0);
    const handleTypeaheadSearch = $cg2C9$react.useCallback((key)=>{
        const search = searchRef.current + key;
        handleSearchChange(search);
        (function updateSearch(value) {
            searchRef.current = value;
            window.clearTimeout(timerRef.current); // Reset `searchRef` 1 second after it was last updated
            if (value !== '') timerRef.current = window.setTimeout(()=>updateSearch('')
            , 1000);
        })(search);
    }, [
        handleSearchChange
    ]);
    const resetTypeahead = $cg2C9$react.useCallback(()=>{
        searchRef.current = '';
        window.clearTimeout(timerRef.current);
    }, []);
    $cg2C9$react.useEffect(()=>{
        return ()=>window.clearTimeout(timerRef.current)
        ;
    }, []);
    return [
        searchRef,
        handleTypeaheadSearch,
        resetTypeahead
    ];
}
/**
 * This is the "meat" of the typeahead matching logic. It takes in a list of items,
 * the search and the current item, and returns the next item (or `undefined`).
 *
 * We normalize the search because if a user has repeatedly pressed a character,
 * we want the exact same behavior as if we only had that one character
 * (ie. cycle through items starting with that character)
 *
 * We also reorder the items by wrapping the array around the current item.
 * This is so we always look forward from the current item, and picking the first
 * item will always be the correct one.
 *
 * Finally, if the normalized search is exactly one character, we exclude the
 * current item from the values because otherwise it would be the first to match always
 * and focus would never move. This is as opposed to the regular case, where we
 * don't want focus to move if the current item still matches.
 */ function $1345bda09ffc1bc6$var$findNextItem(items, search, currentItem) {
    const isRepeated = search.length > 1 && Array.from(search).every((char)=>char === search[0]
    );
    const normalizedSearch = isRepeated ? search[0] : search;
    const currentItemIndex = currentItem ? items.indexOf(currentItem) : -1;
    let wrappedItems = $1345bda09ffc1bc6$var$wrapArray(items, Math.max(currentItemIndex, 0));
    const excludeCurrentItem = normalizedSearch.length === 1;
    if (excludeCurrentItem) wrappedItems = wrappedItems.filter((v)=>v !== currentItem
    );
    const nextItem = wrappedItems.find((item)=>item.textValue.toLowerCase().startsWith(normalizedSearch.toLowerCase())
    );
    return nextItem !== currentItem ? nextItem : undefined;
}
/**
 * Wraps an array around itself at a given start index
 * Example: `wrapArray(['a', 'b', 'c', 'd'], 2) === ['c', 'd', 'a', 'b']`
 */ function $1345bda09ffc1bc6$var$wrapArray(array, startIndex) {
    return array.map((_, index)=>array[(startIndex + index) % array.length]
    );
}
const $1345bda09ffc1bc6$export$be92b6f5f03c0fe9 = $1345bda09ffc1bc6$export$ef9b1a59e592288f;
const $1345bda09ffc1bc6$export$41fb9f06171c75f4 = $1345bda09ffc1bc6$export$3ac1e88a1c0b9f1;
const $1345bda09ffc1bc6$export$4c8d1a57a761ef94 = $1345bda09ffc1bc6$export$e288731fd71264f0;
const $1345bda09ffc1bc6$export$f04a61298a47a40f = $1345bda09ffc1bc6$export$99b400cabb58c515;
const $1345bda09ffc1bc6$export$602eac185826482c = $1345bda09ffc1bc6$export$b2af6c9944296213;
const $1345bda09ffc1bc6$export$7c6e2c02157bb7d2 = $1345bda09ffc1bc6$export$c973a4b3cb86a03d;
const $1345bda09ffc1bc6$export$d5c6c08dc2d3ca7 = $1345bda09ffc1bc6$export$9ed6e7b40248d36d;
const $1345bda09ffc1bc6$export$eb2fcfdbd7ba97d4 = $1345bda09ffc1bc6$export$ee25a334c55de1f4;
const $1345bda09ffc1bc6$export$b04be29aa201d4f5 = $1345bda09ffc1bc6$export$f67338d29bd972f8;
const $1345bda09ffc1bc6$export$6d08773d2e66f8f2 = $1345bda09ffc1bc6$export$13ef48a934230896;
const $1345bda09ffc1bc6$export$d6e5bf9c43ea9319 = $1345bda09ffc1bc6$export$3572fb0fb821ff49;
const $1345bda09ffc1bc6$export$c3468e2714d175fa = $1345bda09ffc1bc6$export$6b9198de19accfe6;
const $1345bda09ffc1bc6$export$2f60d3ec9ad468f2 = $1345bda09ffc1bc6$export$d8117927658af6d7;
const $1345bda09ffc1bc6$export$bf1aedc3039c8d63 = $1345bda09ffc1bc6$export$ff951e476c12189;
const $1345bda09ffc1bc6$export$1ff3c3f08ae963c0 = $1345bda09ffc1bc6$export$eba4b1df07cb1d3;
const $1345bda09ffc1bc6$export$21b07c8f274aebd5 = $1345bda09ffc1bc6$export$314f4cb8f8099628;




//# sourceMappingURL=index.js.map
