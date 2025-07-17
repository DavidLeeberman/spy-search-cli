package tools

// modify the code
/*
	modifier is a modify tool that modify files
*/

type ModifyTool struct {
	Tool
}

func NewModifierTool() ModifyTool {
	return ModifyTool{
		Tool{},
	}
}

type modifyArgs struct{}
