#!/usr/bin/env node

import { XBaseClient } from './client.js';

function usage() {
  console.log(`
xbase Node.js 测试客户端 — 连接 Gate 并发送/接收 packet 消息

用法:
  node src/index.js [options]

连接:
  --host <addr>       Gate 地址 (默认 127.0.0.1)
  --port <num>        Gate 端口 (默认 3553)
  --ws                使用 WebSocket（默认 TCP）
  --endian <BE|LE>    字节序 (默认 BE，与 etc.packet.byteOrder=big 一致)

消息:
  --game <id>         GameID，Gate 选 Node 用 (默认 0)
  --message <id>      MessageID，Node 内路由 (必填，除非 --interactive)
  --body <text>       消息 body，UTF-8 字符串
  --json <text>       以 JSON 字符串作为 body（与 --body 二选一）
  --seq <num>         指定序列号（默认自增）

模式:
  --count <n>         发送 n 条后退出 (默认 1)
  --interval <ms>     多条发送间隔 (默认 0)
  --interactive       交互模式：输入 messageId body
  --listen <sec>      发送后继续监听下行 sec 秒 (默认 5，0=立即退出)

示例:
  node src/index.js --host 127.0.0.1 --port 3553 --game 1 --message 1001 --body hello
  node src/index.js --ws --port 3653 --game 1 --message 1001 --json '{"action":"ping"}'
  node src/index.js --interactive --game 1 --listen 0
`);
}

function parseArgs(argv) {
  const opts = {
    host: '127.0.0.1',
    port: 3553,
    ws: false,
    endian: 'BE',
    gameID: 0,
    messageID: null,
    body: '',
    json: null,
    seq: null,
    count: 1,
    interval: 0,
    interactive: false,
    listen: 5,
  };

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    const next = argv[i + 1];

    switch (arg) {
      case '-h':
      case '--help':
        opts.help = true;
        break;
      case '--host':
        opts.host = next;
        i++;
        break;
      case '--port':
        opts.port = Number(next);
        i++;
        break;
      case '--ws':
        opts.ws = true;
        break;
      case '--endian':
        opts.endian = next === 'LE' ? 'LE' : 'BE';
        i++;
        break;
      case '--game':
        opts.gameID = Number(next);
        i++;
        break;
      case '--message':
        opts.messageID = Number(next);
        i++;
        break;
      case '--body':
        opts.body = next ?? '';
        i++;
        break;
      case '--json':
        opts.json = next ?? '{}';
        i++;
        break;
      case '--seq':
        opts.seq = Number(next);
        i++;
        break;
      case '--count':
        opts.count = Number(next);
        i++;
        break;
      case '--interval':
        opts.interval = Number(next);
        i++;
        break;
      case '--interactive':
        opts.interactive = true;
        break;
      case '--listen':
        opts.listen = Number(next);
        i++;
        break;
      default:
        break;
    }
  }

  return opts;
}

function formatMessage(msg) {
  const bodyText = msg.buffer.length
    ? msg.buffer.toString('utf8')
    : '(empty)';
  return `[recv] seq=${msg.seq} game=${msg.gameID} message=${msg.messageID} body=${bodyText}`;
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function runInteractive(client, opts) {
  const readline = await import('node:readline');
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });

  console.log('交互模式：输入 "<messageID> [body]"，空行退出');
  console.log(`当前 GameID=${opts.gameID}`);

  const prompt = () => {
    rl.question('> ', (line) => {
      const trimmed = line.trim();
      if (!trimmed) {
        rl.close();
        client.close();
        process.exit(0);
        return;
      }

      const [idStr, ...rest] = trimmed.split(/\s+/);
      const messageID = Number(idStr);
      if (Number.isNaN(messageID)) {
        console.log('无效 messageID');
        prompt();
        return;
      }

      const body = rest.join(' ');
      const seq = client.send({ gameID: opts.gameID, messageID, buffer: body });
      console.log(`[send] seq=${seq} game=${opts.gameID} message=${messageID} body=${body}`);
      prompt();
    });
  };

  prompt();
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));

  if (opts.help) {
    usage();
    process.exit(0);
  }

  if (!opts.interactive && opts.messageID == null) {
    usage();
    process.exit(1);
  }

  const client = new XBaseClient({
    host: opts.host,
    port: opts.port,
    protocol: opts.ws ? 'ws' : 'tcp',
    endian: opts.endian,
  });

  client.on('message', (msg) => {
    console.log(formatMessage(msg));
  });

  client.on('error', (err) => {
    console.error('[error]', err.message);
  });

  client.on('close', () => {
    console.log('[close] disconnected');
  });

  try {
    await client.connect();
    console.log(`[connect] ${opts.ws ? 'ws' : 'tcp'}://${opts.host}:${opts.port}`);
  } catch (err) {
    console.error('[connect failed]', err.message);
    process.exit(1);
  }

  if (opts.interactive) {
    await runInteractive(client, opts);
    return;
  }

  const payload = opts.json != null ? opts.json : opts.body;

  for (let i = 0; i < opts.count; i++) {
    const seq = client.send({
      gameID: opts.gameID,
      messageID: opts.messageID,
      buffer: payload,
      seq: opts.seq != null ? opts.seq + i : undefined,
    });

    console.log(
      `[send] seq=${seq} game=${opts.gameID} message=${opts.messageID} body=${payload}`,
    );

    if (opts.interval > 0 && i < opts.count - 1) {
      await sleep(opts.interval);
    }
  }

  if (opts.listen > 0) {
    await sleep(opts.listen * 1000);
  }

  client.close();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
