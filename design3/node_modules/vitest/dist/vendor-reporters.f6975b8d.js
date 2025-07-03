import { performance } from 'node:perf_hooks';
import c from 'picocolors';
import { b as isNode, c as relativePath } from './vendor-index.29282562.js';
import { isAbsolute, relative, dirname, basename, resolve } from 'pathe';
import { s as slash } from './vendor-base.9c08bbd0.js';
import { g as getFullName, h as hasFailedSnapshot } from './vendor-tasks.f9d75aed.js';
import { getSafeTimers, notNullish } from '@vitest/utils';
import { isCI } from 'std-env';
import { getTests, hasFailed, getSuites } from '@vitest/runner/utils';
import { g as getDefaultExportFromCjs } from './vendor-_commonjsHelpers.7d1333e8.js';
import { existsSync, promises } from 'node:fs';
import { parseErrorStacktrace } from '@vitest/utils/source-map';
import { hostname } from 'node:os';
import { createRequire } from 'node:module';

const F_RIGHT = "\u2192";
const F_DOWN = "\u2193";
const F_DOWN_RIGHT = "\u21B3";
const F_POINTER = "\u276F";
const F_DOT = "\xB7";
const F_CHECK = "\u2713";
const F_CROSS = "\xD7";
const F_LONG_DASH = "\u23AF";

function ansiRegex({onlyFirst = false} = {}) {
	const pattern = [
	    '[\\u001B\\u009B][[\\]()#;?]*(?:(?:(?:(?:;[-a-zA-Z\\d\\/#&.:=?%@~_]+)*|[a-zA-Z\\d]+(?:;[-a-zA-Z\\d\\/#&.:=?%@~_]*)*)?\\u0007)',
		'(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-ntqry=><~]))'
	].join('|');

	return new RegExp(pattern, onlyFirst ? undefined : 'g');
}

const regex = ansiRegex();

function stripAnsi(string) {
	if (typeof string !== 'string') {
		throw new TypeError(`Expected a \`string\`, got \`${typeof string}\``);
	}

	// Even though the regex is global, we don't need to reset the `.lastIndex`
	// because unlike `.exec()` and `.test()`, `.replace()` does it automatically
	// and doing it manually has a performance penalty.
	return string.replace(regex, '');
}

const spinnerMap = /* @__PURE__ */ new WeakMap();
const hookSpinnerMap = /* @__PURE__ */ new WeakMap();
const pointer = c.yellow(F_POINTER);
const skipped = c.dim(c.gray(F_DOWN));
function getCols(delta = 0) {
  var _a;
  let length = (_a = process.stdout) == null ? void 0 : _a.columns;
  if (!length || Number.isNaN(length))
    length = 30;
  return Math.max(length + delta, 0);
}
function divider(text, left, right) {
  const cols = getCols();
  if (text) {
    const textLength = stripAnsi(text).length;
    if (left == null && right != null) {
      left = cols - textLength - right;
    } else {
      left = left ?? Math.floor((cols - textLength) / 2);
      right = cols - textLength - left;
    }
    left = Math.max(0, left);
    right = Math.max(0, right);
    return `${F_LONG_DASH.repeat(left)}${text}${F_LONG_DASH.repeat(right)}`;
  }
  return F_LONG_DASH.repeat(cols);
}
function formatTestPath(root, path) {
  var _a;
  if (isAbsolute(path))
    path = relative(root, path);
  const dir = dirname(path);
  const ext = ((_a = path.match(/(\.(spec|test)\.[cm]?[tj]sx?)$/)) == null ? void 0 : _a[0]) || "";
  const base = basename(path, ext);
  return slash(c.dim(`${dir}/`) + c.bold(base)) + c.dim(ext);
}
function renderSnapshotSummary(rootDir, snapshots) {
  const summary = [];
  if (snapshots.added)
    summary.push(c.bold(c.green(`${snapshots.added} written`)));
  if (snapshots.unmatched)
    summary.push(c.bold(c.red(`${snapshots.unmatched} failed`)));
  if (snapshots.updated)
    summary.push(c.bold(c.green(`${snapshots.updated} updated `)));
  if (snapshots.filesRemoved) {
    if (snapshots.didUpdate)
      summary.push(c.bold(c.green(`${snapshots.filesRemoved} files removed `)));
    else
      summary.push(c.bold(c.yellow(`${snapshots.filesRemoved} files obsolete `)));
  }
  if (snapshots.filesRemovedList && snapshots.filesRemovedList.length) {
    const [head, ...tail] = snapshots.filesRemovedList;
    summary.push(`${c.gray(F_DOWN_RIGHT)} ${formatTestPath(rootDir, head)}`);
    tail.forEach((key) => {
      summary.push(`  ${c.gray(F_DOT)} ${formatTestPath(rootDir, key)}`);
    });
  }
  if (snapshots.unchecked) {
    if (snapshots.didUpdate)
      summary.push(c.bold(c.green(`${snapshots.unchecked} removed`)));
    else
      summary.push(c.bold(c.yellow(`${snapshots.unchecked} obsolete`)));
    snapshots.uncheckedKeysByFile.forEach((uncheckedFile) => {
      summary.push(`${c.gray(F_DOWN_RIGHT)} ${formatTestPath(rootDir, uncheckedFile.filePath)}`);
      uncheckedFile.keys.forEach((key) => summary.push(`  ${c.gray(F_DOT)} ${key}`));
    });
  }
  return summary;
}
function countTestErrors(tasks) {
  return tasks.reduce((c2, i) => {
    var _a, _b;
    return c2 + (((_b = (_a = i.result) == null ? void 0 : _a.errors) == null ? void 0 : _b.length) || 0);
  }, 0);
}
function getStateString(tasks, name = "tests", showTotal = true) {
  if (tasks.length === 0)
    return c.dim(`no ${name}`);
  const passed = tasks.filter((i) => {
    var _a;
    return ((_a = i.result) == null ? void 0 : _a.state) === "pass";
  });
  const failed = tasks.filter((i) => {
    var _a;
    return ((_a = i.result) == null ? void 0 : _a.state) === "fail";
  });
  const skipped2 = tasks.filter((i) => i.mode === "skip");
  const todo = tasks.filter((i) => i.mode === "todo");
  return [
    failed.length ? c.bold(c.red(`${failed.length} failed`)) : null,
    passed.length ? c.bold(c.green(`${passed.length} passed`)) : null,
    skipped2.length ? c.yellow(`${skipped2.length} skipped`) : null,
    todo.length ? c.gray(`${todo.length} todo`) : null
  ].filter(Boolean).join(c.dim(" | ")) + (showTotal ? c.gray(` (${tasks.length})`) : "");
}
function getStateSymbol(task) {
  var _a;
  if (task.mode === "skip" || task.mode === "todo")
    return skipped;
  if (!task.result)
    return c.gray("\xB7");
  if (task.result.state === "run") {
    if (task.type === "suite")
      return pointer;
    let spinner = spinnerMap.get(task);
    if (!spinner) {
      spinner = elegantSpinner();
      spinnerMap.set(task, spinner);
    }
    return c.yellow(spinner());
  }
  if (task.result.state === "pass") {
    return ((_a = task.meta) == null ? void 0 : _a.benchmark) ? c.green(F_DOT) : c.green(F_CHECK);
  }
  if (task.result.state === "fail") {
    return task.type === "suite" ? pointer : c.red(F_CROSS);
  }
  return " ";
}
function getHookStateSymbol(task, hookName) {
  var _a, _b;
  const state = (_b = (_a = task.result) == null ? void 0 : _a.hooks) == null ? void 0 : _b[hookName];
  if (state && state === "run") {
    let spinnerMap2 = hookSpinnerMap.get(task);
    if (!spinnerMap2) {
      spinnerMap2 = /* @__PURE__ */ new Map();
      hookSpinnerMap.set(task, spinnerMap2);
    }
    let spinner = spinnerMap2.get(hookName);
    if (!spinner) {
      spinner = elegantSpinner();
      spinnerMap2.set(hookName, spinner);
    }
    return c.yellow(spinner());
  }
}
const spinnerFrames = process.platform === "win32" ? ["-", "\\", "|", "/"] : ["\u280B", "\u2819", "\u2839", "\u2838", "\u283C", "\u2834", "\u2826", "\u2827", "\u2807", "\u280F"];
function elegantSpinner() {
  let index = 0;
  return () => {
    index = ++index % spinnerFrames.length;
    return spinnerFrames[index];
  };
}
function formatTimeString(date) {
  return date.toTimeString().split(" ")[0];
}
function formatProjectName(name, suffix = " ") {
  if (!name)
    return "";
  const index = name.split("").reduce((acc, v, idx) => acc + v.charCodeAt(0) + idx, 0);
  const colors = [
    c.blue,
    c.yellow,
    c.cyan,
    c.green,
    c.magenta
  ];
  return colors[index % colors.length](`|${name}|`) + suffix;
}

