# go_ptor

## 1 章 ― アーキテクチャ概観（詳細版）

### 目的

go_ptor は「多段暗号による秘匿通信を“最小の部品数”と“Go 標準ライブラリのみ”で再現する」ことをゴールにします。
完全な匿名性 や 量的耐解析性 は求めず、理解しやすい実装を優先します。

### 1.0 スコープ & 制約

| 項目         | 方針                                                        |
| ------------ | ----------------------------------------------------------- |
| 言語         | **Go 1.22+**（標準ライブラリ + `golang.org/x/crypto` のみ） |
| 暗号         | RSA-2048 / AES-256-GCM（TLS 依存なし）                      |
| ノード数     | 3 – 5 ホップを上限（学習用）                                |
| 依存プロセス | なし（全て単一バイナリ起動）                                |
| OS           | クロスプラットフォーム（mac / linux / win）                 |
| 部外通信     | ディレクトリ取得以外ゼロ                                    |

### 1.1 ランタイムトポロジ

```txt
ブラウザ ──▶ SOCKS5 (127.0.0.1:9050) ─▶ ptor-client
       ▲                                        │
       └────────── TCP ← AES-GCM cells … ←─────┘
```

- 三層ハンドシェイク（Entry→Middle→Exit）に固定
- HiddenService は Exit にローカル TCP でバインド
  - .ptor アドレス = ED25519 公開鍵の Base32
  - リゾルバは ptor-client 内部にハードコード

### 1.2 コンポーネント別技術ポイント

#### 1.2.1 `ptor-client`

| 観点           | 内容                                                                                 |
| -------------- | ------------------------------------------------------------------------------------ |
| **役割**       | (1) SOCKS5 サーバ, (2) Circuit Builder, (3) Cell Mux                                 |
| **起動順**     | ① ディレクトリ読み込み → ② `hops` 分の RSA 公開鍵取得 → ③ 対称鍵 `K₁…Kₙ` 生成        |
| **GOROUTINE**  | _SOCKS 每接続_ に `forward(up)` / `forward(down)` 2 本                               |
| **エラー処理** | タイムアウトは `context.WithTimeout`。ホップ失敗時は `DESTROY` セル送信 → 全て Close |
| **Config**     | YAML / CLI フラグ両対応 (`-entry`, `-hops`, `-dirurl`)                               |

#### 1.2.2 `ptor-relay`

| 観点                                               | 内容                                                       |
| -------------------------------------------------- | ---------------------------------------------------------- |
| **役割**                                           | セル単位で「復号 → 次ホップへ暗号化転送」                  |
| **内部 Map**                                       | `map[circuitID]*ConnState`（対称鍵・前後コネクション保持） |
| **暗号実装**                                       | `crypto/cipher` の `NewGCM`。ノンスはセル頭 12 byte        |
| **パフォーマンス**                                 | `io.CopyBuffer` で 512 byte 固定読取。                     |
| Back-pressure は `net.Conn.SetReadDeadline` で実装 |                                                            |
| **安全装置**                                       | 不正セル長 / cmd 不正 → 即 `DESTROY`                       |

#### 1.2.3 `ptor-hidden`

| 観点               | 内容                                                          |
| ------------------ | ------------------------------------------------------------- |
| **役割**           | (1) `.ptor` アドレス生成, (2) 127.0.0.1:<port> でアプリを起動 |
| **キー管理**       | PEM (ED25519 private)\*。再起動しても同一アドレス             |
| **サービス実装例** | 内部で `http.ServeMux` or 好きなバイナリを `exec.Command`     |
| **アクセスポリシ** | リレー以外の接続は `--firewall local` で拒否 (標準設定)       |

> - 公開鍵をハッシュして 52 文字の .ptor を生成：
>   addr := base32.StdEncoding.EncodeToString(sha3.Sum256(pub)[:]) + ".ptor"

#### 1.2.4 `ptor-dir`

