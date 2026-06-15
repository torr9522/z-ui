const Protocols = {
    VMESS: 'vmess',
    VLESS: 'vless',
    TROJAN: 'trojan',
    SHADOWSOCKS: 'shadowsocks',
    DOKODEMO: 'dokodemo-door',
    SOCKS: 'socks',
    HTTP: 'http',
    MIXED: 'mixed',
    TUNNEL: 'tunnel',
    MTPROTO: 'mtproto',
};

const LegacyProtocols = [
    Protocols.MTPROTO,
];

const LegacyNetworks = ['kcp', 'quic', 'http', 'splithttp'];

const VmessMethods = {
    AES_128_GCM: 'aes-128-gcm',
    CHACHA20_POLY1305: 'chacha20-poly1305',
    AUTO: 'auto',
    NONE: 'none',
};

const SSMethods = {
    // AES_256_CFB: 'aes-256-cfb',
    // AES_128_CFB: 'aes-128-cfb',
    // CHACHA20: 'chacha20',
    // CHACHA20_IETF: 'chacha20-ietf',
    CHACHA20_POLY1305: 'chacha20-poly1305',
    AES_256_GCM: 'aes-256-gcm',
    AES_128_GCM: 'aes-128-gcm',
};

const RULE_IP = {
    PRIVATE: 'geoip:private',
    CN: 'geoip:cn',
};

const RULE_DOMAIN = {
    ADS: 'geosite:category-ads',
    ADS_ALL: 'geosite:category-ads-all',
    CN: 'geosite:cn',
    GOOGLE: 'geosite:google',
    FACEBOOK: 'geosite:facebook',
    SPEEDTEST: 'geosite:speedtest',
};

Object.freeze(Protocols);
Object.freeze(LegacyProtocols);
Object.freeze(LegacyNetworks);
Object.freeze(VmessMethods);
Object.freeze(SSMethods);
Object.freeze(RULE_IP);
Object.freeze(RULE_DOMAIN);

function getPathValue(obj, path, fallback=null) {
    if (obj == null || ObjectUtil.isEmpty(path)) {
        return fallback;
    }
    let current = obj;
    for (const segment of path.split('.')) {
        if (current == null) {
            return fallback;
        }
        current = current[segment];
    }
    return current == null ? fallback : current;
}

function setPathValue(obj, path, value) {
    const segments = path.split('.');
    let current = obj;
    for (let i = 0; i < segments.length - 1; ++i) {
        const segment = segments[i];
        const next = segments[i + 1];
        if (current[segment] == null) {
            current[segment] = /^\d+$/.test(next) ? [] : {};
        }
        current = current[segment];
    }
    current[segments[segments.length - 1]] = value;
}

function firstNonEmpty(value) {
    if (Array.isArray(value)) {
        for (const item of value) {
            if (!ObjectUtil.isEmpty(item)) {
                return item;
            }
        }
        return '';
    }
    return value || '';
}

function defaultProtocolSettings(protocol) {
    switch (protocol) {
        case Protocols.VMESS:
            return {
                clients: [{ id: RandomUtil.randomUUID() }],
                disableInsecureEncryption: false,
            };
        case Protocols.VLESS:
            return {
                clients: [{ id: RandomUtil.randomUUID(), flow: '' }],
                decryption: 'none',
            };
        case Protocols.TROJAN:
            return {
                clients: [{ password: RandomUtil.randomSeq(10) }],
            };
        case Protocols.SHADOWSOCKS:
            return {
                method: SSMethods.AES_256_GCM,
                password: RandomUtil.randomSeq(10),
                network: 'tcp,udp',
            };
        case Protocols.MIXED:
            return {};
        case Protocols.SOCKS:
            return {
                auth: 'noauth',
                udp: true,
                ip: '127.0.0.1',
            };
        case Protocols.HTTP:
            return {
                auth: false,
            };
        default:
            return {};
    }
}

class XrayCommonClass {

    static toJsonArray(arr) {
        return arr.map(obj => obj.toJson());
    }

    static fromJson() {
        return new XrayCommonClass();
    }

    toJson() {
        return this;
    }

    toString(format=true) {
        return format ? JSON.stringify(this.toJson(), null, 2) : JSON.stringify(this.toJson());
    }

    static toHeaders(v2Headers) {
        let newHeaders = [];
        if (v2Headers) {
            Object.keys(v2Headers).forEach(key => {
                let values = v2Headers[key];
                if (typeof(values) === 'string') {
                    newHeaders.push({ name: key, value: values });
                } else {
                    for (let i = 0; i < values.length; ++i) {
                        newHeaders.push({ name: key, value: values[i] });
                    }
                }
            });
        }
        return newHeaders;
    }

    static toV2Headers(headers, arr=true) {
        let v2Headers = {};
        for (let i = 0; i < headers.length; ++i) {
            let name = headers[i].name;
            let value = headers[i].value;
            if (ObjectUtil.isEmpty(name) || ObjectUtil.isEmpty(value)) {
                continue;
            }
            if (!(name in v2Headers)) {
                v2Headers[name] = arr ? [value] : value;
            } else {
                if (arr) {
                    v2Headers[name].push(value);
                } else {
                    v2Headers[name] = value;
                }
            }
        }
        return v2Headers;
    }
}

