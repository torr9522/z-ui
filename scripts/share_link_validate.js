#!/usr/bin/env node

const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

function loadFrontendModel() {
  const root = path.resolve(__dirname, '..');
  const sources = [
    'web/assets/js/util/common.js',
    'web/assets/js/util/utils.js',
    'web/assets/js/model/xray.js',
  ];

  const sandbox = {
    console,
    URL,
    URLSearchParams,
    Buffer,
    Math,
    Date,
    JSON,
    Array,
    Object,
    String,
    Number,
    Boolean,
    Map,
    Set,
    parseInt,
    parseFloat,
    isNaN,
    encodeURIComponent,
    decodeURIComponent,
    Base64: {
      encode(str) {
        return Buffer.from(str, 'utf8').toString('base64');
      },
    },
    axios: {},
    Vue: { prototype: {} },
    moment() {
      return null;
    },
    location: { hostname: '127.0.0.1' },
  };

  vm.createContext(sandbox);
  const bundle = sources
    .map((relativePath) => fs.readFileSync(path.join(root, relativePath), 'utf8'))
    .join('\n\n');
  vm.runInContext(
    `${bundle}
this.__shareLinkTest = {
  Protocols,
  Inbound,
  StreamSettings,
};`,
    sandbox,
    { filename: 'x-ui-share-link-validate.js' },
  );
  return sandbox.__shareLinkTest;
}

function decodeBase64UrlSafe(value) {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
  const padding = normalized.length % 4;
  const suffix = padding === 0 ? '' : '='.repeat(4 - padding);
  return Buffer.from(normalized + suffix, 'base64').toString('utf8');
}

function parseVmess(link) {
  assert(link.startsWith('vmess://'), `unexpected vmess link: ${link}`);
  const payload = link.slice('vmess://'.length);
  return JSON.parse(Buffer.from(payload, 'base64').toString('utf8'));
}

function parseSS(link) {
  assert(link.startsWith('ss://'), `unexpected ss link: ${link}`);
  const parts = link.slice('ss://'.length).split('#');
  const decoded = decodeBase64UrlSafe(parts[0]);
  const match = decoded.match(/^([^:]+):([^@]+)@(.+):(\d+)$/);
  assert(match, `unable to parse ss payload: ${decoded}`);
  return {
    method: match[1],
    password: match[2],
    host: match[3],
    port: match[4],
    remark: decodeURIComponent(parts[1] || ''),
  };
}

function check(name, fn) {
  try {
    fn();
    console.log(`PASS ${name}`);
  } catch (error) {
    console.error(`FAIL ${name}`);
    console.error(error.stack || error.message || String(error));
    process.exitCode = 1;
  }
}

const { Protocols, Inbound, StreamSettings } = loadFrontendModel();

check('vmess share link format', () => {
  const inbound = new Inbound(443, '', Protocols.VMESS, null, new StreamSettings('xhttp', 'tls'));
  inbound.settings.set('clients.0.id', '11111111-1111-1111-1111-111111111111');
  inbound.settings.set('clients.0.alterId', 0);
  inbound.stream.tls.server = 'vmess.example.com';
  inbound.stream.xhttp.path = '/mesh';
  inbound.stream.xhttp.host = 'edge.example.com';
  inbound.stream.xhttp.mode = 'stream-one';

  const payload = parseVmess(inbound.genLink('45.77.246.87', 'vmess node'));
  assert.equal(payload.add, 'vmess.example.com');
  assert.equal(payload.port, 443);
  assert.equal(payload.id, '11111111-1111-1111-1111-111111111111');
  assert.equal(payload.net, 'xhttp');
  assert.equal(payload.host, 'edge.example.com');
  assert.equal(payload.path, '/mesh');
  assert.equal(payload.tls, 'tls');
});

