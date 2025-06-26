define(['./_bindCb', './_setup'], function (_bindCb, _setup) {

  function unmethodize(method) {
    return _bindCb(_setup.call, method);
  }

  return unmethodize;

});
