package param

/*
**  Creator: pxf
**  Date: 2018/4/10 下午6:25
**  Description: 
*/

type ParamStore struct {
	Current   *ParamMeta
	Futures   []*ParamMeta
	Histories []*ParamMeta
}

