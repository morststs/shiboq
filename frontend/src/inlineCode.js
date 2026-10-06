// サンプルの解説文（samples.json）は、コード部分をバッククォートで囲んで書いている。
// それを「通常の文字列」と「コード」の断片に分ける。HTMLとして解釈はしない
// （{@html}を使わずに描画するため、文面に記号が含まれても安全）。
export function splitInlineCode(text) {
  return (text ?? '').split('`').map((part, i) => ({ code: i % 2 === 1, text: part }));
}
