package main

import (
	"crypto/md5"
	"encoding/hex"
)

func CalculateMD5(content []byte) string {
	// 创建MD5哈希对象
	hash := md5.New()

	// 计算MD5哈希值
	hash.Write(content)
	hashBytes := hash.Sum(nil)
	md5String := hex.EncodeToString(hashBytes)

	return md5String
}
