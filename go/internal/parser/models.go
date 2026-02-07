package parser

type LineInfo struct {
	Num         int
	Str         string
	Type        string
	Param       []string
	Tasks       []string
	DefaultFlag bool
}

type Block struct {
	Tasks        []string
	TaskId       int
	CmdBlock     string
	Comment      string
	StartLineNum int
	WithDiv      bool
	Interpreter  string
}

type MkInfo struct {
	File         string
	BlockList    []*Block
	InitBlock    *Block
	DefaultTasks []string
	Err          int
	ErrMsg       string
}

func NewMkInfo(file string) *MkInfo {
	mk := &MkInfo{
		File:         file,
		BlockList:    make([]*Block, 0),
		DefaultTasks: make([]string, 0),
	}
	mk.ParseFile()
	return mk
}
