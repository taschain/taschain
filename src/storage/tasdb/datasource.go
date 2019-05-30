package tasdb

import "github.com/syndtr/goleveldb/leveldb/opt"

/*
**  Creator: pxf
**  Date: 2019/3/18 上午10:21
**  Description: 
*/

type TasDataSource struct {
	db *LDBDatabase
}

func NewDataSource(file string, options *opt.Options) (*TasDataSource, error) {
	db, err := getInstance(file, options)
	if err != nil {
		return nil, err
	}
	return &TasDataSource{db:db}, nil
}

func (ds *TasDataSource) NewPrefixDatabase(prefix string) (*PrefixedDatabase, error) {
	return &PrefixedDatabase{
		db:     ds.db,
		prefix: prefix,
	}, nil
}