| 観点           | 内容                                                        |
| -------------- | ----------------------------------------------------------- |
| **データ形式** | JSON (上位互換のため version フィールド付き)                |
| **署名**       | v1 は省略。v2 で `ed25519` 署名フィールド追加予定           |
| **キャッシュ** | client 側はメモリ + `~/.ptor/cache.json`                    |
| **配布方法**   | HTTP または `file://` パス。学習環境なら GitHub Gist でも可 |

#### 1.2.5 `ptor-keygen`

| 観点       | 内容                                       |
| ---------- | ------------------------------------------ |
| **生成**   | `rsa.GenerateKey(rand.Reader, 2048)`       |
| **保存**   | `x509.MarshalPKCS1PrivateKey` → PEM        |
| **対称鍵** | Circuit 時に `crypto/rand` で 32 byte 生成 |

### 1.3 ゴルーチン / チャネル設計

```scss
ptor-relay
└─ listener.Accept()
   └─ go handleConn(c)
        ├─ decodeLoop()  // 受信セル解析
        └─ encodeLoop()  // 送信用
```

- 完全フルデュープレックス - 逆方向セルは別チャネル
- メモリ食い防止：セルバッファは chan [][]byte サイズ 32

### 1.4 ポート割り当て & Firewall ルール

| コンポ       | デフォルト | 許可元             |
| ------------ | ---------- | ------------------ |
| relay        | TCP 5000   | 他 relay / client  |
| hidden       | TCP 8080   | localhost のみ     |
| client SOCKS | TCP 9050   | localhost のみ     |
| dir          | TCP 7000   | localhost (学習用) |

### 1.5 セキュリティ境界

1. クライアント ⇔ Entry で TLS は張らない（学習用にパケット観察可）
2. Relay 間 は二重 AES-GCM。MITM でも暗号文のみ観測
3. Dir が改竄されるとルートが崩壊 → 将来の署名追加で対策

### 1.6 拡張フック

| 項目                   | 将来追加案                     |
| ---------------------- | ------------------------------ |
| 認証済み隠しサービス   | `AUTH_COOKIE` 方式を導入       |
| QoS/帯域制御           | `tc` or `netem` で人工遅延注入 |
| トラフィックパディング | 定期ダミーセル送出 Goroutine   |

## 第 2 章 ― プロトコル詳細設計（詳細版）

### 2.1 セル（Cell）構造

Tor の本物の “relay cell” は固定長 512 byte ですが、pTor でも同様にします：

```diff
+------+-----+--------+-------------------+-----------+
| CMD  | VER |  LEN   | PAYLOAD (LEN バイト) | PADDING   |
+------+-----+--------+-------------------+-----------+
  1B     1B     2B           可変              ～512B
```

| フィールド | 説明                                           |
| ---------- | ---------------------------------------------- |
| `CMD`      | 処理内容（EXTEND, CONNECT, DATA, END）         |
| `VER`      | バージョン（`0x01`）                           |
| `LEN`      | 有効 `PAYLOAD` 長（`uint16`）                  |
| `PAYLOAD`  | 暗号化された本体（内容はコマンドごとに異なる） |
| `PADDING`  | 512 バイト固定長になるまでランダム埋め         |

#### 📌 Go での構造体

```golang
type Cell struct {
    Cmd     byte
    Version byte
    Length  uint16
    Payload []byte // Lengthバイト分
}
```

### 2.2 コマンド定義

```golang
const (
    CmdExtend  = 0x01
    CmdConnect = 0x02
    CmdData    = 0x03
    CmdEnd     = 0x04
    CmdDestroy = 0x05
)
```

#### ✉️ 各コマンドの目的

| CMD       | 用途                                        | 内容                        |
| --------- | ------------------------------------------- | --------------------------- |
| `EXTEND`  | 回路延長                                    | 対称鍵の公開 & 次ホップ指定 |
| `CONNECT` | Exit ノード → Hidden サービス接続           |                             |
| `DATA`    | 上位アプリケーションの TCP ストリームデータ |                             |
| `END`     | 回路切断通知                                |                             |
| `DESTROY` | 回路エラー／不正通知（Relay が送る）        |                             |

