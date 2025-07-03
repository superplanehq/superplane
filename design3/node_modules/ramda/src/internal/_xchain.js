var _flatCat =
/*#__PURE__*/
require("./_flatCat.js");

var _xmap =
/*#__PURE__*/
require("./_xmap.js");

function _xchain(f) {
  return function (xf) {
    return _xmap(f)(_flatCat(xf));
  };
}

module.exports = _xchain;