package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/itchyny/gojq"
)

// デフォルトの実行制限。JqServiceのフィールドで上書きできる
// （テストで実際に5秒待たされないよう、テストは小さい値に差し替える）。
const (
	defaultTimeout        = 5 * time.Second
	defaultMaxOutputs     = 10000
	defaultMaxOutputBytes = 10 * 1024 * 1024 // 10MB
)

// JqService は jq クエリの実行・解析をまとめる。
// Timeout/MaxOutputs/MaxOutputBytes はゼロ値ならNewJqServiceのデフォルトが
// 使われる。テストからは直接フィールドを差し替えて小さい値で高速に検証する。
type JqService struct {
	Timeout        time.Duration
	MaxOutputs     int
	MaxOutputBytes int
}

func NewJqService() *JqService {
	return &JqService{
		Timeout:        defaultTimeout,
		MaxOutputs:     defaultMaxOutputs,
		MaxOutputBytes: defaultMaxOutputBytes,
	}
}

// RunResultErrorKind は RunResult.ErrorKind が取り得る値。
// フロントエンドがエラーの出処（JSON入力かjqクエリか）を、日本語メッセージの
// 前方一致に頼らず判定するための判別子。
type RunResultErrorKind string

const (
	ErrorKindJSON    RunResultErrorKind = "json"    // 入力JSON自体が不正
	ErrorKindSyntax  RunResultErrorKind = "syntax"  // jqクエリの構文エラー
	ErrorKindCompile RunResultErrorKind = "compile" // jqクエリのコンパイルエラー（未定義関数など）
	ErrorKindRuntime RunResultErrorKind = "runtime" // 実行時エラー・タイムアウト・出力上限超過
)

// RunResult は RunQuery の結果。Error が空文字なら成功。
type RunResult struct {
	Result    string             `json:"result"`
	Error     string             `json:"error"`
	ErrorKind RunResultErrorKind `json:"errorKind,omitempty"`
}

// RunQuery は元のJSONに対してjqクエリを実行する。
// パースエラー・コンパイルエラー・実行時エラーはそれぞれ日本語の接頭辞を付けて
// Errorに格納し、Resultは空のまま返す。ErrorKindでエラーの種類を判別できる。
// 呼び出し側はErrorが空でない場合、結果ペインを更新してはいけない。
// 複数の出力（`.a, .b` 等）は改行区切りで連結する。
//
// 実行はTimeoutで打ち切られ（gojqはcontextのキャンセルをIter.Next()から
// エラー値として返す）、出力もMaxOutputs件・MaxOutputBytesバイトで打ち切る。
// これらが無いと、500msデバウンスで自動実行される性質上、`repeat(.)`や
// `range(1e9)`のような編集途中のクエリが終了しないgoroutineを量産し、
// メモリを際限なく消費する。
func (s *JqService) RunQuery(jsonText, query string) RunResult {
	var input any
	if err := json.Unmarshal([]byte(jsonText), &input); err != nil {
		return RunResult{Error: "JSONの解析に失敗しました: " + err.Error(), ErrorKind: ErrorKindJSON}
	}

	parsed, err := gojq.Parse(query)
	if err != nil {
		return RunResult{Error: "構文エラー: " + err.Error(), ErrorKind: ErrorKindSyntax}
	}

	code, err := gojq.Compile(parsed)
	if err != nil {
		return RunResult{Error: "コンパイルエラー: " + err.Error(), ErrorKind: ErrorKindCompile}
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	maxOutputs := s.MaxOutputs
	if maxOutputs <= 0 {
		maxOutputs = defaultMaxOutputs
	}
	maxBytes := s.MaxOutputBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxOutputBytes
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	iter := code.RunWithContext(ctx, input)
	var lines []string
	count := 0
	totalBytes := 0
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if haltErr, ok := err.(*gojq.HaltError); ok && haltErr.Value() == nil {
				// `halt` によるエラーで値がnilの場合は正常終了として扱う。
				break
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return RunResult{
					Error:     fmt.Sprintf("実行がタイムアウトしました（%.0f秒以内に完了しませんでした）。無限ループや終了しないクエリになっていないか確認してください。", timeout.Seconds()),
					ErrorKind: ErrorKindRuntime,
				}
			}
			if errors.Is(err, context.Canceled) {
				return RunResult{Error: "実行がキャンセルされました", ErrorKind: ErrorKindRuntime}
			}
			return RunResult{Error: "実行エラー: " + err.Error(), ErrorKind: ErrorKindRuntime}
		}
		count++
		if count > maxOutputs {
			return RunResult{
				Error:     fmt.Sprintf("出力件数が上限（%d件）を超えました。より絞り込んだクエリを使ってください。", maxOutputs),
				ErrorKind: ErrorKindRuntime,
			}
		}
		v = normalizeNonFiniteFloats(v)
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return RunResult{Error: "結果の整形に失敗しました: " + err.Error(), ErrorKind: ErrorKindRuntime}
		}
		totalBytes += len(b)
		if totalBytes > maxBytes {
			return RunResult{
				Error:     fmt.Sprintf("出力サイズが上限（%dMB）を超えました。より絞り込んだクエリを使ってください。", maxBytes/(1024*1024)),
				ErrorKind: ErrorKindRuntime,
			}
		}
		lines = append(lines, string(b))
	}
	return RunResult{Result: strings.Join(lines, "\n")}
}

