import _xfBase from "./_xfBase.js";

var XDropWhile =
/*#__PURE__*/
function () {
  function XDropWhile(f, xf) {
    this.xf = xf;
    this.f = f;
  }

  XDropWhile.prototype['@@transducer/init'] = _xfBase.init;
  XDropWhile.prototype['@@transducer/result'] = _xfBase.result;

  XDropWhile.prototype['@@transducer/step'] = function (result, input) {
    if (this.f) {
      if (this.f(input)) {
        return result;
      }

      this.f = null;
    }

    return this.xf['@@transducer/step'](result, input);
  };

  return XDropWhile;
}();

export default function _xdropWhile(f) {
  return function (xf) {
    return new XDropWhile(f, xf);
  };
}