var _a;
const BADGE_PADDING = "       ";
const HELP_HINT = `${c.dim("press ")}${c.bold("h")}${c.dim(" to show help")}`;
const HELP_UPDATE_SNAP = c.dim("press ") + c.bold(c.yellow("u")) + c.dim(" to update snapshot");
const HELP_QUITE = `${c.dim("press ")}${c.bold("q")}${c.dim(" to quit")}`;
const WAIT_FOR_CHANGE_PASS = `
${c.bold(c.inverse(c.green(" PASS ")))}${c.green(" Waiting for file changes...")}`;
const WAIT_FOR_CHANGE_FAIL = `
${c.bold(c.inverse(c.red(" FAIL ")))}${c.red(" Tests failed. Watching for file changes...")}`;
const WAIT_FOR_CHANGE_CANCELLED = `
${c.bold(c.inverse(c.red(" CANCELLED ")))}${c.red(" Test run cancelled. Watching for file changes...")}`;
const LAST_RUN_LOG_TIMEOUT = 1500;
class BaseReporter {
  start = 0;
  end = 0;
  watchFilters;
  isTTY = isNode && ((_a = process.stdout) == null ? void 0 : _a.isTTY) && !isCI;
  ctx = void 0;
  _filesInWatchMode = /* @__PURE__ */ new Map();
  _lastRunTimeout = 0;
  _lastRunTimer;
  _lastRunCount = 0;
  _timeStart = /* @__PURE__ */ new Date();
  constructor() {
    this.registerUnhandledRejection();
  }
  get mode() {
    return this.ctx.config.mode;
  }
  onInit(ctx) {
    this.ctx = ctx;
    ctx.logger.printBanner();
    this.start = performance.now();
  }
  relative(path) {
    return relativePath(this.ctx.config.root, path);
  }
  async onFinished(files = this.ctx.state.getFiles(), errors = this.ctx.state.getUnhandledErrors()) {
    this.end = performance.now();
    await this.reportSummary(files, errors);
    if (errors.length) {
      if (!this.ctx.config.dangerouslyIgnoreUnhandledErrors)
        process.exitCode = 1;
    }
  }
  onTaskUpdate(packs) {
    var _a2, _b, _c, _d;
    if (this.isTTY)
      return;
    const logger = this.ctx.logger;
    for (const pack of packs) {
      const task = this.ctx.state.idMap.get(pack[0]);
      if (task && "filepath" in task && ((_a2 = task.result) == null ? void 0 : _a2.state) && ((_b = task.result) == null ? void 0 : _b.state) !== "run") {
        const tests = getTests(task);
        const failed = tests.filter((t) => {
          var _a3;
          return ((_a3 = t.result) == null ? void 0 : _a3.state) === "fail";
        });
        const skipped = tests.filter((t) => t.mode === "skip" || t.mode === "todo");
        let state = c.dim(`${tests.length} test${tests.length > 1 ? "s" : ""}`);
        if (failed.length)
          state += ` ${c.dim("|")} ${c.red(`${failed.length} failed`)}`;
        if (skipped.length)
          state += ` ${c.dim("|")} ${c.yellow(`${skipped.length} skipped`)}`;
        let suffix = c.dim(" (") + state + c.dim(")");
        if (task.result.duration) {
          const color = task.result.duration > this.ctx.config.slowTestThreshold ? c.yellow : c.gray;
          suffix += color(` ${Math.round(task.result.duration)}${c.dim("ms")}`);
        }
        if (this.ctx.config.logHeapUsage && task.result.heap != null)
          suffix += c.magenta(` ${Math.floor(task.result.heap / 1024 / 1024)} MB heap used`);
        let title = ` ${getStateSymbol(task)} `;
        if (task.projectName)
          title += formatProjectName(task.projectName);
        title += `${task.name} ${suffix}`;
        logger.log(title);
        for (const test of failed) {
          logger.log(c.red(`   ${pointer} ${getFullName(test, c.dim(" > "))}`));
          (_d = (_c = test.result) == null ? void 0 : _c.errors) == null ? void 0 : _d.forEach((e) => {
            logger.log(c.red(`     ${F_RIGHT} ${e == null ? void 0 : e.message}`));
          });
        }
      }
    }
  }
  async onWatcherStart(files = this.ctx.state.getFiles(), errors = this.ctx.state.getUnhandledErrors()) {
    this.resetLastRunLog();
    const failed = errors.length > 0 || hasFailed(files);
    const failedSnap = hasFailedSnapshot(files);
    const cancelled = this.ctx.isCancelling;
    if (failed)
      this.ctx.logger.log(WAIT_FOR_CHANGE_FAIL);
    else if (cancelled)
      this.ctx.logger.log(WAIT_FOR_CHANGE_CANCELLED);
    else
      this.ctx.logger.log(WAIT_FOR_CHANGE_PASS);
    const hints = [];
    if (this.mode !== "typecheck")
      hints.push(HELP_HINT);
    if (failedSnap)
      hints.unshift(HELP_UPDATE_SNAP);
    else
      hints.push(HELP_QUITE);
    this.ctx.logger.log(BADGE_PADDING + hints.join(c.dim(", ")));
    if (this._lastRunCount) {
      const LAST_RUN_TEXT = `rerun x${this._lastRunCount}`;
      const LAST_RUN_TEXTS = [
        c.blue(LAST_RUN_TEXT),
        c.gray(LAST_RUN_TEXT),
        c.dim(c.gray(LAST_RUN_TEXT))
      ];
      this.ctx.logger.logUpdate(BADGE_PADDING + LAST_RUN_TEXTS[0]);
      this._lastRunTimeout = 0;
      const { setInterval } = getSafeTimers();
      this._lastRunTimer = setInterval(
        () => {
          this._lastRunTimeout += 1;
          if (this._lastRunTimeout >= LAST_RUN_TEXTS.length)
            this.resetLastRunLog();
          else
            this.ctx.logger.logUpdate(BADGE_PADDING + LAST_RUN_TEXTS[this._lastRunTimeout]);
        },
        LAST_RUN_LOG_TIMEOUT / LAST_RUN_TEXTS.length
      );
    }
  }
  resetLastRunLog() {
    const { clearInterval } = getSafeTimers();
    clearInterval(this._lastRunTimer);
    this._lastRunTimer = void 0;
    this.ctx.logger.logUpdate.clear();
  }
  async onWatcherRerun(files, trigger) {
    this.resetLastRunLog();
    this.watchFilters = files;
    files.forEach((filepath) => {
      let reruns = this._filesInWatchMode.get(filepath) ?? 0;
      this._filesInWatchMode.set(filepath, ++reruns);
    });
    const BADGE = c.inverse(c.bold(c.blue(" RERUN ")));
    const TRIGGER = trigger ? c.dim(` ${this.relative(trigger)}`) : "";
    const FILENAME_PATTERN = this.ctx.filenamePattern ? `${BADGE_PADDING} ${c.dim("Filename pattern: ")}${c.blue(this.ctx.filenamePattern)}
` : "";
    const TESTNAME_PATTERN = this.ctx.configOverride.testNamePattern ? `${BADGE_PADDING} ${c.dim("Test name pattern: ")}${c.blue(String(this.ctx.configOverride.testNamePattern))}
` : "";
    if (files.length > 1) {
      this.ctx.logger.clearFullScreen(`
${BADGE}${TRIGGER}
${FILENAME_PATTERN}${TESTNAME_PATTERN}`);
      this._lastRunCount = 0;
    } else if (files.length === 1) {
      const rerun = this._filesInWatchMode.get(files[0]) ?? 1;
      this._lastRunCount = rerun;
      this.ctx.logger.clearFullScreen(`
${BADGE}${TRIGGER} ${c.blue(`x${rerun}`)}
${FILENAME_PATTERN}${TESTNAME_PATTERN}`);
    }
    this._timeStart = /* @__PURE__ */ new Date();
    this.start = performance.now();
  }
  onUserConsoleLog(log) {
    if (!this.shouldLog(log))
      return;
    const task = log.taskId ? this.ctx.state.idMap.get(log.taskId) : void 0;
    const header = c.gray(log.type + c.dim(` | ${task ? getFullName(task, c.dim(" > ")) : "unknown test"}`));
    process[log.type].write(`${header}
${log.content}
`);
  }
  shouldLog(log) {
    var _a2, _b;
    if (this.ctx.config.silent)
      return false;
    const shouldLog = (_b = (_a2 = this.ctx.config).onConsoleLog) == null ? void 0 : _b.call(_a2, log.content, log.type);
    if (shouldLog === false)
      return shouldLog;
    return true;
  }
  onServerRestart(reason) {
    this.ctx.logger.log(c.bold(c.magenta(
      reason === "config" ? "\nRestarting due to config changes..." : "\nRestarting Vitest..."
    )));
  }
  async reportSummary(files, errors) {
    await this.printErrorsSummary(files, errors);
    if (this.mode === "benchmark")
      await this.reportBenchmarkSummary(files);
    else
      await this.reportTestSummary(files, errors);
  }
  async reportTestSummary(files, errors) {
    const tests = getTests(files);
    const logger = this.ctx.logger;
    const executionTime = this.end - this.start;
    const collectTime = files.reduce((acc, test) => acc + Math.max(0, test.collectDuration || 0), 0);
    const setupTime = files.reduce((acc, test) => acc + Math.max(0, test.setupDuration || 0), 0);
    const testsTime = files.reduce((acc, test) => {
      var _a2;
      return acc + Math.max(0, ((_a2 = test.result) == null ? void 0 : _a2.duration) || 0);
    }, 0);
    const transformTime = this.ctx.projects.flatMap((w) => Array.from(w.vitenode.fetchCache.values()).map((i) => i.duration || 0)).reduce((a, b) => a + b, 0);
    const environmentTime = files.reduce((acc, file) => acc + Math.max(0, file.environmentLoad || 0), 0);
    const prepareTime = files.reduce((acc, file) => acc + Math.max(0, file.prepareDuration || 0), 0);
    const threadTime = collectTime + testsTime + setupTime;
    const padTitle = (str) => c.dim(`${str.padStart(11)} `);
    const time = (time2) => {
      if (time2 > 1e3)
        return `${(time2 / 1e3).toFixed(2)}s`;
      return `${Math.round(time2)}ms`;
    };
    const snapshotOutput = renderSnapshotSummary(this.ctx.config.root, this.ctx.snapshot.summary);
    if (snapshotOutput.length) {
      logger.log(snapshotOutput.map(
        (t, i) => i === 0 ? `${padTitle("Snapshots")} ${t}` : `${padTitle("")} ${t}`
      ).join("\n"));
      if (snapshotOutput.length > 1)
        logger.log();
    }
    logger.log(padTitle("Test Files"), getStateString(files));
    logger.log(padTitle("Tests"), getStateString(tests));
    if (this.mode === "typecheck") {
      const failed = tests.filter((t) => {
        var _a2, _b, _c;
        return ((_a2 = t.meta) == null ? void 0 : _a2.typecheck) && ((_c = (_b = t.result) == null ? void 0 : _b.errors) == null ? void 0 : _c.length);
      });
      logger.log(padTitle("Type Errors"), failed.length ? c.bold(c.red(`${failed.length} failed`)) : c.dim("no errors"));
    }
    if (errors.length)
      logger.log(padTitle("Errors"), c.bold(c.red(`${errors.length} error${errors.length > 1 ? "s" : ""}`)));
    logger.log(padTitle("Start at"), formatTimeString(this._timeStart));
    if (this.watchFilters)
      logger.log(padTitle("Duration"), time(threadTime));
    else if (this.mode === "typecheck")
      logger.log(padTitle("Duration"), time(executionTime));
    else
      logger.log(padTitle("Duration"), time(executionTime) + c.dim(` (transform ${time(transformTime)}, setup ${time(setupTime)}, collect ${time(collectTime)}, tests ${time(testsTime)}, environment ${time(environmentTime)}, prepare ${time(prepareTime)})`));
    logger.log();
  }
  async printErrorsSummary(files, errors) {
    const logger = this.ctx.logger;
    const suites = getSuites(files);
    const tests = getTests(files);
    const failedSuites = suites.filter((i) => {
      var _a2;
      return (_a2 = i.result) == null ? void 0 : _a2.errors;
    });
    const failedTests = tests.filter((i) => {
      var _a2;
      return ((_a2 = i.result) == null ? void 0 : _a2.state) === "fail";
    });
    const failedTotal = countTestErrors(failedSuites) + countTestErrors(failedTests);
    let current = 1;
    const errorDivider = () => logger.error(`${c.red(c.dim(divider(`[${current++}/${failedTotal}]`, void 0, 1)))}
`);
    if (failedSuites.length) {
      logger.error(c.red(divider(c.bold(c.inverse(` Failed Suites ${failedSuites.length} `)))));
      logger.error();
      await this.printTaskErrors(failedSuites, errorDivider);
    }
    if (failedTests.length) {
      logger.error(c.red(divider(c.bold(c.inverse(` Failed Tests ${failedTests.length} `)))));
      logger.error();
      await this.printTaskErrors(failedTests, errorDivider);
    }
    if (errors.length) {
      await logger.printUnhandledErrors(errors);
      logger.error();
    }
    return tests;
  }
  async reportBenchmarkSummary(files) {
    const logger = this.ctx.logger;
    const benches = getTests(files);
    const topBenches = benches.filter((i) => {
      var _a2, _b;
      return ((_b = (_a2 = i.result) == null ? void 0 : _a2.benchmark) == null ? void 0 : _b.rank) === 1;
    });
    logger.log(`
${c.cyan(c.inverse(c.bold(" BENCH ")))} ${c.cyan("Summary")}
`);
    for (const bench of topBenches) {
      const group = bench.suite;
      if (!group)
        continue;
      const groupName = getFullName(group, c.dim(" > "));
      logger.log(`  ${bench.name}${c.dim(` - ${groupName}`)}`);
      const siblings = group.tasks.filter((i) => {
        var _a2;
        return ((_a2 = i.result) == null ? void 0 : _a2.benchmark) && i !== bench;
      }).sort((a, b) => a.result.benchmark.rank - b.result.benchmark.rank);
      for (const sibling of siblings) {
        const number = `${(sibling.result.benchmark.mean / bench.result.benchmark.mean).toFixed(2)}x`;
        logger.log(`    ${c.green(number)} ${c.gray("faster than")} ${sibling.name}`);
      }
      logger.log("");
    }
  }
  async printTaskErrors(tasks, errorDivider) {
    var _a2, _b, _c;
    const errorsQueue = [];
    for (const task of tasks) {
      (_b = (_a2 = task.result) == null ? void 0 : _a2.errors) == null ? void 0 : _b.forEach((error) => {
        const errorItem = (error == null ? void 0 : error.stackStr) && errorsQueue.find((i) => {
          var _a3, _b2, _c2, _d;
          const hasStr = ((_a3 = i[0]) == null ? void 0 : _a3.stackStr) === error.stackStr;
          if (!hasStr)
            return false;
          const currentProjectName = (task == null ? void 0 : task.projectName) || ((_b2 = task.file) == null ? void 0 : _b2.projectName);
          const projectName = ((_c2 = i[1][0]) == null ? void 0 : _c2.projectName) || ((_d = i[1][0].file) == null ? void 0 : _d.projectName);
          return projectName === currentProjectName;
        });
        if (errorItem)
          errorItem[1].push(task);
        else
          errorsQueue.push([error, [task]]);
      });
    }
    for (const [error, tasks2] of errorsQueue) {
      for (const task of tasks2) {
        const filepath = (task == null ? void 0 : task.filepath) || "";
        const projectName = (task == null ? void 0 : task.projectName) || ((_c = task.file) == null ? void 0 : _c.projectName);
        let name = getFullName(task, c.dim(" > "));
        if (filepath)
          name = `${name} ${c.dim(`[ ${this.relative(filepath)} ]`)}`;
        this.ctx.logger.error(`${c.red(c.bold(c.inverse(" FAIL ")))} ${formatProjectName(projectName)}${name}`);
      }
      const project = this.ctx.getProjectByTaskId(tasks2[0].id);
      await this.ctx.logger.printError(error, { project });
      errorDivider();
      await Promise.resolve();
    }
  }
  registerUnhandledRejection() {
    process.on("unhandledRejection", async (err) => {
      process.exitCode = 1;
      await this.ctx.logger.printError(err, { fullStack: true, type: "Unhandled Rejection" });
      this.ctx.logger.error("\n\n");
      process.exit(1);
    });
  }
}

class BasicReporter extends BaseReporter {
  isTTY = false;
  reportSummary(files, errors) {
    this.ctx.logger.log();
    return super.reportSummary(files, errors);
  }
}

/* eslint-disable yoda */

function isFullwidthCodePoint(codePoint) {
	if (!Number.isInteger(codePoint)) {
		return false;
	}

	// Code points are derived from:
	// https://unicode.org/Public/UNIDATA/EastAsianWidth.txt
	return codePoint >= 0x1100 && (
		codePoint <= 0x115F || // Hangul Jamo
		codePoint === 0x2329 || // LEFT-POINTING ANGLE BRACKET
		codePoint === 0x232A || // RIGHT-POINTING ANGLE BRACKET
		// CJK Radicals Supplement .. Enclosed CJK Letters and Months
		(0x2E80 <= codePoint && codePoint <= 0x3247 && codePoint !== 0x303F) ||
		// Enclosed CJK Letters and Months .. CJK Unified Ideographs Extension A
		(0x3250 <= codePoint && codePoint <= 0x4DBF) ||
		// CJK Unified Ideographs .. Yi Radicals
		(0x4E00 <= codePoint && codePoint <= 0xA4C6) ||
		// Hangul Jamo Extended-A
		(0xA960 <= codePoint && codePoint <= 0xA97C) ||
		// Hangul Syllables
		(0xAC00 <= codePoint && codePoint <= 0xD7A3) ||
		// CJK Compatibility Ideographs
		(0xF900 <= codePoint && codePoint <= 0xFAFF) ||
		// Vertical Forms
		(0xFE10 <= codePoint && codePoint <= 0xFE19) ||
		// CJK Compatibility Forms .. Small Form Variants
		(0xFE30 <= codePoint && codePoint <= 0xFE6B) ||
		// Halfwidth and Fullwidth Forms
		(0xFF01 <= codePoint && codePoint <= 0xFF60) ||
		(0xFFE0 <= codePoint && codePoint <= 0xFFE6) ||
		// Kana Supplement
		(0x1B000 <= codePoint && codePoint <= 0x1B001) ||
		// Enclosed Ideographic Supplement
		(0x1F200 <= codePoint && codePoint <= 0x1F251) ||
		// CJK Unified Ideographs Extension B .. Tertiary Ideographic Plane
		(0x20000 <= codePoint && codePoint <= 0x3FFFD)
	);
}

const ANSI_BACKGROUND_OFFSET = 10;

const wrapAnsi16 = (offset = 0) => code => `\u001B[${code + offset}m`;

const wrapAnsi256 = (offset = 0) => code => `\u001B[${38 + offset};5;${code}m`;

const wrapAnsi16m = (offset = 0) => (red, green, blue) => `\u001B[${38 + offset};2;${red};${green};${blue}m`;

const styles = {
	modifier: {
		reset: [0, 0],
		// 21 isn't widely supported and 22 does the same thing
		bold: [1, 22],
		dim: [2, 22],
		italic: [3, 23],
		underline: [4, 24],
		overline: [53, 55],
		inverse: [7, 27],
		hidden: [8, 28],
		strikethrough: [9, 29],
	},
	color: {
		black: [30, 39],
		red: [31, 39],
		green: [32, 39],
		yellow: [33, 39],
		blue: [34, 39],
		magenta: [35, 39],
		cyan: [36, 39],
		white: [37, 39],

		// Bright color
		blackBright: [90, 39],
		gray: [90, 39], // Alias of `blackBright`
		grey: [90, 39], // Alias of `blackBright`
		redBright: [91, 39],
		greenBright: [92, 39],
		yellowBright: [93, 39],
		blueBright: [94, 39],
		magentaBright: [95, 39],
		cyanBright: [96, 39],
		whiteBright: [97, 39],
	},
	bgColor: {
		bgBlack: [40, 49],
		bgRed: [41, 49],
		bgGreen: [42, 49],
		bgYellow: [43, 49],
		bgBlue: [44, 49],
		bgMagenta: [45, 49],
		bgCyan: [46, 49],
		bgWhite: [47, 49],

		// Bright color
		bgBlackBright: [100, 49],
		bgGray: [100, 49], // Alias of `bgBlackBright`
		bgGrey: [100, 49], // Alias of `bgBlackBright`
		bgRedBright: [101, 49],
		bgGreenBright: [102, 49],
		bgYellowBright: [103, 49],
		bgBlueBright: [104, 49],
		bgMagentaBright: [105, 49],
		bgCyanBright: [106, 49],
		bgWhiteBright: [107, 49],
	},
};

Object.keys(styles.modifier);
const foregroundColorNames = Object.keys(styles.color);
const backgroundColorNames = Object.keys(styles.bgColor);
[...foregroundColorNames, ...backgroundColorNames];

function assembleStyles() {
	const codes = new Map();

	for (const [groupName, group] of Object.entries(styles)) {
		for (const [styleName, style] of Object.entries(group)) {
			styles[styleName] = {
				open: `\u001B[${style[0]}m`,
				close: `\u001B[${style[1]}m`,
			};

			group[styleName] = styles[styleName];

			codes.set(style[0], style[1]);
		}

		Object.defineProperty(styles, groupName, {
			value: group,
			enumerable: false,
		});
	}

	Object.defineProperty(styles, 'codes', {
		value: codes,
		enumerable: false,
	});

	styles.color.close = '\u001B[39m';
	styles.bgColor.close = '\u001B[49m';

	styles.color.ansi = wrapAnsi16();
	styles.color.ansi256 = wrapAnsi256();
	styles.color.ansi16m = wrapAnsi16m();
	styles.bgColor.ansi = wrapAnsi16(ANSI_BACKGROUND_OFFSET);
	styles.bgColor.ansi256 = wrapAnsi256(ANSI_BACKGROUND_OFFSET);
	styles.bgColor.ansi16m = wrapAnsi16m(ANSI_BACKGROUND_OFFSET);

	// From https://github.com/Qix-/color-convert/blob/3f0e0d4e92e235796ccb17f6e85c72094a651f49/conversions.js
	Object.defineProperties(styles, {
		rgbToAnsi256: {
			value: (red, green, blue) => {
				// We use the extended greyscale palette here, with the exception of
				// black and white. normal palette only has 4 greyscale shades.
				if (red === green && green === blue) {
					if (red < 8) {
						return 16;
					}

					if (red > 248) {
						return 231;
					}

					return Math.round(((red - 8) / 247) * 24) + 232;
				}

				return 16
					+ (36 * Math.round(red / 255 * 5))
					+ (6 * Math.round(green / 255 * 5))
					+ Math.round(blue / 255 * 5);
			},
			enumerable: false,
		},
		hexToRgb: {
			value: hex => {
				const matches = /[a-f\d]{6}|[a-f\d]{3}/i.exec(hex.toString(16));
				if (!matches) {
					return [0, 0, 0];
				}

				let [colorString] = matches;

				if (colorString.length === 3) {
					colorString = [...colorString].map(character => character + character).join('');
				}

				const integer = Number.parseInt(colorString, 16);

				return [
					/* eslint-disable no-bitwise */
					(integer >> 16) & 0xFF,
					(integer >> 8) & 0xFF,
					integer & 0xFF,
					/* eslint-enable no-bitwise */
				];
			},
			enumerable: false,
		},
		hexToAnsi256: {
			value: hex => styles.rgbToAnsi256(...styles.hexToRgb(hex)),
			enumerable: false,
		},
		ansi256ToAnsi: {
			value: code => {
				if (code < 8) {
					return 30 + code;
				}

				if (code < 16) {
					return 90 + (code - 8);
				}

				let red;
				let green;
				let blue;

				if (code >= 232) {
					red = (((code - 232) * 10) + 8) / 255;
					green = red;
					blue = red;
				} else {
					code -= 16;

					const remainder = code % 36;

					red = Math.floor(code / 36) / 5;
					green = Math.floor(remainder / 6) / 5;
					blue = (remainder % 6) / 5;
				}

				const value = Math.max(red, green, blue) * 2;

				if (value === 0) {
					return 30;
				}

				// eslint-disable-next-line no-bitwise
				let result = 30 + ((Math.round(blue) << 2) | (Math.round(green) << 1) | Math.round(red));

				if (value === 2) {
					result += 60;
				}

				return result;
			},
			enumerable: false,
		},
		rgbToAnsi: {
			value: (red, green, blue) => styles.ansi256ToAnsi(styles.rgbToAnsi256(red, green, blue)),
			enumerable: false,
		},
		hexToAnsi: {
			value: hex => styles.ansi256ToAnsi(styles.hexToAnsi256(hex)),
			enumerable: false,
		},
	});

	return styles;
}

