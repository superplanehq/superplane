var $2833058fcd3993f5$exports = require("./focusSafely.main.js");
var $5cb73d0ce355b0dc$exports = require("./useFocus.main.js");
var $892d64db2a3c53b0$exports = require("./useKeyboard.main.js");
var $cvQje$reactariautils = require("@react-aria/utils");
var $cvQje$react = require("react");


function $parcel$interopDefault(a) {
  return a && a.__esModule ? a.default : a;
}

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "FocusableContext", () => $15f8fd80892557ff$export$f9762fab77588ecb);
$parcel$export(module.exports, "FocusableProvider", () => $15f8fd80892557ff$export$13f3202a3e5ddd5);
$parcel$export(module.exports, "useFocusable", () => $15f8fd80892557ff$export$4c014de7c8940b4c);
$parcel$export(module.exports, "Focusable", () => $15f8fd80892557ff$export$35a3bebf7ef2d934);
/*
 * Copyright 2020 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 




let $15f8fd80892557ff$export$f9762fab77588ecb = /*#__PURE__*/ (0, ($parcel$interopDefault($cvQje$react))).createContext(null);
function $15f8fd80892557ff$var$useFocusableContext(ref) {
    let context = (0, $cvQje$react.useContext)($15f8fd80892557ff$export$f9762fab77588ecb) || {};
    (0, $cvQje$reactariautils.useSyncRef)(context, ref);
    // eslint-disable-next-line
    let { ref: _, ...otherProps } = context;
    return otherProps;
}
const $15f8fd80892557ff$export$13f3202a3e5ddd5 = /*#__PURE__*/ (0, ($parcel$interopDefault($cvQje$react))).forwardRef(function FocusableProvider(props, ref) {
    let { children: children, ...otherProps } = props;
    let objRef = (0, $cvQje$reactariautils.useObjectRef)(ref);
    let context = {
        ...otherProps,
        ref: objRef
    };
    return /*#__PURE__*/ (0, ($parcel$interopDefault($cvQje$react))).createElement($15f8fd80892557ff$export$f9762fab77588ecb.Provider, {
        value: context
    }, children);
});
function $15f8fd80892557ff$export$4c014de7c8940b4c(props, domRef) {
    let { focusProps: focusProps } = (0, $5cb73d0ce355b0dc$exports.useFocus)(props);
    let { keyboardProps: keyboardProps } = (0, $892d64db2a3c53b0$exports.useKeyboard)(props);
    let interactions = (0, $cvQje$reactariautils.mergeProps)(focusProps, keyboardProps);
    let domProps = $15f8fd80892557ff$var$useFocusableContext(domRef);
    let interactionProps = props.isDisabled ? {} : domProps;
    let autoFocusRef = (0, $cvQje$react.useRef)(props.autoFocus);
    (0, $cvQje$react.useEffect)(()=>{
        if (autoFocusRef.current && domRef.current) (0, $2833058fcd3993f5$exports.focusSafely)(domRef.current);
        autoFocusRef.current = false;
    }, [
        domRef
    ]);
    // Always set a tabIndex so that Safari allows focusing native buttons and inputs.
    let tabIndex = props.excludeFromTabOrder ? -1 : 0;
    if (props.isDisabled) tabIndex = undefined;
    return {
        focusableProps: (0, $cvQje$reactariautils.mergeProps)({
            ...interactions,
            tabIndex: tabIndex
        }, interactionProps)
    };
}
const $15f8fd80892557ff$export$35a3bebf7ef2d934 = /*#__PURE__*/ (0, $cvQje$react.forwardRef)(({ children: children, ...props }, ref)=>{
    ref = (0, $cvQje$reactariautils.useObjectRef)(ref);
    let { focusableProps: focusableProps } = $15f8fd80892557ff$export$4c014de7c8940b4c(props, ref);
    let child = (0, ($parcel$interopDefault($cvQje$react))).Children.only(children);
    (0, $cvQje$react.useEffect)(()=>{
        if (process.env.NODE_ENV === 'production') return;
        let el = ref.current;
        if (!el || !(el instanceof (0, $cvQje$reactariautils.getOwnerWindow)(el).Element)) {
            console.error('<Focusable> child must forward its ref to a DOM element.');
            return;
        }
        if (!props.isDisabled && !(0, $cvQje$reactariautils.isFocusable)(el)) {
            console.warn('<Focusable> child must be focusable. Please ensure the tabIndex prop is passed through.');
            return;
        }
        if (el.localName !== 'button' && el.localName !== 'input' && el.localName !== 'select' && el.localName !== 'textarea' && el.localName !== 'a' && el.localName !== 'area' && el.localName !== 'summary' && el.localName !== 'img' && el.localName !== 'svg') {
            let role = el.getAttribute('role');
            if (!role) console.warn('<Focusable> child must have an interactive ARIA role.');
            else if (// https://w3c.github.io/aria/#widget_roles
            role !== 'application' && role !== 'button' && role !== 'checkbox' && role !== 'combobox' && role !== 'gridcell' && role !== 'link' && role !== 'menuitem' && role !== 'menuitemcheckbox' && role !== 'menuitemradio' && role !== 'option' && role !== 'radio' && role !== 'searchbox' && role !== 'separator' && role !== 'slider' && role !== 'spinbutton' && role !== 'switch' && role !== 'tab' && role !== 'tabpanel' && role !== 'textbox' && role !== 'treeitem' && // aria-describedby is also announced on these roles
            role !== 'img' && role !== 'meter' && role !== 'progressbar') console.warn(`<Focusable> child must have an interactive ARIA role. Got "${role}".`);
        }
    }, [
        ref,
        props.isDisabled
    ]);
    // @ts-ignore
    let childRef = parseInt((0, ($parcel$interopDefault($cvQje$react))).version, 10) < 19 ? child.ref : child.props.ref;
    return /*#__PURE__*/ (0, ($parcel$interopDefault($cvQje$react))).cloneElement(child, {
        ...(0, $cvQje$reactariautils.mergeProps)(focusableProps, child.props),
        // @ts-ignore
        ref: (0, $cvQje$reactariautils.mergeRefs)(childRef, ref)
    });
});


//# sourceMappingURL=useFocusable.main.js.map
