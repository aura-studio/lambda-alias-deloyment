// Package exitcode defines exit codes for the lad command line tool.
package exitcode

const (
	Success          = 0 // 成功
	ParamError       = 1 // 参数错误
	AWSError         = 2 // AWS 错误
	ResourceNotFound = 3 // 资源不存在
	NetworkError     = 4 // 网络错误
)
