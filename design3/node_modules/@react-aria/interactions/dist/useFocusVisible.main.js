var $625cf83917e112ad$exports = require("./utils.main.js");
var $cR3F8$reactariautils = require("@react-aria/utils");
var $cR3F8$react = require("react");
var $cR3F8$reactariassr = require("@react-aria/ssr");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "addWindowFocusTracking", () => $e77252a287ef94ab$export$2f1888112f558a7d);
$parcel$export(module.exports, "isFocusVisible", () => $e77252a287ef94ab$export$b9b3dfddab17db27);
$parcel$export(module.exports, "getInteractionModality", () => $e77252a287ef94ab$export$630ff653c5ada6a9);
$parcel$export(module.exports, "setInteractionModality", () => $e77252a287ef94ab$export$8397ddfc504fdb9a);
$parcel$export(module.exports, "useInteractionModality", () => $e77252a287ef94ab$export$98e20ec92f614cfe);
$parcel$export(module.exports, "useFocusVisible", () => $e77252a287ef94ab$export$ffd9e5021c1fb2d6);
$parcel$export(module.exports, "useFocusVisibleListener", () => $e77252a287ef94ab$export$ec71b4b83ac08ec3);
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
 */ // Portions of the code in this file are based on code from react.
// Original licensing for the following can be found in the
// NOTICE file in the root directory of this source tree.
// See https://github.com/facebook/react/tree/cc7c1aece46a6b69b41958d731e0fd27c94bfc6c/packages/react-interactions




