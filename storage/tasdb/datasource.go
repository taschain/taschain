package tasdb

/*
**  Creator: pxf
**  Date: 2019/3/18 上午10:21
**  Description:
 */

type TasDataSource struct {
	db *LDBDatabase
}

func NewDataSource(file string) (*TasDataSource, error) {
	db, err := getInstance(file)
	if err != nil {
		return nil, err
	}
	return &TasDataSource{db: db}, nil
}

func (ds *TasDataSource) NewPrefixDatabase(prefix string) (*PrefixedDatabase, error) {
	return &PrefixedDatabase{
		db:     ds.db,
		prefix: prefix,
	}, nil
}
