var $78605a5d7424e31b$exports = require("./useLayoutEffect.main.js");
var $7PjpK$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useDescription", () => $34da4502ea8120db$export$f8aeda7b10753fa1);
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

let $34da4502ea8120db$var$descriptionId = 0;
const $34da4502ea8120db$var$descriptionNodes = new Map();
function $34da4502ea8120db$export$f8aeda7b10753fa1(description) {
    let [id, setId] = (0, $7PjpK$react.useState)();
    (0, $78605a5d7424e31b$exports.useLayoutEffect)(()=>{
        if (!description) return;
        let desc = $34da4502ea8120db$var$descriptionNodes.get(description);
        if (!desc) {
            let id = `react-aria-description-${$34da4502ea8120db$var$descriptionId++}`;
            setId(id);
            let node = document.createElement('div');
            node.id = id;
            node.style.display = 'none';
            node.textContent = description;
            document.body.appendChild(node);
            desc = {
                refCount: 0,
                element: node
            };
            $34da4502ea8120db$var$descriptionNodes.set(description, desc);
        } else setId(desc.element.id);
        desc.refCount++;
        return ()=>{
            if (desc && --desc.refCount === 0) {
                desc.element.remove();
                $34da4502ea8120db$var$descriptionNodes.delete(description);
            }
        };
    }, [
        description
    ]);
    return {
        'aria-describedby': description ? id : undefined
    };
}


//# sourceMappingURL=useDescription.main.js.map
