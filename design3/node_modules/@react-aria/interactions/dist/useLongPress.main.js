var $0294ea432cd92340$exports = require("./usePress.main.js");
var $5sxTM$reactariautils = require("@react-aria/utils");
var $5sxTM$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useLongPress", () => $3cd7b5d0eebf0ca9$export$c24ed0104d07eab9);
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


const $3cd7b5d0eebf0ca9$var$DEFAULT_THRESHOLD = 500;
function $3cd7b5d0eebf0ca9$export$c24ed0104d07eab9(props) {
    let { isDisabled: isDisabled, onLongPressStart: onLongPressStart, onLongPressEnd: onLongPressEnd, onLongPress: onLongPress, threshold: threshold = $3cd7b5d0eebf0ca9$var$DEFAULT_THRESHOLD, accessibilityDescription: accessibilityDescription } = props;
    const timeRef = (0, $5sxTM$react.useRef)(undefined);
    let { addGlobalListener: addGlobalListener, removeGlobalListener: removeGlobalListener } = (0, $5sxTM$reactariautils.useGlobalListeners)();
    let { pressProps: pressProps } = (0, $0294ea432cd92340$exports.usePress)({
        isDisabled: isDisabled,
        onPressStart (e) {
            e.continuePropagation();
            if (e.pointerType === 'mouse' || e.pointerType === 'touch') {
                if (onLongPressStart) onLongPressStart({
                    ...e,
                    type: 'longpressstart'
                });
                timeRef.current = setTimeout(()=>{
                    // Prevent other usePress handlers from also handling this event.
                    e.target.dispatchEvent(new PointerEvent('pointercancel', {
                        bubbles: true
                    }));
                    // Ensure target is focused. On touch devices, browsers typically focus on pointer up.
                    if ((0, $5sxTM$reactariautils.getOwnerDocument)(e.target).activeElement !== e.target) (0, $5sxTM$reactariautils.focusWithoutScrolling)(e.target);
                    if (onLongPress) onLongPress({
                        ...e,
                        type: 'longpress'
                    });
                    timeRef.current = undefined;
                }, threshold);
                // Prevent context menu, which may be opened on long press on touch devices
                if (e.pointerType === 'touch') {
                    let onContextMenu = (e)=>{
                        e.preventDefault();
                    };
                    addGlobalListener(e.target, 'contextmenu', onContextMenu, {
                        once: true
                    });
                    addGlobalListener(window, 'pointerup', ()=>{
                        // If no contextmenu event is fired quickly after pointerup, remove the handler
                        // so future context menu events outside a long press are not prevented.
                        setTimeout(()=>{
                            removeGlobalListener(e.target, 'contextmenu', onContextMenu);
                        }, 30);
                    }, {
                        once: true
                    });
                }
            }
        },
        onPressEnd (e) {
            if (timeRef.current) clearTimeout(timeRef.current);
            if (onLongPressEnd && (e.pointerType === 'mouse' || e.pointerType === 'touch')) onLongPressEnd({
                ...e,
                type: 'longpressend'
            });
        }
    });
    let descriptionProps = (0, $5sxTM$reactariautils.useDescription)(onLongPress && !isDisabled ? accessibilityDescription : undefined);
    return {
        longPressProps: (0, $5sxTM$reactariautils.mergeProps)(pressProps, descriptionProps)
    };
}


//# sourceMappingURL=useLongPress.main.js.map
