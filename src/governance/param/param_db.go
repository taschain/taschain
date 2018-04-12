package param

import "errors"

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



type ParamDB struct {
	//TODO: db accessor
}

//TODO: load from real db
func (db *ParamDB) load() ([]ParamStore, error) {
    return []ParamStore{}, errors.New("no interface")
}
//TODO: store real db
func (db *ParamDB) store(s *ParamStore) error {
	return nil
}

func (db *ParamDB) LoadParams() *ParamDefs {
    stores, err := db.load()
	if err != nil {
		//TODO: log err
		return nil
	}
    defs := newParamDefs()

	for idx, param := range stores {
		def := &ParamDef{
			Current: param.Current,
			Futures: param.Futures,
			Histories: param.Histories,
			Validate: getValidateFunc(getDefaultValueDefs(idx)),
			update: make(chan int),
		}
		defs.AddParam(def)
	}
	return defs
}

func (db *ParamDB) StoreParam(def *ParamDef) error {
    ps := &ParamStore{
    	Current: def.Current,
    	Histories: def.Histories,
    	Futures: def.Futures,
	}

	return db.store(ps)
}

func (db *ParamDB) StoreAll(defs *ParamDefs) error {
	for _, def := range defs.defs {
		if err := db.StoreParam(def); err != nil {
			return err
		}
	}
	return nil
}