class TcpStreamSettings extends XrayCommonClass {
    constructor(type='none',
                request=new TcpStreamSettings.TcpRequest(),
                response=new TcpStreamSettings.TcpResponse(),
                ) {
        super();
        this.type = type;
        this.request = request;
        this.response = response;
    }

    static fromJson(json={}) {
        let header = json.header;
        if (!header) {
            header = {};
        }
        return new TcpStreamSettings(
            header.type,
            TcpStreamSettings.TcpRequest.fromJson(header.request),
            TcpStreamSettings.TcpResponse.fromJson(header.response),
        );
    }

    toJson() {
        return {
            header: {
                type: this.type,
                request: this.type === 'http' ? this.request.toJson() : undefined,
                response: this.type === 'http' ? this.response.toJson() : undefined,
            },
        };
    }
}

TcpStreamSettings.TcpRequest = class extends XrayCommonClass {
    constructor(version='1.1',
                method='GET',
                path=['/'],
                headers=[],
    ) {
        super();
        this.version = version;
        this.method = method;
        this.path = path.length === 0 ? ['/'] : path;
        this.headers = headers;
    }

    addPath(path) {
        this.path.push(path);
    }

    removePath(index) {
        this.path.splice(index, 1);
    }

    addHeader(name, value) {
        this.headers.push({ name: name, value: value });
    }

    getHeader(name) {
        for (const header of this.headers) {
            if (header.name.toLowerCase() === name.toLowerCase()) {
                return header.value;
            }
        }
        return null;
    }

    removeHeader(index) {
        this.headers.splice(index, 1);
    }

    static fromJson(json={}) {
        return new TcpStreamSettings.TcpRequest(
            json.version,
            json.method,
            json.path,
            XrayCommonClass.toHeaders(json.headers),
        );
    }

    toJson() {
        return {
            method: this.method,
            path: ObjectUtil.clone(this.path),
            headers: XrayCommonClass.toV2Headers(this.headers),
        };
    }
};

TcpStreamSettings.TcpResponse = class extends XrayCommonClass {
    constructor(version='1.1',
                status='200',
                reason='OK',
                headers=[],
    ) {
        super();
        this.version = version;
        this.status = status;
        this.reason = reason;
        this.headers = headers;
    }

    addHeader(name, value) {
        this.headers.push({ name: name, value: value });
    }

    removeHeader(index) {
        this.headers.splice(index, 1);
    }

    static fromJson(json={}) {
        return new TcpStreamSettings.TcpResponse(
            json.version,
            json.status,
            json.reason,
            XrayCommonClass.toHeaders(json.headers),
        );
    }

    toJson() {
        return {
            version: this.version,
            status: this.status,
            reason: this.reason,
            headers: XrayCommonClass.toV2Headers(this.headers),
        };
    }
};

class KcpStreamSettings extends XrayCommonClass {
    constructor(mtu=1350, tti=20,
                uplinkCapacity=5,
                downlinkCapacity=20,
                congestion=false,
                readBufferSize=2,
                writeBufferSize=2,
                type='none',
                seed=RandomUtil.randomSeq(10),
                ) {
        super();
        this.mtu = mtu;
        this.tti = tti;
        this.upCap = uplinkCapacity;
        this.downCap = downlinkCapacity;
        this.congestion = congestion;
        this.readBuffer = readBufferSize;
        this.writeBuffer = writeBufferSize;
        this.type = type;
        this.seed = seed;
    }

    static fromJson(json={}) {
        return new KcpStreamSettings(
            json.mtu,
            json.tti,
            json.uplinkCapacity,
            json.downlinkCapacity,
            json.congestion,
            json.readBufferSize,
            json.writeBufferSize,
            ObjectUtil.isEmpty(json.header) ? 'none' : json.header.type,
            json.seed,
        );
    }

    toJson() {
        return {
            mtu: this.mtu,
            tti: this.tti,
            uplinkCapacity: this.upCap,
            downlinkCapacity: this.downCap,
            congestion: this.congestion,
            readBufferSize: this.readBuffer,
            writeBufferSize: this.writeBuffer,
            header: {
                type: this.type,
            },
            seed: this.seed,
        };
    }
}

class WsStreamSettings extends XrayCommonClass {
    constructor(path='/', headers=[]) {
        super();
        this.path = path;
        this.headers = headers;
    }

    addHeader(name, value) {
        this.headers.push({ name: name, value: value });
    }

    getHeader(name) {
        for (const header of this.headers) {
            if (header.name.toLowerCase() === name.toLowerCase()) {
                return header.value;
            }
        }
        return null;
    }

    removeHeader(index) {
        this.headers.splice(index, 1);
    }

    static fromJson(json={}) {
        return new WsStreamSettings(
            json.path,
            XrayCommonClass.toHeaders(json.headers),
        );
    }

    toJson() {
        return {
            path: this.path,
            headers: XrayCommonClass.toV2Headers(this.headers, false),
        };
    }
}

class HttpStreamSettings extends XrayCommonClass {
    constructor(path='/', host=['']) {
        super();
        this.path = path;
        this.host = host.length === 0 ? [''] : host;
    }

    addHost(host) {
        this.host.push(host);
    }