### 2.3 暗号処理

#### 🔐 鍵の種類

| 鍵の種類   | 用途               | サイズ   | 説明                             |
| ---------- | ------------------ | -------- | -------------------------------- |
| RSA 公開鍵 | relay 公開鍵配布用 | 2048 bit | OAEP 形式で暗号化に使用          |
| AES 鍵     | クライアント生成   | 32 byte  | 1 ホップごとの対称鍵（多段）     |
| GCM Nonce  | 暗号用ノンス       | 12 byte  | 毎セルごとに乱数生成（ランダム） |

### 🔁 暗号レイヤー構造

```txt
CLIENT →
    ENC_r1(
        ENC_r2(
            DATA
        )
    )
→ RELAY1 → RELAY2 → …
```

> 多段リレーでは、前段 relay は自身の鍵 K₁ で復号し、次段 relay に中身を丸ごと転送するだけ。

### 2.4 EXTEND セル構造（例）

このコマンドは「次ホップに対して回路を伸ばしたい」という要求。

| 項目                | 内容                         | サイズ |
| ------------------- | ---------------------------- | ------ |
| 次ノードの IP\:Port | `"127.0.0.1:5001"`（文字列） | 可変   |
| AES 鍵（K2）        | 32 バイト                    | 固定   |
| GCM ノンス          | 12 バイト                    | 固定   |

```golang
type ExtendPayload struct {
    NextHop string // ex: "relay2:5001"
    AESKey  []byte // 32B
    Nonce   []byte // 12B
}
```

→ Go ではこれを encoding/gob や msgpack でシリアライズ
→ RSA 公開鍵で AESKey+Nonce を暗号化 → 署名省略（学習用）

### 2.5 DATA セル構造

DATA セルのペイロードは「任意のバイト列」で OK（上位 TCP ストリーム）

- 長さ上限：1 セル = 512B - 4B = 508B
- データがそれ以上の時は 複数セルに分割して送る

```golang
type DataPayload struct {
    StreamID uint16  // SOCKS5経由で複数ストリームサポートも可能
    Data     []byte
}
```

→ 多段暗号が解かれるごとに中身が明らかになる
→ 最終 Relay（Exit）は 127.0.0.1:8080 に接続し、このデータを転送

### 2.6 回路 ID とステート保持

リレーでは以下を Map にして持つ：

```golang
type CircuitState struct {
    CircuitID string     // 固有ID
    AESKey    []byte     // このホップの鍵
    NextHop   net.Conn   // 次段RelayまたはHiddenサービス
    CreatedAt time.Time
}
```

リレーでは：

- `CircuitID` に紐づく鍵を使って復号
- 復号した中身を `NextHop` にそのまま渡す（暗号化せず）

### 2.7 `.ptor` アドレスの生成方式

pTor の Hidden Service では、Tor の .onion と同様に .ptor を生成します：

```golang
func OnionAddr(pub ed25519.PublicKey) string {
    h := sha3.Sum256(pub)
    return base32.StdEncoding.EncodeToString(h[:])[:52] + ".ptor"
}
```

- ローカルの `ptor-hidden` は ED25519 鍵ペアを保持
- クライアントは `.ptor` → 公開鍵ハッシュ → Directory 経由でアドレス探索
- 実装初期では `.ptor` をフラットに管理しても OK

## 第 3 章 ― 通信フロー詳細設計

### 3.1 回路（Circuit）確立フェーズ

🌱 ステップごとの流れ（例：3 ホップ）

```txt
Step 0: client が ptor-dir から relay 公開鍵リストを取得

Step 1: relay1 に EXTEND → K1を暗号化して送信（RSA + GCM nonce）
Step 2: relay1 が K1 を保持。relay2 に転送セル（EXTEND）生成 → K2を転送
Step 3: relay2 が K2 を保持。同様に relay3 に K3 を転送
Step 4: relay3 が K3 を保持。最後の CONNECT セルで hidden service に接続
Step 5: reverse ACK セルが relay3 → relay2 → relay1 → client へ返送

=> 回路完成！
```

