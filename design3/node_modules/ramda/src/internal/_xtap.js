var _xfBase =
/*#__PURE__*/
require("./_xfBase.js");

var XTap =
/*#__PURE__*/
function () {
  function XTap(f, xf) {
    this.xf = xf;
    this.f = f;
  }

  XTap.prototype['@@transducer/init'] = _xfBase.init;
  XTap.prototype['@@transducer/result'] = _xfBase.result;

  XTap.prototype['@@transducer/step'] = function (result, input) {
    this.f(input);
    return this.xf['@@transducer/step'](result, input);
  };

  return XTap;
}();

function _xtap(f) {
  return function (xf) {
    return new XTap(f, xf);
  };
}

module.exports = _xtap;