    removeHost(index) {
        this.host.splice(index, 1);
    }

    static fromJson(json={}) {
        return new HttpStreamSettings(json.path, json.host);
    }

    toJson() {
        let host = [];
        for (let i = 0; i < this.host.length; ++i) {
            if (!ObjectUtil.isEmpty(this.host[i])) {
                host.push(this.host[i]);
            }
        }
        return {
            path: this.path,
            host: host,
        }
    }
}

class XHTTPStreamSettings extends XrayCommonClass {
    constructor(path='/', host='', mode='stream-one', extra='') {
        super();
        this.path = path;
        this.host = host;
        this.mode = mode;
        this.extra = extra;
    }

    static fromJson(json={}) {
        let host = json.host;
        if (Array.isArray(host)) {
            host = host.length > 0 ? host[0] : '';
        }
        let extra = json.extra || '';
        if (!ObjectUtil.isEmpty(extra) && typeof extra !== 'string') {
            extra = JSON.stringify(extra, null, 2);
        }
        return new XHTTPStreamSettings(
            json.path,
            host,
            json.mode || 'stream-one',
            extra,
        );
    }

    toJson() {
        const settings = {
            path: this.path,
            host: this.host,
            mode: this.mode,
        };
        if (!ObjectUtil.isEmpty(this.extra)) {
            try {
                const extra = JSON.parse(this.extra);
                if (extra && typeof extra === 'object' && !Array.isArray(extra)) {
                    Object.assign(settings, extra);
                }
            } catch (_) {
                settings.extra = this.extra;
            }
        }
        return settings;
    }
}

class QuicStreamSettings extends XrayCommonClass {
    constructor(security=VmessMethods.NONE,
                key='', type='none') {
        super();
        this.security = security;
        this.key = key;
        this.type = type;
    }

    static fromJson(json={}) {
        return new QuicStreamSettings(
            json.security,
            json.key,
            json.header ? json.header.type : 'none',
        );
    }

    toJson() {
        return {
            security: this.security,
            key: this.key,
            header: {
                type: this.type,
            }
        }
    }
}

class GrpcStreamSettings extends XrayCommonClass {
    constructor(serviceName="") {
        super();
        this.serviceName = serviceName;
    }

    static fromJson(json={}) {
        return new GrpcStreamSettings(json.serviceName);
    }

    toJson() {
        return {
            serviceName: this.serviceName,
        }
    }
}

class TlsStreamSettings extends XrayCommonClass {
    constructor(serverName='',
                certificates=[new TlsStreamSettings.Cert()]) {
        super();
        this.server = serverName;
        this.certs = certificates;
    }

    addCert(cert) {
        this.certs.push(cert);
    }

    removeCert(index) {
        this.certs.splice(index, 1);
    }

    static fromJson(json={}) {
        let certs;
        if (!ObjectUtil.isEmpty(json.certificates)) {
            certs = json.certificates.map(cert => TlsStreamSettings.Cert.fromJson(cert));
        }
        return new TlsStreamSettings(
            json.serverName,
            certs,
        );
    }

    toJson() {
        return {
            serverName: this.server,
            certificates: TlsStreamSettings.toJsonArray(this.certs),
        };
    }
}

TlsStreamSettings.Cert = class extends XrayCommonClass {
    constructor(useFile=true, certificateFile='', keyFile='', certificate='', key='') {
        super();
        this.useFile = useFile;
        this.certFile = certificateFile;
        this.keyFile = keyFile;
        this.cert = certificate instanceof Array ? certificate.join('\n') : certificate;
        this.key = key instanceof Array ? key.join('\n') : key;
    }

    static fromJson(json={}) {
        if ('certificateFile' in json && 'keyFile' in json) {
            return new TlsStreamSettings.Cert(
                true,
                json.certificateFile,
                json.keyFile,
            );
        } else {
            return new TlsStreamSettings.Cert(
                false, '', '',
                json.certificate.join('\n'),
                json.key.join('\n'),
            );
        }
    }

    toJson() {
        if (this.useFile) {
            return {
                certificateFile: this.certFile,
                keyFile: this.keyFile,
            };
        } else {
            return {
                certificate: this.cert.split('\n'),
                key: this.key.split('\n'),
            };
        }
    }
};

class RealityStreamSettings extends XrayCommonClass {
    constructor(privateKey='',
                publicKey='',
                shortIds=[''],
                serverNames=[''],
                spiderX='/',
                dest='',
                fingerprint='chrome') {
        super();
        this.privateKey = privateKey;
        this.publicKey = publicKey;
        this.shortIds = shortIds && shortIds.length > 0 ? shortIds : [''];
        this.serverNames = serverNames && serverNames.length > 0 ? serverNames : [''];
        this.spiderX = spiderX;
        this.dest = dest;
        this.fingerprint = fingerprint || 'chrome';
    }

    static fromJson(json={}) {
        return new RealityStreamSettings(
            json.privateKey,
            json.publicKey,
            json.shortIds,
            json.serverNames,
            json.spiderX,
            json.dest,
            json.fingerprint,
        );
    }

