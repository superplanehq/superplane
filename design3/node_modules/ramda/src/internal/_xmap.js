var _xfBase =
/*#__PURE__*/
require("./_xfBase.js");

var XMap =
/*#__PURE__*/
function () {
  function XMap(f, xf) {
    this.xf = xf;
    this.f = f;
  }

  XMap.prototype['@@transducer/init'] = _xfBase.init;
  XMap.prototype['@@transducer/result'] = _xfBase.result;

  XMap.prototype['@@transducer/step'] = function (result, input) {
    return this.xf['@@transducer/step'](result, this.f(input));
  };

  return XMap;
}();

var _xmap = function _xmap(f) {
  return function (xf) {
    return new XMap(f, xf);
  };
};

module.exports = _xmap;