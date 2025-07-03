import {isShadowRoot as $431fbd86ca7dc216$export$af51f0f06c0f328a} from "./domHelpers.mjs";
import {shadowDOM as $lcSu5$shadowDOM} from "@react-stately/flags";

// Source: https://github.com/microsoft/tabster/blob/a89fc5d7e332d48f68d03b1ca6e344489d1c3898/src/Shadowdomize/DOMFunctions.ts#L16


function $d4ee10de306f2510$export$4282f70798064fe0(node, otherNode) {
    if (!(0, $lcSu5$shadowDOM)()) return otherNode && node ? node.contains(otherNode) : false;
    if (!node || !otherNode) return false;
    let currentNode = otherNode;
    while(currentNode !== null){
        if (currentNode === node) return true;
        if (currentNode.tagName === 'SLOT' && currentNode.assignedSlot) // Element is slotted
        currentNode = currentNode.assignedSlot.parentNode;
        else if ((0, $431fbd86ca7dc216$export$af51f0f06c0f328a)(currentNode)) // Element is in shadow root
        currentNode = currentNode.host;
        else currentNode = currentNode.parentNode;
    }
    return false;
}
const $d4ee10de306f2510$export$cd4e5573fbe2b576 = (doc = document)=>{
    var _activeElement_shadowRoot;
    if (!(0, $lcSu5$shadowDOM)()) return doc.activeElement;
    let activeElement = doc.activeElement;
    while(activeElement && 'shadowRoot' in activeElement && ((_activeElement_shadowRoot = activeElement.shadowRoot) === null || _activeElement_shadowRoot === void 0 ? void 0 : _activeElement_shadowRoot.activeElement))activeElement = activeElement.shadowRoot.activeElement;
    return activeElement;
};
function $d4ee10de306f2510$export$e58f029f0fbfdb29(event) {
    if ((0, $lcSu5$shadowDOM)() && event.target.shadowRoot) {
        if (event.composedPath) return event.composedPath()[0];
    }
    return event.target;
}


export {$d4ee10de306f2510$export$4282f70798064fe0 as nodeContains, $d4ee10de306f2510$export$cd4e5573fbe2b576 as getActiveElement, $d4ee10de306f2510$export$e58f029f0fbfdb29 as getEventTarget};
//# sourceMappingURL=DOMFunctions.module.js.map