    toJson() {
        return {
            show: false,
            privateKey: this.privateKey,
            publicKey: this.publicKey,
            shortIds: this.shortIds.filter(shortId => !ObjectUtil.isEmpty(shortId)),
            serverNames: this.serverNames.filter(serverName => !ObjectUtil.isEmpty(serverName)),
            spiderX: this.spiderX,
            dest: this.dest,
            fingerprint: this.fingerprint,
        };
    }
}

class StreamSettings extends XrayCommonClass {
    constructor(network='tcp',
                security='none',
                tlsSettings=new TlsStreamSettings(),
                realitySettings=new RealityStreamSettings(),
                tcpSettings=new TcpStreamSettings(),
                kcpSettings=new KcpStreamSettings(),
                wsSettings=new WsStreamSettings(),
                httpSettings=new HttpStreamSettings(),
                xhttpSettings=new XHTTPStreamSettings(),
                quicSettings=new QuicStreamSettings(),
                grpcSettings=new GrpcStreamSettings(),
                ) {
        super();
        this.network = network;
        this.security = security;
        this.tls = tlsSettings;
        this.reality = realitySettings;
        this.tcp = tcpSettings;
        this.kcp = kcpSettings;
        this.ws = wsSettings;
        this.http = httpSettings;
        this.xhttp = xhttpSettings;
        this.quic = quicSettings;
        this.grpc = grpcSettings;
    }

    get isTls() {
        return this.security === 'tls';
    }

    set isTls(isTls) {
        if (isTls) {
            this.security = 'tls';
        } else {
            this.security = 'none';
        }
    }

    get isXTls() {
        return false;
    }

    get isReality() {
        return this.security === "reality";
    }

    set isReality(isReality) {
        if (isReality) {
            this.security = 'reality';
        } else {
            this.security = 'none';
        }
    }

    set isXTls(isXTls) {
        if (isXTls) {
            this.security = 'tls';
        } else {
            this.security = 'none';
        }
    }

    static fromJson(json={}) {
        json = ObjectUtil.clone(json || {});
        if (json.network === 'http' || json.network === 'splithttp') {
            json.network = 'xhttp';
            if (!json.xhttpSettings) {
                json.xhttpSettings = json.splithttpSettings || json.httpSettings;
            }
        }
        let tls;
        if (json.security === "xtls") {
            tls = TlsStreamSettings.fromJson(json.xtlsSettings);
            json.security = 'tls';
        } else {
            tls = TlsStreamSettings.fromJson(json.tlsSettings);
        }
        return new StreamSettings(
            json.network,
            json.security,
            tls,
            RealityStreamSettings.fromJson(json.realitySettings),
            TcpStreamSettings.fromJson(json.tcpSettings),
            KcpStreamSettings.fromJson(json.kcpSettings),
            WsStreamSettings.fromJson(json.wsSettings),
            HttpStreamSettings.fromJson(json.httpSettings),
            XHTTPStreamSettings.fromJson(json.xhttpSettings),
            QuicStreamSettings.fromJson(json.quicSettings),
            GrpcStreamSettings.fromJson(json.grpcSettings),
        );
    }

    toJson() {
        const network = this.network;
        return {
            network: network,
            security: this.security,
            tlsSettings: this.isTls ? this.tls.toJson() : undefined,
            realitySettings: this.isReality ? this.reality.toJson() : undefined,
            tcpSettings: network === 'tcp' ? this.tcp.toJson() : undefined,
            kcpSettings: network === 'kcp' ? this.kcp.toJson() : undefined,
            wsSettings: network === 'ws' ? this.ws.toJson() : undefined,
            xhttpSettings: network === 'xhttp' ? this.xhttp.toJson() : undefined,
            quicSettings: network === 'quic' ? this.quic.toJson() : undefined,
            grpcSettings: network === 'grpc' ? this.grpc.toJson() : undefined,
        };
    }
}

class Sniffing extends XrayCommonClass {
    constructor(enabled=true, destOverride=['http', 'tls']) {
        super();
        this.enabled = enabled;
        this.destOverride = destOverride;
    }

    static fromJson(json={}) {
        let destOverride = ObjectUtil.clone(json.destOverride);
        if (!ObjectUtil.isEmpty(destOverride) && !ObjectUtil.isArrEmpty(destOverride)) {
            if (ObjectUtil.isEmpty(destOverride[0])) {
                destOverride = ['http', 'tls'];
            }
        }
        return new Sniffing(
            !!json.enabled,
            destOverride,
        );
    }
}

class Inbound extends XrayCommonClass {
    constructor(port=RandomUtil.randomIntRange(10000, 60000),
                listen='',
                protocol=Protocols.VMESS,
                settings=null,
                streamSettings=new StreamSettings(),
                tag='',
                sniffing=new Sniffing(),
                ) {
        super();
        this.port = port;
        this.listen = listen;
        this._protocol = protocol;
        this.settings = ObjectUtil.isEmpty(settings) ? Inbound.Settings.getSettings(protocol) : settings;
        this.stream = streamSettings;
        this.tag = tag;
        this.sniffing = sniffing;
    }

    get protocol() {
        return this._protocol;
    }

    set protocol(protocol) {
        this._protocol = protocol;
        this.settings = Inbound.Settings.getSettings(protocol);
    }

    get tls() {
        return this.stream.security === 'tls';
    }

