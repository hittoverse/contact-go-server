#!/bin/bash
# セキュリティ対策テストスクリプト
# Usage: ./test_security.sh [host] [port]

HOST="${1:-contact.hitto.me}"
PORT="${2:-1337}"

echo "=========================================="
echo "セキュリティ対策テスト"
echo "対象: $HOST:$PORT"
echo "=========================================="
echo ""

# 色の定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# テスト1: 正常接続
echo -e "${YELLOW}[テスト1] 正常接続テスト${NC}"
result=$(echo "q" | nc -w 3 "$HOST" "$PORT" 2>&1)
if echo "$result" | grep -q "Welcome"; then
    echo -e "${GREEN}✓ 正常に接続できました${NC}"
else
    echo -e "${RED}✗ 接続に失敗しました${NC}"
    echo "サーバーが起動していないか、接続できません"
    exit 1
fi
echo ""

# テスト2: IP単位レート制限テスト (5接続/10秒)
echo -e "${YELLOW}[テスト2] IP単位レート制限テスト (5接続/10秒)${NC}"
echo "10秒以内に7回接続を試みます..."

success_count=0
rejected_count=0

for i in {1..7}; do
    result=$(echo "q" | nc -w 2 "$HOST" "$PORT" 2>&1)

    if echo "$result" | grep -q "Too many connections"; then
        echo -e "  接続 $i: ${RED}拒否 (レート制限) ✓${NC}"
        ((rejected_count++))
    elif echo "$result" | grep -q "Server is busy"; then
        echo -e "  接続 $i: ${YELLOW}拒否 (同時接続数上限)${NC}"
        ((rejected_count++))
    elif echo "$result" | grep -q "Welcome"; then
        echo -e "  接続 $i: ${GREEN}成功${NC}"
        ((success_count++))
    else
        echo -e "  接続 $i: ${YELLOW}応答なし/タイムアウト${NC}"
    fi
done

echo ""
if [ $rejected_count -gt 0 ]; then
    echo -e "${GREEN}✓ レート制限が機能しています (成功: $success_count, 拒否: $rejected_count)${NC}"
else
    echo -e "${RED}✗ レート制限が機能していません (全て成功: $success_count)${NC}"
    echo "  → 新しいバージョンがデプロイされていない可能性があります"
fi
echo ""

# テスト3: 入力サイズ制限テスト (1KB)
echo -e "${YELLOW}[テスト3] 入力サイズ制限テスト (1KB超過)${NC}"
echo "11秒待機後、2KBの入力を送信..."
sleep 11  # レート制限リセットを待つ

large_input=$(python3 -c "print('A' * 2000)")
result=$(echo "$large_input" | nc -w 3 "$HOST" "$PORT" 2>&1)

if echo "$result" | grep -q "Input too large"; then
    echo -e "${GREEN}✓ 大量入力が正しく拒否されました${NC}"
elif echo "$result" | grep -q "Too many connections"; then
    echo -e "${YELLOW}△ レート制限により接続拒否（入力テスト不可）${NC}"
elif echo "$result" | grep -q "Welcome"; then
    echo -e "${RED}✗ 大量入力が拒否されませんでした${NC}"
    echo "  → 新しいバージョンがデプロイされていない可能性があります"
else
    echo -e "${YELLOW}? 不明な応答${NC}"
    echo "応答内容: $(echo "$result" | tail -3)"
fi
echo ""

# テスト4: タイムアウトテスト（オプション）
echo -e "${YELLOW}[テスト4] 読み込みタイムアウトテスト (30秒) [オプション]${NC}"
read -p "このテストは約35秒かかります。実行しますか? (y/N): " answer
if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
    echo "接続して35秒間入力を送信しません..."
    sleep 11  # レート制限リセットを待つ
    start_time=$(date +%s)
    (sleep 35) | nc -w 60 "$HOST" "$PORT" > /dev/null 2>&1
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))
    if [ $elapsed -lt 40 ]; then
        echo -e "${GREEN}✓ タイムアウト動作確認 (${elapsed}秒で切断)${NC}"
    else
        echo -e "${RED}✗ タイムアウトが機能していない可能性があります${NC}"
    fi
else
    echo "スキップしました"
fi
echo ""

echo "=========================================="
echo "テスト完了"
echo "=========================================="
