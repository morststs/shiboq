import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

// flowbite-svelte はTypeScript入りの.svelteを配布しているため、
// Svelte 5 の組み込みTS除去だけではビルドが失敗する箇所がある。
// esbuildによるTSトランスパイルを明示的に有効化する（sirusitaと同じ対策）。
export default {
  preprocess: vitePreprocess({ script: true }),
}
