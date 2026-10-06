// Web版（GitHub Pages）のバックエンド。wails.jsと同じ関数を同じ形で提供する。
// - jq実行・スキーマ推論: Goをwasmにしたもの（wasm_main.go）をWorkerで動かす
// - 保存項目: IndexedDB（webSaved.js）
// - ファイルを開く: <input type="file">
// - クリップボード: navigator.clipboard
import wasmUrl from './generated/shiboq.wasm?url';
import JqWorker from './jq.worker.js?worker';

export { SaveItem, ListSaved, DeleteSaved } from './webSaved.js';

// Web版にだけ出すもの（アプリ版への案内など）の切り替えに使う。
export const IS_WEB = true;

// Go側のJqService.Timeout（5秒）と同じ値。wasmにはプリエンプションが無く、
// `def f: f; f`のようなCPUを占有し続けるクエリではGo側のタイムアウトが
// 発火しない（実測）。そのためメインスレッド側でWorkerごと打ち切る。
const QUERY_TIMEOUT_MS = 5000;
const TIMEOUT_MESSAGE =
  '実行がタイムアウトしました（5秒以内に完了しませんでした）。無限ループや終了しないクエリになっていないか確認してください。';

let modulePromise = null;
function compiledModule() {
  if (!modulePromise) {
    modulePromise = (async () => {
      try {
        return await WebAssembly.compileStreaming(fetch(wasmUrl));
      } catch (e) {
        // MIMEタイプがapplication/wasmでないサーバー向けのフォールバック
        const res = await fetch(wasmUrl);
        return WebAssembly.compile(await res.arrayBuffer());
      }
    })();
    // 失敗したら次回の呼び出しで取得し直せるようにする
    modulePromise.catch(() => {
      modulePromise = null;
    });
  }
  return modulePromise;
}

// Workerを1つ持ち、id付きのメッセージで呼び出しと応答を対応付ける。
class WorkerClient {
  constructor() {
    this.worker = null;
    this.pending = new Map();
    this.nextId = 0;
  }

  async ensure() {
    if (this.worker) return this.worker;
    const module = await compiledModule();
    if (this.worker) return this.worker;
    const w = new JqWorker();
    w.onmessage = (e) => {
      const { id, result, error } = e.data;
      const p = this.pending.get(id);
      if (!p) return;
      this.pending.delete(id);
      clearTimeout(p.timer);
      if (error != null) p.reject(new Error(error));
      else p.resolve(JSON.parse(result));
    };
    w.postMessage({ type: 'init', module });
    this.worker = w;
    return w;
  }

  // Workerを破棄する。応答待ちの呼び出しはsettle(p)で決着させる。
  reset(settle) {
    if (this.worker) this.worker.terminate();
    this.worker = null;
    const list = [...this.pending.values()];
    this.pending.clear();
    for (const p of list) {
      clearTimeout(p.timer);
      settle(p);
    }
  }

  async call(method, args, { timeoutMs, onTimeout } = {}) {
    const w = await this.ensure();
    return new Promise((resolve, reject) => {
      const id = ++this.nextId;
      const p = { resolve, reject, timer: null };
      if (timeoutMs) {
        p.timer = setTimeout(() => {
          // 応答しないWorkerは止める以外に手が無い。新しいWorkerは次の呼び出しで作る。
          this.reset((q) => (q === p ? onTimeout(q) : q.reject(new Error('中断されました'))));
        }, timeoutMs);
      }
      this.pending.set(id, p);
      w.postMessage({ id, method, args });
    });
  }
}

// クエリ実行と、スキーマ推論等の軽い処理でWorkerを分ける。同じWorkerだと
// 終わらないクエリの後ろにスキーマ推論が詰まり、打ち切りの巻き添えになるため。
const queryWorker = new WorkerClient();
const analysisWorker = new WorkerClient();

export function RunQuery(jsonText, query, raw) {
  // 前のクエリがまだ実行中なら、それは古い編集内容に対するもの（App.svelteの
  // evaluate()は新しい呼び出しで古い結果を捨てる）。終わるのを待つと新しい
  // クエリが最大5秒待たされるので、Workerごと打ち切ってから実行する。
  if (queryWorker.pending.size > 0) {
    queryWorker.reset((p) => p.reject(new Error('新しいクエリの実行により中断されました')));
  }
  return queryWorker.call('RunQuery', [jsonText, query, raw], {
    timeoutMs: QUERY_TIMEOUT_MS,
    onTimeout: (p) => p.resolve({ result: '', error: TIMEOUT_MESSAGE, errorKind: 'runtime' }),
  });
}

export function ExtractUsedKeys(query) {
  return analysisWorker.call('ExtractUsedKeys', [query]);
}

export function InferSchema(jsonText) {
  return analysisWorker.call('InferSchema', [jsonText]);
}

export function SchemaJSON(jsonText) {
  return analysisWorker.call('SchemaJSON', [jsonText]);
}

export function EngineInfo() {
  return analysisWorker.call('EngineInfo', []);
}

// 編集中の内容を失わないよう、外部リンクは別タブで開く。
export function OpenURL(url) {
  window.open(url, '_blank', 'noopener,noreferrer');
}

// デスクトップ版（app.goのmaxOpenJSONFileSize）と同じ上限・同じ文言。
const MAX_OPEN_JSON_FILE_SIZE = 20 * 1024 * 1024;

// キャンセル時は空文字を返す（デスクトップ版と同じ）。
export function OpenJSONFile() {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json,application/json';
    input.addEventListener('cancel', () => resolve(''));
    input.addEventListener('change', async () => {
      const file = input.files?.[0];
      if (!file) {
        resolve('');
        return;
      }
      if (file.size > MAX_OPEN_JSON_FILE_SIZE) {
        reject(
          new Error(
            `ファイルサイズが大きすぎます（${(file.size / (1024 * 1024)).toFixed(1)}MB、上限${
              MAX_OPEN_JSON_FILE_SIZE / (1024 * 1024)
            }MB）。より小さいファイルを選択してください。`,
          ),
        );
        return;
      }
      try {
        resolve(await file.text());
      } catch (e) {
        reject(e);
      }
    });
    input.click();
  });
}

export async function ClipboardSetText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (e) {
    return false;
  }
}