    set tls(isTls) {
        if (isTls) {
            this.stream.security = 'tls';
        } else {
            this.stream.security = 'none';
        }
    }

    get xtls() {
        return false;
    }

    set xtls(isXTls) {
        if (isXTls) {
            this.stream.security = 'tls';
        } else {
            this.stream.security = 'none';
        }
    }

    get network() {
        return this.stream.network;
    }

    set network(network) {
        this.stream.network = network;
    }

    get isTcp() {
        return this.network === "tcp";
    }

    get isWs() {
        return this.network === "ws";
    }

    get isKcp() {
        return this.network === "kcp";
    }

    get isQuic() {
        return this.network === "quic"
    }

    get isGrpc() {
        return this.network === "grpc";
    }

    get isH2() {
        return this.network === "http";
    }

    get isXHTTP() {
        return this.network === "xhttp";
    }

    // VMess & VLess
    get uuid() {
        switch (this.protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
                return this.settings.get('clients.0.id', '');
            default:
                return "";
        }
    }

    // VLess & Trojan
    get flow() {
        switch (this.protocol) {
            case Protocols.VLESS:
                return this.settings.get('clients.0.flow', '');
            default:
                return "";
        }
    }

    legacyFlowWarning() {
        if (this.protocol !== Protocols.VLESS) {
            return '';
        }
        const flow = this.flow;
        if (flow === 'xtls-rprx-origin' || flow === 'xtls-rprx-direct') {
            return 'Legacy Flow Migrated Warning: 保存时将迁移为 xtls-rprx-vision。';
        }
        if (flow === 'xtls-rprx-splice') {
            return 'Legacy Flow Unsupported Warning: xtls-rprx-splice 不再支持，保存时将清空 flow。';
        }
        return '';
    }

    legacyVMessWarning() {
        if (this.protocol !== Protocols.VMESS) {
            return '';
        }
        const alterId = Number(this.alterId || 0);
        if (alterId > 0) {
            return 'Legacy VMess Warning: 保存时 alterId 将归一化为 0。';
        }
        return '';
    }

    // VMess
    get alterId() {
        switch (this.protocol) {
            case Protocols.VMESS:
                return this.settings.get('clients.0.alterId', 0);
            default:
                return "";
        }
    }

    // Socks & HTTP
    get username() {
        const account = this.settings.accounts?.[0];
        switch (this.protocol) {
            case Protocols.SOCKS:
            case Protocols.HTTP:
                return account ? account.user : "";
            default:
                return "";
        }
    }

    // Trojan & Shadowsocks & Socks & HTTP
    get password() {
        switch (this.protocol) {
            case Protocols.TROJAN:
                return this.settings.get('clients.0.password', '');
            case Protocols.SHADOWSOCKS:
                return this.settings.get('password', '');
            case Protocols.SOCKS:
            case Protocols.HTTP:
                return this.settings.accounts?.[0]?.pass || "";
            default:
                return "";
        }
    }

    // Shadowsocks
    get method() {
        switch (this.protocol) {
            case Protocols.SHADOWSOCKS:
                return this.settings.get('method', '');
            default:
                return "";
        }
    }

    get serverName() {
        if (this.stream.isTls) {
            return this.stream.tls.server;
        } else if (this.stream.isReality) {
            return this.stream.reality.serverNames[0];
        }
        return "";
    }

    get host() {
        if (this.isTcp) {
            return this.stream.tcp.request.getHeader("Host");
        } else if (this.isWs) {
            return this.stream.ws.getHeader("Host");
        } else if (this.isH2) {
            return this.stream.http.host[0];
        } else if (this.isXHTTP) {
            return this.stream.xhttp.host;
        }
        return null;
    }

    get path() {
        if (this.isTcp) {
            return this.stream.tcp.request.path[0];
        } else if (this.isWs) {
            return this.stream.ws.path;
        } else if (this.isH2) {
            return this.stream.http.path[0];
        } else if (this.isXHTTP) {
            return this.stream.xhttp.path;
        }
        return null;
    }

    get securityLabel() {
        return this.stream.security;
    }

    get realityPublicKey() {
        return this.stream.reality.publicKey;
    }

    get realityShortId() {
        return firstNonEmpty(this.stream.reality.shortIds);
    }

    get realityServerName() {
        return firstNonEmpty(this.stream.reality.serverNames);
    }

    get realitySpiderX() {
        return this.stream.reality.spiderX;
    }

    get xhttpMode() {
        return this.stream.xhttp.mode;
    }

    get quicSecurity() {
        return this.stream.quic.security;
    }

    get quicKey() {
        return this.stream.quic.key;
    }

    get quicType() {
        return this.stream.quic.type;
    }

    get kcpType() {
        return this.stream.kcp.type;
    }

    get kcpSeed() {
        return this.stream.kcp.seed;
    }

    get serviceName() {
        return this.stream.grpc.serviceName;
    }

    canEnableTls() {
        switch (this.protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
                break;
            default:
                return false;
        }

        switch (this.network) {
            case "tcp":
            case "ws":
            case "quic":
            case "xhttp":
            case "grpc":
                return true;
            default:
                return false;
        }
    }

    canSetTls() {
        return this.canEnableTls();
    }

    canEnableXTls() {
        return false;
    }