> ※ K1, K2, K3 はクライアントが事前に生成（乱数）した AES-256-GCM 鍵

### 3.2 多段鍵配送の仕組み

各 EXTEND セルの Payload は 次ホップ情報 + 次の AES 鍵（暗号化済み）

```txt
Payload: {
  NextHop: "relay2:5001",
  EncAESKey: RSA_ENC(pub_of_relay2, [K2 + nonce])
}
```

relay1 は：

- NextHop を読んで relay2 に TCP 接続
- EncAESKey は そのまま relay2 に送る

relay2 では：

- 自分の秘密鍵で復号し、K2 をメモリに保存

### 3.3 回路確立後のデータ通信

#### 🔁 多段復号の流れ

```txt
[Client]
ENC(K1, ENC(K2, ENC(K3, payload)))

[Relay1]
  └─ Decrypt with K1 → Forward ENC(K2, ENC(K3, payload))
[Relay2]
  └─ Decrypt with K2 → Forward ENC(K3, payload)
[Relay3]
  └─ Decrypt with K3 → Forward payload to hidden-svc
```

> relay たちは「自分のホップ鍵で復号 → 次へ転送」するだけ。
> 中身のアプリケーションデータは絶対に見えない。

### 3.4 回路終了の処理

#### `END` セル

クライアントが接続を閉じると、`END` セルが relay に送られる：

```txt
[Client] → END → [Relay1] → END → [Relay2] → END → [Relay3]
```

各 relay は：

- 回路 ID に紐づく `CircuitState` を削除
- 次ホップへ `END` を転送

#### `DESTROY` セル（異常終了）

何らかの不正が起きた場合：

- セル構造不正
- 復号失敗
- 次ホップ到達不可

などは relay が `DESTROY` を逆方向に送る：

```txt
[Relay2] ← DESTROY ← [Relay3]
↓
[Client] ← DESTROY ← [Relay1]

```

client はこれを受けて回路を強制閉鎖。

### 3.5 多重ストリーム処理（任意）

複数の SOCKS5 接続を一つの回路に多重化するには `StreamID` を追加する必要がある：

```golang
type DataPayload struct {
    StreamID uint16
    Data     []byte
}
```

> pTor の MVP では単一ストリームで OK。
> 拡張時に map[StreamID]\*Conn を relay/client に持てばよい。

### 3.6 タイムアウトと監視

#### ⏳ クライアント側

- 回路確立: `ctx, cancel := context.WithTimeout(...)`
- データ受信なし: `SetReadDeadline(time.Now().Add(10 \* time.Second))`

#### 🧼 Relay 側

- `CircuitState.CreatedAt` から 60 秒以上経過 → 自動破棄

```golang
// GC goroutine
for {
  time.Sleep(10 * time.Second)
  for id, st := range circuits {
    if time.Since(st.CreatedAt) > 1*time.Minute {
      st.Close()
      delete(circuits, id)
    }
  }
}
```

### 3.7 再接続とリトライ

- `EXTEND` で次ホップ接続失敗時 → `DESTROY` を送る
- クライアントはディレクトリから別の構成で回路再構築を試みる

→ 信頼性の高い学習用実装では `retry up to N times` を用意

## 第 4 章 - Go 実装計画

### 4.1 プロジェクト・レイアウト

```txt
/
├── cmd/
│   ├── client/       // ptor-client main
│   ├── relay/        // ptor-relay  main
│   ├── hidden/       // ptor-hidden main
│   ├── directory/    // ptor-dir    main
│   └── keygen/       // ptor-keygen main
├── internal/
│   ├── cell/         // Cell encode/decode
│   ├── crypto/       // RSA・AES ラッパ
│   ├── circuit/      // 回路ステート管理
│   ├── socksproxy/   // 最小 SOCKS5 実装
│   ├── dirclient/    // Directory JSON クライアント
│   ├── logger/       // zap ラッパ（optional）
│   └── util/         // 汎用ヘルパ
└── go.mod

```