const ansiStyles = assembleStyles();

const astralRegex = /^[\uD800-\uDBFF][\uDC00-\uDFFF]$/;

const ESCAPES = [
	'\u001B',
	'\u009B'
];

const wrapAnsi = code => `${ESCAPES[0]}[${code}m`;

const checkAnsi = (ansiCodes, isEscapes, endAnsiCode) => {
	let output = [];
	ansiCodes = [...ansiCodes];

	for (let ansiCode of ansiCodes) {
		const ansiCodeOrigin = ansiCode;
		if (ansiCode.includes(';')) {
			ansiCode = ansiCode.split(';')[0][0] + '0';
		}

		const item = ansiStyles.codes.get(Number.parseInt(ansiCode, 10));
		if (item) {
			const indexEscape = ansiCodes.indexOf(item.toString());
			if (indexEscape === -1) {
				output.push(wrapAnsi(isEscapes ? item : ansiCodeOrigin));
			} else {
				ansiCodes.splice(indexEscape, 1);
			}
		} else if (isEscapes) {
			output.push(wrapAnsi(0));
			break;
		} else {
			output.push(wrapAnsi(ansiCodeOrigin));
		}
	}

	if (isEscapes) {
		output = output.filter((element, index) => output.indexOf(element) === index);

		if (endAnsiCode !== undefined) {
			const fistEscapeCode = wrapAnsi(ansiStyles.codes.get(Number.parseInt(endAnsiCode, 10)));
			// TODO: Remove the use of `.reduce` here.
			// eslint-disable-next-line unicorn/no-array-reduce
			output = output.reduce((current, next) => next === fistEscapeCode ? [next, ...current] : [...current, next], []);
		}
	}

	return output.join('');
};

function sliceAnsi(string, begin, end) {
	const characters = [...string];
	const ansiCodes = [];

	let stringEnd = typeof end === 'number' ? end : characters.length;
	let isInsideEscape = false;
	let ansiCode;
	let visible = 0;
	let output = '';

	for (const [index, character] of characters.entries()) {
		let leftEscape = false;

		if (ESCAPES.includes(character)) {
			const code = /\d[^m]*/.exec(string.slice(index, index + 18));
			ansiCode = code && code.length > 0 ? code[0] : undefined;

			if (visible < stringEnd) {
				isInsideEscape = true;

				if (ansiCode !== undefined) {
					ansiCodes.push(ansiCode);
				}
			}
		} else if (isInsideEscape && character === 'm') {
			isInsideEscape = false;
			leftEscape = true;
		}

		if (!isInsideEscape && !leftEscape) {
			visible++;
		}

		if (!astralRegex.test(character) && isFullwidthCodePoint(character.codePointAt())) {
			visible++;

			if (typeof end !== 'number') {
				stringEnd++;
			}
		}

		if (visible > begin && visible <= stringEnd) {
			output += character;
		} else if (visible === begin && !isInsideEscape && ansiCode !== undefined) {
			output = checkAnsi(ansiCodes);
		} else if (visible >= stringEnd) {
			output += checkAnsi(ansiCodes, true, ansiCode);
			break;
		}
	}

	return output;
}

var eastasianwidth = {exports: {}};

(function (module) {
	var eaw = {};

	{
	  module.exports = eaw;
	}

	eaw.eastAsianWidth = function(character) {
	  var x = character.charCodeAt(0);
	  var y = (character.length == 2) ? character.charCodeAt(1) : 0;
	  var codePoint = x;
	  if ((0xD800 <= x && x <= 0xDBFF) && (0xDC00 <= y && y <= 0xDFFF)) {
	    x &= 0x3FF;
	    y &= 0x3FF;
	    codePoint = (x << 10) | y;
	    codePoint += 0x10000;
	  }

	  if ((0x3000 == codePoint) ||
	      (0xFF01 <= codePoint && codePoint <= 0xFF60) ||
	      (0xFFE0 <= codePoint && codePoint <= 0xFFE6)) {
	    return 'F';
	  }
	  if ((0x20A9 == codePoint) ||
	      (0xFF61 <= codePoint && codePoint <= 0xFFBE) ||
	      (0xFFC2 <= codePoint && codePoint <= 0xFFC7) ||
	      (0xFFCA <= codePoint && codePoint <= 0xFFCF) ||
	      (0xFFD2 <= codePoint && codePoint <= 0xFFD7) ||
	      (0xFFDA <= codePoint && codePoint <= 0xFFDC) ||
	      (0xFFE8 <= codePoint && codePoint <= 0xFFEE)) {
	    return 'H';
	  }
	  if ((0x1100 <= codePoint && codePoint <= 0x115F) ||
	      (0x11A3 <= codePoint && codePoint <= 0x11A7) ||
	      (0x11FA <= codePoint && codePoint <= 0x11FF) ||
	      (0x2329 <= codePoint && codePoint <= 0x232A) ||
	      (0x2E80 <= codePoint && codePoint <= 0x2E99) ||
	      (0x2E9B <= codePoint && codePoint <= 0x2EF3) ||
	      (0x2F00 <= codePoint && codePoint <= 0x2FD5) ||
	      (0x2FF0 <= codePoint && codePoint <= 0x2FFB) ||
	      (0x3001 <= codePoint && codePoint <= 0x303E) ||
	      (0x3041 <= codePoint && codePoint <= 0x3096) ||
	      (0x3099 <= codePoint && codePoint <= 0x30FF) ||
	      (0x3105 <= codePoint && codePoint <= 0x312D) ||
	      (0x3131 <= codePoint && codePoint <= 0x318E) ||
	      (0x3190 <= codePoint && codePoint <= 0x31BA) ||
	      (0x31C0 <= codePoint && codePoint <= 0x31E3) ||
	      (0x31F0 <= codePoint && codePoint <= 0x321E) ||
	      (0x3220 <= codePoint && codePoint <= 0x3247) ||
	      (0x3250 <= codePoint && codePoint <= 0x32FE) ||
	      (0x3300 <= codePoint && codePoint <= 0x4DBF) ||
	      (0x4E00 <= codePoint && codePoint <= 0xA48C) ||
	      (0xA490 <= codePoint && codePoint <= 0xA4C6) ||
	      (0xA960 <= codePoint && codePoint <= 0xA97C) ||
	      (0xAC00 <= codePoint && codePoint <= 0xD7A3) ||
	      (0xD7B0 <= codePoint && codePoint <= 0xD7C6) ||
	      (0xD7CB <= codePoint && codePoint <= 0xD7FB) ||
	      (0xF900 <= codePoint && codePoint <= 0xFAFF) ||
	      (0xFE10 <= codePoint && codePoint <= 0xFE19) ||
	      (0xFE30 <= codePoint && codePoint <= 0xFE52) ||
	      (0xFE54 <= codePoint && codePoint <= 0xFE66) ||
	      (0xFE68 <= codePoint && codePoint <= 0xFE6B) ||
	      (0x1B000 <= codePoint && codePoint <= 0x1B001) ||
	      (0x1F200 <= codePoint && codePoint <= 0x1F202) ||
	      (0x1F210 <= codePoint && codePoint <= 0x1F23A) ||
	      (0x1F240 <= codePoint && codePoint <= 0x1F248) ||
	      (0x1F250 <= codePoint && codePoint <= 0x1F251) ||
	      (0x20000 <= codePoint && codePoint <= 0x2F73F) ||
	      (0x2B740 <= codePoint && codePoint <= 0x2FFFD) ||
	      (0x30000 <= codePoint && codePoint <= 0x3FFFD)) {
	    return 'W';
	  }
	  if ((0x0020 <= codePoint && codePoint <= 0x007E) ||
	      (0x00A2 <= codePoint && codePoint <= 0x00A3) ||
	      (0x00A5 <= codePoint && codePoint <= 0x00A6) ||
	      (0x00AC == codePoint) ||
	      (0x00AF == codePoint) ||
	      (0x27E6 <= codePoint && codePoint <= 0x27ED) ||
	      (0x2985 <= codePoint && codePoint <= 0x2986)) {
	    return 'Na';
	  }
	  if ((0x00A1 == codePoint) ||
	      (0x00A4 == codePoint) ||
	      (0x00A7 <= codePoint && codePoint <= 0x00A8) ||
	      (0x00AA == codePoint) ||
	      (0x00AD <= codePoint && codePoint <= 0x00AE) ||
	      (0x00B0 <= codePoint && codePoint <= 0x00B4) ||
	      (0x00B6 <= codePoint && codePoint <= 0x00BA) ||
	      (0x00BC <= codePoint && codePoint <= 0x00BF) ||
	      (0x00C6 == codePoint) ||
	      (0x00D0 == codePoint) ||
	      (0x00D7 <= codePoint && codePoint <= 0x00D8) ||
	      (0x00DE <= codePoint && codePoint <= 0x00E1) ||
	      (0x00E6 == codePoint) ||
	      (0x00E8 <= codePoint && codePoint <= 0x00EA) ||
	      (0x00EC <= codePoint && codePoint <= 0x00ED) ||
	      (0x00F0 == codePoint) ||
	      (0x00F2 <= codePoint && codePoint <= 0x00F3) ||
	      (0x00F7 <= codePoint && codePoint <= 0x00FA) ||
	      (0x00FC == codePoint) ||
	      (0x00FE == codePoint) ||
	      (0x0101 == codePoint) ||
	      (0x0111 == codePoint) ||
	      (0x0113 == codePoint) ||
	      (0x011B == codePoint) ||
	      (0x0126 <= codePoint && codePoint <= 0x0127) ||
	      (0x012B == codePoint) ||
	      (0x0131 <= codePoint && codePoint <= 0x0133) ||
	      (0x0138 == codePoint) ||
	      (0x013F <= codePoint && codePoint <= 0x0142) ||
	      (0x0144 == codePoint) ||
	      (0x0148 <= codePoint && codePoint <= 0x014B) ||
	      (0x014D == codePoint) ||
	      (0x0152 <= codePoint && codePoint <= 0x0153) ||
	      (0x0166 <= codePoint && codePoint <= 0x0167) ||
	      (0x016B == codePoint) ||
	      (0x01CE == codePoint) ||
	      (0x01D0 == codePoint) ||
	      (0x01D2 == codePoint) ||
	      (0x01D4 == codePoint) ||
	      (0x01D6 == codePoint) ||
	      (0x01D8 == codePoint) ||
	      (0x01DA == codePoint) ||
	      (0x01DC == codePoint) ||
	      (0x0251 == codePoint) ||
	      (0x0261 == codePoint) ||
	      (0x02C4 == codePoint) ||
	      (0x02C7 == codePoint) ||
	      (0x02C9 <= codePoint && codePoint <= 0x02CB) ||
	      (0x02CD == codePoint) ||
	      (0x02D0 == codePoint) ||
	      (0x02D8 <= codePoint && codePoint <= 0x02DB) ||
	      (0x02DD == codePoint) ||
	      (0x02DF == codePoint) ||
	      (0x0300 <= codePoint && codePoint <= 0x036F) ||
	      (0x0391 <= codePoint && codePoint <= 0x03A1) ||
	      (0x03A3 <= codePoint && codePoint <= 0x03A9) ||
	      (0x03B1 <= codePoint && codePoint <= 0x03C1) ||
	      (0x03C3 <= codePoint && codePoint <= 0x03C9) ||
	      (0x0401 == codePoint) ||
	      (0x0410 <= codePoint && codePoint <= 0x044F) ||
	      (0x0451 == codePoint) ||
	      (0x2010 == codePoint) ||
	      (0x2013 <= codePoint && codePoint <= 0x2016) ||
	      (0x2018 <= codePoint && codePoint <= 0x2019) ||
	      (0x201C <= codePoint && codePoint <= 0x201D) ||
	      (0x2020 <= codePoint && codePoint <= 0x2022) ||
	      (0x2024 <= codePoint && codePoint <= 0x2027) ||
	      (0x2030 == codePoint) ||
	      (0x2032 <= codePoint && codePoint <= 0x2033) ||
	      (0x2035 == codePoint) ||
	      (0x203B == codePoint) ||
	      (0x203E == codePoint) ||
	      (0x2074 == codePoint) ||
	      (0x207F == codePoint) ||
	      (0x2081 <= codePoint && codePoint <= 0x2084) ||
	      (0x20AC == codePoint) ||
	      (0x2103 == codePoint) ||
	      (0x2105 == codePoint) ||
	      (0x2109 == codePoint) ||
	      (0x2113 == codePoint) ||
	      (0x2116 == codePoint) ||
	      (0x2121 <= codePoint && codePoint <= 0x2122) ||
	      (0x2126 == codePoint) ||
	      (0x212B == codePoint) ||
	      (0x2153 <= codePoint && codePoint <= 0x2154) ||
	      (0x215B <= codePoint && codePoint <= 0x215E) ||
	      (0x2160 <= codePoint && codePoint <= 0x216B) ||
	      (0x2170 <= codePoint && codePoint <= 0x2179) ||
	      (0x2189 == codePoint) ||
	      (0x2190 <= codePoint && codePoint <= 0x2199) ||
	      (0x21B8 <= codePoint && codePoint <= 0x21B9) ||
	      (0x21D2 == codePoint) ||
	      (0x21D4 == codePoint) ||
	      (0x21E7 == codePoint) ||
	      (0x2200 == codePoint) ||
	      (0x2202 <= codePoint && codePoint <= 0x2203) ||
	      (0x2207 <= codePoint && codePoint <= 0x2208) ||
	      (0x220B == codePoint) ||
	      (0x220F == codePoint) ||
	      (0x2211 == codePoint) ||
	      (0x2215 == codePoint) ||
	      (0x221A == codePoint) ||
	      (0x221D <= codePoint && codePoint <= 0x2220) ||
	      (0x2223 == codePoint) ||
	      (0x2225 == codePoint) ||
	      (0x2227 <= codePoint && codePoint <= 0x222C) ||
	      (0x222E == codePoint) ||
	      (0x2234 <= codePoint && codePoint <= 0x2237) ||
	      (0x223C <= codePoint && codePoint <= 0x223D) ||
	      (0x2248 == codePoint) ||
	      (0x224C == codePoint) ||
	      (0x2252 == codePoint) ||
	      (0x2260 <= codePoint && codePoint <= 0x2261) ||
	      (0x2264 <= codePoint && codePoint <= 0x2267) ||
	      (0x226A <= codePoint && codePoint <= 0x226B) ||
	      (0x226E <= codePoint && codePoint <= 0x226F) ||
	      (0x2282 <= codePoint && codePoint <= 0x2283) ||
	      (0x2286 <= codePoint && codePoint <= 0x2287) ||
	      (0x2295 == codePoint) ||
	      (0x2299 == codePoint) ||
	      (0x22A5 == codePoint) ||
	      (0x22BF == codePoint) ||
	      (0x2312 == codePoint) ||
	      (0x2460 <= codePoint && codePoint <= 0x24E9) ||
	      (0x24EB <= codePoint && codePoint <= 0x254B) ||
	      (0x2550 <= codePoint && codePoint <= 0x2573) ||
	      (0x2580 <= codePoint && codePoint <= 0x258F) ||
	      (0x2592 <= codePoint && codePoint <= 0x2595) ||
	      (0x25A0 <= codePoint && codePoint <= 0x25A1) ||
	      (0x25A3 <= codePoint && codePoint <= 0x25A9) ||
	      (0x25B2 <= codePoint && codePoint <= 0x25B3) ||
	      (0x25B6 <= codePoint && codePoint <= 0x25B7) ||
	      (0x25BC <= codePoint && codePoint <= 0x25BD) ||
	      (0x25C0 <= codePoint && codePoint <= 0x25C1) ||
	      (0x25C6 <= codePoint && codePoint <= 0x25C8) ||
	      (0x25CB == codePoint) ||
	      (0x25CE <= codePoint && codePoint <= 0x25D1) ||
	      (0x25E2 <= codePoint && codePoint <= 0x25E5) ||
	      (0x25EF == codePoint) ||
	      (0x2605 <= codePoint && codePoint <= 0x2606) ||
	      (0x2609 == codePoint) ||
	      (0x260E <= codePoint && codePoint <= 0x260F) ||
	      (0x2614 <= codePoint && codePoint <= 0x2615) ||
	      (0x261C == codePoint) ||
	      (0x261E == codePoint) ||
	      (0x2640 == codePoint) ||
	      (0x2642 == codePoint) ||
	      (0x2660 <= codePoint && codePoint <= 0x2661) ||
	      (0x2663 <= codePoint && codePoint <= 0x2665) ||
	      (0x2667 <= codePoint && codePoint <= 0x266A) ||
	      (0x266C <= codePoint && codePoint <= 0x266D) ||
	      (0x266F == codePoint) ||
	      (0x269E <= codePoint && codePoint <= 0x269F) ||
	      (0x26BE <= codePoint && codePoint <= 0x26BF) ||
	      (0x26C4 <= codePoint && codePoint <= 0x26CD) ||
	      (0x26CF <= codePoint && codePoint <= 0x26E1) ||
	      (0x26E3 == codePoint) ||
	      (0x26E8 <= codePoint && codePoint <= 0x26FF) ||
	      (0x273D == codePoint) ||
	      (0x2757 == codePoint) ||
	      (0x2776 <= codePoint && codePoint <= 0x277F) ||
	      (0x2B55 <= codePoint && codePoint <= 0x2B59) ||
	      (0x3248 <= codePoint && codePoint <= 0x324F) ||
	      (0xE000 <= codePoint && codePoint <= 0xF8FF) ||
	      (0xFE00 <= codePoint && codePoint <= 0xFE0F) ||
	      (0xFFFD == codePoint) ||
	      (0x1F100 <= codePoint && codePoint <= 0x1F10A) ||
	      (0x1F110 <= codePoint && codePoint <= 0x1F12D) ||
	      (0x1F130 <= codePoint && codePoint <= 0x1F169) ||
	      (0x1F170 <= codePoint && codePoint <= 0x1F19A) ||
	      (0xE0100 <= codePoint && codePoint <= 0xE01EF) ||
	      (0xF0000 <= codePoint && codePoint <= 0xFFFFD) ||
	      (0x100000 <= codePoint && codePoint <= 0x10FFFD)) {
	    return 'A';
	  }

	  return 'N';
	};

	eaw.characterLength = function(character) {
	  var code = this.eastAsianWidth(character);
	  if (code == 'F' || code == 'W' || code == 'A') {
	    return 2;
	  } else {
	    return 1;
	  }
	};

	// Split a string considering surrogate-pairs.
	function stringToArray(string) {
	  return string.match(/[\uD800-\uDBFF][\uDC00-\uDFFF]|[^\uD800-\uDFFF]/g) || [];
	}

	eaw.length = function(string) {
	  var characters = stringToArray(string);
	  var len = 0;
	  for (var i = 0; i < characters.length; i++) {
	    len = len + this.characterLength(characters[i]);
	  }
	  return len;
	};

	eaw.slice = function(text, start, end) {
	  textLen = eaw.length(text);
	  start = start ? start : 0;
	  end = end ? end : 1;
	  if (start < 0) {
	      start = textLen + start;
	  }
	  if (end < 0) {
	      end = textLen + end;
	  }
	  var result = '';
	  var eawLen = 0;
	  var chars = stringToArray(text);
	  for (var i = 0; i < chars.length; i++) {
	    var char = chars[i];
	    var charLen = eaw.length(char);
	    if (eawLen >= start - (charLen == 2 ? 1 : 0)) {
	        if (eawLen + charLen <= end) {
	            result += char;
	        } else {
	            break;
	        }
	    }
	    eawLen += charLen;
	  }
	  return result;
	}; 
} (eastasianwidth));