    canEnableReality() {
        switch (this.protocol) {
            case Protocols.VLESS:
            case Protocols.TROJAN:
                break;
            default:
                return false;
        }
        return ["tcp", "http", "xhttp", "grpc"].includes(this.network);
    }

    canEnableVision() {
        return this.protocol === Protocols.VLESS && ["tls", "reality"].includes(this.stream.security);
    }

    canEnableStream() {
        switch (this.protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
                return true;
            default:
                return false;
        }
    }

    canSniffing() {
        return false;
    }

    reset() {
        this.port = RandomUtil.randomIntRange(10000, 60000);
        this.listen = '';
        this.protocol = Protocols.VMESS;
        this.settings = Inbound.Settings.getSettings(Protocols.VMESS);
        this.stream = new StreamSettings();
        this.tag = '';
        this.sniffing = new Sniffing();
    }

    isLegacyProtocol() {
        return LegacyProtocols.includes(this.protocol);
    }

    isLegacyNetwork() {
        return LegacyNetworks.includes(this.network);
    }

    applyTransportParams(params) {
        const type = this.stream.network;
        switch (type) {
            case "tcp":
                const tcp = this.stream.tcp;
                if (tcp.type === 'http') {
                    const request = tcp.request;
                    params.set("path", request.path.join(','));
                    const host = request.getHeader("Host");
                    if (!ObjectUtil.isEmpty(host)) {
                        params.set("host", host);
                    }
                    params.set("headerType", "http");
                }
                break;
            case "kcp":
                const kcp = this.stream.kcp;
                params.set("headerType", kcp.type);
                params.set("seed", kcp.seed);
                break;
            case "ws":
                const ws = this.stream.ws;
                params.set("path", ws.path);
                const wsHost = ws.getHeader("Host");
                if (!ObjectUtil.isEmpty(wsHost)) {
                    params.set("host", wsHost);
                }
                break;
            case "http":
                const http = this.stream.http;
                params.set("path", http.path);
                params.set("host", firstNonEmpty(http.host));
                break;
            case "xhttp":
                const xhttp = this.stream.xhttp;
                params.set("path", xhttp.path);
                params.set("host", xhttp.host);
                params.set("mode", xhttp.mode);
                break;
            case "quic":
                const quic = this.stream.quic;
                params.set("quicSecurity", quic.security);
                params.set("key", quic.key);
                params.set("headerType", quic.type);
                break;
            case "grpc":
                const grpc = this.stream.grpc;
                params.set("serviceName", grpc.serviceName);
                break;
        }
    }

    applySecurityParams(params, address='') {
        let nextAddress = address;
        if (this.stream.security === 'tls') {
            if (!ObjectUtil.isEmpty(this.stream.tls.server)) {
                nextAddress = this.stream.tls.server;
                params.set("sni", nextAddress);
            }
        } else if (this.stream.security === 'reality') {
            const sni = this.realityServerName;
            if (!ObjectUtil.isEmpty(sni)) {
                params.set("sni", sni);
            }
            if (!ObjectUtil.isEmpty(this.realityPublicKey)) {
                params.set("pbk", this.realityPublicKey);
            }
            if (!ObjectUtil.isEmpty(this.realityShortId)) {
                params.set("sid", this.realityShortId);
            }
            if (!ObjectUtil.isEmpty(this.realitySpiderX)) {
                params.set("spx", this.realitySpiderX);
            }
        }
        return nextAddress;
    }

    genVmessLink(address='', remark='') {
        if (this.protocol !== Protocols.VMESS) {
            return '';
        }
        let network = this.stream.network;
        let type = 'none';
        let host = '';
        let path = '';
        if (network === 'tcp') {
            let tcp = this.stream.tcp;
            type = tcp.type;
            if (type === 'http') {
                let request = tcp.request;
                path = request.path.join(',');
                let index = request.headers.findIndex(header => header.name.toLowerCase() === 'host');
                if (index >= 0) {
                    host = request.headers[index].value;
                }
            }
        } else if (network === 'kcp') {
            let kcp = this.stream.kcp;
            type = kcp.type;
            path = kcp.seed;
        } else if (network === 'ws') {
            let ws = this.stream.ws;
            path = ws.path;
            let index = ws.headers.findIndex(header => header.name.toLowerCase() === 'host');
            if (index >= 0) {
                host = ws.headers[index].value;
            }
        } else if (network === 'http') {
            network = 'h2';
            path = this.stream.http.path;
            host = this.stream.http.host.join(',');
        } else if (network === 'xhttp') {
            path = this.stream.xhttp.path;
            host = this.stream.xhttp.host;
        } else if (network === 'quic') {
            type = this.stream.quic.type;
            host = this.stream.quic.security;
            path = this.stream.quic.key;
        } else if (network === 'grpc') {
            path = this.stream.grpc.serviceName;
        }

        if (this.stream.security === 'tls') {
            if (!ObjectUtil.isEmpty(this.stream.tls.server)) {
                address = this.stream.tls.server;
            }
        }

        let obj = {
            v: '2',
            ps: remark,
            add: address,
            port: this.port,
            id: this.settings.get('clients.0.id', ''),
            aid: this.settings.get('clients.0.alterId', 0),
            net: network,
            type: type,
            host: host,
            path: path,
            tls: this.stream.security,
        };
        return 'vmess://' + base64(JSON.stringify(obj, null, 2));
    }

