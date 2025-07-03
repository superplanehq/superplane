var $2Y3Ap$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "inertValue", () => $13915169b1e4142c$export$a9d04c5684123369);

function $13915169b1e4142c$export$a9d04c5684123369(value) {
    const pieces = (0, $2Y3Ap$react.version).split('.');
    const major = parseInt(pieces[0], 10);
    if (major >= 19) return value;
    // compatibility with React < 19
    return value ? 'true' : undefined;
}


//# sourceMappingURL=inertValue.main.js.map