var eastasianwidthExports = eastasianwidth.exports;
var eastAsianWidth = /*@__PURE__*/getDefaultExportFromCjs(eastasianwidthExports);

var emojiRegex = function () {
  // https://mths.be/emoji
  return /\uD83C\uDFF4\uDB40\uDC67\uDB40\uDC62(?:\uDB40\uDC77\uDB40\uDC6C\uDB40\uDC73|\uDB40\uDC73\uDB40\uDC63\uDB40\uDC74|\uDB40\uDC65\uDB40\uDC6E\uDB40\uDC67)\uDB40\uDC7F|(?:\uD83E\uDDD1\uD83C\uDFFF\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D)?\uD83E\uDDD1|\uD83D\uDC69\uD83C\uDFFF\u200D\uD83E\uDD1D\u200D(?:\uD83D[\uDC68\uDC69]))(?:\uD83C[\uDFFB-\uDFFE])|(?:\uD83E\uDDD1\uD83C\uDFFE\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D)?\uD83E\uDDD1|\uD83D\uDC69\uD83C\uDFFE\u200D\uD83E\uDD1D\u200D(?:\uD83D[\uDC68\uDC69]))(?:\uD83C[\uDFFB-\uDFFD\uDFFF])|(?:\uD83E\uDDD1\uD83C\uDFFD\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D)?\uD83E\uDDD1|\uD83D\uDC69\uD83C\uDFFD\u200D\uD83E\uDD1D\u200D(?:\uD83D[\uDC68\uDC69]))(?:\uD83C[\uDFFB\uDFFC\uDFFE\uDFFF])|(?:\uD83E\uDDD1\uD83C\uDFFC\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D)?\uD83E\uDDD1|\uD83D\uDC69\uD83C\uDFFC\u200D\uD83E\uDD1D\u200D(?:\uD83D[\uDC68\uDC69]))(?:\uD83C[\uDFFB\uDFFD-\uDFFF])|(?:\uD83E\uDDD1\uD83C\uDFFB\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D)?\uD83E\uDDD1|\uD83D\uDC69\uD83C\uDFFB\u200D\uD83E\uDD1D\u200D(?:\uD83D[\uDC68\uDC69]))(?:\uD83C[\uDFFC-\uDFFF])|\uD83D\uDC68(?:\uD83C\uDFFB(?:\u200D(?:\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D\uD83D\uDC68(?:\uD83C[\uDFFB-\uDFFF])|\uD83D\uDC68(?:\uD83C[\uDFFB-\uDFFF]))|\uD83E\uDD1D\u200D\uD83D\uDC68(?:\uD83C[\uDFFC-\uDFFF])|[\u2695\u2696\u2708]\uFE0F|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD]))?|(?:\uD83C[\uDFFC-\uDFFF])\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D\uD83D\uDC68(?:\uD83C[\uDFFB-\uDFFF])|\uD83D\uDC68(?:\uD83C[\uDFFB-\uDFFF]))|\u200D(?:\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D)?\uD83D\uDC68|(?:\uD83D[\uDC68\uDC69])\u200D(?:\uD83D\uDC66\u200D\uD83D\uDC66|\uD83D\uDC67\u200D(?:\uD83D[\uDC66\uDC67]))|\uD83D\uDC66\u200D\uD83D\uDC66|\uD83D\uDC67\u200D(?:\uD83D[\uDC66\uDC67])|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFF\u200D(?:\uD83E\uDD1D\u200D\uD83D\uDC68(?:\uD83C[\uDFFB-\uDFFE])|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFE\u200D(?:\uD83E\uDD1D\u200D\uD83D\uDC68(?:\uD83C[\uDFFB-\uDFFD\uDFFF])|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFD\u200D(?:\uD83E\uDD1D\u200D\uD83D\uDC68(?:\uD83C[\uDFFB\uDFFC\uDFFE\uDFFF])|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFC\u200D(?:\uD83E\uDD1D\u200D\uD83D\uDC68(?:\uD83C[\uDFFB\uDFFD-\uDFFF])|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|(?:\uD83C\uDFFF\u200D[\u2695\u2696\u2708]|\uD83C\uDFFE\u200D[\u2695\u2696\u2708]|\uD83C\uDFFD\u200D[\u2695\u2696\u2708]|\uD83C\uDFFC\u200D[\u2695\u2696\u2708]|\u200D[\u2695\u2696\u2708])\uFE0F|\u200D(?:(?:\uD83D[\uDC68\uDC69])\u200D(?:\uD83D[\uDC66\uDC67])|\uD83D[\uDC66\uDC67])|\uD83C\uDFFF|\uD83C\uDFFE|\uD83C\uDFFD|\uD83C\uDFFC)?|(?:\uD83D\uDC69(?:\uD83C\uDFFB\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D(?:\uD83D[\uDC68\uDC69])|\uD83D[\uDC68\uDC69])|(?:\uD83C[\uDFFC-\uDFFF])\u200D\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D(?:\uD83D[\uDC68\uDC69])|\uD83D[\uDC68\uDC69]))|\uD83E\uDDD1(?:\uD83C[\uDFFB-\uDFFF])\u200D\uD83E\uDD1D\u200D\uD83E\uDDD1)(?:\uD83C[\uDFFB-\uDFFF])|\uD83D\uDC69\u200D\uD83D\uDC69\u200D(?:\uD83D\uDC66\u200D\uD83D\uDC66|\uD83D\uDC67\u200D(?:\uD83D[\uDC66\uDC67]))|\uD83D\uDC69(?:\u200D(?:\u2764\uFE0F\u200D(?:\uD83D\uDC8B\u200D(?:\uD83D[\uDC68\uDC69])|\uD83D[\uDC68\uDC69])|\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFF\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFE\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFD\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFC\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFB\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD]))|\uD83E\uDDD1(?:\u200D(?:\uD83E\uDD1D\u200D\uD83E\uDDD1|\uD83C[\uDF3E\uDF73\uDF7C\uDF84\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFF\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF84\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFE\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF84\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFD\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF84\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFC\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF84\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD])|\uD83C\uDFFB\u200D(?:\uD83C[\uDF3E\uDF73\uDF7C\uDF84\uDF93\uDFA4\uDFA8\uDFEB\uDFED]|\uD83D[\uDCBB\uDCBC\uDD27\uDD2C\uDE80\uDE92]|\uD83E[\uDDAF-\uDDB3\uDDBC\uDDBD]))|\uD83D\uDC69\u200D\uD83D\uDC66\u200D\uD83D\uDC66|\uD83D\uDC69\u200D\uD83D\uDC69\u200D(?:\uD83D[\uDC66\uDC67])|\uD83D\uDC69\u200D\uD83D\uDC67\u200D(?:\uD83D[\uDC66\uDC67])|(?:\uD83D\uDC41\uFE0F\u200D\uD83D\uDDE8|\uD83E\uDDD1(?:\uD83C\uDFFF\u200D[\u2695\u2696\u2708]|\uD83C\uDFFE\u200D[\u2695\u2696\u2708]|\uD83C\uDFFD\u200D[\u2695\u2696\u2708]|\uD83C\uDFFC\u200D[\u2695\u2696\u2708]|\uD83C\uDFFB\u200D[\u2695\u2696\u2708]|\u200D[\u2695\u2696\u2708])|\uD83D\uDC69(?:\uD83C\uDFFF\u200D[\u2695\u2696\u2708]|\uD83C\uDFFE\u200D[\u2695\u2696\u2708]|\uD83C\uDFFD\u200D[\u2695\u2696\u2708]|\uD83C\uDFFC\u200D[\u2695\u2696\u2708]|\uD83C\uDFFB\u200D[\u2695\u2696\u2708]|\u200D[\u2695\u2696\u2708])|\uD83D\uDE36\u200D\uD83C\uDF2B|\uD83C\uDFF3\uFE0F\u200D\u26A7|\uD83D\uDC3B\u200D\u2744|(?:(?:\uD83C[\uDFC3\uDFC4\uDFCA]|\uD83D[\uDC6E\uDC70\uDC71\uDC73\uDC77\uDC81\uDC82\uDC86\uDC87\uDE45-\uDE47\uDE4B\uDE4D\uDE4E\uDEA3\uDEB4-\uDEB6]|\uD83E[\uDD26\uDD35\uDD37-\uDD39\uDD3D\uDD3E\uDDB8\uDDB9\uDDCD-\uDDCF\uDDD4\uDDD6-\uDDDD])(?:\uD83C[\uDFFB-\uDFFF])|\uD83D\uDC6F|\uD83E[\uDD3C\uDDDE\uDDDF])\u200D[\u2640\u2642]|(?:\u26F9|\uD83C[\uDFCB\uDFCC]|\uD83D\uDD75)(?:\uFE0F|\uD83C[\uDFFB-\uDFFF])\u200D[\u2640\u2642]|\uD83C\uDFF4\u200D\u2620|(?:\uD83C[\uDFC3\uDFC4\uDFCA]|\uD83D[\uDC6E\uDC70\uDC71\uDC73\uDC77\uDC81\uDC82\uDC86\uDC87\uDE45-\uDE47\uDE4B\uDE4D\uDE4E\uDEA3\uDEB4-\uDEB6]|\uD83E[\uDD26\uDD35\uDD37-\uDD39\uDD3D\uDD3E\uDDB8\uDDB9\uDDCD-\uDDCF\uDDD4\uDDD6-\uDDDD])\u200D[\u2640\u2642]|[\xA9\xAE\u203C\u2049\u2122\u2139\u2194-\u2199\u21A9\u21AA\u2328\u23CF\u23ED-\u23EF\u23F1\u23F2\u23F8-\u23FA\u24C2\u25AA\u25AB\u25B6\u25C0\u25FB\u25FC\u2600-\u2604\u260E\u2611\u2618\u2620\u2622\u2623\u2626\u262A\u262E\u262F\u2638-\u263A\u2640\u2642\u265F\u2660\u2663\u2665\u2666\u2668\u267B\u267E\u2692\u2694-\u2697\u2699\u269B\u269C\u26A0\u26A7\u26B0\u26B1\u26C8\u26CF\u26D1\u26D3\u26E9\u26F0\u26F1\u26F4\u26F7\u26F8\u2702\u2708\u2709\u270F\u2712\u2714\u2716\u271D\u2721\u2733\u2734\u2744\u2747\u2763\u27A1\u2934\u2935\u2B05-\u2B07\u3030\u303D\u3297\u3299]|\uD83C[\uDD70\uDD71\uDD7E\uDD7F\uDE02\uDE37\uDF21\uDF24-\uDF2C\uDF36\uDF7D\uDF96\uDF97\uDF99-\uDF9B\uDF9E\uDF9F\uDFCD\uDFCE\uDFD4-\uDFDF\uDFF5\uDFF7]|\uD83D[\uDC3F\uDCFD\uDD49\uDD4A\uDD6F\uDD70\uDD73\uDD76-\uDD79\uDD87\uDD8A-\uDD8D\uDDA5\uDDA8\uDDB1\uDDB2\uDDBC\uDDC2-\uDDC4\uDDD1-\uDDD3\uDDDC-\uDDDE\uDDE1\uDDE3\uDDE8\uDDEF\uDDF3\uDDFA\uDECB\uDECD-\uDECF\uDEE0-\uDEE5\uDEE9\uDEF0\uDEF3])\uFE0F|\uD83C\uDFF3\uFE0F\u200D\uD83C\uDF08|\uD83D\uDC69\u200D\uD83D\uDC67|\uD83D\uDC69\u200D\uD83D\uDC66|\uD83D\uDE35\u200D\uD83D\uDCAB|\uD83D\uDE2E\u200D\uD83D\uDCA8|\uD83D\uDC15\u200D\uD83E\uDDBA|\uD83E\uDDD1(?:\uD83C\uDFFF|\uD83C\uDFFE|\uD83C\uDFFD|\uD83C\uDFFC|\uD83C\uDFFB)?|\uD83D\uDC69(?:\uD83C\uDFFF|\uD83C\uDFFE|\uD83C\uDFFD|\uD83C\uDFFC|\uD83C\uDFFB)?|\uD83C\uDDFD\uD83C\uDDF0|\uD83C\uDDF6\uD83C\uDDE6|\uD83C\uDDF4\uD83C\uDDF2|\uD83D\uDC08\u200D\u2B1B|\u2764\uFE0F\u200D(?:\uD83D\uDD25|\uD83E\uDE79)|\uD83D\uDC41\uFE0F|\uD83C\uDFF3\uFE0F|\uD83C\uDDFF(?:\uD83C[\uDDE6\uDDF2\uDDFC])|\uD83C\uDDFE(?:\uD83C[\uDDEA\uDDF9])|\uD83C\uDDFC(?:\uD83C[\uDDEB\uDDF8])|\uD83C\uDDFB(?:\uD83C[\uDDE6\uDDE8\uDDEA\uDDEC\uDDEE\uDDF3\uDDFA])|\uD83C\uDDFA(?:\uD83C[\uDDE6\uDDEC\uDDF2\uDDF3\uDDF8\uDDFE\uDDFF])|\uD83C\uDDF9(?:\uD83C[\uDDE6\uDDE8\uDDE9\uDDEB-\uDDED\uDDEF-\uDDF4\uDDF7\uDDF9\uDDFB\uDDFC\uDDFF])|\uD83C\uDDF8(?:\uD83C[\uDDE6-\uDDEA\uDDEC-\uDDF4\uDDF7-\uDDF9\uDDFB\uDDFD-\uDDFF])|\uD83C\uDDF7(?:\uD83C[\uDDEA\uDDF4\uDDF8\uDDFA\uDDFC])|\uD83C\uDDF5(?:\uD83C[\uDDE6\uDDEA-\uDDED\uDDF0-\uDDF3\uDDF7-\uDDF9\uDDFC\uDDFE])|\uD83C\uDDF3(?:\uD83C[\uDDE6\uDDE8\uDDEA-\uDDEC\uDDEE\uDDF1\uDDF4\uDDF5\uDDF7\uDDFA\uDDFF])|\uD83C\uDDF2(?:\uD83C[\uDDE6\uDDE8-\uDDED\uDDF0-\uDDFF])|\uD83C\uDDF1(?:\uD83C[\uDDE6-\uDDE8\uDDEE\uDDF0\uDDF7-\uDDFB\uDDFE])|\uD83C\uDDF0(?:\uD83C[\uDDEA\uDDEC-\uDDEE\uDDF2\uDDF3\uDDF5\uDDF7\uDDFC\uDDFE\uDDFF])|\uD83C\uDDEF(?:\uD83C[\uDDEA\uDDF2\uDDF4\uDDF5])|\uD83C\uDDEE(?:\uD83C[\uDDE8-\uDDEA\uDDF1-\uDDF4\uDDF6-\uDDF9])|\uD83C\uDDED(?:\uD83C[\uDDF0\uDDF2\uDDF3\uDDF7\uDDF9\uDDFA])|\uD83C\uDDEC(?:\uD83C[\uDDE6\uDDE7\uDDE9-\uDDEE\uDDF1-\uDDF3\uDDF5-\uDDFA\uDDFC\uDDFE])|\uD83C\uDDEB(?:\uD83C[\uDDEE-\uDDF0\uDDF2\uDDF4\uDDF7])|\uD83C\uDDEA(?:\uD83C[\uDDE6\uDDE8\uDDEA\uDDEC\uDDED\uDDF7-\uDDFA])|\uD83C\uDDE9(?:\uD83C[\uDDEA\uDDEC\uDDEF\uDDF0\uDDF2\uDDF4\uDDFF])|\uD83C\uDDE8(?:\uD83C[\uDDE6\uDDE8\uDDE9\uDDEB-\uDDEE\uDDF0-\uDDF5\uDDF7\uDDFA-\uDDFF])|\uD83C\uDDE7(?:\uD83C[\uDDE6\uDDE7\uDDE9-\uDDEF\uDDF1-\uDDF4\uDDF6-\uDDF9\uDDFB\uDDFC\uDDFE\uDDFF])|\uD83C\uDDE6(?:\uD83C[\uDDE8-\uDDEC\uDDEE\uDDF1\uDDF2\uDDF4\uDDF6-\uDDFA\uDDFC\uDDFD\uDDFF])|[#\*0-9]\uFE0F\u20E3|\u2764\uFE0F|(?:\uD83C[\uDFC3\uDFC4\uDFCA]|\uD83D[\uDC6E\uDC70\uDC71\uDC73\uDC77\uDC81\uDC82\uDC86\uDC87\uDE45-\uDE47\uDE4B\uDE4D\uDE4E\uDEA3\uDEB4-\uDEB6]|\uD83E[\uDD26\uDD35\uDD37-\uDD39\uDD3D\uDD3E\uDDB8\uDDB9\uDDCD-\uDDCF\uDDD4\uDDD6-\uDDDD])(?:\uD83C[\uDFFB-\uDFFF])|(?:\u26F9|\uD83C[\uDFCB\uDFCC]|\uD83D\uDD75)(?:\uFE0F|\uD83C[\uDFFB-\uDFFF])|\uD83C\uDFF4|(?:[\u270A\u270B]|\uD83C[\uDF85\uDFC2\uDFC7]|\uD83D[\uDC42\uDC43\uDC46-\uDC50\uDC66\uDC67\uDC6B-\uDC6D\uDC72\uDC74-\uDC76\uDC78\uDC7C\uDC83\uDC85\uDC8F\uDC91\uDCAA\uDD7A\uDD95\uDD96\uDE4C\uDE4F\uDEC0\uDECC]|\uD83E[\uDD0C\uDD0F\uDD18-\uDD1C\uDD1E\uDD1F\uDD30-\uDD34\uDD36\uDD77\uDDB5\uDDB6\uDDBB\uDDD2\uDDD3\uDDD5])(?:\uD83C[\uDFFB-\uDFFF])|(?:[\u261D\u270C\u270D]|\uD83D[\uDD74\uDD90])(?:\uFE0F|\uD83C[\uDFFB-\uDFFF])|[\u270A\u270B]|\uD83C[\uDF85\uDFC2\uDFC7]|\uD83D[\uDC08\uDC15\uDC3B\uDC42\uDC43\uDC46-\uDC50\uDC66\uDC67\uDC6B-\uDC6D\uDC72\uDC74-\uDC76\uDC78\uDC7C\uDC83\uDC85\uDC8F\uDC91\uDCAA\uDD7A\uDD95\uDD96\uDE2E\uDE35\uDE36\uDE4C\uDE4F\uDEC0\uDECC]|\uD83E[\uDD0C\uDD0F\uDD18-\uDD1C\uDD1E\uDD1F\uDD30-\uDD34\uDD36\uDD77\uDDB5\uDDB6\uDDBB\uDDD2\uDDD3\uDDD5]|\uD83C[\uDFC3\uDFC4\uDFCA]|\uD83D[\uDC6E\uDC70\uDC71\uDC73\uDC77\uDC81\uDC82\uDC86\uDC87\uDE45-\uDE47\uDE4B\uDE4D\uDE4E\uDEA3\uDEB4-\uDEB6]|\uD83E[\uDD26\uDD35\uDD37-\uDD39\uDD3D\uDD3E\uDDB8\uDDB9\uDDCD-\uDDCF\uDDD4\uDDD6-\uDDDD]|\uD83D\uDC6F|\uD83E[\uDD3C\uDDDE\uDDDF]|[\u231A\u231B\u23E9-\u23EC\u23F0\u23F3\u25FD\u25FE\u2614\u2615\u2648-\u2653\u267F\u2693\u26A1\u26AA\u26AB\u26BD\u26BE\u26C4\u26C5\u26CE\u26D4\u26EA\u26F2\u26F3\u26F5\u26FA\u26FD\u2705\u2728\u274C\u274E\u2753-\u2755\u2757\u2795-\u2797\u27B0\u27BF\u2B1B\u2B1C\u2B50\u2B55]|\uD83C[\uDC04\uDCCF\uDD8E\uDD91-\uDD9A\uDE01\uDE1A\uDE2F\uDE32-\uDE36\uDE38-\uDE3A\uDE50\uDE51\uDF00-\uDF20\uDF2D-\uDF35\uDF37-\uDF7C\uDF7E-\uDF84\uDF86-\uDF93\uDFA0-\uDFC1\uDFC5\uDFC6\uDFC8\uDFC9\uDFCF-\uDFD3\uDFE0-\uDFF0\uDFF8-\uDFFF]|\uD83D[\uDC00-\uDC07\uDC09-\uDC14\uDC16-\uDC3A\uDC3C-\uDC3E\uDC40\uDC44\uDC45\uDC51-\uDC65\uDC6A\uDC79-\uDC7B\uDC7D-\uDC80\uDC84\uDC88-\uDC8E\uDC90\uDC92-\uDCA9\uDCAB-\uDCFC\uDCFF-\uDD3D\uDD4B-\uDD4E\uDD50-\uDD67\uDDA4\uDDFB-\uDE2D\uDE2F-\uDE34\uDE37-\uDE44\uDE48-\uDE4A\uDE80-\uDEA2\uDEA4-\uDEB3\uDEB7-\uDEBF\uDEC1-\uDEC5\uDED0-\uDED2\uDED5-\uDED7\uDEEB\uDEEC\uDEF4-\uDEFC\uDFE0-\uDFEB]|\uD83E[\uDD0D\uDD0E\uDD10-\uDD17\uDD1D\uDD20-\uDD25\uDD27-\uDD2F\uDD3A\uDD3F-\uDD45\uDD47-\uDD76\uDD78\uDD7A-\uDDB4\uDDB7\uDDBA\uDDBC-\uDDCB\uDDD0\uDDE0-\uDDFF\uDE70-\uDE74\uDE78-\uDE7A\uDE80-\uDE86\uDE90-\uDEA8\uDEB0-\uDEB6\uDEC0-\uDEC2\uDED0-\uDED6]|(?:[\u231A\u231B\u23E9-\u23EC\u23F0\u23F3\u25FD\u25FE\u2614\u2615\u2648-\u2653\u267F\u2693\u26A1\u26AA\u26AB\u26BD\u26BE\u26C4\u26C5\u26CE\u26D4\u26EA\u26F2\u26F3\u26F5\u26FA\u26FD\u2705\u270A\u270B\u2728\u274C\u274E\u2753-\u2755\u2757\u2795-\u2797\u27B0\u27BF\u2B1B\u2B1C\u2B50\u2B55]|\uD83C[\uDC04\uDCCF\uDD8E\uDD91-\uDD9A\uDDE6-\uDDFF\uDE01\uDE1A\uDE2F\uDE32-\uDE36\uDE38-\uDE3A\uDE50\uDE51\uDF00-\uDF20\uDF2D-\uDF35\uDF37-\uDF7C\uDF7E-\uDF93\uDFA0-\uDFCA\uDFCF-\uDFD3\uDFE0-\uDFF0\uDFF4\uDFF8-\uDFFF]|\uD83D[\uDC00-\uDC3E\uDC40\uDC42-\uDCFC\uDCFF-\uDD3D\uDD4B-\uDD4E\uDD50-\uDD67\uDD7A\uDD95\uDD96\uDDA4\uDDFB-\uDE4F\uDE80-\uDEC5\uDECC\uDED0-\uDED2\uDED5-\uDED7\uDEEB\uDEEC\uDEF4-\uDEFC\uDFE0-\uDFEB]|\uD83E[\uDD0C-\uDD3A\uDD3C-\uDD45\uDD47-\uDD78\uDD7A-\uDDCB\uDDCD-\uDDFF\uDE70-\uDE74\uDE78-\uDE7A\uDE80-\uDE86\uDE90-\uDEA8\uDEB0-\uDEB6\uDEC0-\uDEC2\uDED0-\uDED6])|(?:[#\*0-9\xA9\xAE\u203C\u2049\u2122\u2139\u2194-\u2199\u21A9\u21AA\u231A\u231B\u2328\u23CF\u23E9-\u23F3\u23F8-\u23FA\u24C2\u25AA\u25AB\u25B6\u25C0\u25FB-\u25FE\u2600-\u2604\u260E\u2611\u2614\u2615\u2618\u261D\u2620\u2622\u2623\u2626\u262A\u262E\u262F\u2638-\u263A\u2640\u2642\u2648-\u2653\u265F\u2660\u2663\u2665\u2666\u2668\u267B\u267E\u267F\u2692-\u2697\u2699\u269B\u269C\u26A0\u26A1\u26A7\u26AA\u26AB\u26B0\u26B1\u26BD\u26BE\u26C4\u26C5\u26C8\u26CE\u26CF\u26D1\u26D3\u26D4\u26E9\u26EA\u26F0-\u26F5\u26F7-\u26FA\u26FD\u2702\u2705\u2708-\u270D\u270F\u2712\u2714\u2716\u271D\u2721\u2728\u2733\u2734\u2744\u2747\u274C\u274E\u2753-\u2755\u2757\u2763\u2764\u2795-\u2797\u27A1\u27B0\u27BF\u2934\u2935\u2B05-\u2B07\u2B1B\u2B1C\u2B50\u2B55\u3030\u303D\u3297\u3299]|\uD83C[\uDC04\uDCCF\uDD70\uDD71\uDD7E\uDD7F\uDD8E\uDD91-\uDD9A\uDDE6-\uDDFF\uDE01\uDE02\uDE1A\uDE2F\uDE32-\uDE3A\uDE50\uDE51\uDF00-\uDF21\uDF24-\uDF93\uDF96\uDF97\uDF99-\uDF9B\uDF9E-\uDFF0\uDFF3-\uDFF5\uDFF7-\uDFFF]|\uD83D[\uDC00-\uDCFD\uDCFF-\uDD3D\uDD49-\uDD4E\uDD50-\uDD67\uDD6F\uDD70\uDD73-\uDD7A\uDD87\uDD8A-\uDD8D\uDD90\uDD95\uDD96\uDDA4\uDDA5\uDDA8\uDDB1\uDDB2\uDDBC\uDDC2-\uDDC4\uDDD1-\uDDD3\uDDDC-\uDDDE\uDDE1\uDDE3\uDDE8\uDDEF\uDDF3\uDDFA-\uDE4F\uDE80-\uDEC5\uDECB-\uDED2\uDED5-\uDED7\uDEE0-\uDEE5\uDEE9\uDEEB\uDEEC\uDEF0\uDEF3-\uDEFC\uDFE0-\uDFEB]|\uD83E[\uDD0C-\uDD3A\uDD3C-\uDD45\uDD47-\uDD78\uDD7A-\uDDCB\uDDCD-\uDDFF\uDE70-\uDE74\uDE78-\uDE7A\uDE80-\uDE86\uDE90-\uDEA8\uDEB0-\uDEB6\uDEC0-\uDEC2\uDED0-\uDED6])\uFE0F|(?:[\u261D\u26F9\u270A-\u270D]|\uD83C[\uDF85\uDFC2-\uDFC4\uDFC7\uDFCA-\uDFCC]|\uD83D[\uDC42\uDC43\uDC46-\uDC50\uDC66-\uDC78\uDC7C\uDC81-\uDC83\uDC85-\uDC87\uDC8F\uDC91\uDCAA\uDD74\uDD75\uDD7A\uDD90\uDD95\uDD96\uDE45-\uDE47\uDE4B-\uDE4F\uDEA3\uDEB4-\uDEB6\uDEC0\uDECC]|\uD83E[\uDD0C\uDD0F\uDD18-\uDD1F\uDD26\uDD30-\uDD39\uDD3C-\uDD3E\uDD77\uDDB5\uDDB6\uDDB8\uDDB9\uDDBB\uDDCD-\uDDCF\uDDD1-\uDDDD])/g;
};

