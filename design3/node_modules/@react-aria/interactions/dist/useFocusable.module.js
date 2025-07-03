import {focusSafely as $3ad3f6e1647bc98d$export$80f3e147d781571c} from "./focusSafely.module.js";
import {useFocus as $a1ea59d68270f0dd$export$f8168d8dd8fd66e6} from "./useFocus.module.js";
import {useKeyboard as $46d819fcbaf35654$export$8f71654801c2f7cd} from "./useKeyboard.module.js";
import {useSyncRef as $fcPuG$useSyncRef, useObjectRef as $fcPuG$useObjectRef, mergeProps as $fcPuG$mergeProps, getOwnerWindow as $fcPuG$getOwnerWindow, isFocusable as $fcPuG$isFocusable, mergeRefs as $fcPuG$mergeRefs} from "@react-aria/utils";
import $fcPuG$react, {useContext as $fcPuG$useContext, useRef as $fcPuG$useRef, useEffect as $fcPuG$useEffect, forwardRef as $fcPuG$forwardRef} from "react";

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




let $f645667febf57a63$export$f9762fab77588ecb = /*#__PURE__*/ (0, $fcPuG$react).createContext(null);
function $f645667febf57a63$var$useFocusableContext(ref) {
    let context = (0, $fcPuG$useContext)($f645667febf57a63$export$f9762fab77588ecb) || {};
    (0, $fcPuG$useSyncRef)(context, ref);
    // eslint-disable-next-line
    let { ref: _, ...otherProps } = context;
    return otherProps;
}
const $f645667febf57a63$export$13f3202a3e5ddd5 = /*#__PURE__*/ (0, $fcPuG$react).forwardRef(function FocusableProvider(props, ref) {
    let { children: children, ...otherProps } = props;
    let objRef = (0, $fcPuG$useObjectRef)(ref);
    let context = {
        ...otherProps,
        ref: objRef
    };
    return /*#__PURE__*/ (0, $fcPuG$react).createElement($f645667febf57a63$export$f9762fab77588ecb.Provider, {
        value: context
    }, children);
});
function $f645667febf57a63$export$4c014de7c8940b4c(props, domRef) {
    let { focusProps: focusProps } = (0, $a1ea59d68270f0dd$export$f8168d8dd8fd66e6)(props);
    let { keyboardProps: keyboardProps } = (0, $46d819fcbaf35654$export$8f71654801c2f7cd)(props);
    let interactions = (0, $fcPuG$mergeProps)(focusProps, keyboardProps);
    let domProps = $f645667febf57a63$var$useFocusableContext(domRef);
    let interactionProps = props.isDisabled ? {} : domProps;
    let autoFocusRef = (0, $fcPuG$useRef)(props.autoFocus);
    (0, $fcPuG$useEffect)(()=>{
        if (autoFocusRef.current && domRef.current) (0, $3ad3f6e1647bc98d$export$80f3e147d781571c)(domRef.current);
        autoFocusRef.current = false;
    }, [
        domRef
    ]);
    // Always set a tabIndex so that Safari allows focusing native buttons and inputs.
    let tabIndex = props.excludeFromTabOrder ? -1 : 0;
    if (props.isDisabled) tabIndex = undefined;
    return {
        focusableProps: (0, $fcPuG$mergeProps)({
            ...interactions,
            tabIndex: tabIndex
        }, interactionProps)
    };
}
const $f645667febf57a63$export$35a3bebf7ef2d934 = /*#__PURE__*/ (0, $fcPuG$forwardRef)(({ children: children, ...props }, ref)=>{
    ref = (0, $fcPuG$useObjectRef)(ref);
    let { focusableProps: focusableProps } = $f645667febf57a63$export$4c014de7c8940b4c(props, ref);
    let child = (0, $fcPuG$react).Children.only(children);
    (0, $fcPuG$useEffect)(()=>{
        if (process.env.NODE_ENV === 'production') return;
        let el = ref.current;
        if (!el || !(el instanceof (0, $fcPuG$getOwnerWindow)(el).Element)) {
            console.error('<Focusable> child must forward its ref to a DOM element.');
            return;
        }
        if (!props.isDisabled && !(0, $fcPuG$isFocusable)(el)) {
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
    let childRef = parseInt((0, $fcPuG$react).version, 10) < 19 ? child.ref : child.props.ref;
    return /*#__PURE__*/ (0, $fcPuG$react).cloneElement(child, {
        ...(0, $fcPuG$mergeProps)(focusableProps, child.props),
        // @ts-ignore
        ref: (0, $fcPuG$mergeRefs)(childRef, ref)
    });
});


export {$f645667febf57a63$export$f9762fab77588ecb as FocusableContext, $f645667febf57a63$export$13f3202a3e5ddd5 as FocusableProvider, $f645667febf57a63$export$4c014de7c8940b4c as useFocusable, $f645667febf57a63$export$35a3bebf7ef2d934 as Focusable};
//# sourceMappingURL=useFocusable.module.js.map
