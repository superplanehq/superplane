var $aaa611146751592e$exports = require("./domHelpers.main.js");
var $loak6$reactstatelyflags = require("@react-stately/flags");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "nodeContains", () => $d723bea02f3e2567$export$4282f70798064fe0);
$parcel$export(module.exports, "getActiveElement", () => $d723bea02f3e2567$export$cd4e5573fbe2b576);
$parcel$export(module.exports, "getEventTarget", () => $d723bea02f3e2567$export$e58f029f0fbfdb29);
// Source: https://github.com/microsoft/tabster/blob/a89fc5d7e332d48f68d03b1ca6e344489d1c3898/src/Shadowdomize/DOMFunctions.ts#L16


function $d723bea02f3e2567$export$4282f70798064fe0(node, otherNode) {
    if (!(0, $loak6$reactstatelyflags.shadowDOM)()) return otherNode && node ? node.contains(otherNode) : false;
    if (!node || !otherNode) return false;
    let currentNode = otherNode;
    while(currentNode !== null){
        if (currentNode === node) return true;
        if (currentNode.tagName === 'SLOT' && currentNode.assignedSlot) // Element is slotted
        currentNode = currentNode.assignedSlot.parentNode;
        else if ((0, $aaa611146751592e$exports.isShadowRoot)(currentNode)) // Element is in shadow root
        currentNode = currentNode.host;
        else currentNode = currentNode.parentNode;
    }
    return false;
}
const $d723bea02f3e2567$export$cd4e5573fbe2b576 = (doc = document)=>{
    var _activeElement_shadowRoot;
    if (!(0, $loak6$reactstatelyflags.shadowDOM)()) return doc.activeElement;
    let activeElement = doc.activeElement;
    while(activeElement && 'shadowRoot' in activeElement && ((_activeElement_shadowRoot = activeElement.shadowRoot) === null || _activeElement_shadowRoot === void 0 ? void 0 : _activeElement_shadowRoot.activeElement))activeElement = activeElement.shadowRoot.activeElement;
    return activeElement;
};
function $d723bea02f3e2567$export$e58f029f0fbfdb29(event) {
    if ((0, $loak6$reactstatelyflags.shadowDOM)() && event.target.shadowRoot) {
        if (event.composedPath) return event.composedPath()[0];
    }
    return event.target;
}


//# sourceMappingURL=DOMFunctions.main.js.map