var emojiRegex$1 = /*@__PURE__*/getDefaultExportFromCjs(emojiRegex);

function stringWidth(string, options = {}) {
	if (typeof string !== 'string' || string.length === 0) {
		return 0;
	}

	options = {
		ambiguousIsNarrow: true,
		...options
	};

	string = stripAnsi(string);

	if (string.length === 0) {
		return 0;
	}

	string = string.replace(emojiRegex$1(), '  ');

	const ambiguousCharacterWidth = options.ambiguousIsNarrow ? 1 : 2;
	let width = 0;

	for (const character of string) {
		const codePoint = character.codePointAt(0);

		// Ignore control characters
		if (codePoint <= 0x1F || (codePoint >= 0x7F && codePoint <= 0x9F)) {
			continue;
		}

		// Ignore combining characters
		if (codePoint >= 0x300 && codePoint <= 0x36F) {
			continue;
		}

		const code = eastAsianWidth.eastAsianWidth(character);
		switch (code) {
			case 'F':
			case 'W':
				width += 2;
				break;
			case 'A':
				width += ambiguousCharacterWidth;
				break;
			default:
				width += 1;
		}
	}

	return width;
}

function getIndexOfNearestSpace(string, wantedIndex, shouldSearchRight) {
	if (string.charAt(wantedIndex) === ' ') {
		return wantedIndex;
	}

	for (let index = 1; index <= 3; index++) {
		if (shouldSearchRight) {
			if (string.charAt(wantedIndex + index) === ' ') {
				return wantedIndex + index;
			}
		} else if (string.charAt(wantedIndex - index) === ' ') {
			return wantedIndex - index;
		}
	}

	return wantedIndex;
}

