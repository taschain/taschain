package tasdb

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
