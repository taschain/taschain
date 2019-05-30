//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