function cliTruncate(text, columns, options) {
	options = {
		position: 'end',
		preferTruncationOnSpace: false,
		truncationCharacter: '…',
		...options,
	};

	const {position, space, preferTruncationOnSpace} = options;
	let {truncationCharacter} = options;

	if (typeof text !== 'string') {
		throw new TypeError(`Expected \`input\` to be a string, got ${typeof text}`);
	}

	if (typeof columns !== 'number') {
		throw new TypeError(`Expected \`columns\` to be a number, got ${typeof columns}`);
	}

	if (columns < 1) {
		return '';
	}

	if (columns === 1) {
		return truncationCharacter;
	}

	const length = stringWidth(text);

	if (length <= columns) {
		return text;
	}

	if (position === 'start') {
		if (preferTruncationOnSpace) {
			const nearestSpace = getIndexOfNearestSpace(text, length - columns + 1, true);
			return truncationCharacter + sliceAnsi(text, nearestSpace, length).trim();
		}

		if (space === true) {
			truncationCharacter += ' ';
		}

		return truncationCharacter + sliceAnsi(text, length - columns + stringWidth(truncationCharacter), length);
	}

	if (position === 'middle') {
		if (space === true) {
			truncationCharacter = ` ${truncationCharacter} `;
		}

		const half = Math.floor(columns / 2);

		if (preferTruncationOnSpace) {
			const spaceNearFirstBreakPoint = getIndexOfNearestSpace(text, half);
			const spaceNearSecondBreakPoint = getIndexOfNearestSpace(text, length - (columns - half) + 1, true);
			return sliceAnsi(text, 0, spaceNearFirstBreakPoint) + truncationCharacter + sliceAnsi(text, spaceNearSecondBreakPoint, length).trim();
		}

		return (
			sliceAnsi(text, 0, half)
				+ truncationCharacter
				+ sliceAnsi(text, length - (columns - half) + stringWidth(truncationCharacter), length)
		);
	}

	if (position === 'end') {
		if (preferTruncationOnSpace) {
			const nearestSpace = getIndexOfNearestSpace(text, columns - 1);
			return sliceAnsi(text, 0, nearestSpace) + truncationCharacter;
		}

		if (space === true) {
			truncationCharacter = ` ${truncationCharacter}`;
		}

		return sliceAnsi(text, 0, columns - stringWidth(truncationCharacter)) + truncationCharacter;
	}

	throw new Error(`Expected \`options.position\` to be either \`start\`, \`middle\` or \`end\`, got ${position}`);
}

const DURATION_LONG$1 = 300;
const outputMap$1 = /* @__PURE__ */ new WeakMap();
function formatFilepath$1(path) {
  const lastSlash = Math.max(path.lastIndexOf("/") + 1, 0);
  const basename = path.slice(lastSlash);
  let firstDot = basename.indexOf(".");
  if (firstDot < 0)
    firstDot = basename.length;
  firstDot += lastSlash;
  return c.dim(path.slice(0, lastSlash)) + path.slice(lastSlash, firstDot) + c.dim(path.slice(firstDot));
}
function formatNumber$1(number) {
  const res = String(number.toFixed(number < 100 ? 4 : 2)).split(".");
  return res[0].replace(/(?=(?:\d{3})+$)(?!\b)/g, ",") + (res[1] ? `.${res[1]}` : "");
}
function renderHookState(task, hookName, level = 0) {
  var _a, _b;
  const state = (_b = (_a = task.result) == null ? void 0 : _a.hooks) == null ? void 0 : _b[hookName];
  if (state && state === "run")
    return `${"  ".repeat(level)} ${getHookStateSymbol(task, hookName)} ${c.dim(`[ ${hookName} ]`)}`;
  return "";
}
function renderBenchmarkItems$1(result) {
  return [
    result.name,
    formatNumber$1(result.hz || 0),
    formatNumber$1(result.p99 || 0),
    `\xB1${result.rme.toFixed(2)}%`,
    result.samples.length.toString()
  ];
}
function renderBenchmark$1(task, tasks) {
  var _a;
  const result = (_a = task.result) == null ? void 0 : _a.benchmark;
  if (!result)
    return task.name;
  const benches = tasks.map((i) => {
    var _a2, _b;
    return ((_a2 = i.meta) == null ? void 0 : _a2.benchmark) ? (_b = i.result) == null ? void 0 : _b.benchmark : void 0;
  }).filter(notNullish);
  const allItems = benches.map(renderBenchmarkItems$1);
  const items = renderBenchmarkItems$1(result);
  const padded = items.map((i, idx) => {
    const width = Math.max(...allItems.map((i2) => i2[idx].length));
    return idx ? i.padStart(width, " ") : i.padEnd(width, " ");
  });
  return [
    padded[0],
    // name
    c.dim("  "),
    c.blue(padded[1]),
    c.dim(" ops/sec "),
    c.cyan(padded[3]),
    c.dim(` (${padded[4]} samples)`),
    result.rank === 1 ? c.bold(c.green(" fastest")) : result.rank === benches.length && benches.length > 2 ? c.bold(c.gray(" slowest")) : ""
  ].join("");
}
function renderTree$1(tasks, options, level = 0, maxRows) {
  var _a, _b, _c, _d, _e, _f, _g, _h, _i;
  const output = [];
  let currentRowCount = 0;
  for (const task of [...tasks].reverse()) {
    const taskOutput = [];
    let suffix = "";
    let prefix = ` ${getStateSymbol(task)} `;
    if (level === 0 && task.type === "suite" && task.projectName)
      prefix += formatProjectName(task.projectName);
    if (task.type === "test" && ((_a = task.result) == null ? void 0 : _a.retryCount) && task.result.retryCount > 0)
      suffix += c.yellow(` (retry x${task.result.retryCount})`);
    if (task.type === "suite" && !((_b = task.meta) == null ? void 0 : _b.typecheck)) {
      const tests = getTests(task);
      suffix += c.dim(` (${tests.length})`);
    }
    if (task.mode === "skip" || task.mode === "todo")
      suffix += ` ${c.dim(c.gray("[skipped]"))}`;
    if (task.type === "test" && ((_c = task.result) == null ? void 0 : _c.repeatCount) && task.result.repeatCount > 0)
      suffix += c.yellow(` (repeat x${task.result.repeatCount})`);
    if (((_d = task.result) == null ? void 0 : _d.duration) != null) {
      if (task.result.duration > DURATION_LONG$1)
        suffix += c.yellow(` ${Math.round(task.result.duration)}${c.dim("ms")}`);
    }
    if (options.showHeap && ((_e = task.result) == null ? void 0 : _e.heap) != null)
      suffix += c.magenta(` ${Math.floor(task.result.heap / 1024 / 1024)} MB heap used`);
    let name = task.name;
    if (level === 0)
      name = formatFilepath$1(name);
    const padding = "  ".repeat(level);
    const body = ((_f = task.meta) == null ? void 0 : _f.benchmark) ? renderBenchmark$1(task, tasks) : name;
    taskOutput.push(padding + prefix + body + suffix);
    if (((_g = task.result) == null ? void 0 : _g.state) !== "pass" && outputMap$1.get(task) != null) {
      let data = outputMap$1.get(task);
      if (typeof data === "string") {
        data = stripAnsi(data.trim().split("\n").filter(Boolean).pop());
        if (data === "")
          data = void 0;
      }
      if (data != null) {
        const out = `${"  ".repeat(level)}${F_RIGHT} ${data}`;
        taskOutput.push(`   ${c.gray(cliTruncate(out, getCols(-3)))}`);
      }
    }
    taskOutput.push(renderHookState(task, "beforeAll", level + 1));
    taskOutput.push(renderHookState(task, "beforeEach", level + 1));
    if (task.type === "suite" && task.tasks.length > 0) {
      if (((_h = task.result) == null ? void 0 : _h.state) === "fail" || ((_i = task.result) == null ? void 0 : _i.state) === "run" || options.renderSucceed) {
        if (options.logger.ctx.config.hideSkippedTests) {
          const filteredTasks = task.tasks.filter((t) => t.mode !== "skip" && t.mode !== "todo");
          taskOutput.push(renderTree$1(filteredTasks, options, level + 1, maxRows));
        } else {
          taskOutput.push(renderTree$1(task.tasks, options, level + 1, maxRows));
        }
      }
    }
    taskOutput.push(renderHookState(task, "afterAll", level + 1));
    taskOutput.push(renderHookState(task, "afterEach", level + 1));
    const rows = taskOutput.filter(Boolean);
    output.push(rows.join("\n"));
    currentRowCount += rows.length;
    if (maxRows && currentRowCount >= maxRows)
      break;
  }
  return output.reverse().join("\n");
}
function createListRenderer(_tasks, options) {
  let tasks = _tasks;
  let timer;
  const log = options.logger.logUpdate;
  function update() {
    if (options.logger.ctx.config.hideSkippedTests) {
      const filteredTasks = tasks.filter((t) => t.mode !== "skip" && t.mode !== "todo");
      log(renderTree$1(
        filteredTasks,
        options,
        0,
        // log-update already limits the amount of printed rows to fit the current terminal
        // but we can optimize performance by doing it ourselves
        process.stdout.rows
      ));
    } else {
      log(renderTree$1(
        tasks,
        options,
        0,
        // log-update already limits the amount of printed rows to fit the current terminal
        // but we can optimize performance by doing it ourselves
        process.stdout.rows
      ));
    }
  }
  return {
    start() {
      if (timer)
        return this;
      timer = setInterval(update, 16);
      return this;
    },
    update(_tasks2) {
      tasks = _tasks2;
      return this;
    },
    async stop() {
      if (timer) {
        clearInterval(timer);
        timer = void 0;
      }
      log.clear();
      if (options.logger.ctx.config.hideSkippedTests) {
        const filteredTasks = tasks.filter((t) => t.mode !== "skip" && t.mode !== "todo");
        options.logger.log(renderTree$1(filteredTasks, options));
      } else {
        options.logger.log(renderTree$1(tasks, options));
      }
      return this;
    },
    clear() {
      log.clear();
    }
  };
}

class DefaultReporter extends BaseReporter {
  renderer;
  rendererOptions = {};
  renderSucceedDefault;
  onPathsCollected(paths = []) {
    if (this.isTTY) {
      if (this.renderSucceedDefault === void 0)
        this.renderSucceedDefault = !!this.rendererOptions.renderSucceed;
      if (this.renderSucceedDefault !== true)
        this.rendererOptions.renderSucceed = paths.length <= 1;
    }
  }
  async onTestRemoved(trigger) {
    await this.stopListRender();
    this.ctx.logger.clearScreen(c.yellow("Test removed...") + (trigger ? c.dim(` [ ${this.relative(trigger)} ]
`) : ""), true);
    const files = this.ctx.state.getFiles(this.watchFilters);
    createListRenderer(files, this.rendererOptions).stop();
    this.ctx.logger.log();
    await super.reportSummary(files, this.ctx.state.getUnhandledErrors());
    super.onWatcherStart();
  }
  onCollected() {
    if (this.isTTY) {
      this.rendererOptions.logger = this.ctx.logger;
      this.rendererOptions.showHeap = this.ctx.config.logHeapUsage;
      this.rendererOptions.mode = this.mode;
      const files = this.ctx.state.getFiles(this.watchFilters);
      if (!this.renderer)
        this.renderer = createListRenderer(files, this.rendererOptions).start();
      else
        this.renderer.update(files);
    }
  }
  async onFinished(files = this.ctx.state.getFiles(), errors = this.ctx.state.getUnhandledErrors()) {
    await this.stopListRender();
    this.ctx.logger.log();
    await super.onFinished(files, errors);
  }
  async onWatcherStart(files = this.ctx.state.getFiles(), errors = this.ctx.state.getUnhandledErrors()) {
    await this.stopListRender();
    await super.onWatcherStart(files, errors);
  }
  async stopListRender() {
    var _a;
    await ((_a = this.renderer) == null ? void 0 : _a.stop());
    this.renderer = void 0;
  }
  async onWatcherRerun(files, trigger) {
    await this.stopListRender();
    await super.onWatcherRerun(files, trigger);
  }
  onUserConsoleLog(log) {
    var _a;
    if (!this.shouldLog(log))
      return;
    (_a = this.renderer) == null ? void 0 : _a.clear();
    super.onUserConsoleLog(log);
  }
}

const check = { char: "\xB7", color: c.green };
const cross = { char: "x", color: c.red };
const pending = { char: "*", color: c.yellow };
const skip = { char: "-", color: (char) => c.dim(c.gray(char)) };
function getIcon(task) {
  var _a;
  if (task.mode === "skip" || task.mode === "todo")
    return skip;
  switch ((_a = task.result) == null ? void 0 : _a.state) {
    case "pass":
      return check;
    case "fail":
      return cross;
    default:
      return pending;
  }
}
function render(tasks, width) {
  const all = getTests(tasks);
  let currentIcon = pending;
  let currentTasks = 0;
  let previousLineWidth = 0;
  let output = "";
  const addOutput = () => {
    const { char, color } = currentIcon;
    const availableWidth = width - previousLineWidth;
    if (availableWidth > currentTasks) {
      output += color(char.repeat(currentTasks));
      previousLineWidth += currentTasks;
    } else {
      let buf = `${char.repeat(availableWidth)}
`;
      const remaining = currentTasks - availableWidth;
      const fullRows = Math.floor(remaining / width);
      buf += `${char.repeat(width)}
`.repeat(fullRows);
      const partialRow = remaining % width;
      if (partialRow > 0) {
        buf += char.repeat(partialRow);
        previousLineWidth = partialRow;
      } else {
        previousLineWidth = 0;
      }
      output += color(buf);
    }
  };
  for (const task of all) {
    const icon = getIcon(task);
    if (icon === currentIcon) {
      currentTasks++;
      continue;
    }
    addOutput();
    currentTasks = 1;
    currentIcon = icon;
  }
  addOutput();
  return output;
}
function createDotRenderer(_tasks, options) {
  let tasks = _tasks;
  let timer;
  const { logUpdate: log, outputStream } = options.logger;
  function update() {
    log(render(tasks, outputStream.columns));
  }
  return {
    start() {
      if (timer)
        return this;
      timer = setInterval(update, 16);
      return this;
    },
    update(_tasks2) {
      tasks = _tasks2;
      return this;
    },
    async stop() {
      if (timer) {
        clearInterval(timer);
        timer = void 0;
      }
      log.clear();
      options.logger.log(render(tasks, outputStream.columns));
      return this;
    },
    clear() {
      log.clear();
    }
  };
}