// normalizeNonFiniteFloats はfloat64のNaN/±Infを再帰的にnilへ置き換える。
// encoding/jsonはNaN/Infを丸められないため、そのままMarshalIndentに渡すと
// 「結果の整形に失敗しました」になってしまう（本物のjqは`nan`/`infinite`
// 組み込み関数の結果を`null`として表示する）。
func normalizeNonFiniteFloats(v any) any {
	switch x := v.(type) {
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return nil
		}
		return x
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalizeNonFiniteFloats(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = normalizeNonFiniteFloats(e)
		}
		return out
	default:
		return v
	}
}

// InferSchema はJSONからキーツリー（型付き）を推論する（schema.go参照）。
func (s *JqService) InferSchema(jsonText string) ([]SchemaNode, error) {
	return InferSchema(jsonText)
}

// ExtractUsedKeys はjqクエリのAST（抽象構文木）を解析し、
// `.foo` のようなドットによるフィールドアクセスで参照されているキー名の
// 一覧を返す（重複排除、順不同）。`.["foo"]` のようなブラケット記法は対象外。
func (s *JqService) ExtractUsedKeys(query string) ([]string, error) {
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	walkQueryForKeys(q, set)
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	return keys, nil
}

func walkQueryForKeys(q *gojq.Query, set map[string]bool) {
	if q == nil {
		return
	}
	walkTermForKeys(q.Term, set)
	for _, fd := range q.FuncDefs {
		walkQueryForKeys(fd.Body, set)
	}
	walkQueryForKeys(q.Left, set)
	walkQueryForKeys(q.Right, set)
	for _, p := range q.Patterns {
		walkPatternForKeys(p, set)
	}
}

func walkTermForKeys(t *gojq.Term, set map[string]bool) {
	if t == nil {
		return
	}
	collectIndexKey(t.Index, set)
	for _, sfx := range t.SuffixList {
		collectIndexKey(sfx.Index, set)
	}
	if t.Func != nil {
		for _, a := range t.Func.Args {
			walkQueryForKeys(a, set)
		}
	}
	if t.Object != nil {
		for _, kv := range t.Object.KeyVals {
			if kv.Val == nil {
				// `{age}` のような省略記法は `.age` の参照とみなす。
				if kv.Key != "" {
					set[kv.Key] = true
				}
				continue
			}
			walkQueryForKeys(kv.KeyQuery, set)
			walkQueryForKeys(kv.Val, set)
		}
	}
	if t.Array != nil {
		walkQueryForKeys(t.Array.Query, set)
	}
	if t.If != nil {
		walkQueryForKeys(t.If.Cond, set)
		walkQueryForKeys(t.If.Then, set)
		for _, e := range t.If.Elif {
			walkQueryForKeys(e.Cond, set)
			walkQueryForKeys(e.Then, set)
		}
		walkQueryForKeys(t.If.Else, set)
	}
	if t.Try != nil {
		walkQueryForKeys(t.Try.Body, set)
		walkQueryForKeys(t.Try.Catch, set)
	}
	if t.Reduce != nil {
		walkQueryForKeys(t.Reduce.Query, set)
		walkPatternForKeys(t.Reduce.Pattern, set)
		walkQueryForKeys(t.Reduce.Start, set)
		walkQueryForKeys(t.Reduce.Update, set)
	}
	if t.Foreach != nil {
		walkQueryForKeys(t.Foreach.Query, set)
		walkPatternForKeys(t.Foreach.Pattern, set)
		walkQueryForKeys(t.Foreach.Start, set)
		walkQueryForKeys(t.Foreach.Update, set)
		walkQueryForKeys(t.Foreach.Extract, set)
	}
	if t.Label != nil {
		walkQueryForKeys(t.Label.Body, set)
	}
	if t.Str != nil {
		for _, sq := range t.Str.Queries {
			walkQueryForKeys(sq, set)
		}
	}
	if t.Unary != nil {
		walkTermForKeys(t.Unary.Term, set)
	}
	walkQueryForKeys(t.Query, set)
}

func collectIndexKey(idx *gojq.Index, set map[string]bool) {
	if idx == nil {
		return
	}
	if idx.Name != "" {
		set[idx.Name] = true
	}
	walkQueryForKeys(idx.Start, set)
	walkQueryForKeys(idx.End, set)
}

func walkPatternForKeys(p *gojq.Pattern, set map[string]bool) {
	if p == nil {
		return
	}
	for _, ap := range p.Array {
		walkPatternForKeys(ap, set)
	}
	for _, po := range p.Object {
		walkPatternObjectForKeys(po, set)
	}
}

func walkPatternObjectForKeys(po *gojq.PatternObject, set map[string]bool) {
	if po == nil {
		return
	}
	if po.Key != "" {
		set[po.Key] = true
	}
	if po.KeyString != nil {
		for _, sq := range po.KeyString.Queries {
			walkQueryForKeys(sq, set)
		}
	}
	walkQueryForKeys(po.KeyQuery, set)
	walkPatternForKeys(po.Val, set)
}
