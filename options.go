package xnetwork

import (
	"github.com/totem-project/xtls"
)

var NetOptions *ClientOptions

type ClientOptions struct {
	FailRetries int                 `mapstructure:"fail_retries" json:"fail_retries" yaml:"fail_retries" #:"请求失败的重试次数，0 则不重试"`
	ReadSize    int                 `mapstructure:"read_size" json:"read_size" yaml:"read_size" #:"响应读取长度, 默认 2048"`
	DialTimeout int                 `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout" #:"建立 tcp 连接的超时时间"`
	KeepAlive   int                 `mapstructure:"keep_alive" json:"keep_alive" yaml:"keep_alive" #:"tcp keep_alive 时间"`
	ReadTimeout int                 `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout" #:"响应读取超时时间"`
	TlsOptions  *xtls.ClientOptions `mapstructure:"tls_options" json:"tls_options" yaml:"tls_options" #:"tls 配置"`
	Debug       bool                `mapstructure:"net_debug" json:"net_debug" yaml:"net_debug" #:"是否启用 debug 模式, 开启 request trace"`
}

func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Debug:       false,
		FailRetries: 0, // 默认改为0，否则如果配置文件指定了0，会不生效。 "nil value" 的问题
		ReadSize:    2048,
		DialTimeout: 10,
		KeepAlive:   10,
		ReadTimeout: 30,
		TlsOptions:  xtls.DefaultClientOptions(),
	}
}

func GetNetOptions() *ClientOptions {
	if NetOptions != nil {
		return NetOptions
	} else {
		return DefaultClientOptions()
	}
}
