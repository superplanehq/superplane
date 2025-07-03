import {getOwnerDocument as $hpDQO$getOwnerDocument, getActiveElement as $hpDQO$getActiveElement} from "@react-aria/utils";


function $55f9b1ae81f22853$export$76e4e37e5339496d(to) {
    let from = $55f9b1ae81f22853$export$759df0d867455a91((0, $hpDQO$getOwnerDocument)(to));
    if (from !== to) {
        if (from) $55f9b1ae81f22853$export$6c5dc7e81d2cc29a(from, to);
        if (to) $55f9b1ae81f22853$export$2b35b76d2e30e129(to, from);
    }
}
function $55f9b1ae81f22853$export$6c5dc7e81d2cc29a(from, to) {
    from.dispatchEvent(new FocusEvent('blur', {
        relatedTarget: to
    }));
    from.dispatchEvent(new FocusEvent('focusout', {
        bubbles: true,
        relatedTarget: to
    }));
}
function $55f9b1ae81f22853$export$2b35b76d2e30e129(to, from) {
    to.dispatchEvent(new FocusEvent('focus', {
        relatedTarget: from
    }));
    to.dispatchEvent(new FocusEvent('focusin', {
        bubbles: true,
        relatedTarget: from
    }));
}
function $55f9b1ae81f22853$export$759df0d867455a91(document) {
    let activeElement = (0, $hpDQO$getActiveElement)(document);
    let activeDescendant = activeElement === null || activeElement === void 0 ? void 0 : activeElement.getAttribute('aria-activedescendant');
    if (activeDescendant) return document.getElementById(activeDescendant) || activeElement;
    return activeElement;
}


export {$55f9b1ae81f22853$export$76e4e37e5339496d as moveVirtualFocus, $55f9b1ae81f22853$export$759df0d867455a91 as getVirtuallyFocusedElement, $55f9b1ae81f22853$export$6c5dc7e81d2cc29a as dispatchVirtualBlur, $55f9b1ae81f22853$export$2b35b76d2e30e129 as dispatchVirtualFocus};
//# sourceMappingURL=virtualFocus.module.js.map
