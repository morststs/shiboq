<script>
  // jq の使い方に困ったとき用のヘルプ。記載内容は内蔵エンジン（gojq）で
  // 実際に動作を確かめたもの（使えない機能の一覧も実測に基づく）。
  let { engineInfo = null, onClose, onOpenUrl, onShowSamples } = $props();

  const sections = [
    { id: 'help-version', title: 'バージョン情報' },
    { id: 'help-usage', title: 'shiboq の使い方' },
    { id: 'help-syntax', title: '構文の早見表' },
    { id: 'help-functions', title: 'よく使う関数' },
    { id: 'help-trouble', title: '困ったときは' },
    { id: 'help-limits', title: '使えない機能・jq との違い' },
    { id: 'help-more', title: 'もっと学ぶ' },
  ];

  let body;

  function jump(id) {
    body?.querySelector(`#${id}`)?.scrollIntoView({ block: 'start' });
  }

  const JQ_MANUAL_URL = $derived(
    `https://jqlang.org/manual/v${engineInfo?.jqLanguageVersion ?? '1.7'}/`,
  );
  const GOJQ_URL = $derived(`https://github.com/itchyny/gojq/tree/${engineInfo?.engineVersion ?? 'main'}`);
  const GOJQ_DIFF_URL = $derived(`${GOJQ_URL}#difference-to-jq`);

  const syntax = [
    ['.', '入力そのもの', '.'],
    ['.key', 'キーの値（無ければ null）', '.name'],
    ['."キー" / .["キー"]', '日本語・記号・空白を含むキー', '."氏名"'],
    ['.[n] / .[-1]', '配列の n 番目（0 始まり）／最後', '.items[0]'],
    ['.[a:b]', '配列・文字列の範囲（b の手前まで）', '.items[1:3]'],
    ['.[]', '配列・オブジェクトの値を 1 つずつ出す', '.items[]'],
    ['a | b', 'a の結果を b に渡す', '.items[] | .id'],
    ['a, b', '結果を複数出す', '.id, .name'],
    ['[ ... ]', '結果を配列にまとめる', '[.items[].id]'],
    ['{a, b: .x}', 'オブジェクトを作る（a は a: .a の省略）', '{name, n: .count}'],
    ['{(式): 値}', '式の結果をキーにする', '{(.id): .name}'],
    ['a // b', 'a が null・false・空なら b', '.nick // .name'],
    ['式?', 'エラーを無視する', '.[]?'],
    ['try a catch b', 'エラーなら b', 'try tonumber catch 0'],
    ['if c then a elif d then b else e end', '条件分岐（else は省略可）', 'if .n > 0 then "+" end'],
    ['式 as $x | ...', '変数に入れる', '.id as $i | ...'],
    ['reduce g as $x (init; 更新)', '畳み込み', 'reduce .[] as $x (0; . + $x)'],
    ['def f: 本体;', '関数を定義（名前は英数字と _）', 'def double: . * 2;'],
    ['"文字 \\(式)"', '文字列に埋め込む', '"\\(.id): \\(.name)"'],
    ['.a |= f / .a += 1', 'その場所を更新する', '.price |= . * 2'],
    ['..', 'すべての階層をたどる', '.. | .id? // empty'],
    ['== != < <= > >=', '比較', 'select(.n >= 10)'],
    ['and or not', '論理演算（not は後置）', 'select(.a and (.b | not))'],
  ];

  const functions = [
    ['配列', 'length, map(f), select(条件), sort, sort_by(f), group_by(f), unique, unique_by(f), reverse, min, max, min_by(f), max_by(f), add, any, all, flatten, first, last, limit(n; f), index(x), contains(x), inside(x), range(n), indices(x), IN(x), INDEX(f; key)'],
    ['オブジェクト', 'keys, has(key), to_entries, from_entries, with_entries(f), del(path), pick(path), paths, paths(f), getpath(p), setpath(p; v), path(f)'],
    ['文字列', 'split(s), join(s), ascii_downcase, ascii_upcase, ltrimstr(s), rtrimstr(s), trimstr(s), trim, ltrim, rtrim, startswith(s), endswith(s), tostring, tonumber, toboolean, length, explode, implode'],
    ['正規表現', 'test(re), match(re), capture(re), scan(re), splits(re), sub(re; s), gsub(re; s)（フラグは g / i / m のみ）'],
    ['型・判定', 'type, arrays, objects, strings, numbers, booleans, nulls, iterables, scalars, values, empty, error(msg)'],
    ['数値', 'floor, ceil, round, abs, sqrt, pow(a; b), log, min, max, add, tostring'],
    ['形式（@）', '@csv, @tsv, @json, @text, @html, @uri, @urid, @sh, @base64, @base64d（文字列の前に置くと埋め込み部分だけに適用: @uri "q=\\(.x)"）'],
    ['日付', 'now, todate, fromdate, todateiso8601, fromdateiso8601, strftime(fmt), strptime(fmt)'],
    ['その他', 'tojson, fromjson, tostream, fromstream, recurse, repeat(f), while(c; f), until(c; f), env, $ENV'],
  ];

  const troubles = [
    {
      q: '結果が null になる',
      a: 'キー名の綴りと大文字小文字を確認してください（存在しないキーはエラーではなく null になります）。配列の中のキーなら `.items.name` ではなく `.items[].name` のように `[]` が必要です。スキーマペインで実際のキー名と型を確認できます。',
    },
    {
      q: '構文エラー: unexpected token "会"（日本語のキーや関数名）',
      a: '英数字と `_` 以外を含むキー名は `.会員` と書けません。`."会員"` または `.["会員"]` と書いてください。オブジェクトを作るときのキーも `{"商品": .name}` のように引用符が必要です。関数名（def）には英数字と `_` しか使えません。',
    },
    {
      q: 'cannot iterate over: null / expected an object but got: array',
      a: '前者は null に `[]` を使った、後者は配列に `.name` を使ったときのエラーです。直前までのクエリの結果を結果ペインで確認し、配列なら `.[]` で展開してからキーを指定します。値が無いことがあるなら `.items[]?` や `(.items // [])[]` で避けられます。',
    },
    {
      q: '結果が複数行に分かれて出る／配列にしたい',
      a: '`.[]` やカンマは結果を 1 つずつ出力します（jq の仕様です）。1 つの配列にまとめたいときはクエリ全体を `[ ]` で囲みます。',
    },
    {
      q: '文字列に引用符が付く／CSV が "..." で囲まれる',
      a: '結果ペインの「生出力 (-r)」をオンにしてください。結果が文字列のときだけ、引用符とエスケープを外して表示します（jq の `-r` と同じ）。',
    },
    {
      q: 'select で何も出てこない',
      a: '比較している値の型を確認してください。`"10"`（文字列）と `10`（数値）は等しくありません。`select(.n == "10")` か `select((.n | tonumber) == 10)` のように型を合わせます。`type` で型を確認できます。',
    },
    {
      q: 'cannot add: string and number（文字列と数値を + でつなげない）',
      a: '`"ID: " + .id` は .id が数値だとエラーになります。`"ID: " + (.id | tostring)` とするか、文字列埋め込み `"ID: \\(.id)"` を使ってください。',
    },
    {
      q: 'キーの順番が入力と変わる',
      a: '内蔵エンジン（gojq）は出力時にオブジェクトのキーをアルファベット順に並べます（jq の `-S` と同じ）。キーの順番を保つ方法はありません（`keys_unsorted` もありません）。',
    },
    {
      q: '実行がタイムアウトした／出力が多すぎると言われた',
      a: '5 秒以内に終わらないクエリ（`def f: f; f` や `repeat(.)` など）は打ち切ります。出力は 10,000 件・10MB までです。`limit(10; ...)` や `first(...)` で件数を絞ってください。',
    },
    {
      q: 'どこにあるキーか分からない',
      a: '`[.. | .キー名? // empty]` ですべての階層から探せます。`[paths | map(tostring) | join(".")]` とすると、値のある場所の一覧が見られます（サンプルの「10. 更新と探索」参照）。',
    },
    {
      q: '正規表現のフラグ x・s などがエラーになる',
      a: '使えるフラグは g（全体一致）・i（大文字小文字を無視）・m（. を改行にも一致）の 3 つだけです。後読み・先読み・後方参照も使えません（内蔵エンジンが Go の正規表現を使っているため）。',
    },
  ];
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="overlay" onclick={() => onClose?.()}>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="dialog" role="dialog" tabindex="-1" aria-modal="true" aria-labelledby="help-title" onclick={(e) => e.stopPropagation()}>
    <div class="head">
      <h2 id="help-title">jq ヘルプ</h2>
      <button class="close" aria-label="閉じる" title="閉じる (Esc)" onclick={() => onClose?.()}>×</button>
    </div>
    <div class="layout">
      <nav class="toc" aria-label="目次">
        {#each sections as s (s.id)}
          <button onclick={() => jump(s.id)}>{s.title}</button>
        {/each}
      </nav>
      <div class="body" bind:this={body}>
        <section id="help-version">
          <h3>バージョン情報</h3>
          <table class="kv">
            <tbody>
              <tr>
                <th>準拠する jq の仕様</th>
                <td><strong>jq {engineInfo?.jqLanguageVersion ?? '…'}</strong>（jq 1.8 で追加された <code>trimstr</code>・<code>trim</code>・<code>toboolean</code>・<code>add(f)</code>・<code>skip</code> も使えます）</td>
              </tr>
              <tr>
                <th>jq エンジン</th>
                <td>{engineInfo ? `${engineInfo.engine} ${engineInfo.engineVersion}` : '…'}（Go で書かれた jq の実装。jq コマンドのインストールは不要です）</td>
              </tr>
            </tbody>
          </table>
          <p class="note">
            jq 本家との細かな違いは「使えない機能・jq との違い」を参照してください。
            公式マニュアルは jq {engineInfo?.jqLanguageVersion ?? '1.7'} の版が目安になります。
          </p>
        </section>

        <section id="help-usage">
          <h3>shiboq の使い方</h3>
          <ul>
            <li>JSON とクエリは入力が止まってから 0.5 秒後に自動で実行されます。実行ボタンはありません。</li>
            <li>クエリがエラーの間、結果ペインには直前に成功した結果が薄く残ります（「前回の成功結果」と表示）。</li>
            <li>スキーマペインには JSON のキーと型が表示され、クエリで使っているキーが色付きになります。</li>
            <li>クエリペインでは JSON のキー名と関数名が補完されます（<kbd>Ctrl</kbd>+<kbd>Space</kbd> で候補を表示）。</li>
            <li>「生出力 (-r)」で、文字列の結果を引用符なしで表示します（@csv などに便利）。</li>
            <li>「＋ 保存」または <kbd>Ctrl</kbd>+<kbd>S</kbd> で、JSON・クエリ・結果を保存できます。</li>
            <li>左の「サンプル」タブに、jq を順を追って学べる例題があります。
              <button class="link" onclick={() => onShowSamples?.()}>サンプルを開く</button>
            </li>
          </ul>
        </section>

        <section id="help-syntax">
          <h3>構文の早見表</h3>
          <table class="cheat">
            <thead><tr><th>書き方</th><th>意味</th><th>例</th></tr></thead>
            <tbody>
              {#each syntax as [form, meaning, example]}
                <tr><td><code>{form}</code></td><td>{meaning}</td><td><code>{example}</code></td></tr>
              {/each}
            </tbody>
          </table>
        </section>

        <section id="help-functions">
          <h3>よく使う関数</h3>
          <table class="cheat">
            <tbody>
              {#each functions as [group, list]}
                <tr><th>{group}</th><td class="mono">{list}</td></tr>
              {/each}
            </tbody>
          </table>
        </section>

        <section id="help-trouble">
          <h3>困ったときは</h3>
          {#each troubles as t}
            <details>
              <summary>{t.q}</summary>
              <p>
                {#each t.a.split('`') as part, i}{#if i % 2 === 1}<code>{part}</code>{:else}{part}{/if}{/each}
              </p>
            </details>
          {/each}
        </section>

        <section id="help-limits">
          <h3>使えない機能・jq との違い</h3>
          <ul>
            <li><strong>入力は 1 つだけ</strong>: <code>input</code> / <code>inputs</code> / <code>input_filename</code> は使えません。</li>
            <li><strong>デバッグ出力</strong>: <code>debug</code> / <code>stderr</code> はありません。途中の値は、クエリをそこで切って結果ペインで確認してください。<code>halt</code> / <code>halt_error</code> は何も出力しません。</li>
            <li><strong>未対応の関数・形式</strong>: <code>keys_unsorted</code>、<code>$__loc__</code>、<code>input_line_number</code>、<code>get_jq_origin</code>、jq 1.8 の <code>have_literal_numbers</code>・<code>have_decnum</code> など。</li>
            <li><strong>キーの順番</strong>: 出力時にオブジェクトのキーはアルファベット順に並びます。</li>
            <li><strong>正規表現</strong>: フラグは g / i / m のみ。後読み・先読み・後方参照は使えません。</li>
            <li><strong>大きな整数</strong>: 足し算などでは桁が落ちません（jq より正確）。ただし <code>floor</code> などの数学関数は小数として計算します。</li>
            <li><strong>JSON の拡張</strong>: 入力 JSON に <code>NaN</code> や <code>Infinity</code> は書けません。結果の <code>nan</code> / <code>infinite</code> は null と表示します。</li>
            <li><strong>文字列の添字</strong>: <code>"abcde"[2]</code> のように 1 文字を取り出せます（jq では未対応）。</li>
            <li><strong>実行の制限</strong>: 5 秒・10,000 件・10MB で打ち切ります（自動実行のため）。</li>
          </ul>
          <p class="note">
            詳しくは gojq の
            <button class="link" onclick={() => onOpenUrl?.(GOJQ_DIFF_URL)}>Difference to jq</button>
            （英語）を参照してください。
          </p>
        </section>

        <section id="help-more">
          <h3>もっと学ぶ</h3>
          <ul>
            <li>
              <button class="link" onclick={() => onOpenUrl?.(JQ_MANUAL_URL)}>jq {engineInfo?.jqLanguageVersion ?? '1.7'} 公式マニュアル</button>（英語）
            </li>
            <li><button class="link" onclick={() => onOpenUrl?.(GOJQ_URL)}>gojq（内蔵エンジン）</button>（英語）</li>
            <li>左の「サンプル」タブ: 値の取り出しから集計・正規表現・パス操作まで 10 章の例題</li>
          </ul>
          <p class="note">リンクは外部のサイトをブラウザで開きます。</p>
        </section>
      </div>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1100;
  }
  .dialog {
    display: flex;
    flex-direction: column;
    width: min(960px, calc(100vw - 40px));
    height: min(720px, calc(100vh - 40px));
    background: #252526;
    border: 1px solid #3c3c3c;
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    overflow: hidden;
  }
  .head {
    display: flex;
    align-items: center;
    padding: 12px 18px;
    border-bottom: 1px solid #3c3c3c;
  }
  h2 {
    font-size: 15px;
    color: #e7e7e7;
  }
  .close {
    margin-left: auto;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 4px;
    background: none;
    color: #969696;
    font-size: 20px;
    line-height: 1;
    cursor: pointer;
  }
  .close:hover {
    background: #3a3a3a;
    color: #ffffff;
  }
  .layout {
    display: flex;
    flex: 1;
    min-height: 0;
  }
  .toc {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    width: 200px;
    padding: 8px 0;
    border-right: 1px solid #3c3c3c;
    overflow-y: auto;
  }
  .toc button {
    padding: 6px 16px;
    border: none;
    background: none;
    color: #cccccc;
    font-family: inherit;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }
  .toc button:hover {
    background: #2a2d2e;
    color: #ffffff;
  }
  .body {
    flex: 1;
    min-width: 0;
    padding: 4px 24px 24px;
    overflow-y: auto;
    font-size: 13px;
    line-height: 1.7;
    color: #cccccc;
  }
  section {
    padding-top: 16px;
  }
  h3 {
    margin-bottom: 8px;
    padding-bottom: 4px;
    border-bottom: 1px solid #3c3c3c;
    font-size: 14px;
    color: #e7e7e7;
  }
  ul {
    padding-left: 20px;
  }
  li {
    margin: 2px 0;
  }
  .note {
    margin-top: 8px;
    font-size: 12px;
    color: #969696;
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  .cheat th,
  .cheat td,
  .kv th,
  .kv td {
    padding: 4px 8px;
    border: 1px solid #3c3c3c;
    vertical-align: top;
    text-align: left;
  }
  .cheat thead th,
  .cheat tbody th,
  .kv th {
    background: #2d2d2d;
    font-weight: normal;
    color: #969696;
    white-space: nowrap;
  }
  .mono {
    font-family: "Source Code Pro", "SFMono-Regular", Consolas, monospace;
    font-size: 12px;
  }
  code,
  kbd {
    padding: 0 3px;
    border-radius: 3px;
    background: #333;
    color: #ce9178;
    font-family: "Source Code Pro", "SFMono-Regular", Consolas, monospace;
    font-size: 12px;
  }
  kbd {
    color: #e7e7e7;
    border: 1px solid #555;
  }
  details {
    border-bottom: 1px solid #333;
  }
  summary {
    padding: 6px 0;
    color: #e7e7e7;
    cursor: pointer;
  }
  details p {
    padding: 0 0 10px 16px;
  }
  .link {
    padding: 0;
    border: none;
    background: none;
    color: #4fc1ff;
    font-family: inherit;
    font-size: inherit;
    text-decoration: underline;
    cursor: pointer;
  }
  .link:hover {
    color: #9cdcfe;
  }
</style>
