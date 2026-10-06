<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';
  import { registerJqCompletions } from './jqCompletions.js';

  let { value = '', onChange, errorMessage = '', getKeys, onHelp } = $props();

  let container;
  let editor;
  let applyingExternal = false;
  let completionDisposable;

  onMount(() => {
    editor = monaco.editor.create(container, {
      value,
      language: 'jq',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: '"Source Code Pro", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      tabSize: 2,
    });

    completionDisposable = registerJqCompletions(monaco, getKeys ?? (() => []));

    editor.onDidChangeModelContent(() => {
      if (applyingExternal) return;
      onChange?.(editor.getValue());
    });
  });

  onDestroy(() => {
    editor?.dispose();
    completionDisposable?.dispose();
  });

  $effect(() => {
    const v = value;
    if (editor && editor.getValue() !== v) {
      applyingExternal = true;
      editor.setValue(v);
      applyingExternal = false;
    }
  });
</script>

<div class="pane">
  <div class="pane-header">
    <span>jqクエリ</span>
    <button class="help-btn" title="jq の書き方・困ったとき・バージョン情報" onclick={() => onHelp?.()}>？ ヘルプ</button>
  </div>
  <div class="editor" bind:this={container}></div>
  {#if errorMessage}
    <div class="error-bar">⚠ {errorMessage}</div>
  {/if}
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
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .help-btn {
    margin-left: auto;
    padding: 0 8px;
    border: 1px solid #3c3c3c;
    border-radius: 4px;
    background: #2d2d2d;
    color: #cccccc;
    font-family: inherit;
    font-size: 12px;
    cursor: pointer;
  }
  .help-btn:hover {
    border-color: #007acc;
    color: #ffffff;
  }
  .editor {
    flex: 1;
    min-height: 0;
  }
  .error-bar {
    flex-shrink: 0;
    padding: 6px 10px;
    background: #3a1414;
    color: #f48771;
    font-size: 12px;
    white-space: pre-wrap;
    border-top: 1px solid #5a1d1d;
  }
</style>
