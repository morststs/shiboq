// エディタ本体 + JSON言語サービス（検証・補完・ホバー等）のみを取り込み、
// 全言語版は避けてバンドルサイズを抑える。JSON言語サービスは専用のworkerを
// 必要とするため、getWorker()はlabelで振り分けてEditorWorkerとJsonWorkerを
// 返す。この振り分けは必須で、labelを無視して常にEditorWorkerを返すと
// JSON検証・補完等がworker経由で実行時に失敗する（過去に実際発生した不具合）。
// jqクエリ用には専用の言語IDだけを登録し、補完機能のみ提供する
// （QueryPane.svelte / jqCompletions.js参照）。
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';

self.MonacoEnvironment = {
  getWorker(_moduleId, label) {
    if (label === 'json') return new JsonWorker();
    return new EditorWorker();
  },
};

monaco.languages.register({ id: 'jq' });

// 結果ペイン（ResultPane.svelte）用の言語ID。
// RunQueryは複数出力（`.a, .b` や `.[]` 等）を改行区切りで連結して返すため、
// 出力全体としては有効なJSON文書にならないことがある（jqとしては正しい仕様）。
// 'json'言語をそのまま使うと、monaco.contributionが提供するフル機能のJSON
// 言語サービスが構文検証してしまい、正当な複数出力に「End of file expected」
// の赤波線が付く。JSON入力ペイン（JsonPane.svelte）の検証は欲しい機能なので
// 言語サービス自体は無効化せず、結果ペイン専用に検証を持たない軽量な
// Monarchトークナイザ（ハイライトのみ）を 'jq-result' として登録する。
monaco.languages.register({ id: 'jq-result' });
monaco.languages.setMonarchTokensProvider('jq-result', {
  tokenizer: {
    root: [
      [/"(?:[^"\\]|\\.)*"/, 'string'],
      [/-?\d+(\.\d+)?([eE][+-]?\d+)?/, 'number'],
      [/\btrue\b|\bfalse\b/, 'keyword'],
      [/\bnull\b/, 'keyword'],
      [/[{}[\]]/, '@brackets'],
      [/[,:]/, 'delimiter'],
    ],
  },
});

export default monaco;
