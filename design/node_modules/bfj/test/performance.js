#!/usr/bin/env node

'use strict'

const fs = require('fs')
const path = require('path')
const check = require('check-types')
const bfj = require('../src')

const inPath = getDataPath('.json');

let time = process.hrtime()

if (process.argv.length === 4) {
  const stuff = []
  const stream = bfj.match(fs.createReadStream(inPath), process.argv[3])
  stream.on('data', thing => stuff.push(thing))
  stream.on('end', () => {
    reportTime()
    console.log('hooray!', stuff.length)
    fs.writeFileSync(getDataPath('-result.ndjson'), stuff.map(s => JSON.stringify(s)).join('\n'), {
      encoding: 'utf8',
    })
    process.exit(0)
  })
  stream.on('error', error => {
    console.error('error!', error.stack)
    process.exit(1)
  })
  stream.on('dataError', error => {
    console.error('dataError!', error.stack)
    process.exit(2)
  })
} else {
  console.log('reading json')
  bfj.read(inPath)
    .then(data => {
      reportTime()
      console.log('writing json')
      return bfj.write(getDataPath('-result.json'), data)
    })
    .then(() => done('succeeded'))
    .catch(error => done(error.stack, 1))
}

function getDataPath (suffix) {
  return path.resolve(__dirname, process.argv[2] + suffix)
}

function reportTime () {
  let interimTime = process.hrtime(time)
  console.log('%d seconds and %d nanoseconds', interimTime[0], interimTime[1])
  time = process.hrtime()
}

function done (message, code) {
  reportTime()
  console.log(message)
  process.exit(code)
}

