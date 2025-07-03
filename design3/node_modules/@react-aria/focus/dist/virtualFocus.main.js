var $jMsK8$reactariautils = require("@react-aria/utils");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "moveVirtualFocus", () => $a756eb2d3b28d089$export$76e4e37e5339496d);
$parcel$export(module.exports, "getVirtuallyFocusedElement", () => $a756eb2d3b28d089$export$759df0d867455a91);
$parcel$export(module.exports, "dispatchVirtualBlur", () => $a756eb2d3b28d089$export$6c5dc7e81d2cc29a);
$parcel$export(module.exports, "dispatchVirtualFocus", () => $a756eb2d3b28d089$export$2b35b76d2e30e129);

function $a756eb2d3b28d089$export$76e4e37e5339496d(to) {
    let from = $a756eb2d3b28d089$export$759df0d867455a91((0, $jMsK8$reactariautils.getOwnerDocument)(to));
    if (from !== to) {
        if (from) $a756eb2d3b28d089$export$6c5dc7e81d2cc29a(from, to);
        if (to) $a756eb2d3b28d089$export$2b35b76d2e30e129(to, from);
    }
}
function $a756eb2d3b28d089$export$6c5dc7e81d2cc29a(from, to) {
    from.dispatchEvent(new FocusEvent('blur', {
        relatedTarget: to
    }));
    from.dispatchEvent(new FocusEvent('focusout', {
        bubbles: true,
        relatedTarget: to
    }));
}
function $a756eb2d3b28d089$export$2b35b76d2e30e129(to, from) {
    to.dispatchEvent(new FocusEvent('focus', {
        relatedTarget: from
    }));
    to.dispatchEvent(new FocusEvent('focusin', {
        bubbles: true,
        relatedTarget: from
    }));
}
function $a756eb2d3b28d089$export$759df0d867455a91(document) {
    let activeElement = (0, $jMsK8$reactariautils.getActiveElement)(document);
    let activeDescendant = activeElement === null || activeElement === void 0 ? void 0 : activeElement.getAttribute('aria-activedescendant');
    if (activeDescendant) return document.getElementById(activeDescendant) || activeElement;
    return activeElement;
}


//# sourceMappingURL=virtualFocus.main.js.map
