<script>
  let {
    items = [],
    selectedId = null,
    canSave = false,
    saveDisabledReason = '',
    onSelect,
    onRequestSave,
    onRequestDelete,
  } = $props();

  function formatTime(iso) {
    const d = new Date(iso);
    return d.toLocaleString('ja-JP', { hour12: false });
  }

  // 名前は任意入力なので、空ならクエリを見出し代わりに使う。
  function displayLabel(item) {
    return item.name?.trim() ? item.name : item.query;
  }
</script>

<div class="pane">
  <div class="pane-header">
    <span>保存</span>
    <button
      class="save-btn"
      disabled={!canSave}
      title={canSave ? '現在のJSON・クエリ・結果を保存します (Ctrl+S)' : saveDisabledReason}
      onclick={() => onRequestSave?.()}
    >＋ 保存</button>
  </div>
  <ul class="list">
    {#each items as item (item.id)}
      <li class="row" class:selected={item.id === selectedId}>
        <button class="item" onclick={() => onSelect?.(item.id)}>
          <div class="label">{displayLabel(item)}</div>
          <div class="time">{formatTime(item.createdAt)}</div>
        </button>
        <!-- 行のホバーで現れる削除ボタン。常時表示だと一覧が煩雑になるため。 -->
        <button
          class="delete-btn"
          title="削除"
          aria-label="{displayLabel(item)} を削除"
          onclick={() => onRequestDelete?.(item.id)}
        >×</button>
      </li>
    {:else}
      <li class="empty">保存された項目はありません</li>
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
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .save-btn {
    margin-left: auto;
    padding: 2px 8px;
    border: 1px solid #3c3c3c;
    border-radius: 4px;
    background: #2d2d2d;
    color: #cccccc;
    font-family: inherit;
    font-size: 12px;
    cursor: pointer;
    white-space: nowrap;
  }
  .save-btn:hover:not(:disabled) {
    border-color: #007acc;
    color: #ffffff;
  }
  .save-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }
  .list {
    list-style: none;
    overflow-y: auto;
    flex: 1;
  }
  .row {
    display: flex;
    align-items: stretch;
    border-bottom: 1px solid #333;
  }
  .row:hover {
    background: #2a2d2e;
  }
  .row.selected {
    background: #094771;
  }
  .item {
    flex: 1;
    min-width: 0;
    text-align: left;
    background: none;
    border: none;
    padding: 8px 10px;
    cursor: pointer;
    color: #cccccc;
  }
  .label {
    font-family: "Source Code Pro", monospace;
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .time {
    font-size: 11px;
    color: #969696;
  }
  .delete-btn {
    flex-shrink: 0;
    width: 28px;
    border: none;
    background: none;
    color: #969696;
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
    /* ホバー時だけ見せる。キーボード操作でも到達できるよう:focus-visibleも含める。 */
    opacity: 0;
  }
  .row:hover .delete-btn,
  .delete-btn:focus-visible {
    opacity: 1;
  }
  .delete-btn:hover {
    color: #f48771;
  }
  .empty {
    padding: 10px;
    color: #6a6a6a;
    font-size: 12px;
  }
</style>
