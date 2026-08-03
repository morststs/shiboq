// jqクエリペイン用のMonaco補完プロバイダ。
// 現在のJSONから収集したキー名（フラット・重複排除）と、
// 代表的なjq組み込み関数を候補として表示する。
const BUILTIN_FUNCTIONS = [
  'map', 'select', 'keys', 'length', 'sort_by', 'group_by',
  'has', 'to_entries', 'from_entries', 'unique', 'flatten',
  'add', 'min_by', 'max_by', 'reverse', 'empty', 'not', 'type',
];

// getKeys: () => string[] を渡すと、補完要求のたびに最新のキー一覧を反映する。
export function registerJqCompletions(monaco, getKeys) {
  return monaco.languages.registerCompletionItemProvider('jq', {
    triggerCharacters: ['.'],
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };
      const keySuggestions = getKeys().map((key) => ({
        label: key,
        kind: monaco.languages.CompletionItemKind.Field,
        insertText: key,
        detail: 'JSONのキー',
        range,
      }));
      const functionSuggestions = BUILTIN_FUNCTIONS.map((name) => ({
        label: name,
        kind: monaco.languages.CompletionItemKind.Function,
        insertText: name,
        detail: 'jq組み込み関数',
        range,
      }));
      return { suggestions: [...keySuggestions, ...functionSuggestions] };
    },
  });
}
