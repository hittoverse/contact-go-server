package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	port = ":1337"

	// Security settings
	maxConnections = 100              // 最大同時接続数
	maxInputSize   = 1024             // 入力読み込み上限 (1KB)
	connTimeout    = 1 * time.Minute  // 接続タイムアウト
	readTimeout    = 30 * time.Second // 読み込みタイムアウト

	// Rate limiting settings
	rateLimitWindow  = 10 * time.Second // レート制限の時間枠
	rateLimitMax     = 5                // 時間枠内の最大接続数
	rateLimitCleanup = 1 * time.Minute  // レート制限情報のクリーンアップ間隔
)

// 同時接続数を管理するセマフォ
var (
	connSemaphore = make(chan struct{}, maxConnections)
	activeConns   int64 // 現在のアクティブ接続数（ログ用）
)

// IP単位のレート制限を管理
type rateLimiter struct {
	connections sync.Map // map[string]*ipRateInfo
}

type ipRateInfo struct {
	mu         sync.Mutex
	timestamps []time.Time
}

var limiter = &rateLimiter{}

// checkRateLimit はIPアドレスのレート制限をチェック
func (r *rateLimiter) checkRateLimit(ip string) bool {
	now := time.Now()
	windowStart := now.Add(-rateLimitWindow)

	val, _ := r.connections.LoadOrStore(ip, &ipRateInfo{})
	info := val.(*ipRateInfo)

	info.mu.Lock()
	defer info.mu.Unlock()

	// 古いタイムスタンプを削除
	valid := info.timestamps[:0]
	for _, t := range info.timestamps {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	info.timestamps = valid

	// レート制限チェック
	if len(info.timestamps) >= rateLimitMax {
		return false
	}

	// 新しい接続を記録
	info.timestamps = append(info.timestamps, now)
	return true
}

// cleanup は古いレート制限情報を削除
func (r *rateLimiter) cleanup() {
	cutoff := time.Now().Add(-rateLimitWindow * 2)
	r.connections.Range(func(key, value interface{}) bool {
		info := value.(*ipRateInfo)
		info.mu.Lock()
		// タイムスタンプが全て古い場合は削除
		allOld := true
		for _, t := range info.timestamps {
			if t.After(cutoff) {
				allOld = false
				break
			}
		}
		info.mu.Unlock()
		if allOld {
			r.connections.Delete(key)
		}
		return true
	})
}

// extractIP は接続からIPアドレスを抽出（ポート番号を除去）
func extractIP(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

var contacts = []struct {
	Num  int
	Name string
	URL  string
}{
	{1, "Twitter/X", "x.com/hitto_kun"},
	{2, "GitHub", "github.com/hitto-hub"},
	{3, "Zenn", "zenn.dev/hitto"},
	{4, "Qiita", "qiita.com/hitto"},
	{5, "Blog", "hitto-kun.hatenablog.com"},
}

const banner = `
 _     _ _   _
| |__ (_) |_| |_ ___
| '_ \| | __| __/ _ \
| | | | | |_| || (_) |
|_| |_|_|\__|\__\___/

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Welcome to hitto's contact server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Available endpoints:

`

func main() {
	// Graceful shutdown の設定
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// シグナルハンドリング
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Contact server listening on %s (max connections: %d, rate limit: %d/%s)",
		port, maxConnections, rateLimitMax, rateLimitWindow)

	// レート制限のクリーンアップ用goroutine
	go func() {
		ticker := time.NewTicker(rateLimitCleanup)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				limiter.cleanup()
			}
		}
	}()

	// シャットダウン処理用goroutine
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping new connections...")
		listener.Close()
		cancel()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// シャットダウン時のエラーは正常終了
			select {
			case <-ctx.Done():
				log.Printf("Waiting for %d active connections to close...", atomic.LoadInt64(&activeConns))
				// アクティブ接続が終了するまで待機（最大10秒）
				for i := 0; i < 100; i++ {
					if atomic.LoadInt64(&activeConns) == 0 {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
				log.Println("Server shutdown complete")
				return
			default:
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
		}

		ip := extractIP(conn.RemoteAddr())

		// IP単位のレート制限チェック
		if !limiter.checkRateLimit(ip) {
			log.Printf("Connection rejected from %s: rate limit exceeded", ip)
			fmt.Fprint(conn, "Too many connections. Please wait and try again.\n")
			conn.Close()
			continue
		}

		// 同時接続数の制限チェック (ノンブロッキング)
		select {
		case connSemaphore <- struct{}{}:
			// 接続を受け入れ
			count := atomic.AddInt64(&activeConns, 1)
			log.Printf("New connection from %s (active: %d/%d)", ip, count, maxConnections)
			go handleConnection(conn)
		default:
			// 接続数上限に達している場合は拒否
			log.Printf("Connection rejected from %s: max connections reached (%d)", ip, maxConnections)
			fmt.Fprint(conn, "Server is busy. Please try again later.\n")
			conn.Close()
		}
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		<-connSemaphore // セマフォを解放
		count := atomic.AddInt64(&activeConns, -1)
		log.Printf("Connection closed from %s (active: %d/%d)", conn.RemoteAddr(), count, maxConnections)
	}()

	// 接続全体のタイムアウト設定
	conn.SetDeadline(time.Now().Add(connTimeout))

	// Send banner
	fmt.Fprint(conn, banner)

	// List contacts
	for _, c := range contacts {
		fmt.Fprintf(conn, "  [%d] %-10s → %s\n", c.Num, c.Name, c.URL)
	}

	fmt.Fprint(conn, "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprint(conn, "> Select [1-5] or 'q' to quit: ")

	scanner := bufio.NewScanner(conn)
	// 1行あたりの最大サイズを制限
	scanner.Buffer(make([]byte, maxInputSize), maxInputSize)

	for {
		// 読み込みごとにタイムアウトをリセット
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				if err == bufio.ErrTooLong {
					log.Printf("Input size limit exceeded from %s", conn.RemoteAddr())
					fmt.Fprint(conn, "\nInput too large. Connection closed.\n")
				}
			}
			return
		}

		input := scanner.Text()

		if input == "q" || input == "quit" || input == "exit" {
			fmt.Fprint(conn, "\nConnection closed. See you! 👋\n")
			return
		}

		var num int
		if _, err := fmt.Sscanf(input, "%d", &num); err == nil {
			if num >= 1 && num <= len(contacts) {
				c := contacts[num-1]
				fmt.Fprintf(conn, "\n→ Opening %s: https://%s\n\n", c.Name, c.URL)
				fmt.Fprint(conn, "> Select [1-5] or 'q' to quit: ")
				continue
			}
		}

		fmt.Fprint(conn, "Invalid input. Select [1-5] or 'q' to quit: ")
	}
}
