package tasdb

type TasDataSource struct {
	db *LDBDatabase
}

// NewDataSource create levedb instance by file
func NewDataSource(file string) (*TasDataSource, error) {
	db, err := getInstance(file)
	if err != nil {
		return nil, err
	}
	return &TasDataSource{db: db}, nil
}

// NewPrefixDatabase create logical database by prefix
func (ds *TasDataSource) NewPrefixDatabase(prefix string) (*PrefixedDatabase, error) {
	return &PrefixedDatabase{
		db:     ds.db,
		prefix: prefix,
	}, nil
}
