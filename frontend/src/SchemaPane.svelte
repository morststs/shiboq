<script>
  import SchemaTreeNode from './SchemaTreeNode.svelte';

  let { nodes = [], usedKeys = new Set(), errorMessage = '' } = $props();
</script>

<div class="pane">
  <div class="pane-header">スキーマ</div>
  <div class="tree">
    {#if errorMessage}
      <div class="error-text">{errorMessage}</div>
    {:else if nodes.length === 0}
      <div class="empty">キーがありません</div>
    {:else}
      <ul>
        {#each nodes as node (node.key)}
          <SchemaTreeNode {node} {usedKeys} />
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: #1e1e1e;
  }
  .pane-header {
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .tree {
    flex: 1;
    overflow: auto;
    padding: 8px;
  }
  .empty,
  .error-text {
    color: #6a6a6a;
    font-size: 13px;
    padding: 8px;
  }
  .error-text {
    color: #f48771;
  }
</style>
