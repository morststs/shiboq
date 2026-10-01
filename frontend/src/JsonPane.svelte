<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';
  import { Button } from 'flowbite-svelte';

  let { value = '', onChange, onOpenFile } = $props();

  let container;
  let editor;
  let applyingExternal = false;

  onMount(() => {
    editor = monaco.editor.create(container, {
      value,
      language: 'json',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: '"Source Code Pro", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      tabSize: 2,
    });

    editor.onDidChangeModelContent(() => {
      if (applyingExternal) return;
      onChange?.(editor.getValue());
    });
  });

  onDestroy(() => {
    editor?.dispose();
  });

  // Monaco標準の整形アクション（JSON言語サービスのフォーマッタ）を使う。
  // JSON.parse→JSON.stringifyにしないのは、大きな整数の精度落ち・数値表記の
  // 変化（1.0→1等）を避け、Ctrl+Zで整形前に戻せるようにするため。
  // 編集として扱われるのでonDidChangeModelContent経由で親にも反映される。
  function format() {
    // Monacoは読み込んだ内容からインデント幅を自動検出する（4スペースやタブの
    // JSONを開くとそれに合わせる）ため、整形結果が常に2スペースになるよう固定する。
    editor?.getModel()?.updateOptions({ tabSize: 2, insertSpaces: true });
    editor?.getAction('editor.action.formatDocument')?.run();
  }

  // 親（履歴の復元・ファイル読み込み）からの反映。編集中の自己ループを避けるため、
  // 値が実際に異なる場合のみ setValue する。
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
    <span>JSON</span>
    <div class="actions">
      <Button size="xs" color="alternative" onclick={format}>整形</Button>
      <Button size="xs" color="alternative" onclick={() => onOpenFile?.()}>ファイルを開く</Button>
    </div>
  </div>
  <div class="editor" bind:this={container}></div>
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
    justify-content: space-between;
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .actions {
    display: flex;
    gap: 6px;
  }
  .editor {
    flex: 1;
    min-height: 0;
  }
</style>