### 4.2 主要パッケージ詳細

| パッケージ     | 公開型 / 役割                                                                                                                                                                      | 技術ポイント                                                               |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| **cell**       | `type Cell struct`<br>`func Encode(c *Cell) []byte`<br>`func Decode([]byte) (*Cell, error)`                                                                                        | 固定長 512B / `sync.Pool` でバッファ再利用                                 |
| **crypto**     | `func RSAEncrypt(pub *rsa.PublicKey, in []byte) ([]byte, error)`<br>`func AESSeal(k, nonce, plain []byte) ([]byte, error)`<br>`func AESOpen(k, nonce, enc []byte) ([]byte, error)` | `crypto/rsa` + `crypto/cipher` GCM<br>エラーをラップ (`errors.Join`)       |
| **circuit**    | `type State struct {ID string; Key []byte; Next net.Conn}`<br>`type Table struct { sync.RWMutex; m map[string]*State }`                                                            | GC ゴルーチンを内包<br>`Destroy(id)` で END / DESTROY 送出                 |
| **socksproxy** | `func Start(laddr string, dial func(dest string) net.Conn)`                                                                                                                        | 最小 SOCKS5 (CONNECT のみ)                                                 |
| **dirclient**  | `func Fetch(url string) ([]RelayInfo, error)`                                                                                                                                      | JSON → `[]RelayInfo{ID,Addr,PubKey}`<br>FS/HTTP 判定は `strings.HasPrefix` |
| **logger**     | `type Logger struct { *zap.SugaredLogger }`                                                                                                                                        | 差し替え自由 (学習用には fmt.Printf でも OK)                               |

### 4.3 コンポーネント main の骨格

#### `cmd/client/main.go`

```golang
func main() {
    cfg := loadFlags()                              // entry, hops, dirURL
    relays, err := dirclient.Fetch(cfg.DirURL)
    must(err)

    cir, err := circuit.Build(relays, cfg.Hops)    // 回路確立 (= dial chain)
    must(err)
    defer cir.Close()

    // SOCKS5
    ln, _ := net.Listen("tcp", "127.0.0.1:9050")
    for {
        c, _ := ln.Accept()
        go socksproxy.Handle(c, cir.Dial)          // Dial(streamID) → Cell転送
    }
}

```

#### `cmd/relay/main.go`

```golang
func main() {
    key := crypto.LoadRSAPriv(flag.Arg("key"))
    tbl := circuit.NewTable()

    ln, _ := net.Listen("tcp", flag.Arg("listen"))
    for {
        c, _ := ln.Accept()
        go handleConn(c, key, tbl)
    }
}

func handleConn(c net.Conn, priv *rsa.PrivateKey, tbl *circuit.Table) {
    defer c.Close()
    buf := make([]byte, 512)
    for {
        if _, err := io.ReadFull(c, buf); err != nil { return }
        cell, _ := cell.Decode(buf)

        switch cell.Cmd {
        case cell.CmdExtend:
            tbl.HandleExtend(cell, c, priv)
        case cell.CmdData:
            tbl.Forward(cell)
        case cell.CmdEnd:
            tbl.Teardown(cell.CircuitID)
        default:
            tbl.SendDestroy(cell.CircuitID)
        }
    }
}

```

### `cmd/hidden/main.go`

```golang
func main() {
    hsk := crypto.LoadEdPriv(flag.Arg("key"))
    addr := crypto.HiddenAddr(hsk.Public()) // xxx.ptor
    fmt.Println("Hidden address:", addr)

    go http.ListenAndServe("127.0.0.1:8080", demoMux())

    // Relay動作：単ホップ扱い（前段からTCP受け取る）
    ln, _ := net.Listen("tcp", flag.Arg("listen")) // :5000
    for {
        c, _ := ln.Accept()
        go io.Copy(c, c) // loopback echo (デモ)
    }
}
```

## 第 5 章 ― .ptor Hidden Service 設計

