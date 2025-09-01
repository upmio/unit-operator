package project

import "testing"

func TestGenerateAES256Key(t *testing.T) {
	key, err := generateAES256Key()
	if err != nil {
		t.Fatalf("generateAES256Key 返回错误: %v", err)
	}
	
	if len(key) != 32 {

		t.Errorf("生成的 key 长度不是 32，实际为: %d", len(key))
	}
	// 检查是否为有效的 32 字节十六进制字符串
	if !isValidHex32(key) {
		t.Errorf("生成的 key 不是有效的 32 字节十六进制字符串: %s", string(key))
	}
}
