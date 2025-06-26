define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var push = _unmethodize(_setup.ArrayProto.push);

	return push;

});