    genVLESSLink(address = '', remark='') {
        const settings = this.settings;
        const uuid = settings.get('clients.0.id', '');
        const port = this.port;
        const params = new Map();
        params.set("type", this.stream.network);
        params.set("security", this.stream.security);
        this.applyTransportParams(params);
        address = this.applySecurityParams(params, address);

        if (this.protocol === Protocols.VLESS && this.canEnableVision() && !ObjectUtil.isEmpty(this.settings.get('clients.0.flow', ''))) {
            params.set("flow", this.settings.get('clients.0.flow', ''));
        }

        const link = `vless://${uuid}@${address}:${port}`;
        const url = new URL(link);
        for (const [key, value] of params) {
            url.searchParams.set(key, value)
        }
        url.hash = encodeURIComponent(remark);
        return url.toString();
    }

    genSSLink(address='', remark='') {
        let settings = this.settings;
        const server = this.stream.tls.server;
        if (!ObjectUtil.isEmpty(server)) {
            address = server;
        }
        return 'ss://' + safeBase64(settings.get('method', '') + ':' + settings.get('password', '') + '@' + address + ':' + this.port)
            + '#' + encodeURIComponent(remark);
    }

    genTrojanLink(address='', remark='') {
        let settings = this.settings;
        const url = new URL(`trojan://${settings.get('clients.0.password', '')}@${address}:${this.port}`);
        url.searchParams.set("type", this.stream.network);
        url.searchParams.set("security", this.stream.security);
        this.applyTransportParams(url.searchParams);
        address = this.applySecurityParams(url.searchParams, address);
        url.username = settings.get('clients.0.password', '');
        url.hostname = address;
        url.hash = encodeURIComponent(remark);
        return url.toString();
    }

    genLink(address='', remark='') {
        switch (this.protocol) {
            case Protocols.VMESS: return this.genVmessLink(address, remark);
            case Protocols.VLESS: return this.genVLESSLink(address, remark);
            case Protocols.SHADOWSOCKS: return this.genSSLink(address, remark);
            case Protocols.TROJAN: return this.genTrojanLink(address, remark);
            default: return '';
        }
    }

    static fromJson(json={}) {
        const protocol = (json.protocol || '').toLowerCase();
        return new Inbound(
            json.port,
            json.listen,
            protocol,
            Inbound.Settings.fromJson(protocol, json.settings),
            StreamSettings.fromJson(json.streamSettings),
            json.tag,
            Sniffing.fromJson(json.sniffing),
        )
    }

    toJson() {
        let streamSettings;
        if (this.canEnableStream()) {
            streamSettings = this.stream.toJson();
        }
        return {
            port: this.port,
            listen: this.listen,
            protocol: this.protocol,
            settings: this.settings instanceof XrayCommonClass ? this.settings.toJson() : this.settings,
            streamSettings: streamSettings,
            tag: this.tag,
            sniffing: {},
        };
    }
}

Inbound.Settings = class extends XrayCommonClass {
    constructor(protocol) {
        super();
        this.protocol = protocol;
    }

    static getSettings(protocol) {
        switch (protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
            case Protocols.MIXED:
                return new Inbound.ProtocolSettings(protocol);
            case Protocols.DOKODEMO: return new Inbound.DokodemoSettings(protocol);
            case Protocols.TUNNEL: return new Inbound.DokodemoSettings(protocol);
            case Protocols.MTPROTO: return new Inbound.MtprotoSettings(protocol);
            case Protocols.SOCKS: return new Inbound.SocksSettings(protocol);
            case Protocols.HTTP: return new Inbound.HttpSettings(protocol);
            default: return null;
        }
    }

    static fromJson(protocol, json) {
        switch (protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
            case Protocols.MIXED:
                return Inbound.ProtocolSettings.fromJson(protocol, json);
            case Protocols.DOKODEMO: return Inbound.DokodemoSettings.fromJson(json);
            case Protocols.TUNNEL: return Inbound.DokodemoSettings.fromJson(json, Protocols.TUNNEL);
            case Protocols.MTPROTO: return Inbound.MtprotoSettings.fromJson(json);
            case Protocols.SOCKS: return Inbound.SocksSettings.fromJson(json);
            case Protocols.HTTP: return Inbound.HttpSettings.fromJson(json);
            default: return null;
        }
    }

    toJson() {
        return {};
    }
};

Inbound.ProtocolSettings = class extends Inbound.Settings {
    constructor(protocol, data=null) {
        super(protocol);
        this.data = ObjectUtil.isEmpty(data) ? defaultProtocolSettings(protocol) : data;
        this.ensureDefaults();
    }

    ensureDefaults() {
        const defaults = defaultProtocolSettings(this.protocol);
        for (const key of Object.keys(defaults)) {
            if (this.data[key] == null) {
                this.data[key] = ObjectUtil.clone(defaults[key]);
            }
        }
    }

    get(path, fallback=null) {
        return getPathValue(this.data, path, fallback);
    }

    set(path, value) {
        setPathValue(this.data, path, value);
    }

    toJson() {
        return this.data;
    }

    static fromJson(protocol, json={}) {
        return new Inbound.ProtocolSettings(protocol, json || {});
    }
};

