package tools

// modify the code
/*
	modifier
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
