/**
 * xbase packet 编解码，对齐 Go 端 packet/defaultPacker（默认 Big Endian）。
 *
 * 帧结构:
 *   [4B totalLen][4B dataBit=0][4B gameID][4B messageID][4B seq][body...]
 * totalLen = 20 + body.length（不含最外层 4 字节 length 前缀）
 */

const HEADER_SIZE = 20;
const DATA_BIT = 0;

export class PacketCodec {
  /**
   * @param {'BE'|'LE'} endian 默认 BE，与 etc.packet.byteOrder=big 一致
   */
  constructor(endian = 'BE') {
    this.endian = endian;
    this._buf = Buffer.alloc(0);
  }

  _u32(buf, offset, value) {
    if (this.endian === 'LE') {
      buf.writeUInt32LE(value, offset);
    } else {
      buf.writeUInt32BE(value, offset);
    }
  }

  _i32(buf, offset, value) {
    if (this.endian === 'LE') {
      buf.writeInt32LE(value, offset);
    } else {
      buf.writeInt32BE(value, offset);
    }
  }

  _readU32(buf, offset) {
    return this.endian === 'LE'
      ? buf.readUInt32LE(offset)
      : buf.readUInt32BE(offset);
  }

  _readI32(buf, offset) {
    return this.endian === 'LE'
      ? buf.readInt32LE(offset)
      : buf.readInt32BE(offset);
  }

  /**
   * @param {{ seq?: number, gameID?: number, messageID?: number, buffer?: Buffer|string }} msg
   * @returns {Buffer}
   */
  pack({ seq = 0, gameID = 0, messageID = 0, buffer = Buffer.alloc(0) }) {
    const body = Buffer.isBuffer(buffer) ? buffer : Buffer.from(String(buffer), 'utf8');
    const totalLen = HEADER_SIZE + body.length;
    const out = Buffer.alloc(4 + totalLen);

    this._u32(out, 0, totalLen);
    this._u32(out, 4, DATA_BIT);
    this._i32(out, 8, gameID);
    this._i32(out, 12, messageID);
    this._i32(out, 16, seq);
    body.copy(out, 20);

    return out;
  }

  /**
   * @param {Buffer} data 完整帧（含 4 字节 length 前缀）
   */
  unpack(data) {
    if (data.length < 4) {
      return null;
    }

    const size = this._readU32(data, 0);
    if (size <= HEADER_SIZE || data.length < 4 + size) {
      return null;
    }

    const dataBit = this._readU32(data, 4);
    if ((dataBit & DATA_BIT) !== DATA_BIT) {
      throw new Error('invalid dataBit');
    }

    return {
      seq: this._readI32(data, 16),
      gameID: this._readI32(data, 8),
      messageID: this._readI32(data, 12),
      buffer: data.subarray(20, size),
      rawSize: size,
    };
  }

  /**
   * 粘包/半包解析，feed 网络 chunk 返回完整 message 列表
   * @param {Buffer} chunk
   * @returns {Array<{seq, gameID, messageID, buffer}>}
   */
  feed(chunk) {
    this._buf = Buffer.concat([this._buf, chunk]);
    const messages = [];

    while (this._buf.length >= 4) {
      const size = this._readU32(this._buf, 0);
      if (size === 0) {
        this._buf = this._buf.subarray(4);
        continue;
      }

      if (this._buf.length < 4 + size) {
        break;
      }

      const frame = this._buf.subarray(0, 4 + size);
      this._buf = this._buf.subarray(4 + size);

      const msg = this.unpack(frame);
      if (msg) {
        messages.push(msg);
      }
    }

    return messages;
  }

  reset() {
    this._buf = Buffer.alloc(0);
  }
}
