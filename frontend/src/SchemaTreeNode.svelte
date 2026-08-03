<script>
  import SchemaTreeNode from './SchemaTreeNode.svelte';
  import { Badge } from 'flowbite-svelte';

  let { node, usedKeys } = $props();
  let expanded = $state(true);
  let isUsed = $derived(usedKeys.has(node.key));
  let hasChildren = $derived(!!(node.children && node.children.length > 0));
</script>

<li>
  <div class="row" class:used={isUsed}>
    {#if hasChildren}
      <button class="toggle" onclick={() => (expanded = !expanded)}>{expanded ? '▾' : '▸'}</button>
    {:else}
      <span class="toggle-spacer"></span>
    {/if}
    <span class="key">{node.key}</span>
    <span class="type">: {node.type}</span>
    {#if isUsed}
      <Badge color="blue" class="ml-1">使用中</Badge>
    {/if}
  </div>
  {#if hasChildren && expanded}
    <ul>
      {#each node.children as child (child.key)}
        <SchemaTreeNode node={child} {usedKeys} />
      {/each}
    </ul>
  {/if}
</li>

<style>
  li {
    list-style: none;
  }
  ul {
    margin-left: 16px;
    padding-left: 8px;
    border-left: 1px dashed #3c3c3c;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 2px 4px;
    border-radius: 3px;
  }
  .row.used {
    background: #094771;
  }
  .toggle {
    background: none;
    border: none;
    color: #969696;
    cursor: pointer;
    width: 14px;
    flex-shrink: 0;
    padding: 0;
  }
  .toggle-spacer {
    width: 14px;
    flex-shrink: 0;
    display: inline-block;
  }
  .key {
    color: #9cdcfe;
    font-family: "Source Code Pro", monospace;
  }
  .type {
    color: #6a9955;
    font-family: "Source Code Pro", monospace;
  }
</style>