check('vless share link includes reality xhttp and vision params', () => {
  const inbound = new Inbound(443, '', Protocols.VLESS, null, new StreamSettings('xhttp', 'reality'));
  inbound.settings.set('clients.0.id', '22222222-2222-2222-2222-222222222222');
  inbound.settings.set('clients.0.flow', 'xtls-rprx-vision');
  inbound.settings.set('decryption', 'none');
  inbound.stream.reality.publicKey = 'pubkey';
  inbound.stream.reality.shortIds = ['abcd1234'];
  inbound.stream.reality.serverNames = ['reality.example.com'];
  inbound.stream.reality.spiderX = '/spider';
  inbound.stream.xhttp.path = '/xhttp';
  inbound.stream.xhttp.host = 'edge.example.com';
  inbound.stream.xhttp.mode = 'stream-one';

  const link = inbound.genLink('45.77.246.87', 'vless node');
  const url = new URL(link);
  assert.equal(url.protocol, 'vless:');
  assert.equal(decodeURIComponent(url.hash.slice(1)), 'vless node');
  assert.equal(url.username, '22222222-2222-2222-2222-222222222222');
  assert.equal(url.hostname, '45.77.246.87');
  assert.equal(url.port, '443');
  assert.equal(url.searchParams.get('type'), 'xhttp');
  assert.equal(url.searchParams.get('security'), 'reality');
  assert.equal(url.searchParams.get('path'), '/xhttp');
  assert.equal(url.searchParams.get('host'), 'edge.example.com');
  assert.equal(url.searchParams.get('mode'), 'stream-one');
  assert.equal(url.searchParams.get('sni'), 'reality.example.com');
  assert.equal(url.searchParams.get('pbk'), 'pubkey');
  assert.equal(url.searchParams.get('sid'), 'abcd1234');
  assert.equal(url.searchParams.get('spx'), '/spider');
  assert.equal(url.searchParams.get('flow'), 'xtls-rprx-vision');
});

check('trojan share link omits flow while keeping reality xhttp params', () => {
  const inbound = new Inbound(8443, '', Protocols.TROJAN, null, new StreamSettings('xhttp', 'reality'));
  inbound.settings.set('clients.0.password', 'secret-pass');
  inbound.settings.set('clients.0.flow', 'xtls-rprx-vision');
  inbound.stream.reality.publicKey = 'pubkey';
  inbound.stream.reality.shortIds = ['ef567890'];
  inbound.stream.reality.serverNames = ['trojan.example.com'];
  inbound.stream.reality.spiderX = '/jump';
  inbound.stream.xhttp.path = '/gateway';
  inbound.stream.xhttp.host = 'trojan-edge.example.com';
  inbound.stream.xhttp.mode = 'stream-one';

  const link = inbound.genLink('45.77.246.87', 'trojan node');
  const url = new URL(link);
  assert.equal(url.protocol, 'trojan:');
  assert.equal(url.username, 'secret-pass');
  assert.equal(url.hostname, '45.77.246.87');
  assert.equal(url.port, '8443');
  assert.equal(url.searchParams.get('type'), 'xhttp');
  assert.equal(url.searchParams.get('security'), 'reality');
  assert.equal(url.searchParams.get('path'), '/gateway');
  assert.equal(url.searchParams.get('host'), 'trojan-edge.example.com');
  assert.equal(url.searchParams.get('mode'), 'stream-one');
  assert.equal(url.searchParams.get('sni'), 'trojan.example.com');
  assert.equal(url.searchParams.get('pbk'), 'pubkey');
  assert.equal(url.searchParams.get('sid'), 'ef567890');
  assert.equal(url.searchParams.get('spx'), '/jump');
  assert.equal(url.searchParams.has('flow'), false);
});

check('shadowsocks share link format', () => {
  const inbound = new Inbound(8388, '', Protocols.SHADOWSOCKS, null, new StreamSettings('tcp', 'tls'));
  inbound.settings.set('method', 'aes-256-gcm');
  inbound.settings.set('password', 'ss-secret');
  inbound.stream.tls.server = 'ss.example.com';

  const payload = parseSS(inbound.genLink('45.77.246.87', 'ss node'));
  assert.equal(payload.method, 'aes-256-gcm');
  assert.equal(payload.password, 'ss-secret');
  assert.equal(payload.host, 'ss.example.com');
  assert.equal(payload.port, '8388');
  assert.equal(payload.remark, 'ss node');
});

if (process.exitCode && process.exitCode !== 0) {
  process.exit(process.exitCode);
}

console.log('FORMAT VALIDATED, CLIENT IMPORT NOT VERIFIED');
