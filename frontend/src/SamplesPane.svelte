<script>
  import { untrack } from 'svelte';
  import { splitInlineCode } from './inlineCode.js';

  // chapters: samples.json の chapters。selectedKey は "章ID/サンプルID"。
  let { chapters = [], selectedKey = null, onSelect } = $props();

  // 開いている章。最初は第1章だけ開き、サンプルが選ばれたらその章も開く。
  // 開閉は利用者の操作を優先するため、どちらも「きっかけ」のときだけ開き、
  // 閉じた章を勝手に開き直さない（openChaptersはuntrackで読み、依存にしない）。
  let openChapters = $state(new Set());
  let initialized = false;

  $effect(() => {
    if (!initialized && chapters.length > 0) {
      initialized = true;
      openChapters = new Set([chapters[0].id]);
    }
  });

  $effect(() => {
    const chapterId = selectedKey?.split('/')[0];
    if (!chapterId) return;
    untrack(() => {
      if (!openChapters.has(chapterId)) openChapters = new Set([...openChapters, chapterId]);
    });
  });

  function toggle(id) {
    const next = new Set(openChapters);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    openChapters = next;
  }
</script>

<div class="pane">
  <p class="intro">
    選ぶと JSON とクエリに読み込まれます。解説を読みながら、クエリを書き換えて試してください。
  </p>
  <ul class="chapters">
    {#each chapters as ch (ch.id)}
      <li>
        <button class="chapter" aria-expanded={openChapters.has(ch.id)} onclick={() => toggle(ch.id)}>
          <span class="caret">{openChapters.has(ch.id) ? '▾' : '▸'}</span>
          {ch.title}
        </button>
        {#if openChapters.has(ch.id)}
          <ul class="samples">
            {#each ch.samples as s (s.id)}
              {@const key = `${ch.id}/${s.id}`}
              <li class="sample" class:selected={key === selectedKey}>
                <button class="sample-btn" onclick={() => onSelect?.(ch.id, s.id)}>{s.title}</button>
                {#if key === selectedKey}
                  <div class="detail">
                    <p>
                      {#each splitInlineCode(s.explain) as part}{#if part.code}<code>{part.text}</code>{:else}{part.text}{/if}{/each}
                    </p>
                    {#if s.try}
                      <p class="try">
                        <span class="try-label">やってみよう</span>
                        {#each splitInlineCode(s.try) as part}{#if part.code}<code>{part.text}</code>{:else}{part.text}{/if}{/each}
                      </p>
                    {/if}
                  </div>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </li>
    {/each}
  </ul>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow-y: auto;
    background: #252526;
  }
  .intro {
    padding: 8px 10px;
    font-size: 12px;
    line-height: 1.6;
    color: #969696;
    border-bottom: 1px solid #333;
  }
  .chapters,
  .samples {
    list-style: none;
  }
  .chapter {
    display: flex;
    gap: 6px;
    width: 100%;
    padding: 7px 10px;
    border: none;
    border-bottom: 1px solid #333;
    background: #2d2d2d;
    color: #e7e7e7;
    font-family: inherit;
    font-size: 13px;
    font-weight: bold;
    text-align: left;
    cursor: pointer;
  }
  .chapter:hover {
    background: #333;
  }
  .caret {
    width: 10px;
    color: #969696;
  }
  .sample {
    border-bottom: 1px solid #333;
  }
  .sample-btn {
    display: block;
    width: 100%;
    padding: 6px 10px 6px 26px;
    border: none;
    background: none;
    color: #cccccc;
    font-family: inherit;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }
  .sample-btn:hover {
    background: #2a2d2e;
  }
  .sample.selected .sample-btn {
    background: #094771;
    color: #ffffff;
  }
  .detail {
    padding: 8px 10px 10px 26px;
    background: #1e1e1e;
    font-size: 12px;
    line-height: 1.7;
    color: #cccccc;
  }
  .try {
    margin-top: 8px;
    padding: 6px 8px;
    border-left: 3px solid #0e639c;
    background: #252526;
  }
  .try-label {
    display: block;
    font-size: 11px;
    font-weight: bold;
    color: #4fc1ff;
  }
  code {
    padding: 0 3px;
    border-radius: 3px;
    background: #333;
    color: #ce9178;
    font-family: "Source Code Pro", "SFMono-Regular", Consolas, monospace;
    font-size: 12px;
    word-break: break-all;
  }
</style>