let $e77252a287ef94ab$var$currentModality = null;
let $e77252a287ef94ab$var$changeHandlers = new Set();
let $e77252a287ef94ab$export$d90243b58daecda7 = new Map(); // We use a map here to support setting event listeners across multiple document objects.
let $e77252a287ef94ab$var$hasEventBeforeFocus = false;
let $e77252a287ef94ab$var$hasBlurredWindowRecently = false;
// Only Tab or Esc keys will make focus visible on text input elements
const $e77252a287ef94ab$var$FOCUS_VISIBLE_INPUT_KEYS = {
    Tab: true,
    Escape: true
};
function $e77252a287ef94ab$var$triggerChangeHandlers(modality, e) {
    for (let handler of $e77252a287ef94ab$var$changeHandlers)handler(modality, e);
}
/**
 * Helper function to determine if a KeyboardEvent is unmodified and could make keyboard focus styles visible.
 */ function $e77252a287ef94ab$var$isValidKey(e) {
    // Control and Shift keys trigger when navigating back to the tab with keyboard.
    return !(e.metaKey || !(0, $cR3F8$reactariautils.isMac)() && e.altKey || e.ctrlKey || e.key === 'Control' || e.key === 'Shift' || e.key === 'Meta');
}
function $e77252a287ef94ab$var$handleKeyboardEvent(e) {
    $e77252a287ef94ab$var$hasEventBeforeFocus = true;
    if ($e77252a287ef94ab$var$isValidKey(e)) {
        $e77252a287ef94ab$var$currentModality = 'keyboard';
        $e77252a287ef94ab$var$triggerChangeHandlers('keyboard', e);
    }
}
function $e77252a287ef94ab$var$handlePointerEvent(e) {
    $e77252a287ef94ab$var$currentModality = 'pointer';
    if (e.type === 'mousedown' || e.type === 'pointerdown') {
        $e77252a287ef94ab$var$hasEventBeforeFocus = true;
        $e77252a287ef94ab$var$triggerChangeHandlers('pointer', e);
    }
}
function $e77252a287ef94ab$var$handleClickEvent(e) {
    if ((0, $cR3F8$reactariautils.isVirtualClick)(e)) {
        $e77252a287ef94ab$var$hasEventBeforeFocus = true;
        $e77252a287ef94ab$var$currentModality = 'virtual';
    }
}
function $e77252a287ef94ab$var$handleFocusEvent(e) {
    // Firefox fires two extra focus events when the user first clicks into an iframe:
    // first on the window, then on the document. We ignore these events so they don't
    // cause keyboard focus rings to appear.
    if (e.target === window || e.target === document || (0, $625cf83917e112ad$exports.ignoreFocusEvent) || !e.isTrusted) return;
    // If a focus event occurs without a preceding keyboard or pointer event, switch to virtual modality.
    // This occurs, for example, when navigating a form with the next/previous buttons on iOS.
    if (!$e77252a287ef94ab$var$hasEventBeforeFocus && !$e77252a287ef94ab$var$hasBlurredWindowRecently) {
        $e77252a287ef94ab$var$currentModality = 'virtual';
        $e77252a287ef94ab$var$triggerChangeHandlers('virtual', e);
    }
    $e77252a287ef94ab$var$hasEventBeforeFocus = false;
    $e77252a287ef94ab$var$hasBlurredWindowRecently = false;
}
function $e77252a287ef94ab$var$handleWindowBlur() {
    if (0, $625cf83917e112ad$exports.ignoreFocusEvent) return;
    // When the window is blurred, reset state. This is necessary when tabbing out of the window,
    // for example, since a subsequent focus event won't be fired.
    $e77252a287ef94ab$var$hasEventBeforeFocus = false;
    $e77252a287ef94ab$var$hasBlurredWindowRecently = true;
}
/**
 * Setup global event listeners to control when keyboard focus style should be visible.
 */ function $e77252a287ef94ab$var$setupGlobalFocusEvents(element) {
    if (typeof window === 'undefined' || typeof document === 'undefined' || $e77252a287ef94ab$export$d90243b58daecda7.get((0, $cR3F8$reactariautils.getOwnerWindow)(element))) return;
    const windowObject = (0, $cR3F8$reactariautils.getOwnerWindow)(element);
    const documentObject = (0, $cR3F8$reactariautils.getOwnerDocument)(element);
    // Programmatic focus() calls shouldn't affect the current input modality.
    // However, we need to detect other cases when a focus event occurs without
    // a preceding user event (e.g. screen reader focus). Overriding the focus
    // method on HTMLElement.prototype is a bit hacky, but works.
    let focus = windowObject.HTMLElement.prototype.focus;
    windowObject.HTMLElement.prototype.focus = function() {
        $e77252a287ef94ab$var$hasEventBeforeFocus = true;
        focus.apply(this, arguments);
    };
    documentObject.addEventListener('keydown', $e77252a287ef94ab$var$handleKeyboardEvent, true);
    documentObject.addEventListener('keyup', $e77252a287ef94ab$var$handleKeyboardEvent, true);
    documentObject.addEventListener('click', $e77252a287ef94ab$var$handleClickEvent, true);
    // Register focus events on the window so they are sure to happen
    // before React's event listeners (registered on the document).
    windowObject.addEventListener('focus', $e77252a287ef94ab$var$handleFocusEvent, true);
    windowObject.addEventListener('blur', $e77252a287ef94ab$var$handleWindowBlur, false);
    if (typeof PointerEvent !== 'undefined') {
        documentObject.addEventListener('pointerdown', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.addEventListener('pointermove', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.addEventListener('pointerup', $e77252a287ef94ab$var$handlePointerEvent, true);
    } else if (process.env.NODE_ENV === 'test') {
        documentObject.addEventListener('mousedown', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.addEventListener('mousemove', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.addEventListener('mouseup', $e77252a287ef94ab$var$handlePointerEvent, true);
    }
    // Add unmount handler
    windowObject.addEventListener('beforeunload', ()=>{
        $e77252a287ef94ab$var$tearDownWindowFocusTracking(element);
    }, {
        once: true
    });
    $e77252a287ef94ab$export$d90243b58daecda7.set(windowObject, {
        focus: focus
    });
}
const $e77252a287ef94ab$var$tearDownWindowFocusTracking = (element, loadListener)=>{
    const windowObject = (0, $cR3F8$reactariautils.getOwnerWindow)(element);
    const documentObject = (0, $cR3F8$reactariautils.getOwnerDocument)(element);
    if (loadListener) documentObject.removeEventListener('DOMContentLoaded', loadListener);
    if (!$e77252a287ef94ab$export$d90243b58daecda7.has(windowObject)) return;
    windowObject.HTMLElement.prototype.focus = $e77252a287ef94ab$export$d90243b58daecda7.get(windowObject).focus;
    documentObject.removeEventListener('keydown', $e77252a287ef94ab$var$handleKeyboardEvent, true);
    documentObject.removeEventListener('keyup', $e77252a287ef94ab$var$handleKeyboardEvent, true);
    documentObject.removeEventListener('click', $e77252a287ef94ab$var$handleClickEvent, true);
    windowObject.removeEventListener('focus', $e77252a287ef94ab$var$handleFocusEvent, true);
    windowObject.removeEventListener('blur', $e77252a287ef94ab$var$handleWindowBlur, false);
    if (typeof PointerEvent !== 'undefined') {
        documentObject.removeEventListener('pointerdown', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.removeEventListener('pointermove', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.removeEventListener('pointerup', $e77252a287ef94ab$var$handlePointerEvent, true);
    } else if (process.env.NODE_ENV === 'test') {
        documentObject.removeEventListener('mousedown', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.removeEventListener('mousemove', $e77252a287ef94ab$var$handlePointerEvent, true);
        documentObject.removeEventListener('mouseup', $e77252a287ef94ab$var$handlePointerEvent, true);
    }
    $e77252a287ef94ab$export$d90243b58daecda7.delete(windowObject);
};
function $e77252a287ef94ab$export$2f1888112f558a7d(element) {
    const documentObject = (0, $cR3F8$reactariautils.getOwnerDocument)(element);
    let loadListener;
    if (documentObject.readyState !== 'loading') $e77252a287ef94ab$var$setupGlobalFocusEvents(element);
    else {
        loadListener = ()=>{
            $e77252a287ef94ab$var$setupGlobalFocusEvents(element);
        };
        documentObject.addEventListener('DOMContentLoaded', loadListener);
    }
    return ()=>$e77252a287ef94ab$var$tearDownWindowFocusTracking(element, loadListener);
}
// Server-side rendering does not have the document object defined
// eslint-disable-next-line no-restricted-globals
if (typeof document !== 'undefined') $e77252a287ef94ab$export$2f1888112f558a7d();
function $e77252a287ef94ab$export$b9b3dfddab17db27() {
    return $e77252a287ef94ab$var$currentModality !== 'pointer';
}
function $e77252a287ef94ab$export$630ff653c5ada6a9() {
    return $e77252a287ef94ab$var$currentModality;
}
function $e77252a287ef94ab$export$8397ddfc504fdb9a(modality) {
    $e77252a287ef94ab$var$currentModality = modality;
    $e77252a287ef94ab$var$triggerChangeHandlers(modality, null);
}
function $e77252a287ef94ab$export$98e20ec92f614cfe() {
    $e77252a287ef94ab$var$setupGlobalFocusEvents();
    let [modality, setModality] = (0, $cR3F8$react.useState)($e77252a287ef94ab$var$currentModality);
    (0, $cR3F8$react.useEffect)(()=>{
        let handler = ()=>{
            setModality($e77252a287ef94ab$var$currentModality);
        };
        $e77252a287ef94ab$var$changeHandlers.add(handler);
        return ()=>{
            $e77252a287ef94ab$var$changeHandlers.delete(handler);
        };
    }, []);
    return (0, $cR3F8$reactariassr.useIsSSR)() ? null : modality;
}
const $e77252a287ef94ab$var$nonTextInputTypes = new Set([
    'checkbox',
    'radio',
    'range',
    'color',
    'file',
    'image',
    'button',
    'submit',
    'reset'
]);
/**
 * If this is attached to text input component, return if the event is a focus event (Tab/Escape keys pressed) so that
 * focus visible style can be properly set.
 */ function $e77252a287ef94ab$var$isKeyboardFocusEvent(isTextInput, modality, e) {
    let document1 = (0, $cR3F8$reactariautils.getOwnerDocument)(e === null || e === void 0 ? void 0 : e.target);
    const IHTMLInputElement = typeof window !== 'undefined' ? (0, $cR3F8$reactariautils.getOwnerWindow)(e === null || e === void 0 ? void 0 : e.target).HTMLInputElement : HTMLInputElement;
    const IHTMLTextAreaElement = typeof window !== 'undefined' ? (0, $cR3F8$reactariautils.getOwnerWindow)(e === null || e === void 0 ? void 0 : e.target).HTMLTextAreaElement : HTMLTextAreaElement;
    const IHTMLElement = typeof window !== 'undefined' ? (0, $cR3F8$reactariautils.getOwnerWindow)(e === null || e === void 0 ? void 0 : e.target).HTMLElement : HTMLElement;
    const IKeyboardEvent = typeof window !== 'undefined' ? (0, $cR3F8$reactariautils.getOwnerWindow)(e === null || e === void 0 ? void 0 : e.target).KeyboardEvent : KeyboardEvent;
    // For keyboard events that occur on a non-input element that will move focus into input element (aka ArrowLeft going from Datepicker button to the main input group)
    // we need to rely on the user passing isTextInput into here. This way we can skip toggling focus visiblity for said input element
    isTextInput = isTextInput || document1.activeElement instanceof IHTMLInputElement && !$e77252a287ef94ab$var$nonTextInputTypes.has(document1.activeElement.type) || document1.activeElement instanceof IHTMLTextAreaElement || document1.activeElement instanceof IHTMLElement && document1.activeElement.isContentEditable;
    return !(isTextInput && modality === 'keyboard' && e instanceof IKeyboardEvent && !$e77252a287ef94ab$var$FOCUS_VISIBLE_INPUT_KEYS[e.key]);
}
function $e77252a287ef94ab$export$ffd9e5021c1fb2d6(props = {}) {
    let { isTextInput: isTextInput, autoFocus: autoFocus } = props;
    let [isFocusVisibleState, setFocusVisible] = (0, $cR3F8$react.useState)(autoFocus || $e77252a287ef94ab$export$b9b3dfddab17db27());
    $e77252a287ef94ab$export$ec71b4b83ac08ec3((isFocusVisible)=>{
        setFocusVisible(isFocusVisible);
    }, [
        isTextInput
    ], {
        isTextInput: isTextInput
    });
    return {
        isFocusVisible: isFocusVisibleState
    };
}
function $e77252a287ef94ab$export$ec71b4b83ac08ec3(fn, deps, opts) {
    $e77252a287ef94ab$var$setupGlobalFocusEvents();
    (0, $cR3F8$react.useEffect)(()=>{
        let handler = (modality, e)=>{
            // We want to early return for any keyboard events that occur inside text inputs EXCEPT for Tab and Escape
            if (!$e77252a287ef94ab$var$isKeyboardFocusEvent(!!(opts === null || opts === void 0 ? void 0 : opts.isTextInput), modality, e)) return;
            fn($e77252a287ef94ab$export$b9b3dfddab17db27());
        };
        $e77252a287ef94ab$var$changeHandlers.add(handler);
        return ()=>{
            $e77252a287ef94ab$var$changeHandlers.delete(handler);
        };
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps);
}


//# sourceMappingURL=useFocusVisible.main.js.map
