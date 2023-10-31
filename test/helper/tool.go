package helper

import (
	"encoding/hex"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func RandomString() string {
	rand.Seed(time.Now().UnixNano())
	//随机生成字符数组
	b := make([]byte, 6)
	//整合
	rand.Read(b)
	//转换为string
	str := hex.EncodeToString(b)
	return str
}

// DefinedRetry is the recommended retry for a conflict where multiple clients
// are making changes to the same resource.
var DefinedRetry = wait.Backoff{
	Steps: 5,
	// clustersynchro-manager 同步资源信息时延比较大，对于并发update 时，重试间隔时间需调大到1秒
	Duration: 1 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}
