import net from 'node:net';
import { EventEmitter } from 'node:events';
import { PacketCodec } from './packet.js';

/**
 * 连接 xbase Gate 的测试客户端（TCP 或 WebSocket 二进制帧）
 */
export class XBaseClient extends EventEmitter {
  /**
   * @param {object} opts
   * @param {string} opts.host
   * @param {number} opts.port
   * @param {'tcp'|'ws'} [opts.protocol='tcp']
   * @param {'BE'|'LE'} [opts.endian='BE']
   */
  constructor(opts) {
    super();
    this.host = opts.host ?? '127.0.0.1';
    this.port = opts.port ?? 3553;
    this.protocol = opts.protocol ?? 'tcp';
    this.codec = new PacketCodec(opts.endian ?? 'BE');
    this.socket = null;
    this.seq = 1;
    this.connected = false;
  }

  async connect() {
    if (this.protocol === 'ws') {
      return this._connectWS();
    }
    return this._connectTCP();
  }

  _connectTCP() {
    return new Promise((resolve, reject) => {
      const socket = net.createConnection({ host: this.host, port: this.port }, () => {
        this.connected = true;
        this.emit('connect');
        resolve();
      });

      socket.setNoDelay(true);
      socket.on('data', (chunk) => this._onData(chunk));
      socket.on('error', (err) => {
        this.emit('error', err);
        reject(err);
      });
      socket.on('close', () => {
        this.connected = false;
        this.emit('close');
      });

      this.socket = socket;
    });
  }

  async _connectWS() {
    const { default: WebSocket } = await import('ws');
    const url = `ws://${this.host}:${this.port}`;

    return new Promise((resolve, reject) => {
      const ws = new WebSocket(url, { binaryType: 'nodebuffer' });

      ws.on('open', () => {
        this.connected = true;
        this.emit('connect');
        resolve();
      });

      ws.on('message', (data, isBinary) => {
        if (!isBinary) {
          return;
        }
        this._onData(Buffer.from(data));
      });

      ws.on('error', (err) => {
        this.emit('error', err);
        reject(err);
      });

      ws.on('close', () => {
        this.connected = false;
        this.emit('close');
      });

      this.socket = ws;
    });
  }

  _onData(chunk) {
    const messages = this.codec.feed(chunk);
    for (const msg of messages) {
      this.emit('message', msg);
    }
  }

  /**
   * @param {{ gameID?: number, messageID: number, buffer?: Buffer|string, seq?: number }} opts
   */
  send(opts) {
    if (!this.connected) {
      throw new Error('not connected');
    }

    const seq = opts.seq ?? this.seq++;
    const frame = this.codec.pack({
      seq,
      gameID: opts.gameID ?? 0,
      messageID: opts.messageID,
      buffer: opts.buffer ?? '',
    });

    if (this.protocol === 'ws') {
      this.socket.send(frame, { binary: true });
    } else {
      this.socket.write(frame);
    }

    return seq;
  }

  close() {
    if (!this.socket) {
      return;
    }

    if (this.protocol === 'ws') {
      this.socket.close();
    } else {
      this.socket.end();
      this.socket.destroy();
    }

    this.socket = null;
    this.connected = false;
  }
}