### 5.1 `.ptor` アドレスとは何か？

Tor の `.onion` と同様、公開鍵から導出される自己認証型アドレスです。

pTor では .ptor という TLD を用い、以下のように定義します：

```golang
func HiddenAddr(pub ed25519.PublicKey) string {
    hash := sha3.Sum256(pub)
    return base32.StdEncoding.EncodeToString(hash[:])[:52] + ".ptor"
}
```

| 特徴       | 内容                                        |
| ---------- | ------------------------------------------- |
| アドレス長 | 常に 52 文字（Base32 エンコード）           |
| 安全性     | 公開鍵からしか生成不可（衝突困難）          |
| 解決方法   | クライアント内で Directory を参照して逆引き |

### 5.2 Hidden Service の実態

#### ✅ 特性

| 項目             | 説明                                     |
| ---------------- | ---------------------------------------- |
| 接続先           | localhost のポート（例：127.0.0.1:8080） |
| 公開鍵           | `.ptor` アドレスに対応する ED25519       |
| プライベートキー | ファイルとして PEM 形式で管理            |
| 接続手段         | relay3 → TCP connect to local app        |

#### ✅ デモ構成

```txt
client
 └── ptor-client
       └── relay1 ─ relay2 ─ relay3 ─▶ ptor-hidden
                                        └── HTTP server (127.0.0.1:8080)
```

> relay3 は .ptor に対応する公開鍵を持つ → 対称鍵交換完了済み
> → あとは直接 ptor-hidden にローカル TCP 転送するだけ

### 5.3 鍵管理方式

#### 秘密鍵生成

```golang
pub, priv, _ := ed25519.GenerateKey(rand.Reader)
```

#### PEM 保存と読み込み

```golang
func SaveEDPriv(path string, priv ed25519.PrivateKey) error {
    block := &pem.Block{
        Type:  "ED25519 PRIVATE KEY",
        Bytes: priv,
    }
    return os.WriteFile(path, pem.EncodeToMemory(block), 0600)
}

func LoadEDPriv(path string) (ed25519.PrivateKey, error) {
    pemBytes, _ := os.ReadFile(path)
    block, _ := pem.Decode(pemBytes)
    return ed25519.PrivateKey(block.Bytes), nil
}

```

> ※ Go の ed25519 キーは 64 byte 固定で扱いやすい

### 5.4 Hidden Service の動作構造（内部）

```golang
func main() {
    priv := crypto.LoadEDPriv("hidden.pem")
    addr := crypto.HiddenAddr(priv.Public().(ed25519.PublicKey))

    fmt.Println("公開アドレス:", addr) // -> xxxx…xxxx.ptor

    // 内部アプリケーションの起動
    go http.ListenAndServe("127.0.0.1:8080", handler())

    // リレー接続受け入れ
    ln, _ := net.Listen("tcp", ":5003")
    for {
        conn, _ := ln.Accept()
        go proxyLoop(conn)
    }
}
```

### 5.5 セキュリティとアクセス制限

#### ✅ 外部からの直接アクセスを防ぐには

1. `ptor-hidden` は localhost 限定で Listen
2. `firewalld` / `ufw` で tcp/8080 などを外部ブロック
3. `relay3` 以外からの接続は拒否（IP 制限 or 鍵認証）

### 5.6 Directory 登録と解決

#### 🔍 クライアントの .ptor 解決手順

1. client は "xxxx...xxxx.ptor" を要求
2. ptor-client は directory から対応する公開鍵を探す
3. 対応する relay ノード（exit）と通信確立
4. 最終 hop が hidden-service に接続

Directory は .ptor アドレス → Relay ID のマッピングを JSON で持つ。

例：

```json
{
  "hidden_services": {
    "x7vb24ytkrs5zvv5e3syflrck5ns7ugte6r2yotql2lyqehmdy4a.ptor": {
      "relay": "relay3-id",
      "pubkey": "BASE64_ENCODED_ED25519_PUB"
    }
  }
}
```
