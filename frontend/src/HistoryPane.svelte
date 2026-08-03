<script>
  let { entries = [], selectedId = null, onSelect } = $props();

  function formatTime(iso) {
    const d = new Date(iso);
    return d.toLocaleString('ja-JP', { hour12: false });
  }
</script>

<div class="pane">
  <div class="pane-header">履歴</div>
  <ul class="list">
    {#each entries as entry (entry.id)}
      <li>
        <button class="item" class:selected={entry.id === selectedId} onclick={() => onSelect?.(entry.id)}>
          <div class="time">{formatTime(entry.createdAt)}</div>
          <div class="query">{entry.query}</div>
        </button>
      </li>
    {:else}
      <li class="empty">まだ実行履歴がありません</li>
    {/each}
  </ul>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: #252526;
  }
  .pane-header {
    padding: 6px 10px;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .list {
    list-style: none;
    overflow-y: auto;
    flex: 1;
  }
  .item {
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    border-bottom: 1px solid #333;
    padding: 8px 10px;
    cursor: pointer;
    color: #cccccc;
  }
  .item:hover {
    background: #2a2d2e;
  }
  .item.selected {
    background: #094771;
  }
  .time {
    font-size: 11px;
    color: #969696;
  }
  .query {
    font-family: "Source Code Pro", monospace;
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .empty {
    padding: 10px;
    color: #6a6a6a;
    font-size: 12px;
  }
</style>
