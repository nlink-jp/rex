# rex — アーキテクチャ

## 目的

正規表現を使ってテキストまたはJSONストリームから名前付きフィールドを抽出し、
構造化JSONとして出力するパイプ対応CLIツール。Splunkの`rex`コマンドに相当。

## 動作モード

### テキストモード（デフォルト）

```
stdin (プレーンテキスト行)
  → 各行に正規表現を適用
  → 1件以上マッチした行のみJSON出力（マッチなしの行はスキップ）
```

### JSONモード（`--field`）

```
stdin (JSONL)
  → 各行をJSONオブジェクトとしてパース
  → 対象フィールドをドット記法で解決
  → フィールド値に正規表現を適用
  → 抽出フィールドを元のオブジェクトにマージ
  → エンリッチされたJSONを出力（マッチなしでも常に出力）
```

## モジュール構成

```
main.go      CLIフラグ、バリデーション、I/Oセットアップ、モード分岐
process.go   applyRegexes(), processLines(), processJSON()
field.go     getNestedField() — ドット記法フィールド解決
```

### 依存関係

```
main.go
  ├── process.go
  │     └── field.go    (JSONモードのみ)
  └── encoding/json     (設定ファイル読み込み)
```

外部依存なし。標準ライブラリのみ。

## データフロー

### テキストモード

```
reader ──► bufio.Scanner ──► line ──► applyRegexes() ──► map[string]interface{}
                                                              │
                                                  len > 0 ?   │
                                                  ┌───yes──────┘
                                                  ▼
                                          json.Marshal() ──► writer
```

### JSONモード

```
reader ──► bufio.Scanner ──► line ──► json.Unmarshal() ──► map[string]interface{}
                                                              │
                                                  getNestedField(obj, fieldPath)
                                                              │
                                              ┌── 未検出/非文字列 ──► パススルー
                                              ▼
                                     applyRegexes(fieldValue)
                                              │
                                     トップレベルobjにマージ
                                              │
                                     json.Marshal(obj) ──► writer
```

## 主要な動作仕様

### 複数値フィールド

同じ名前付きグループが複数パターンに出現する場合:

1. 初回キャプチャ → 文字列
2. 2回目キャプチャ → `[]string{1回目, 2回目}`
3. 以降 → スライスに追加

`-u` 指定時: 追加前に重複を抑制。

### フィールド上書き（JSONモード）

抽出フィールドは元のJSONオブジェクトの既存キーを上書きする。
マージや配列結合は行わない。

### エラー処理

| 条件 | 動作 |
|------|------|
| 無効な正規表現 | 即時終了 |
| 名前付きグループなし | 即時終了 |
| `--field`モードで非JSON | 行番号付きで即時終了 |
| マッチなし（テキストモード） | サイレントスキップ |
| フィールド未検出/非文字列（JSON） | 変更なしでパススルー |
| JSONマーシャル失敗 | 警告ログ、行スキップ |

## 設定

### パターンソース

パターンは以下から提供可能:
- `-r` フラグ（複数指定可）: インライン正規表現
- `-f` フラグ: `{"patterns": [...]}` 形式のJSONファイル
- 両方併用: パターンは結合される

### I/O

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-i` | stdin | 入力元 |
| `-o` | stdout | 出力先 |

## テスト戦略

### ユニットテスト可能な関数

| 関数 | ファイル | 入力 | 依存 |
|------|---------|------|------|
| `applyRegexes` | process.go | 文字列, 正規表現, uniqueフラグ | なし |
| `processLines` | process.go | io.Writer, io.Reader, 正規表現, unique | なし |
| `processJSON` | process.go | io.Writer, io.Reader, 正規表現, unique, fieldPath | `getNestedField` |
| `getNestedField` | field.go | map, パス文字列 | なし |

全コアロジックが`io.Reader`/`io.Writer`を受け取り、ファイルI/Oなしで完全にテスト可能。

### main.go（結合テストのみ）

CLIフラグ解析、ファイル開閉、モード分岐。ビルド済みバイナリ経由か、
`run()`関数抽出によりテスト。

### カバレッジ目標

| モジュール | 現在 | 目標 |
|-----------|------|------|
| field.go | 100% | 100% |
| process.go | 70-83% | 90%+ |
| main.go | 0% | 60%+（`run()`抽出による） |
| **全体** | 44% | **80%+** |
