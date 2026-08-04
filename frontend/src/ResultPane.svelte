<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';

  let { value = '', stale = false, raw = false, onRawChange } = $props();

  let container;
  let editor;

  onMount(() => {
    editor = monaco.editor.create(container, {
      value,
      // 'json' ではなく専用の 'jq-result'（monaco.js参照）を使う。RunQuery は
      // 複数出力を改行区切りで連結するため、全体としては有効なJSONにならない
      // ことがあり（`.[]` 等）、'json' 言語のフル機能の検証だと正当な出力に
      // 「End of file expected」の赤波線が付いてしまう。
      language: 'jq-result',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: '"Source Code Pro", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      readOnly: true,
      tabSize: 2,
    });
  });

  onDestroy(() => {
    editor?.dispose();
  });

  $effect(() => {
    const v = value;
    if (editor && editor.getValue() !== v) {
      editor.setValue(v);
    }
  });
</script>

<div class="pane">
  <div class="pane-header">
    <span>実行結果</span>
    {#if stale}
      <span class="stale-badge">前回の成功結果</span>
    {/if}
    <!-- jq の -r 相当。結果が文字列のときだけ引用符とエスケープを外す。
         @csv / @tsv の結果が二重に引用符で包まれるのを避けるためのモード。 -->
    <label class="raw-toggle" title="結果が文字列のとき、引用符とエスケープを外して表示します（jq の -r 相当）">
      <input type="checkbox" checked={raw} onchange={(e) => onRawChange?.(e.currentTarget.checked)} />
      <span>生出力 (-r)</span>
    </label>
  </div>
  <div class="editor" class:stale bind:this={container}></div>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .pane-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .stale-badge {
    padding: 1px 6px;
    border-radius: 3px;
    background: #5a4a1d;
    color: #e8c877;
    font-size: 11px;
  }
  /* 見出しの右端に寄せる。バッジの有無でトグルの位置が動かないようにする。 */
  .raw-toggle {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
  }
  .raw-toggle input {
    margin: 0;
    cursor: pointer;
  }
  .raw-toggle:hover {
    color: #cccccc;
  }
  .editor {
    flex: 1;
    min-height: 0;
    transition: opacity 0.15s;
  }
  .editor.stale {
    opacity: 0.55;
  }
</style>
