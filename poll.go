package nnet

type Poll interface {
	Wait() error
	Close() error
	Trigger() error
	Control(operator *FDOperator, event PollEvent) error

	Alloc() (operator *FDOperator)
	Free(operator *FDOperator)
}

type PollEvent int

const (
	PollReadable PollEvent = 0x1 // 新注册fd 可以读取了
	PollWritable PollEvent = 0x2 // 新注册fd 可以写入了
	PollDetach   PollEvent = 0x3 // fd 移除
	PollR2RW     PollEvent = 0x5 // 在原本可读fd上追加可写
	PollRW2R     PollEvent = 0x6 // 撤掉可写 回到只读
)
