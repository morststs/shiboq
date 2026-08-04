<script>
  import HistoryPane from './HistoryPane.svelte';
  import JsonPane from './JsonPane.svelte';
  import SchemaPane from './SchemaPane.svelte';
  import QueryPane from './QueryPane.svelte';
  import ResultPane from './ResultPane.svelte';
  import { onMount, onDestroy } from 'svelte';
  import { OpenJSONFile } from '../wailsjs/go/main/App';
  import { RunQuery, ExtractUsedKeys, InferSchema } from '../wailsjs/go/main/JqService';
  import { SaveHistory, ListHistory } from '../wailsjs/go/main/HistoryService';

  const SAMPLE_JSON = '{\n  "name": "taro",\n  "age": 20,\n  "tags": ["admin", "user"],\n  "address": {\n    "city": "tokyo"\n  }\n}';

  let jsonText = $state(SAMPLE_JSON);
  let query = $state('.name');
  let result = $state('');
  let errorMessage = $state('');
  // RunQueryが失敗している間、resultは直近成功時の内容を保持したまま更新しない
  // （意図的な仕様）。しかしそれをそのまま表示すると最新結果であるかのように
  // 見えてしまうため、「古い結果である」ことを示すフラグを別途持つ。
  // errorMessageと連動させない理由: JSON入力エラー（errorKind === 'json'）は
  // クエリペインのエラーバーには出さない（下記evaluate()参照）が、その場合も
  // RunQueryは失敗しresultは更新されないため、resultStaleは独立して真にする。
  let resultStale = $state(false);
  // jq の -r 相当。結果が文字列のときだけ引用符とエスケープを外して表示する。
  // 既定はオフ（jq の既定の挙動に合わせる）。選択はlocalStorageに保存する。
  let rawOutput = $state(false);
  let schemaNodes = $state([]);
  let schemaError = $state('');
  let usedKeys = $state(new Set());
  let historyEntries = $state([]);
  let selectedHistoryId = $state(null);
  let toastMessage = $state('');
  let toastTimer = null;

  let flatKeys = $derived(flattenKeys(schemaNodes));

  function flattenKeys(nodes) {
    const set = new Set();
    const walk = (list) => {
      for (const n of list) {
        set.add(n.key);
        if (n.children) walk(n.children);
      }
    };
    walk(nodes);
    return set;
  }

  let debounceTimer = null;

  // 直近に開始された evaluate() 呼び出しだけが状態を書き込めるようにするためのトークン。
  // evaluate() はバックエンドへの複数回のawaitを直列で行うため、編集が重なると複数回の
  // evaluate() が同時に飛行中になり得る。await から戻るたびにトークンを検証し、
  // 自分より新しい呼び出しが既に始まっていれば（トークンが変わっていれば）、
  // 古い呼び出しは以降の状態更新を一切行わずに黙って中断する。
  let evaluateToken = 0;

  // 復元直後の内容をそのまま履歴に再保存しないためのシグネチャ。
  // 単純なbooleanだと「復元後・次に実行されるevaluate()」がどんな内容でも
  // 無条件に保存をスキップしてしまい、復元後に別の内容へ編集した場合の
  // 正当な実行結果まで保存されずに失われる。復元した内容そのものの
  // シグネチャを保持し、evaluate()が実際に評価する内容と一致する場合に限り
  // スキップすることで、内容が変化すれば自然に「復元済み」とはみなされなく
  // なるようにする（明示的にnullへ戻す必要はない）。
  //
  // 初期値をSAMPLE_JSON/初期クエリの署名にしておくことが重要。そうしないと、
  // 起動直後（未編集）にマウント時の最初のevaluate()が走った際、この初期状態を
  // 「新規の実行結果」として履歴に保存してしまい、アプリを起動して何も操作
  // しなくても履歴が1件増えてしまう（起動50回で同一内容の履歴が50件になる）。
  let restoredSignature = signatureOf(SAMPLE_JSON, '.name');

  function signatureOf(json, q) {
    return JSON.stringify([json, q]);
  }

  function showToast(message) {
    toastMessage = message;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
      toastMessage = '';
    }, 3000);
  }

  // refreshHistory()自身の呼び出しごとに採番するトークン。
  // onMount と evaluate() の両方から呼ばれるため、古い呼び出しの
  // ListHistory() 応答が新しい呼び出しの応答より後に返ってきても、
  // 一覧を古い内容で上書きしないようにする。
  let historyRequestToken = 0;

  async function refreshHistory() {
    const requestToken = ++historyRequestToken;
    try {
      const list = (await ListHistory()) ?? [];
      if (requestToken !== historyRequestToken) return;
      historyEntries = list;
    } catch (e) {
      if (requestToken !== historyRequestToken) return;
      historyEntries = [];
    }
  }

  // evaluate() はJSON/クエリの変更のたびに500ms後に1回だけ呼ばれる（下の$effect参照）。
  // スキーマ再推論・使用キー抽出・クエリ実行をまとめて行う。
  // パースエラー・実行時エラー時は result を更新しない（前回成功時の内容を保持する）。
  // クエリ実行が成功した場合のみ、履歴として自動保存する。
  async function evaluate() {
    const token = ++evaluateToken;
    const currentJson = jsonText;
    const currentQuery = query;
    // rawもawait前に確定させる。await中にトグルが変わっても、この実行は
    // 開始時点の設定で一貫した結果を返す（新しい設定での実行は新しい
    // evaluate()が担当し、tokenガードで古い方が捨てられる）。
    const currentRaw = rawOutput;
    const shouldSkipSave = restoredSignature !== null && restoredSignature === signatureOf(currentJson, currentQuery);

    try {
      const nodes = (await InferSchema(currentJson)) ?? [];
      if (token !== evaluateToken) return;
      schemaNodes = nodes;
      schemaError = '';
    } catch (e) {
      if (token !== evaluateToken) return;
      schemaNodes = [];
      // バックエンド（schema.go）のエラーには「invalid character 'x' at offset N」
      // のような実際に直しどころが分かる詳細が含まれているので、汎用メッセージで
      // 潰さずそのまま出す。
      schemaError = e?.message || String(e) || 'JSONが不正です';
    }

    try {
      const keys = await ExtractUsedKeys(currentQuery);
      if (token !== evaluateToken) return;
      usedKeys = new Set(keys ?? []);
    } catch (e) {
      if (token !== evaluateToken) return;
      // ここでusedKeysを空にリセットしない: クエリ入力の途中（構文が一時的に
      // 不完全な状態）でハイライトが消えて再度点灯する「点滅」を防ぐため、
      // 前回成功時のキー集合をそのまま保持する。
    }

    let r;
    try {
      r = await RunQuery(currentJson, currentQuery, currentRaw);
    } catch (e) {
      // RunQueryはIPC層のエラーで例外を投げることはまず無いはずだが、
      // 万一rejectされた場合にpanesが無反応のまま固まらないようにする。
      if (token !== evaluateToken) return;
      resultStale = true;
      errorMessage = 'クエリの実行に失敗しました';
      return;
    }
    if (token !== evaluateToken) return;
    if (r.error) {
      // resultは意図的に更新しない（前回成功時の内容を保持する）。
      // ただし「今表示されている内容は最新ではない」ことをResultPaneに伝える。
      resultStale = true;
      // JSON自体の不正はJsonPane/SchemaPane側（schemaError）で既に案内済みなので、
      // クエリペインのエラーバーには出さない（クエリ自体は悪くない）。
      errorMessage = r.errorKind === 'json' ? '' : r.error;
      return;
    }
    errorMessage = '';
    resultStale = false;
    result = r.result;

    if (!shouldSkipSave) {
      try {
        const saved = await SaveHistory({ json: currentJson, query: currentQuery, result: r.result });
        if (token !== evaluateToken) return;
        selectedHistoryId = saved.id;
        await refreshHistory();
      } catch (e) {
        // 履歴保存に失敗しても実行結果自体は表示され続ける。ただし気づけないと
        // ユーザーは履歴が保存されていると誤解し続けるので、控えめに通知する。
        if (token !== evaluateToken) return;
        showToast('履歴を保存できませんでした');
      }
    }
  }

  $effect(() => {
    const _ = jsonText + ' ' + query;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(evaluate, 500);
    return () => clearTimeout(debounceTimer);
  });

  // 生出力トグル。チェックボックスなので500msのデバウンスを待たせず、
  // 保留中のタイマーを潰して即座に再実行する。tokenガードがあるため、
  // 進行中のevaluate()と競合しても古い方の結果は書き込まれない。
  function handleRawChange(next) {
    rawOutput = next;
    try {
      localStorage.setItem(RAW_OUTPUT_STORAGE_KEY, next ? '1' : '0');
    } catch (e) {
      // localStorageが使えなくても表示自体は成立するので黙って続行する
    }
    clearTimeout(debounceTimer);
    evaluate();
  }

  function restoreHistory(id) {
    const entry = historyEntries.find((e) => e.id === id);
    if (!entry) return;
    restoredSignature = signatureOf(entry.json, entry.query);
    jsonText = entry.json;
    query = entry.query;
    result = entry.result;
    errorMessage = '';
    resultStale = false;
    selectedHistoryId = entry.id;
  }

  async function handleOpenFile() {
    try {
      const content = await OpenJSONFile();
      // キャンセル時はOpenJSONFileが空文字とnilエラーを返す（rejectしない）ため、
      // ここには到達しない。UI状態を変えずに何もしない。
      if (content) jsonText = content;
    } catch (e) {
      // ファイルサイズ超過等、実際のエラー時（reject）のみここに来る。
      // 通知しないとユーザーはファイルを選んだのに何も起きなかったように見える。
      showToast(e?.message || String(e) || 'ファイルを開けませんでした');
    }
  }

  // 履歴ペインの幅（スプリッターでリサイズ可能）。参考プロジェクト sirusita
  // (https://github.com/morststs/sirusita) のstartDrag/onDrag/stopDragパターンを
  // 踏襲し、localStorageへ幅を永続化する。
  const HISTORY_WIDTH_STORAGE_KEY = 'shiboq.historyWidth';
  const RAW_OUTPUT_STORAGE_KEY = 'shiboq.rawOutput';
  const HISTORY_MIN_WIDTH = 160;
  const HISTORY_MAX_WIDTH = 500;
  let historyWidth = $state(260);
  let draggingHistory = $state(false);

  function clampHistoryWidth(w) {
    return Math.min(HISTORY_MAX_WIDTH, Math.max(HISTORY_MIN_WIDTH, w));
  }

  function startHistoryDrag(e) {
    draggingHistory = true;
    e.preventDefault();
    window.addEventListener('mousemove', onHistoryDrag);
    window.addEventListener('mouseup', stopHistoryDrag);
  }

  function onHistoryDrag(e) {
    historyWidth = clampHistoryWidth(e.clientX);
  }

  function stopHistoryDrag() {
    if (!draggingHistory) return;
    draggingHistory = false;
    localStorage.setItem(HISTORY_WIDTH_STORAGE_KEY, String(historyWidth));
    window.removeEventListener('mousemove', onHistoryDrag);
    window.removeEventListener('mouseup', stopHistoryDrag);
  }

  onMount(() => {
    const savedWidth = parseInt(localStorage.getItem(HISTORY_WIDTH_STORAGE_KEY), 10);
    if (!isNaN(savedWidth)) {
      historyWidth = clampHistoryWidth(savedWidth);
    }
    rawOutput = localStorage.getItem(RAW_OUTPUT_STORAGE_KEY) === '1';
    refreshHistory();
  });

  onDestroy(() => {
    clearTimeout(toastTimer);
    stopHistoryDrag();
  });
