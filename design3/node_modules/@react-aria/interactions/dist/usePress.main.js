var $625cf83917e112ad$exports = require("./utils.main.js");
var $f7e14e656343df57$exports = require("./textSelection.main.js");
var $01d3f539e91688c8$exports = require("./context.main.js");
var $bBqCQ$swchelperscjs_class_private_field_getcjs = require("@swc/helpers/cjs/_class_private_field_get.cjs");
var $bBqCQ$swchelperscjs_class_private_field_initcjs = require("@swc/helpers/cjs/_class_private_field_init.cjs");
var $bBqCQ$swchelperscjs_class_private_field_setcjs = require("@swc/helpers/cjs/_class_private_field_set.cjs");
var $bBqCQ$reactariautils = require("@react-aria/utils");
var $bBqCQ$reactdom = require("react-dom");
var $bBqCQ$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "usePress", () => $0294ea432cd92340$export$45712eceda6fad21);
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









function $0294ea432cd92340$var$usePressResponderContext(props) {
    // Consume context from <PressResponder> and merge with props.
    let context = (0, $bBqCQ$react.useContext)((0, $01d3f539e91688c8$exports.PressResponderContext));
    if (context) {
        let { register: register, ...contextProps } = context;
        props = (0, $bBqCQ$reactariautils.mergeProps)(contextProps, props);
        register();
    }
    (0, $bBqCQ$reactariautils.useSyncRef)(context, props.ref);
    return props;
}
var $0294ea432cd92340$var$_shouldStopPropagation = /*#__PURE__*/ new WeakMap();
class $0294ea432cd92340$var$PressEvent {
    continuePropagation() {
        (0, $bBqCQ$swchelperscjs_class_private_field_setcjs._)(this, $0294ea432cd92340$var$_shouldStopPropagation, false);
    }
    get shouldStopPropagation() {
        return (0, $bBqCQ$swchelperscjs_class_private_field_getcjs._)(this, $0294ea432cd92340$var$_shouldStopPropagation);
    }
    constructor(type, pointerType, originalEvent, state){
        (0, $bBqCQ$swchelperscjs_class_private_field_initcjs._)(this, $0294ea432cd92340$var$_shouldStopPropagation, {
            writable: true,
            value: void 0
        });
        (0, $bBqCQ$swchelperscjs_class_private_field_setcjs._)(this, $0294ea432cd92340$var$_shouldStopPropagation, true);
        var _state_target;
        let currentTarget = (_state_target = state === null || state === void 0 ? void 0 : state.target) !== null && _state_target !== void 0 ? _state_target : originalEvent.currentTarget;
        const rect = currentTarget === null || currentTarget === void 0 ? void 0 : currentTarget.getBoundingClientRect();
        let x, y = 0;
        let clientX, clientY = null;
        if (originalEvent.clientX != null && originalEvent.clientY != null) {
            clientX = originalEvent.clientX;
            clientY = originalEvent.clientY;
        }
        if (rect) {
            if (clientX != null && clientY != null) {
                x = clientX - rect.left;
                y = clientY - rect.top;
            } else {
                x = rect.width / 2;
                y = rect.height / 2;
            }
        }
        this.type = type;
        this.pointerType = pointerType;
        this.target = originalEvent.currentTarget;
        this.shiftKey = originalEvent.shiftKey;
        this.metaKey = originalEvent.metaKey;
        this.ctrlKey = originalEvent.ctrlKey;
        this.altKey = originalEvent.altKey;
        this.x = x;
        this.y = y;
    }
}
const $0294ea432cd92340$var$LINK_CLICKED = Symbol('linkClicked');
const $0294ea432cd92340$var$STYLE_ID = 'react-aria-pressable-style';
const $0294ea432cd92340$var$PRESSABLE_ATTRIBUTE = 'data-react-aria-pressable';
function $0294ea432cd92340$export$45712eceda6fad21(props) {
    let { onPress: onPress, onPressChange: onPressChange, onPressStart: onPressStart, onPressEnd: onPressEnd, onPressUp: onPressUp, onClick: onClick, isDisabled: isDisabled, isPressed: isPressedProp, preventFocusOnPress: preventFocusOnPress, shouldCancelOnPointerExit: shouldCancelOnPointerExit, allowTextSelectionOnPress: allowTextSelectionOnPress, ref: domRef, ...domProps } = $0294ea432cd92340$var$usePressResponderContext(props);
    let [isPressed, setPressed] = (0, $bBqCQ$react.useState)(false);
    let ref = (0, $bBqCQ$react.useRef)({
        isPressed: false,
        ignoreEmulatedMouseEvents: false,
        didFirePressStart: false,
        isTriggeringEvent: false,
        activePointerId: null,
        target: null,
        isOverTarget: false,
        pointerType: null,
        disposables: []
    });
    let { addGlobalListener: addGlobalListener, removeAllGlobalListeners: removeAllGlobalListeners } = (0, $bBqCQ$reactariautils.useGlobalListeners)();
    let triggerPressStart = (0, $bBqCQ$reactariautils.useEffectEvent)((originalEvent, pointerType)=>{
        let state = ref.current;
        if (isDisabled || state.didFirePressStart) return false;
        let shouldStopPropagation = true;
        state.isTriggeringEvent = true;
        if (onPressStart) {
            let event = new $0294ea432cd92340$var$PressEvent('pressstart', pointerType, originalEvent);
            onPressStart(event);
            shouldStopPropagation = event.shouldStopPropagation;
        }
        if (onPressChange) onPressChange(true);
        state.isTriggeringEvent = false;
        state.didFirePressStart = true;
        setPressed(true);
        return shouldStopPropagation;
    });
    let triggerPressEnd = (0, $bBqCQ$reactariautils.useEffectEvent)((originalEvent, pointerType, wasPressed = true)=>{
        let state = ref.current;
        if (!state.didFirePressStart) return false;
        state.didFirePressStart = false;
        state.isTriggeringEvent = true;
        let shouldStopPropagation = true;
        if (onPressEnd) {
            let event = new $0294ea432cd92340$var$PressEvent('pressend', pointerType, originalEvent);
            onPressEnd(event);
            shouldStopPropagation = event.shouldStopPropagation;
        }
        if (onPressChange) onPressChange(false);
        setPressed(false);
        if (onPress && wasPressed && !isDisabled) {
            let event = new $0294ea432cd92340$var$PressEvent('press', pointerType, originalEvent);
            onPress(event);
            shouldStopPropagation && (shouldStopPropagation = event.shouldStopPropagation);
        }
        state.isTriggeringEvent = false;
        return shouldStopPropagation;
    });
    let triggerPressUp = (0, $bBqCQ$reactariautils.useEffectEvent)((originalEvent, pointerType)=>{
        let state = ref.current;
        if (isDisabled) return false;
        if (onPressUp) {
            state.isTriggeringEvent = true;
            let event = new $0294ea432cd92340$var$PressEvent('pressup', pointerType, originalEvent);
            onPressUp(event);
            state.isTriggeringEvent = false;
            return event.shouldStopPropagation;
        }
        return true;
    });
    let cancel = (0, $bBqCQ$reactariautils.useEffectEvent)((e)=>{
        let state = ref.current;
        if (state.isPressed && state.target) {
            if (state.didFirePressStart && state.pointerType != null) triggerPressEnd($0294ea432cd92340$var$createEvent(state.target, e), state.pointerType, false);
            state.isPressed = false;
            state.isOverTarget = false;
            state.activePointerId = null;
            state.pointerType = null;
            removeAllGlobalListeners();
            if (!allowTextSelectionOnPress) (0, $f7e14e656343df57$exports.restoreTextSelection)(state.target);
            for (let dispose of state.disposables)dispose();
            state.disposables = [];
        }
    });
    let cancelOnPointerExit = (0, $bBqCQ$reactariautils.useEffectEvent)((e)=>{
        if (shouldCancelOnPointerExit) cancel(e);
    });
    let triggerClick = (0, $bBqCQ$reactariautils.useEffectEvent)((e)=>{
        onClick === null || onClick === void 0 ? void 0 : onClick(e);
    });
    let triggerSyntheticClick = (0, $bBqCQ$reactariautils.useEffectEvent)((e, target)=>{
        // Some third-party libraries pass in onClick instead of onPress.
        // Create a fake mouse event and trigger onClick as well.
        // This matches the browser's native activation behavior for certain elements (e.g. button).
        // https://html.spec.whatwg.org/#activation
        // https://html.spec.whatwg.org/#fire-a-synthetic-pointer-event
        if (onClick) {
            let event = new MouseEvent('click', e);
            (0, $625cf83917e112ad$exports.setEventTarget)(event, target);
            onClick((0, $625cf83917e112ad$exports.createSyntheticEvent)(event));
        }
    });
    let pressProps = (0, $bBqCQ$react.useMemo)(()=>{
        let state = ref.current;
        let pressProps = {
            onKeyDown (e) {
                if ($0294ea432cd92340$var$isValidKeyboardEvent(e.nativeEvent, e.currentTarget) && (0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) {
                    var _state_metaKeyEvents;
                    if ($0294ea432cd92340$var$shouldPreventDefaultKeyboard((0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent), e.key)) e.preventDefault();
                    // If the event is repeating, it may have started on a different element
                    // after which focus moved to the current element. Ignore these events and
                    // only handle the first key down event.
                    let shouldStopPropagation = true;
                    if (!state.isPressed && !e.repeat) {
                        state.target = e.currentTarget;
                        state.isPressed = true;
                        state.pointerType = 'keyboard';
                        shouldStopPropagation = triggerPressStart(e, 'keyboard');
                        // Focus may move before the key up event, so register the event on the document
                        // instead of the same element where the key down event occurred. Make it capturing so that it will trigger
                        // before stopPropagation from useKeyboard on a child element may happen and thus we can still call triggerPress for the parent element.
                        let originalTarget = e.currentTarget;
                        let pressUp = (e)=>{
                            if ($0294ea432cd92340$var$isValidKeyboardEvent(e, originalTarget) && !e.repeat && (0, $bBqCQ$reactariautils.nodeContains)(originalTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e)) && state.target) triggerPressUp($0294ea432cd92340$var$createEvent(state.target, e), 'keyboard');
                        };
                        addGlobalListener((0, $bBqCQ$reactariautils.getOwnerDocument)(e.currentTarget), 'keyup', (0, $bBqCQ$reactariautils.chain)(pressUp, onKeyUp), true);
                    }
                    if (shouldStopPropagation) e.stopPropagation();
                    // Keep track of the keydown events that occur while the Meta (e.g. Command) key is held.
                    // macOS has a bug where keyup events are not fired while the Meta key is down.
                    // When the Meta key itself is released we will get an event for that, and we'll act as if
                    // all of these other keys were released as well.
                    // https://bugs.chromium.org/p/chromium/issues/detail?id=1393524
                    // https://bugs.webkit.org/show_bug.cgi?id=55291
                    // https://bugzilla.mozilla.org/show_bug.cgi?id=1299553
                    if (e.metaKey && (0, $bBqCQ$reactariautils.isMac)()) (_state_metaKeyEvents = state.metaKeyEvents) === null || _state_metaKeyEvents === void 0 ? void 0 : _state_metaKeyEvents.set(e.key, e.nativeEvent);
                } else if (e.key === 'Meta') state.metaKeyEvents = new Map();
            },
            onClick (e) {
                if (e && !(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                if (e && e.button === 0 && !state.isTriggeringEvent && !(0, $bBqCQ$reactariautils.openLink).isOpening) {
                    let shouldStopPropagation = true;
                    if (isDisabled) e.preventDefault();
                    // If triggered from a screen reader or by using element.click(),
                    // trigger as if it were a keyboard click.
                    if (!state.ignoreEmulatedMouseEvents && !state.isPressed && (state.pointerType === 'virtual' || (0, $bBqCQ$reactariautils.isVirtualClick)(e.nativeEvent))) {
                        let stopPressStart = triggerPressStart(e, 'virtual');
                        let stopPressUp = triggerPressUp(e, 'virtual');
                        let stopPressEnd = triggerPressEnd(e, 'virtual');
                        triggerClick(e);
                        shouldStopPropagation = stopPressStart && stopPressUp && stopPressEnd;
                    } else if (state.isPressed && state.pointerType !== 'keyboard') {
                        let pointerType = state.pointerType || e.nativeEvent.pointerType || 'virtual';
                        let stopPressUp = triggerPressUp($0294ea432cd92340$var$createEvent(e.currentTarget, e), pointerType);
                        let stopPressEnd = triggerPressEnd($0294ea432cd92340$var$createEvent(e.currentTarget, e), pointerType, true);
                        shouldStopPropagation = stopPressUp && stopPressEnd;
                        state.isOverTarget = false;
                        triggerClick(e);
                        cancel(e);
                    }
                    state.ignoreEmulatedMouseEvents = false;
                    if (shouldStopPropagation) e.stopPropagation();
                }
            }
        };
        let onKeyUp = (e)=>{
            var _state_metaKeyEvents;
            if (state.isPressed && state.target && $0294ea432cd92340$var$isValidKeyboardEvent(e, state.target)) {
                var _state_metaKeyEvents1;
                if ($0294ea432cd92340$var$shouldPreventDefaultKeyboard((0, $bBqCQ$reactariautils.getEventTarget)(e), e.key)) e.preventDefault();
                let target = (0, $bBqCQ$reactariautils.getEventTarget)(e);
                let wasPressed = (0, $bBqCQ$reactariautils.nodeContains)(state.target, (0, $bBqCQ$reactariautils.getEventTarget)(e));
                triggerPressEnd($0294ea432cd92340$var$createEvent(state.target, e), 'keyboard', wasPressed);
                if (wasPressed) triggerSyntheticClick(e, state.target);
                removeAllGlobalListeners();
                // If a link was triggered with a key other than Enter, open the URL ourselves.
                // This means the link has a role override, and the default browser behavior
                // only applies when using the Enter key.
                if (e.key !== 'Enter' && $0294ea432cd92340$var$isHTMLAnchorLink(state.target) && (0, $bBqCQ$reactariautils.nodeContains)(state.target, target) && !e[$0294ea432cd92340$var$LINK_CLICKED]) {
                    // Store a hidden property on the event so we only trigger link click once,
                    // even if there are multiple usePress instances attached to the element.
                    e[$0294ea432cd92340$var$LINK_CLICKED] = true;
                    (0, $bBqCQ$reactariautils.openLink)(state.target, e, false);
                }
                state.isPressed = false;
                (_state_metaKeyEvents1 = state.metaKeyEvents) === null || _state_metaKeyEvents1 === void 0 ? void 0 : _state_metaKeyEvents1.delete(e.key);
            } else if (e.key === 'Meta' && ((_state_metaKeyEvents = state.metaKeyEvents) === null || _state_metaKeyEvents === void 0 ? void 0 : _state_metaKeyEvents.size)) {
                var _state_target;
                // If we recorded keydown events that occurred while the Meta key was pressed,
                // and those haven't received keyup events already, fire keyup events ourselves.
                // See comment above for more info about the macOS bug causing this.
                let events = state.metaKeyEvents;
                state.metaKeyEvents = undefined;
                for (let event of events.values())(_state_target = state.target) === null || _state_target === void 0 ? void 0 : _state_target.dispatchEvent(new KeyboardEvent('keyup', event));
            }
        };
        if (typeof PointerEvent !== 'undefined') {
            pressProps.onPointerDown = (e)=>{
                // Only handle left clicks, and ignore events that bubbled through portals.
                if (e.button !== 0 || !(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                // iOS safari fires pointer events from VoiceOver with incorrect coordinates/target.
                // Ignore and let the onClick handler take care of it instead.
                // https://bugs.webkit.org/show_bug.cgi?id=222627
                // https://bugs.webkit.org/show_bug.cgi?id=223202
                if ((0, $bBqCQ$reactariautils.isVirtualPointerEvent)(e.nativeEvent)) {
                    state.pointerType = 'virtual';
                    return;
                }
                state.pointerType = e.pointerType;
                let shouldStopPropagation = true;
                if (!state.isPressed) {
                    state.isPressed = true;
                    state.isOverTarget = true;
                    state.activePointerId = e.pointerId;
                    state.target = e.currentTarget;
                    if (!allowTextSelectionOnPress) (0, $f7e14e656343df57$exports.disableTextSelection)(state.target);
                    shouldStopPropagation = triggerPressStart(e, state.pointerType);
                    // Release pointer capture so that touch interactions can leave the original target.
                    // This enables onPointerLeave and onPointerEnter to fire.
                    let target = (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent);
                    if ('releasePointerCapture' in target) target.releasePointerCapture(e.pointerId);
                    addGlobalListener((0, $bBqCQ$reactariautils.getOwnerDocument)(e.currentTarget), 'pointerup', onPointerUp, false);
                    addGlobalListener((0, $bBqCQ$reactariautils.getOwnerDocument)(e.currentTarget), 'pointercancel', onPointerCancel, false);
                }
                if (shouldStopPropagation) e.stopPropagation();
            };
            pressProps.onMouseDown = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                if (e.button === 0) {
                    if (preventFocusOnPress) {
                        let dispose = (0, $625cf83917e112ad$exports.preventFocus)(e.target);
                        if (dispose) state.disposables.push(dispose);
                    }
                    e.stopPropagation();
                }
            };
            pressProps.onPointerUp = (e)=>{
                // iOS fires pointerup with zero width and height, so check the pointerType recorded during pointerdown.
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent)) || state.pointerType === 'virtual') return;
                // Only handle left clicks. If isPressed is true, delay until onClick.
                if (e.button === 0 && !state.isPressed) triggerPressUp(e, state.pointerType || e.pointerType);
            };
            pressProps.onPointerEnter = (e)=>{
                if (e.pointerId === state.activePointerId && state.target && !state.isOverTarget && state.pointerType != null) {
                    state.isOverTarget = true;
                    triggerPressStart($0294ea432cd92340$var$createEvent(state.target, e), state.pointerType);
                }
            };
            pressProps.onPointerLeave = (e)=>{
                if (e.pointerId === state.activePointerId && state.target && state.isOverTarget && state.pointerType != null) {
                    state.isOverTarget = false;
                    triggerPressEnd($0294ea432cd92340$var$createEvent(state.target, e), state.pointerType, false);
                    cancelOnPointerExit(e);
                }
            };
            let onPointerUp = (e)=>{
                if (e.pointerId === state.activePointerId && state.isPressed && e.button === 0 && state.target) {
                    if ((0, $bBqCQ$reactariautils.nodeContains)(state.target, (0, $bBqCQ$reactariautils.getEventTarget)(e)) && state.pointerType != null) {
                        // Wait for onClick to fire onPress. This avoids browser issues when the DOM
                        // is mutated between onPointerUp and onClick, and is more compatible with third party libraries.
                        // https://github.com/adobe/react-spectrum/issues/1513
                        // https://issues.chromium.org/issues/40732224
                        // However, iOS and Android do not focus or fire onClick after a long press.
                        // We work around this by triggering a click ourselves after a timeout.
                        // This timeout is canceled during the click event in case the real one fires first.
                        // The timeout must be at least 32ms, because Safari on iOS delays the click event on
                        // non-form elements without certain ARIA roles (for hover emulation).
                        // https://github.com/WebKit/WebKit/blob/dccfae42bb29bd4bdef052e469f604a9387241c0/Source/WebKit/WebProcess/WebPage/ios/WebPageIOS.mm#L875-L892
                        let clicked = false;
                        let timeout = setTimeout(()=>{
                            if (state.isPressed && state.target instanceof HTMLElement) {
                                if (clicked) cancel(e);
                                else {
                                    (0, $bBqCQ$reactariautils.focusWithoutScrolling)(state.target);
                                    state.target.click();
                                }
                            }
                        }, 80);
                        // Use a capturing listener to track if a click occurred.
                        // If stopPropagation is called it may never reach our handler.
                        addGlobalListener(e.currentTarget, 'click', ()=>clicked = true, true);
                        state.disposables.push(()=>clearTimeout(timeout));
                    } else cancel(e);
                    // Ignore subsequent onPointerLeave event before onClick on touch devices.
                    state.isOverTarget = false;
                }
            };
            let onPointerCancel = (e)=>{
                cancel(e);
            };
            pressProps.onDragStart = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                // Safari does not call onPointerCancel when a drag starts, whereas Chrome and Firefox do.
                cancel(e);
            };
        } else if (process.env.NODE_ENV === 'test') {
            // NOTE: this fallback branch is entirely used by unit tests.
            // All browsers now support pointer events, but JSDOM still does not.
            pressProps.onMouseDown = (e)=>{
                // Only handle left clicks
                if (e.button !== 0 || !(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                if (state.ignoreEmulatedMouseEvents) {
                    e.stopPropagation();
                    return;
                }
                state.isPressed = true;
                state.isOverTarget = true;
                state.target = e.currentTarget;
                state.pointerType = (0, $bBqCQ$reactariautils.isVirtualClick)(e.nativeEvent) ? 'virtual' : 'mouse';
                // Flush sync so that focus moved during react re-renders occurs before we yield back to the browser.
                let shouldStopPropagation = (0, $bBqCQ$reactdom.flushSync)(()=>triggerPressStart(e, state.pointerType));
                if (shouldStopPropagation) e.stopPropagation();
                if (preventFocusOnPress) {
                    let dispose = (0, $625cf83917e112ad$exports.preventFocus)(e.target);
                    if (dispose) state.disposables.push(dispose);
                }
                addGlobalListener((0, $bBqCQ$reactariautils.getOwnerDocument)(e.currentTarget), 'mouseup', onMouseUp, false);
            };
            pressProps.onMouseEnter = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                let shouldStopPropagation = true;
                if (state.isPressed && !state.ignoreEmulatedMouseEvents && state.pointerType != null) {
                    state.isOverTarget = true;
                    shouldStopPropagation = triggerPressStart(e, state.pointerType);
                }
                if (shouldStopPropagation) e.stopPropagation();
            };
            pressProps.onMouseLeave = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                let shouldStopPropagation = true;
                if (state.isPressed && !state.ignoreEmulatedMouseEvents && state.pointerType != null) {
                    state.isOverTarget = false;
                    shouldStopPropagation = triggerPressEnd(e, state.pointerType, false);
                    cancelOnPointerExit(e);
                }
                if (shouldStopPropagation) e.stopPropagation();
            };
            pressProps.onMouseUp = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                if (!state.ignoreEmulatedMouseEvents && e.button === 0 && !state.isPressed) triggerPressUp(e, state.pointerType || 'mouse');
            };
            let onMouseUp = (e)=>{
                // Only handle left clicks
                if (e.button !== 0) return;
                if (state.ignoreEmulatedMouseEvents) {
                    state.ignoreEmulatedMouseEvents = false;
                    return;
                }
                if (state.target && state.target.contains(e.target) && state.pointerType != null) ;
                else cancel(e);
                state.isOverTarget = false;
            };
            pressProps.onTouchStart = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                let touch = $0294ea432cd92340$var$getTouchFromEvent(e.nativeEvent);
                if (!touch) return;
                state.activePointerId = touch.identifier;
                state.ignoreEmulatedMouseEvents = true;
                state.isOverTarget = true;
                state.isPressed = true;
                state.target = e.currentTarget;
                state.pointerType = 'touch';
                if (!allowTextSelectionOnPress) (0, $f7e14e656343df57$exports.disableTextSelection)(state.target);
                let shouldStopPropagation = triggerPressStart($0294ea432cd92340$var$createTouchEvent(state.target, e), state.pointerType);
                if (shouldStopPropagation) e.stopPropagation();
                addGlobalListener((0, $bBqCQ$reactariautils.getOwnerWindow)(e.currentTarget), 'scroll', onScroll, true);
            };
            pressProps.onTouchMove = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                if (!state.isPressed) {
                    e.stopPropagation();
                    return;
                }
                let touch = $0294ea432cd92340$var$getTouchById(e.nativeEvent, state.activePointerId);
                let shouldStopPropagation = true;
                if (touch && $0294ea432cd92340$var$isOverTarget(touch, e.currentTarget)) {
                    if (!state.isOverTarget && state.pointerType != null) {
                        state.isOverTarget = true;
                        shouldStopPropagation = triggerPressStart($0294ea432cd92340$var$createTouchEvent(state.target, e), state.pointerType);
                    }
                } else if (state.isOverTarget && state.pointerType != null) {
                    state.isOverTarget = false;
                    shouldStopPropagation = triggerPressEnd($0294ea432cd92340$var$createTouchEvent(state.target, e), state.pointerType, false);
                    cancelOnPointerExit($0294ea432cd92340$var$createTouchEvent(state.target, e));
                }
                if (shouldStopPropagation) e.stopPropagation();
            };
            pressProps.onTouchEnd = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                if (!state.isPressed) {
                    e.stopPropagation();
                    return;
                }
                let touch = $0294ea432cd92340$var$getTouchById(e.nativeEvent, state.activePointerId);
                let shouldStopPropagation = true;
                if (touch && $0294ea432cd92340$var$isOverTarget(touch, e.currentTarget) && state.pointerType != null) {
                    triggerPressUp($0294ea432cd92340$var$createTouchEvent(state.target, e), state.pointerType);
                    shouldStopPropagation = triggerPressEnd($0294ea432cd92340$var$createTouchEvent(state.target, e), state.pointerType);
                    triggerSyntheticClick(e.nativeEvent, state.target);
                } else if (state.isOverTarget && state.pointerType != null) shouldStopPropagation = triggerPressEnd($0294ea432cd92340$var$createTouchEvent(state.target, e), state.pointerType, false);
                if (shouldStopPropagation) e.stopPropagation();
                state.isPressed = false;
                state.activePointerId = null;
                state.isOverTarget = false;
                state.ignoreEmulatedMouseEvents = true;
                if (state.target && !allowTextSelectionOnPress) (0, $f7e14e656343df57$exports.restoreTextSelection)(state.target);
                removeAllGlobalListeners();
            };
            pressProps.onTouchCancel = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                e.stopPropagation();
                if (state.isPressed) cancel($0294ea432cd92340$var$createTouchEvent(state.target, e));
            };
            let onScroll = (e)=>{
                if (state.isPressed && (0, $bBqCQ$reactariautils.nodeContains)((0, $bBqCQ$reactariautils.getEventTarget)(e), state.target)) cancel({
                    currentTarget: state.target,
                    shiftKey: false,
                    ctrlKey: false,
                    metaKey: false,
                    altKey: false
                });
            };
            pressProps.onDragStart = (e)=>{
                if (!(0, $bBqCQ$reactariautils.nodeContains)(e.currentTarget, (0, $bBqCQ$reactariautils.getEventTarget)(e.nativeEvent))) return;
                cancel(e);
            };
        }
        return pressProps;
    }, [
        addGlobalListener,
        isDisabled,
        preventFocusOnPress,
        removeAllGlobalListeners,
        allowTextSelectionOnPress,
        cancel,
        cancelOnPointerExit,
        triggerPressEnd,
        triggerPressStart,
        triggerPressUp,
        triggerClick,
        triggerSyntheticClick
    ]);
    // Avoid onClick delay for double tap to zoom by default.
    (0, $bBqCQ$react.useEffect)(()=>{
        if (!domRef || process.env.NODE_ENV === 'test') return;
        const ownerDocument = (0, $bBqCQ$reactariautils.getOwnerDocument)(domRef.current);
        if (!ownerDocument || !ownerDocument.head || ownerDocument.getElementById($0294ea432cd92340$var$STYLE_ID)) return;
        const style = ownerDocument.createElement('style');
        style.id = $0294ea432cd92340$var$STYLE_ID;
        // touchAction: 'manipulation' is supposed to be equivalent, but in
        // Safari it causes onPointerCancel not to fire on scroll.
        // https://bugs.webkit.org/show_bug.cgi?id=240917
        style.textContent = `
@layer {
  [${$0294ea432cd92340$var$PRESSABLE_ATTRIBUTE}] {
    touch-action: pan-x pan-y pinch-zoom;
  }
}
    `.trim();
        ownerDocument.head.prepend(style);
    }, [
        domRef
    ]);
    // Remove user-select: none in case component unmounts immediately after pressStart
    (0, $bBqCQ$react.useEffect)(()=>{
        let state = ref.current;
        return ()=>{
            var _state_target;
            if (!allowTextSelectionOnPress) (0, $f7e14e656343df57$exports.restoreTextSelection)((_state_target = state.target) !== null && _state_target !== void 0 ? _state_target : undefined);
            for (let dispose of state.disposables)dispose();
            state.disposables = [];
        };
    }, [
        allowTextSelectionOnPress
    ]);
    return {
        isPressed: isPressedProp || isPressed,
        pressProps: (0, $bBqCQ$reactariautils.mergeProps)(domProps, pressProps, {
            [$0294ea432cd92340$var$PRESSABLE_ATTRIBUTE]: true
        })
    };
}
function $0294ea432cd92340$var$isHTMLAnchorLink(target) {
    return target.tagName === 'A' && target.hasAttribute('href');
}
function $0294ea432cd92340$var$isValidKeyboardEvent(event, currentTarget) {
    const { key: key, code: code } = event;
    const element = currentTarget;
    const role = element.getAttribute('role');
    // Accessibility for keyboards. Space and Enter only.
    // "Spacebar" is for IE 11
    return (key === 'Enter' || key === ' ' || key === 'Spacebar' || code === 'Space') && !(element instanceof (0, $bBqCQ$reactariautils.getOwnerWindow)(element).HTMLInputElement && !$0294ea432cd92340$var$isValidInputKey(element, key) || element instanceof (0, $bBqCQ$reactariautils.getOwnerWindow)(element).HTMLTextAreaElement || element.isContentEditable) && // Links should only trigger with Enter key
    !((role === 'link' || !role && $0294ea432cd92340$var$isHTMLAnchorLink(element)) && key !== 'Enter');
}
function $0294ea432cd92340$var$getTouchFromEvent(event) {
    const { targetTouches: targetTouches } = event;
    if (targetTouches.length > 0) return targetTouches[0];
    return null;
}
function $0294ea432cd92340$var$getTouchById(event, pointerId) {
    const changedTouches = event.changedTouches;
    for(let i = 0; i < changedTouches.length; i++){
        const touch = changedTouches[i];
        if (touch.identifier === pointerId) return touch;
    }
    return null;
}
function $0294ea432cd92340$var$createTouchEvent(target, e) {
    let clientX = 0;
    let clientY = 0;
    if (e.targetTouches && e.targetTouches.length === 1) {
        clientX = e.targetTouches[0].clientX;
        clientY = e.targetTouches[0].clientY;
    }
    return {
        currentTarget: target,
        shiftKey: e.shiftKey,
        ctrlKey: e.ctrlKey,
        metaKey: e.metaKey,
        altKey: e.altKey,
        clientX: clientX,
        clientY: clientY
    };
}
function $0294ea432cd92340$var$createEvent(target, e) {
    let clientX = e.clientX;
    let clientY = e.clientY;
    return {
        currentTarget: target,
        shiftKey: e.shiftKey,
        ctrlKey: e.ctrlKey,
        metaKey: e.metaKey,
        altKey: e.altKey,
        clientX: clientX,
        clientY: clientY
    };
}
function $0294ea432cd92340$var$getPointClientRect(point) {
    let offsetX = 0;
    let offsetY = 0;
    if (point.width !== undefined) offsetX = point.width / 2;
    else if (point.radiusX !== undefined) offsetX = point.radiusX;
    if (point.height !== undefined) offsetY = point.height / 2;
    else if (point.radiusY !== undefined) offsetY = point.radiusY;
    return {
        top: point.clientY - offsetY,
        right: point.clientX + offsetX,
        bottom: point.clientY + offsetY,
        left: point.clientX - offsetX
    };
}
function $0294ea432cd92340$var$areRectanglesOverlapping(a, b) {
    // check if they cannot overlap on x axis
    if (a.left > b.right || b.left > a.right) return false;
    // check if they cannot overlap on y axis
    if (a.top > b.bottom || b.top > a.bottom) return false;
    return true;
}
function $0294ea432cd92340$var$isOverTarget(point, target) {
    let rect = target.getBoundingClientRect();
    let pointRect = $0294ea432cd92340$var$getPointClientRect(point);
    return $0294ea432cd92340$var$areRectanglesOverlapping(rect, pointRect);
}
function $0294ea432cd92340$var$shouldPreventDefaultUp(target) {
    if (target instanceof HTMLInputElement) return false;
    if (target instanceof HTMLButtonElement) return target.type !== 'submit' && target.type !== 'reset';
    if ($0294ea432cd92340$var$isHTMLAnchorLink(target)) return false;
    return true;
}
function $0294ea432cd92340$var$shouldPreventDefaultKeyboard(target, key) {
    if (target instanceof HTMLInputElement) return !$0294ea432cd92340$var$isValidInputKey(target, key);
    return $0294ea432cd92340$var$shouldPreventDefaultUp(target);
}
const $0294ea432cd92340$var$nonTextInputTypes = new Set([
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
function $0294ea432cd92340$var$isValidInputKey(target, key) {
    // Only space should toggle checkboxes and radios, not enter.
    return target.type === 'checkbox' || target.type === 'radio' ? key === ' ' : $0294ea432cd92340$var$nonTextInputTypes.has(target.type);
}


//# sourceMappingURL=usePress.main.js.map
