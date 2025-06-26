define(['./_setup', './_unmethodize'], function (_setup, _unmethodize) {

	var push = _unmethodize(_setup.ArrayProto.push);

	return push;

});