</script>

<div class="app-layout" class:dragging={draggingHistory}>
  <div class="history-col" style="width: {historyWidth}px">
    <HistoryPane entries={historyEntries} selectedId={selectedHistoryId} onSelect={restoreHistory} />
  </div>
  <!-- ドラッグ操作のみのリサイズハンドル。sirusita（参考プロジェクト）のsplitterと同じ
       パターンで、キーボード操作の代替手段は無い（既知の制約、将来的にはrole="separator"
       + 矢印キー操作を追加する余地がある）。 -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="splitter"
    class:active={draggingHistory}
    onmousedown={startHistoryDrag}
    title="ドラッグで幅を調整"
  ></div>
  <div class="col">
    <div class="pane-slot"><JsonPane value={jsonText} onChange={(v) => (jsonText = v)} onOpenFile={handleOpenFile} /></div>
    <div class="pane-slot"><SchemaPane nodes={schemaNodes} {usedKeys} errorMessage={schemaError} /></div>
  </div>
  <div class="col">
    <div class="pane-slot"><QueryPane value={query} onChange={(v) => (query = v)} {errorMessage} getKeys={() => Array.from(flatKeys)} /></div>
    <div class="pane-slot"><ResultPane value={result} stale={resultStale} raw={rawOutput} onRawChange={handleRawChange} /></div>
  </div>
</div>

{#if toastMessage}
  <div class="toast">{toastMessage}</div>
{/if}

<style>
  .app-layout {
    display: flex;
    height: 100vh;
  }
  /* ドラッグ中はテキスト選択を抑止し、ペイン内へマウスが入ってもカーソルを統一する */
  .app-layout.dragging {
    cursor: col-resize;
    user-select: none;
  }
  .history-col {
    flex-shrink: 0;
  }
  .splitter {
    width: 5px;
    flex-shrink: 0;
    cursor: col-resize;
    background: #3c3c3c;
    transition: background 0.15s;
  }
  .splitter:hover,
  .splitter.active {
    background: #007acc;
  }
  .col {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #3c3c3c;
  }
  .col:last-child {
    border-right: none;
  }
  .pane-slot {
    flex: 1;
    min-height: 0;
    border-bottom: 1px solid #3c3c3c;
  }
  .pane-slot:last-child {
    border-bottom: none;
  }
  .toast {
    position: fixed;
    bottom: 20px;
    right: 20px;
    background: #007acc;
    color: #ffffff;
    padding: 12px 20px;
    border-radius: 6px;
    font-size: 13px;
    z-index: 1000;
  }
</style>
