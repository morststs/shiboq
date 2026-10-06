// デスクトップ版（Wails）のバックエンド。Goのバインディングをそのまま公開する。
// Web版はweb.jsで、どちらを使うかはvite.config.jsの'$backend'エイリアスで
// ビルド時に切り替える（App.svelteは'$backend'からimportするだけ）。
export { OpenJSONFile } from '../../wailsjs/go/main/App';
export { RunQuery, ExtractUsedKeys, InferSchema, SchemaJSON, EngineInfo } from '../../wailsjs/go/main/JqService';
export { SaveItem, ListSaved, DeleteSaved } from '../../wailsjs/go/main/SavedService';
export { ClipboardSetText } from '../../wailsjs/runtime/runtime';
// WebView内でリンクを開くとアプリの画面が外部サイトに置き換わってしまうため、
// 既定のブラウザで開く。
export { BrowserOpenURL as OpenURL } from '../../wailsjs/runtime/runtime';
