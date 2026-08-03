<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';
  import { registerJqCompletions } from './jqCompletions.js';

  let { value = '', onChange, errorMessage = '', getKeys } = $props();

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
  <div class="pane-header">jqクエリ</div>
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
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
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