class DotReporter extends BaseReporter {
  renderer;
  onCollected() {
    if (this.isTTY) {
      const files = this.ctx.state.getFiles(this.watchFilters);
      if (!this.renderer)
        this.renderer = createDotRenderer(files, { logger: this.ctx.logger }).start();
      else
        this.renderer.update(files);
    }
  }
  async onFinished(files = this.ctx.state.getFiles(), errors = this.ctx.state.getUnhandledErrors()) {
    await this.stopListRender();
    this.ctx.logger.log();
    await super.onFinished(files, errors);
  }
  async onWatcherStart() {
    await this.stopListRender();
    super.onWatcherStart();
  }
  async stopListRender() {
    var _a;
    (_a = this.renderer) == null ? void 0 : _a.stop();
    this.renderer = void 0;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  async onWatcherRerun(files, trigger) {
    await this.stopListRender();
    await super.onWatcherRerun(files, trigger);
  }
  onUserConsoleLog(log) {
    var _a;
    (_a = this.renderer) == null ? void 0 : _a.clear();
    super.onUserConsoleLog(log);
  }
}

function getOutputFile(config, reporter) {
  if (!(config == null ? void 0 : config.outputFile))
    return;
  if (typeof config.outputFile === "string")
    return config.outputFile;
  return config.outputFile[reporter];
}

const StatusMap = {
  fail: "failed",
  only: "pending",
  pass: "passed",
  run: "pending",
  skip: "skipped",
  todo: "todo"
};
let JsonReporter$1 = class JsonReporter {
  start = 0;
  ctx;
  onInit(ctx) {
    this.ctx = ctx;
    this.start = Date.now();
  }
  async logTasks(files) {
    var _a, _b, _c;
    const suites = getSuites(files);
    const numTotalTestSuites = suites.length;
    const tests = getTests(files);
    const numTotalTests = tests.length;
    const numFailedTestSuites = suites.filter((s) => {
      var _a2;
      return (_a2 = s.result) == null ? void 0 : _a2.errors;
    }).length;
    const numPassedTestSuites = numTotalTestSuites - numFailedTestSuites;
    const numPendingTestSuites = suites.filter((s) => {
      var _a2;
      return ((_a2 = s.result) == null ? void 0 : _a2.state) === "run";
    }).length;
    const numFailedTests = tests.filter((t) => {
      var _a2;
      return ((_a2 = t.result) == null ? void 0 : _a2.state) === "fail";
    }).length;
    const numPassedTests = numTotalTests - numFailedTests;
    const numPendingTests = tests.filter((t) => {
      var _a2;
      return ((_a2 = t.result) == null ? void 0 : _a2.state) === "run";
    }).length;
    const numTodoTests = tests.filter((t) => t.mode === "todo").length;
    const testResults = [];
    const success = numFailedTestSuites === 0 && numFailedTests === 0;
    for (const file of files) {
      const tests2 = getTests([file]);
      let startTime = tests2.reduce((prev, next) => {
        var _a2;
        return Math.min(prev, ((_a2 = next.result) == null ? void 0 : _a2.startTime) ?? Number.POSITIVE_INFINITY);
      }, Number.POSITIVE_INFINITY);
      if (startTime === Number.POSITIVE_INFINITY)
        startTime = this.start;
      const endTime = tests2.reduce((prev, next) => {
        var _a2, _b2;
        return Math.max(prev, (((_a2 = next.result) == null ? void 0 : _a2.startTime) ?? 0) + (((_b2 = next.result) == null ? void 0 : _b2.duration) ?? 0));
      }, startTime);
      const assertionResults = await Promise.all(tests2.map(async (t) => {
        var _a2, _b2, _c2, _d;
        const ancestorTitles = [];
        let iter = t.suite;
        while (iter) {
          ancestorTitles.push(iter.name);
          iter = iter.suite;
        }
        ancestorTitles.reverse();
        return {
          ancestorTitles,
          fullName: ancestorTitles.length > 0 ? `${ancestorTitles.join(" ")} ${t.name}` : t.name,
          status: StatusMap[((_a2 = t.result) == null ? void 0 : _a2.state) || t.mode] || "skipped",
          title: t.name,
          duration: (_b2 = t.result) == null ? void 0 : _b2.duration,
          failureMessages: ((_d = (_c2 = t.result) == null ? void 0 : _c2.errors) == null ? void 0 : _d.map((e) => e.message)) || [],
          location: await this.getFailureLocation(t)
        };
      }));
      if (tests2.some((t) => {
        var _a2;
        return ((_a2 = t.result) == null ? void 0 : _a2.state) === "run";
      })) {
        this.ctx.logger.warn("WARNING: Some tests are still running when generating the JSON report.This is likely an internal bug in Vitest.Please report it to https://github.com/vitest-dev/vitest/issues");
      }
      testResults.push({
        assertionResults,
        startTime,
        endTime,
        status: tests2.some((t) => {
          var _a2;
          return ((_a2 = t.result) == null ? void 0 : _a2.state) === "fail";
        }) ? "failed" : "passed",
        message: ((_c = (_b = (_a = file.result) == null ? void 0 : _a.errors) == null ? void 0 : _b[0]) == null ? void 0 : _c.message) ?? "",
        name: file.filepath
      });
    }
    const result = {
      numTotalTestSuites,
      numPassedTestSuites,
      numFailedTestSuites,
      numPendingTestSuites,
      numTotalTests,
      numPassedTests,
      numFailedTests,
      numPendingTests,
      numTodoTests,
      startTime: this.start,
      success,
      testResults
    };
    await this.writeReport(JSON.stringify(result, null, 2));
  }
  async onFinished(files = this.ctx.state.getFiles()) {
    await this.logTasks(files);
  }
  /**
   * Writes the report to an output file if specified in the config,
   * or logs it to the console otherwise.
   * @param report
   */
  async writeReport(report) {
    const outputFile = getOutputFile(this.ctx.config, "json");
    if (outputFile) {
      const reportFile = resolve(this.ctx.config.root, outputFile);
      const outputDirectory = dirname(reportFile);
      if (!existsSync(outputDirectory))
        await promises.mkdir(outputDirectory, { recursive: true });
      await promises.writeFile(reportFile, report, "utf-8");
      this.ctx.logger.log(`JSON report written to ${reportFile}`);
    } else {
      this.ctx.logger.log(report);
    }
  }
  async getFailureLocation(test) {
    var _a, _b;
    const error = (_b = (_a = test.result) == null ? void 0 : _a.errors) == null ? void 0 : _b[0];
    if (!error)
      return;
    const project = this.ctx.getProjectByTaskId(test.id);
    const stack = parseErrorStacktrace(error, {
      getSourceMap: (file) => project.getBrowserSourceMapModuleById(file)
    });
    const frame = stack[0];
    if (!frame)
      return;
    return { line: frame.line, column: frame.column };
  }
};

class VerboseReporter extends DefaultReporter {
  constructor() {
    super();
    this.rendererOptions.renderSucceed = true;
  }
  onTaskUpdate(packs) {
    var _a, _b, _c, _d;
    if (this.isTTY)
      return;
    for (const pack of packs) {
      const task = this.ctx.state.idMap.get(pack[0]);
      if (task && task.type === "test" && ((_a = task.result) == null ? void 0 : _a.state) && ((_b = task.result) == null ? void 0 : _b.state) !== "run") {
        let title = ` ${getStateSymbol(task)} `;
        if ((_c = task.suite) == null ? void 0 : _c.projectName)
          title += formatProjectName(task.suite.projectName);
        title += getFullName(task, c.dim(" > "));
        if (this.ctx.config.logHeapUsage && task.result.heap != null)
          title += c.magenta(` ${Math.floor(task.result.heap / 1024 / 1024)} MB heap used`);
        this.ctx.logger.log(title);
        if (task.result.state === "fail") {
          (_d = task.result.errors) == null ? void 0 : _d.forEach((error) => {
            this.ctx.logger.log(c.red(`   ${F_RIGHT} ${error == null ? void 0 : error.message}`));
          });
        }
      }
    }
  }
}

class IndentedLogger {
  constructor(baseLog) {
    this.baseLog = baseLog;
  }
  currentIndent = "";
  indent() {
    this.currentIndent += "    ";
  }
  unindent() {
    this.currentIndent = this.currentIndent.substring(0, this.currentIndent.length - 4);
  }
  log(text) {
    return this.baseLog(this.currentIndent + text);
  }
}

function yamlString(str) {
  return `"${str.replace(/"/g, '\\"')}"`;
}
function tapString(str) {
  return str.replace(/\\/g, "\\\\").replace(/#/g, "\\#").replace(/\n/g, " ");
}
class TapReporter {
  ctx;
  logger;
  onInit(ctx) {
    this.ctx = ctx;
    this.logger = new IndentedLogger(ctx.logger.log.bind(ctx.logger));
  }
  static getComment(task) {
    var _a;
    if (task.mode === "skip")
      return " # SKIP";
    else if (task.mode === "todo")
      return " # TODO";
    else if (((_a = task.result) == null ? void 0 : _a.duration) != null)
      return ` # time=${task.result.duration.toFixed(2)}ms`;
    else
      return "";
  }
  logErrorDetails(error, stack) {
    this.logger.log(`name: ${yamlString(error.name)}`);
    this.logger.log(`message: ${yamlString(error.message)}`);
    if (stack) {
      this.logger.log(`stack: ${yamlString(`${stack.file}:${stack.line}:${stack.column}`)}`);
    }
  }
  logTasks(tasks) {
    var _a, _b;
    this.logger.log(`1..${tasks.length}`);
    for (const [i, task] of tasks.entries()) {
      const id = i + 1;
      const ok = ((_a = task.result) == null ? void 0 : _a.state) === "pass" || task.mode === "skip" || task.mode === "todo" ? "ok" : "not ok";
      const comment = TapReporter.getComment(task);
      if (task.type === "suite" && task.tasks.length > 0) {
        this.logger.log(`${ok} ${id} - ${tapString(task.name)}${comment} {`);
        this.logger.indent();
        this.logTasks(task.tasks);
        this.logger.unindent();
        this.logger.log("}");
      } else {
        this.logger.log(`${ok} ${id} - ${tapString(task.name)}${comment}`);
        const project = this.ctx.getProjectByTaskId(task.id);
        if (((_b = task.result) == null ? void 0 : _b.state) === "fail" && task.result.errors) {
          this.logger.indent();
          task.result.errors.forEach((error) => {
            const stacks = parseErrorStacktrace(error, {
              getSourceMap: (file) => project.getBrowserSourceMapModuleById(file)
            });
            const stack = stacks[0];
            this.logger.log("---");
            this.logger.log("error:");
            this.logger.indent();
            this.logErrorDetails(error);
            this.logger.unindent();
            if (stack)
              this.logger.log(`at: ${yamlString(`${stack.file}:${stack.line}:${stack.column}`)}`);
            if (error.showDiff) {
              this.logger.log(`actual: ${yamlString(error.actual)}`);
              this.logger.log(`expected: ${yamlString(error.expected)}`);
            }
          });
          this.logger.log("...");
          this.logger.unindent();
        }
      }
    }
  }
  async onFinished(files = this.ctx.state.getFiles()) {
    this.logger.log("TAP version 13");
    this.logTasks(files);
  }
}

function flattenTasks$1(task, baseName = "") {
  const base = baseName ? `${baseName} > ` : "";
  if (task.type === "suite") {
    return task.tasks.flatMap((child) => flattenTasks$1(child, `${base}${task.name}`));
  } else {
    return [{
      ...task,
      name: `${base}${task.name}`
    }];
  }
}
function removeInvalidXMLCharacters(value, removeDiscouragedChars) {
  let regex = /((?:[\0-\x08\x0B\f\x0E-\x1F\uFFFD\uFFFE\uFFFF]|[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?:[^\uD800-\uDBFF]|^)[\uDC00-\uDFFF]))/g;
  value = String(value || "").replace(regex, "");
  if (removeDiscouragedChars) {
    regex = new RegExp(
      "([\\x7F-\\x84]|[\\x86-\\x9F]|[\\uFDD0-\\uFDEF]|(?:\\uD83F[\\uDFFE\\uDFFF])|(?:\\uD87F[\\uDFFE\\uDFFF])|(?:\\uD8BF[\\uDFFE\\uDFFF])|(?:\\uD8FF[\\uDFFE\\uDFFF])|(?:\\uD93F[\\uDFFE\\uDFFF])|(?:\\uD97F[\\uDFFE\\uDFFF])|(?:\\uD9BF[\\uDFFE\\uDFFF])|(?:\\uD9FF[\\uDFFE\\uDFFF])|(?:\\uDA3F[\\uDFFE\\uDFFF])|(?:\\uDA7F[\\uDFFE\\uDFFF])|(?:\\uDABF[\\uDFFE\\uDFFF])|(?:\\uDAFF[\\uDFFE\\uDFFF])|(?:\\uDB3F[\\uDFFE\\uDFFF])|(?:\\uDB7F[\\uDFFE\\uDFFF])|(?:\\uDBBF[\\uDFFE\\uDFFF])|(?:\\uDBFF[\\uDFFE\\uDFFF])(?:[\\0-\\t\\x0B\\f\\x0E-\\u2027\\u202A-\\uD7FF\\uE000-\\uFFFF]|[\\uD800-\\uDBFF][\\uDC00-\\uDFFF]|[\\uD800-\\uDBFF](?![\\uDC00-\\uDFFF])|(?:[^\\uD800-\\uDBFF]|^)[\\uDC00-\\uDFFF]))",
      "g"
    );
    value = value.replace(regex, "");
  }
  return value;
}
function escapeXML(value) {
  return removeInvalidXMLCharacters(
    String(value).replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/'/g, "&apos;").replace(/</g, "&lt;").replace(/>/g, "&gt;"),
    true
  );
}
function executionTime(durationMS) {
  return (durationMS / 1e3).toLocaleString("en-US", { useGrouping: false, maximumFractionDigits: 10 });
}
function getDuration(task) {
  var _a;
  const duration = ((_a = task.result) == null ? void 0 : _a.duration) ?? 0;
  return executionTime(duration);
}
class JUnitReporter {
  ctx;
  reportFile;
  baseLog;
  logger;
  _timeStart = /* @__PURE__ */ new Date();
  fileFd;
  async onInit(ctx) {
    this.ctx = ctx;
    const outputFile = getOutputFile(this.ctx.config, "junit");
    if (outputFile) {
      this.reportFile = resolve(this.ctx.config.root, outputFile);
      const outputDirectory = dirname(this.reportFile);
      if (!existsSync(outputDirectory))
        await promises.mkdir(outputDirectory, { recursive: true });
      const fileFd = await promises.open(this.reportFile, "w+");
      this.fileFd = fileFd;
      this.baseLog = async (text) => {
        if (!this.fileFd)
          this.fileFd = await promises.open(this.reportFile, "w+");
        await promises.writeFile(this.fileFd, `${text}
`);
      };
    } else {
      this.baseLog = async (text) => this.ctx.logger.log(text);
    }
    this._timeStart = /* @__PURE__ */ new Date();
    this.logger = new IndentedLogger(this.baseLog);
  }
  async writeElement(name, attrs, children) {
    const pairs = [];
    for (const key in attrs) {
      const attr = attrs[key];
      if (attr === void 0)
        continue;
      pairs.push(`${key}="${escapeXML(attr)}"`);
    }
    await this.logger.log(`<${name}${pairs.length ? ` ${pairs.join(" ")}` : ""}>`);
    this.logger.indent();
    await children.call(this);
    this.logger.unindent();
    await this.logger.log(`</${name}>`);
  }
  async writeErrorDetails(task, error) {
    const errorName = error.name ?? error.nameStr ?? "Unknown Error";
    const errorDetails = `${errorName}: ${error.message}`;
    await this.baseLog(escapeXML(errorDetails));
    const project = this.ctx.getProjectByTaskId(task.id);
    const stack = parseErrorStacktrace(error, {
      getSourceMap: (file) => project.getBrowserSourceMapModuleById(file)
    });
    for (const frame of stack) {
      const path = relative(this.ctx.config.root, frame.file);
      await this.baseLog(escapeXML(` ${F_POINTER} ${[frame.method, `${path}:${frame.line}:${frame.column}`].filter(Boolean).join(" ")}`));
      if (frame.file in this.ctx.state.filesMap)
        break;
    }
  }
  async writeLogs(task, type) {
    if (task.logs == null || task.logs.length === 0)
      return;
    const logType = type === "err" ? "stderr" : "stdout";
    const logs = task.logs.filter((log) => log.type === logType);
    if (logs.length === 0)
      return;
    await this.writeElement(`system-${type}`, {}, async () => {
      for (const log of logs)
        await this.baseLog(escapeXML(log.content));
    });
  }
  async writeTasks(tasks, filename) {
    for (const task of tasks) {
      await this.writeElement("testcase", {
        classname: process.env.VITEST_JUNIT_CLASSNAME ?? filename,
        name: task.name,
        time: getDuration(task)
      }, async () => {
        var _a, _b;
        await this.writeLogs(task, "out");
        await this.writeLogs(task, "err");
        if (task.mode === "skip" || task.mode === "todo")
          await this.logger.log("<skipped/>");
        if (((_a = task.result) == null ? void 0 : _a.state) === "fail") {
          const errors = ((_b = task.result.errors) == null ? void 0 : _b.length) ? task.result.errors : [task.result.error];
          for (const error of errors) {
            await this.writeElement("failure", {
              message: error == null ? void 0 : error.message,
              type: (error == null ? void 0 : error.name) ?? (error == null ? void 0 : error.nameStr)
            }, async () => {
              if (!error)
                return;
              await this.writeErrorDetails(task, error);
            });
          }
        }
      });
    }
  }
  async onFinished(files = this.ctx.state.getFiles()) {
    var _a;
    await this.logger.log('<?xml version="1.0" encoding="UTF-8" ?>');
    const transformed = files.map((file) => {
      var _a2;
      const tasks = file.tasks.flatMap((task) => flattenTasks$1(task));
      const stats2 = tasks.reduce(
        (stats3, task) => {
          var _a3, _b;
          return {
            passed: stats3.passed + Number(((_a3 = task.result) == null ? void 0 : _a3.state) === "pass"),
            failures: stats3.failures + Number(((_b = task.result) == null ? void 0 : _b.state) === "fail"),
            skipped: stats3.skipped + Number(task.mode === "skip" || task.mode === "todo")
          };
        },
        {
          passed: 0,
          failures: 0,
          skipped: 0
        }
      );
      if (tasks.length === 0 && ((_a2 = file.result) == null ? void 0 : _a2.state) === "fail") {
        stats2.failures = 1;
        tasks.push({
          id: file.id,
          type: "test",
          name: file.name,
          mode: "run",
          result: file.result,
          meta: {},
          // NOTE: not used in JUnitReporter
          context: null,
          suite: null
        });
      }
      return {
        ...file,
        tasks,
        stats: stats2
      };
    });
    const stats = transformed.reduce((stats2, file) => {
      stats2.tests += file.tasks.length;
      stats2.failures += file.stats.failures;
      return stats2;
    }, {
      name: process.env.VITEST_JUNIT_SUITE_NAME || "vitest tests",
      tests: 0,
      failures: 0,
      errors: 0,
      // we cannot detect those
      time: executionTime((/* @__PURE__ */ new Date()).getTime() - this._timeStart.getTime())
    });
    await this.writeElement("testsuites", stats, async () => {
      for (const file of transformed) {
        await this.writeElement("testsuite", {
          name: file.name,
          timestamp: (/* @__PURE__ */ new Date()).toISOString(),
          hostname: hostname(),
          tests: file.tasks.length,
          failures: file.stats.failures,
          errors: 0,
          // An errored test is one that had an unanticipated problem. We cannot detect those.
          skipped: file.stats.skipped,
          time: getDuration(file)
        }, async () => {
          await this.writeTasks(file.tasks, file.name);
        });
      }
    });
    if (this.reportFile)
      this.ctx.logger.log(`JUNIT report written to ${this.reportFile}`);
    await ((_a = this.fileFd) == null ? void 0 : _a.close());
    this.fileFd = void 0;
  }
}

function flattenTasks(task, baseName = "") {
  const base = baseName ? `${baseName} > ` : "";
  if (task.type === "suite" && task.tasks.length > 0) {
    return task.tasks.flatMap((child) => flattenTasks(child, `${base}${task.name}`));
  } else {
    return [{
      ...task,
      name: `${base}${task.name}`
    }];
  }
}
class TapFlatReporter extends TapReporter {
  onInit(ctx) {
    super.onInit(ctx);
  }
  async onFinished(files = this.ctx.state.getFiles()) {
    this.ctx.logger.log("TAP version 13");
    const flatTasks = files.flatMap((task) => flattenTasks(task));
    this.logTasks(flatTasks);
  }
}

class HangingProcessReporter {
  whyRunning;
  onInit() {
    const _require = createRequire(import.meta.url);
    this.whyRunning = _require("why-is-node-running");
  }
  onProcessTimeout() {
    var _a;
    (_a = this.whyRunning) == null ? void 0 : _a.call(this);
  }
}

class JsonReporter {
  start = 0;
  ctx;
  onInit(ctx) {
    this.ctx = ctx;
  }
  async logTasks(files) {
    var _a;
    const suites = getSuites(files);
    const numTotalTestSuites = suites.length;
    const tests = getTests(files);
    const numTotalTests = tests.length;
    const testResults = {};
    const outputFile = getOutputFile(this.ctx.config.benchmark, "json");
    for (const file of files) {
      const tests2 = getTests([file]);
      for (const test of tests2) {
        const res = (_a = test.result) == null ? void 0 : _a.benchmark;
        if (!res || test.mode === "skip")
          continue;
        if (!outputFile)
          res.samples = "ignore on terminal";
        testResults[test.suite.name] = (testResults[test.suite.name] || []).concat(res);
      }
      if (tests2.some((t) => {
        var _a2;
        return ((_a2 = t.result) == null ? void 0 : _a2.state) === "run";
      })) {
        this.ctx.logger.warn("WARNING: Some tests are still running when generating the json report.This is likely an internal bug in Vitest.Please report it to https://github.com/vitest-dev/vitest/issues");
      }
    }
    const result = {
      numTotalTestSuites,
      numTotalTests,
      testResults
    };
    await this.writeReport(JSON.stringify(result, null, 2));
  }
  async onFinished(files = this.ctx.state.getFiles()) {
    await this.logTasks(files);
  }
  /**
   * Writes the report to an output file if specified in the config,
   * or logs it to the console otherwise.
   * @param report
   */
  async writeReport(report) {
    const outputFile = getOutputFile(this.ctx.config.benchmark, "json");
    if (outputFile) {
      const reportFile = resolve(this.ctx.config.root, outputFile);
      const outputDirectory = dirname(reportFile);
      if (!existsSync(outputDirectory))
        await promises.mkdir(outputDirectory, { recursive: true });
      await promises.writeFile(reportFile, report, "utf-8");
      this.ctx.logger.log(`json report written to ${reportFile}`);
    } else {
      this.ctx.logger.log(report);
    }
  }
}

const DURATION_LONG = 300;
const outputMap = /* @__PURE__ */ new WeakMap();
function formatFilepath(path) {
  const lastSlash = Math.max(path.lastIndexOf("/") + 1, 0);
  const basename = path.slice(lastSlash);
  let firstDot = basename.indexOf(".");
  if (firstDot < 0)
    firstDot = basename.length;
  firstDot += lastSlash;
  return c.dim(path.slice(0, lastSlash)) + path.slice(lastSlash, firstDot) + c.dim(path.slice(firstDot));
}
function formatNumber(number) {
  const res = String(number.toFixed(number < 100 ? 4 : 2)).split(".");
  return res[0].replace(/(?=(?:\d{3})+$)(?!\b)/g, ",") + (res[1] ? `.${res[1]}` : "");
}
const tableHead = ["name", "hz", "min", "max", "mean", "p75", "p99", "p995", "p999", "rme", "samples"];
function renderTableHead(tasks) {
  const benches = tasks.map((i) => {
    var _a, _b;
    return ((_a = i.meta) == null ? void 0 : _a.benchmark) ? (_b = i.result) == null ? void 0 : _b.benchmark : void 0;
  }).filter(notNullish);
  const allItems = benches.map(renderBenchmarkItems).concat([tableHead]);
  return `${" ".repeat(3)}${tableHead.map((i, idx) => {
    const width = Math.max(...allItems.map((i2) => i2[idx].length));
    return idx ? i.padStart(width, " ") : i.padEnd(width, " ");
  }).map(c.bold).join("  ")}`;
}
function renderBenchmarkItems(result) {
  return [
    result.name,
    formatNumber(result.hz || 0),
    formatNumber(result.min || 0),
    formatNumber(result.max || 0),
    formatNumber(result.mean || 0),
    formatNumber(result.p75 || 0),
    formatNumber(result.p99 || 0),
    formatNumber(result.p995 || 0),
    formatNumber(result.p999 || 0),
    `\xB1${(result.rme || 0).toFixed(2)}%`,
    result.samples.length.toString()
  ];
}
function renderBenchmark(task, tasks) {
  var _a;
  const result = (_a = task.result) == null ? void 0 : _a.benchmark;
  if (!result)
    return task.name;
  const benches = tasks.map((i) => {
    var _a2, _b;
    return ((_a2 = i.meta) == null ? void 0 : _a2.benchmark) ? (_b = i.result) == null ? void 0 : _b.benchmark : void 0;
  }).filter(notNullish);
  const allItems = benches.map(renderBenchmarkItems).concat([tableHead]);
  const items = renderBenchmarkItems(result);
  const padded = items.map((i, idx) => {
    const width = Math.max(...allItems.map((i2) => i2[idx].length));
    return idx ? i.padStart(width, " ") : i.padEnd(width, " ");
  });
  return [
    padded[0],
    // name
    c.blue(padded[1]),
    // hz
    c.cyan(padded[2]),
    // min
    c.cyan(padded[3]),
    // max
    c.cyan(padded[4]),
    // mean
    c.cyan(padded[5]),
    // p75
    c.cyan(padded[6]),
    // p99
    c.cyan(padded[7]),
    // p995
    c.cyan(padded[8]),
    // p999
    c.dim(padded[9]),
    // rem
    c.dim(padded[10]),
    // sample
    result.rank === 1 ? c.bold(c.green(" fastest")) : result.rank === benches.length && benches.length > 2 ? c.bold(c.gray(" slowest")) : ""
  ].join("  ");
}
function renderTree(tasks, options, level = 0) {
  var _a, _b, _c, _d, _e, _f;
  const output = [];
  let idx = 0;
  for (const task of tasks) {
    const padding = "  ".repeat(level ? 1 : 0);
    let prefix = "";
    if (idx === 0 && ((_a = task.meta) == null ? void 0 : _a.benchmark))
      prefix += `${renderTableHead(tasks)}
${padding}`;
    prefix += ` ${getStateSymbol(task)} `;
    let suffix = "";
    if (task.type === "suite")
      suffix += c.dim(` (${getTests(task).length})`);
    if (task.mode === "skip" || task.mode === "todo")
      suffix += ` ${c.dim(c.gray("[skipped]"))}`;
    if (((_b = task.result) == null ? void 0 : _b.duration) != null) {
      if (task.result.duration > DURATION_LONG)
        suffix += c.yellow(` ${Math.round(task.result.duration)}${c.dim("ms")}`);
    }
    if (options.showHeap && ((_c = task.result) == null ? void 0 : _c.heap) != null)
      suffix += c.magenta(` ${Math.floor(task.result.heap / 1024 / 1024)} MB heap used`);
    let name = task.name;
    if (level === 0)
      name = formatFilepath(name);
    const body = ((_d = task.meta) == null ? void 0 : _d.benchmark) ? renderBenchmark(task, tasks) : name;
    output.push(padding + prefix + body + suffix);
    if (((_e = task.result) == null ? void 0 : _e.state) !== "pass" && outputMap.get(task) != null) {
      let data = outputMap.get(task);
      if (typeof data === "string") {
        data = stripAnsi(data.trim().split("\n").filter(Boolean).pop());
        if (data === "")
          data = void 0;
      }
      if (data != null) {
        const out = `${"  ".repeat(level)}${F_RIGHT} ${data}`;
        output.push(`   ${c.gray(cliTruncate(out, getCols(-3)))}`);
      }
    }
    if (task.type === "suite" && task.tasks.length > 0) {
      if ((_f = task.result) == null ? void 0 : _f.state)
        output.push(renderTree(task.tasks, options, level + 1));
    }
    idx++;
  }
  return output.filter(Boolean).join("\n");
}
function createTableRenderer(_tasks, options) {
  let tasks = _tasks;
  let timer;
  const log = options.logger.logUpdate;
  function update() {
    log(renderTree(tasks, options));
  }
  return {
    start() {
      if (timer)
        return this;
      timer = setInterval(update, 200);
      return this;
    },
    update(_tasks2) {
      tasks = _tasks2;
      update();
      return this;
    },
    async stop() {
      if (timer) {
        clearInterval(timer);
        timer = void 0;
      }
      log.clear();
      options.logger.log(renderTree(tasks, options));
      return this;
    },
    clear() {
      log.clear();
    }
  };
}

class TableReporter extends BaseReporter {
  renderer;
  rendererOptions = {};
  async onTestRemoved(trigger) {
    await this.stopListRender();
    this.ctx.logger.clearScreen(c.yellow("Test removed...") + (trigger ? c.dim(` [ ${this.relative(trigger)} ]
`) : ""), true);
    const files = this.ctx.state.getFiles(this.watchFilters);
    createTableRenderer(files, this.rendererOptions).stop();
    this.ctx.logger.log();
    await super.reportSummary(files, this.ctx.state.getUnhandledErrors());
    super.onWatcherStart();
  }
  onCollected() {
    if (this.isTTY) {
      this.rendererOptions.logger = this.ctx.logger;
      this.rendererOptions.showHeap = this.ctx.config.logHeapUsage;
      const files = this.ctx.state.getFiles(this.watchFilters);
      if (!this.renderer)
        this.renderer = createTableRenderer(files, this.rendererOptions).start();
      else
        this.renderer.update(files);
    }
  }
  async onFinished(files = this.ctx.state.getFiles(), errors = this.ctx.state.getUnhandledErrors()) {
    await this.stopListRender();
    this.ctx.logger.log();
    await super.onFinished(files, errors);
  }
  async onWatcherStart() {
    await this.stopListRender();
    await super.onWatcherStart();
  }
  async stopListRender() {
    var _a;
    await ((_a = this.renderer) == null ? void 0 : _a.stop());
    this.renderer = void 0;
  }
  async onWatcherRerun(files, trigger) {
    await this.stopListRender();
    await super.onWatcherRerun(files, trigger);
  }
  onUserConsoleLog(log) {
    var _a;
    if (!this.shouldLog(log))
      return;
    (_a = this.renderer) == null ? void 0 : _a.clear();
    super.onUserConsoleLog(log);
  }
}

const BenchmarkReportsMap = {
  default: TableReporter,
  verbose: VerboseReporter,
  json: JsonReporter
};

const ReportersMap = {
  "default": DefaultReporter,
  "basic": BasicReporter,
  "verbose": VerboseReporter,
  "dot": DotReporter,
  "json": JsonReporter$1,
  "tap": TapReporter,
  "tap-flat": TapFlatReporter,
  "junit": JUnitReporter,
  "hanging-process": HangingProcessReporter
};

export { BenchmarkReportsMap as B, DefaultReporter as D, F_POINTER as F, HangingProcessReporter as H, JsonReporter$1 as J, ReportersMap as R, TapReporter as T, VerboseReporter as V, ansiStyles as a, stripAnsi as b, sliceAnsi as c, divider as d, cliTruncate as e, BasicReporter as f, DotReporter as g, JUnitReporter as h, TapFlatReporter as i, stringWidth as s };
