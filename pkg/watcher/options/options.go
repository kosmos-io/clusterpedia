package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

type MiddlerwareOptions struct {
	Name                        string
	ServerIp                    string
	ServerPort                  int
	ConnectUser                 string // rabbitmq连接用户名
	ConnectPassword             string // rabbitmq连接密码
	MaxConnections              int    // rabbitmq tcp长连接数量（默认3）
	Suffix                      string // rabbitmq url 后缀（如果有的话配上）
	ExpiresPerSend              int    // 单次发送超时时间（秒，默认3）
	QueueExpires                int64  // 当Queue(队列)在指定的时间未被访问，则队列将被自动删除。
	BindingControllerConfigPath string
	CacheSize                   int
}

func NewMiddlerwareOptions() *MiddlerwareOptions {
	return &MiddlerwareOptions{Name: "apiserver", CacheSize: 100}
}

func (o *MiddlerwareOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errors []error
	if o.ServerPort == 0 {
		errors = append(errors, fmt.Errorf("ServerPort is %d", o.ServerPort))
	}

	if o.CacheSize == 0 {
		o.CacheSize = 100
	}

	if o.ConnectPassword == "" {
		errors = append(errors, fmt.Errorf("Server PassWord is null"))
	}

	return errors
}

func (o *MiddlerwareOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "middleware-name", o.Name, "middlerware name")
	fs.StringVar(&o.ServerIp, "middleware-serverIp", o.ServerIp, "middlerware server Ip")
	fs.IntVar(&o.ServerPort, "middleware-serverPort", o.ServerPort, "middlerware server port")
	fs.StringVar(&o.BindingControllerConfigPath, "binding-controller-config-path", o.BindingControllerConfigPath, ""+
		"binding controller config path.")
	fs.IntVar(&o.CacheSize, "cache-size", o.CacheSize, "middlerware cache size")
	fs.StringVar(&o.ConnectUser, "middleware-user", o.ConnectUser, "middlerware connect user")
	fs.StringVar(&o.ConnectPassword, "middleware-password", o.ConnectPassword, "middlerware connect password")
	fs.IntVar(&o.ExpiresPerSend, "middleware-send-expires", o.ExpiresPerSend, "middlerware expires send")
	fs.Int64Var(&o.QueueExpires, "middleware-queue-expires", o.QueueExpires, "middlereare queue expires")
}
