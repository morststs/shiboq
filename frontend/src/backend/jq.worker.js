// Web版のjq実行用Worker。Goをwasmにビルドしたもの（wasm_main.go）を動かす。
// コンパイル済みのWebAssembly.Moduleはメインスレッドから'init'で受け取る
// （Workerを作り直すたびにダウンロード・コンパイルし直さないため。web.js参照）。
import './generated/wasm_exec.js';

let ready = null;

async function start(module) {
  const go = new Go();
  const instance = await WebAssembly.instantiate(module, go.importObject);
  // go.run()はmain()がブロックするまで同期的に実行するので、戻った時点で
  // shiboqCallは登録済み。run自体のPromiseはプロセス終了まで解決しない。
  go.run(instance);
}

self.onmessage = async (e) => {
  const msg = e.data;
  if (msg.type === 'init') {
    ready = start(msg.module);
    return;
  }
  try {
    await ready;
  } catch (err) {
    self.postMessage({ id: msg.id, error: `jqエンジンを起動できませんでした: ${err?.message || err}` });
    return;
  }
  self.shiboqCall(msg.method, JSON.stringify(msg.args), (result, error) => {
    self.postMessage({ id: msg.id, result, error });
  });
};
