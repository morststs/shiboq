<script>
  import SavedPane from './SavedPane.svelte';
  import JsonPane from './JsonPane.svelte';
  import SchemaPane from './SchemaPane.svelte';
  import QueryPane from './QueryPane.svelte';
  import ResultPane from './ResultPane.svelte';
  import { onMount, onDestroy } from 'svelte';
  import { OpenJSONFile } from '../wailsjs/go/main/App';
  import { RunQuery, ExtractUsedKeys, InferSchema } from '../wailsjs/go/main/JqService';
  import { SaveItem, ListSaved, DeleteSaved } from '../wailsjs/go/main/SavedService';

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
  let savedItems = $state([]);
  let selectedSavedId = $state(null);
  // 名前入力ダイアログ・削除確認ダイアログの状態。
  let saveDialogOpen = $state(false);
  let saveDialogName = $state('');
  let deleteTargetId = $state(null);
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

  function showToast(message) {
    toastMessage = message;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
      toastMessage = '';
    }, 3000);
  }

  // refreshSaved()自身の呼び出しごとに採番するトークン。
  // onMount と保存/削除の後から呼ばれるため、古い呼び出しの
  // ListSaved() 応答が新しい呼び出しの応答より後に返ってきても、
  // 一覧を古い内容で上書きしないようにする。
  let savedRequestToken = 0;

  async function refreshSaved() {
    const requestToken = ++savedRequestToken;
    try {
      const list = (await ListSaved()) ?? [];
      if (requestToken !== savedRequestToken) return;
      savedItems = list;
    } catch (e) {
      if (requestToken !== savedRequestToken) return;
      savedItems = [];
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

  function restoreSaved(id) {
    const item = savedItems.find((i) => i.id === id);
    if (!item) return;
    jsonText = item.json;
    query = item.query;
    result = item.result;
    errorMessage = '';
    resultStale = false;
    selectedSavedId = item.id;
  }

  // --- 保存 ---

  // 実行がエラー中は保存させない。表示中の結果は前回成功時のもので、
  // いま画面にあるJSON/クエリと対応しないため、保存すると内容が食い違う。
  let canSave = $derived(!resultStale && errorMessage === '' && schemaError === '');
  const SAVE_DISABLED_REASON = 'エラーを解消してから保存してください（表示中の結果は現在の内容と対応していません）';

  function openSaveDialog() {
    if (!canSave) return;
    saveDialogName = '';
    saveDialogOpen = true;
  }

  function cancelSave() {
    saveDialogOpen = false;
  }

  async function confirmSave() {
    saveDialogOpen = false;
    try {
      const saved = await SaveItem({
        name: saveDialogName.trim(),
        json: jsonText,
        query,
        result,
      });
      selectedSavedId = saved.id;
      await refreshSaved();
    } catch (e) {
      showToast('保存できませんでした');
    }
  }

  // --- 削除 ---

  let deleteTarget = $derived(savedItems.find((i) => i.id === deleteTargetId) ?? null);

  function requestDelete(id) {
    deleteTargetId = id;
  }

  function cancelDelete() {
    deleteTargetId = null;
  }

  async function confirmDelete() {
    const id = deleteTargetId;
    deleteTargetId = null;
    if (!id) return;
    try {
      await DeleteSaved(id);
      // 選択中のものを消したら選択を外す（存在しないIDを指したままにしない）。
      if (selectedSavedId === id) selectedSavedId = null;
      await refreshSaved();
    } catch (e) {
      showToast('削除できませんでした');
    }
  }

  // Ctrl+S でも保存ダイアログを開く。ブラウザ/WebViewの既定動作（ページ保存）は抑止する。
  function handleKeydown(e) {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      openSaveDialog();
      return;
    }
    if (e.key === 'Escape') {
      if (saveDialogOpen) cancelSave();
      else if (deleteTargetId) cancelDelete();
    }
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

  // 保存ペインの幅（スプリッターでリサイズ可能）。参考プロジェクト sirusita
  // (https://github.com/morststs/sirusita) のstartDrag/onDrag/stopDragパターンを
  // 踏襲し、localStorageへ幅を永続化する。
  const SAVED_WIDTH_STORAGE_KEY = 'shiboq.savedWidth';
  const RAW_OUTPUT_STORAGE_KEY = 'shiboq.rawOutput';
  const SAVED_MIN_WIDTH = 160;
  const SAVED_MAX_WIDTH = 500;
  let savedWidth = $state(260);
  let draggingSaved = $state(false);

  function clampSavedWidth(w) {
    return Math.min(SAVED_MAX_WIDTH, Math.max(SAVED_MIN_WIDTH, w));
  }

  function startSavedDrag(e) {
    draggingSaved = true;
    e.preventDefault();
    window.addEventListener('mousemove', onSavedDrag);
    window.addEventListener('mouseup', stopSavedDrag);
  }

  function onSavedDrag(e) {
    savedWidth = clampSavedWidth(e.clientX);
  }

  function stopSavedDrag() {
    if (!draggingSaved) return;
    draggingSaved = false;
    localStorage.setItem(SAVED_WIDTH_STORAGE_KEY, String(savedWidth));
    window.removeEventListener('mousemove', onSavedDrag);
    window.removeEventListener('mouseup', stopSavedDrag);
  }

  onMount(() => {
    const storedWidth = parseInt(localStorage.getItem(SAVED_WIDTH_STORAGE_KEY), 10);
    if (!isNaN(storedWidth)) {
      savedWidth = clampSavedWidth(storedWidth);
    }
    rawOutput = localStorage.getItem(RAW_OUTPUT_STORAGE_KEY) === '1';
    refreshSaved();
  });

  onDestroy(() => {
    clearTimeout(toastTimer);
    stopSavedDrag();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="app-layout" class:dragging={draggingSaved}>
  <div class="saved-col" style="width: {savedWidth}px">
    <SavedPane
      items={savedItems}
      selectedId={selectedSavedId}
      {canSave}
      saveDisabledReason={SAVE_DISABLED_REASON}
      onSelect={restoreSaved}
      onRequestSave={openSaveDialog}
      onRequestDelete={requestDelete}
    />
  </div>
  <!-- ドラッグ操作のみのリサイズハンドル。sirusita（参考プロジェクト）のsplitterと同じ
       パターンで、キーボード操作の代替手段は無い（既知の制約、将来的にはrole="separator"
       + 矢印キー操作を追加する余地がある）。 -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="splitter"
    class:active={draggingSaved}
    onmousedown={startSavedDrag}
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

{#if saveDialogOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="modal-overlay" onclick={cancelSave}>
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-title">保存</div>
      <div class="modal-body">
        <label class="modal-label" for="save-name">名前（省略可）</label>
        <!-- svelte-ignore a11y_autofocus -->
        <input
          id="save-name"
          class="modal-input"
          bind:value={saveDialogName}
          placeholder={query}
          autofocus
          onkeydown={(e) => { if (e.key === 'Enter') confirmSave(); }}
        />
        <p class="modal-note">空のままにすると、一覧にはクエリが表示されます。</p>
      </div>
      <div class="modal-actions">
        <button class="modal-btn cancel" onclick={cancelSave}>キャンセル</button>
        <button class="modal-btn primary" onclick={confirmSave}>保存する</button>
      </div>
    </div>
  </div>
{/if}

{#if deleteTarget}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="modal-overlay" onclick={cancelDelete}>
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-title">削除の確認</div>
      <div class="modal-body">
        「{deleteTarget.name?.trim() ? deleteTarget.name : deleteTarget.query}」を削除します。<br />
        この操作は元に戻せません。よろしいですか？
      </div>
      <div class="modal-actions">
        <button class="modal-btn cancel" onclick={cancelDelete}>キャンセル</button>
        <button class="modal-btn danger" onclick={confirmDelete}>削除する</button>
      </div>
    </div>
  </div>
{/if}

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
  .saved-col {
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
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1100;
  }
  .modal {
    width: 380px;
    max-width: calc(100vw - 40px);
    background: #252526;
    border: 1px solid #3c3c3c;
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    overflow: hidden;
  }
  .modal-title {
    padding: 14px 18px;
    font-size: 15px;
    font-weight: bold;
    color: #e7e7e7;
    border-bottom: 1px solid #3c3c3c;
  }
  .modal-body {
    padding: 18px;
    color: #cccccc;
    font-size: 14px;
    line-height: 1.7;
  }
  .modal-label {
    display: block;
    font-size: 12px;
    color: #969696;
    margin-bottom: 6px;
  }
  .modal-input {
    width: 100%;
    padding: 6px 8px;
    background: #1e1e1e;
    border: 1px solid #3c3c3c;
    border-radius: 4px;
    color: #cccccc;
    font-family: inherit;
    font-size: 14px;
  }
  .modal-input:focus {
    outline: none;
    border-color: #007acc;
  }
  .modal-note {
    margin-top: 8px;
    font-size: 12px;
    color: #6a6a6a;
    line-height: 1.5;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px 18px;
    border-top: 1px solid #3c3c3c;
  }
  .modal-btn {
    padding: 6px 16px;
    border-radius: 4px;
    border: 1px solid #3c3c3c;
    cursor: pointer;
    font-family: inherit;
    font-size: 13px;
  }
  .modal-btn.cancel {
    background: #2d2d2d;
    color: #cccccc;
  }
  .modal-btn.cancel:hover {
    background: #3a3a3a;
    color: #ffffff;
  }
  .modal-btn.primary {
    background: #0e639c;
    border-color: #0e639c;
    color: #ffffff;
  }
  .modal-btn.primary:hover {
    background: #1177bb;
    border-color: #1177bb;
  }
  .modal-btn.danger {
    background: #a1260d;
    border-color: #a1260d;
    color: #ffffff;
  }
  .modal-btn.danger:hover {
    background: #c4341a;
    border-color: #c4341a;
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