Inbound.DokodemoSettings = class extends Inbound.Settings {
    constructor(protocol, address='', port=0, network='tcp,udp', followRedirect=false) {
        super(protocol);
        this.address = address;
        this.port = port;
        this.network = network;
        this.followRedirect = followRedirect;
    }

    static fromJson(json={}, protocol=Protocols.DOKODEMO) {
        return new Inbound.DokodemoSettings(
            protocol,
            json.address,
            json.port,
            json.network,
            !!json.followRedirect,
        );
    }

    toJson() {
        return {
            address: this.address,
            port: this.port,
            network: this.network,
            followRedirect: this.followRedirect,
        };
    }
};

Inbound.MtprotoSettings = class extends Inbound.Settings {
    constructor(protocol, users=[new Inbound.MtprotoSettings.MtUser()]) {
        super(protocol);
        this.users = users;
    }

    static fromJson(json={}) {
        return new Inbound.MtprotoSettings(
            Protocols.MTPROTO,
            json.users.map(user => Inbound.MtprotoSettings.MtUser.fromJson(user)),
        );
    }

    toJson() {
        return {
            users: XrayCommonClass.toJsonArray(this.users),
        };
    }
};
Inbound.MtprotoSettings.MtUser = class extends XrayCommonClass {
    constructor(secret=RandomUtil.randomMTSecret()) {
        super();
        this.secret = secret;
    }

    static fromJson(json={}) {
        return new Inbound.MtprotoSettings.MtUser(json.secret);
    }
};

Inbound.SocksSettings = class extends Inbound.Settings {
    constructor(protocol, auth='noauth', accounts=[new Inbound.SocksSettings.SocksAccount()], udp=true, ip='127.0.0.1') {
        super(protocol);
        this.auth = auth;
        this.accounts = Array.isArray(accounts) && accounts.length > 0 ? accounts : [new Inbound.SocksSettings.SocksAccount()];
        this.udp = udp;
        this.ip = ip;
    }

    addAccount(account) {
        this.accounts.push(account);
    }

    delAccount(index) {
        this.accounts.splice(index, 1);
    }

    static fromJson(json={}) {
        let accounts = [new Inbound.SocksSettings.SocksAccount()];
        if (json.auth === 'password') {
            accounts = (json.accounts || []).map(
                account => Inbound.SocksSettings.SocksAccount.fromJson(account)
            );
            if (accounts.length === 0) {
                accounts = [new Inbound.SocksSettings.SocksAccount()];
            }
        }
        return new Inbound.SocksSettings(
            Protocols.SOCKS,
            json.auth || 'noauth',
            accounts,
            json.udp === undefined ? true : !!json.udp,
            json.ip || '127.0.0.1',
        );
    }

    toJson() {
        const payload = {
            auth: this.auth,
            accounts: this.auth === 'password' ? this.accounts.map(account => account.toJson()) : undefined,
            udp: this.udp,
            ip: this.ip,
        };
        if (this.auth !== 'password') {
            delete payload.accounts;
        }
        return payload;
    }
};
Inbound.SocksSettings.SocksAccount = class extends XrayCommonClass {
    constructor(user=RandomUtil.randomSeq(10), pass=RandomUtil.randomSeq(10)) {
        super();
        this.user = user;
        this.pass = pass;
    }

    static fromJson(json={}) {
        return new Inbound.SocksSettings.SocksAccount(json.user, json.pass);
    }
};

Inbound.HttpSettings = class extends Inbound.Settings {
    constructor(protocol, auth=false, accounts=[new Inbound.HttpSettings.HttpAccount()], allowTransparent=false) {
        super(protocol);
        this.auth = auth;
        this.accounts = Array.isArray(accounts) && accounts.length > 0 ? accounts : [new Inbound.HttpSettings.HttpAccount()];
        this.allowTransparent = allowTransparent;
    }

    addAccount(account) {
        this.accounts.push(account);
    }

    delAccount(index) {
        this.accounts.splice(index, 1);
    }

    static fromJson(json={}) {
        const accounts = (json.accounts || []).map(account => Inbound.HttpSettings.HttpAccount.fromJson(account));
        const auth = json.auth !== undefined ? !!json.auth : accounts.length > 0;
        return new Inbound.HttpSettings(
            Protocols.HTTP,
            auth,
            accounts.length > 0 ? accounts : [new Inbound.HttpSettings.HttpAccount()],
            !!json.allowTransparent,
        );
    }

    toJson() {
        const payload = {
            auth: this.auth,
            accounts: this.auth ? Inbound.HttpSettings.toJsonArray(this.accounts) : undefined,
            allowTransparent: this.allowTransparent,
        };
        if (!this.auth) {
            delete payload.accounts;
        }
        return payload;
    }
};

Inbound.HttpSettings.HttpAccount = class extends XrayCommonClass {
    constructor(user=RandomUtil.randomSeq(10), pass=RandomUtil.randomSeq(10)) {
        super();
        this.user = user;
        this.pass = pass;
    }

    static fromJson(json={}) {
        return new Inbound.HttpSettings.HttpAccount(json.user, json.pass);
    }
};
