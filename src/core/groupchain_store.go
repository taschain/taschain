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

package core

import (
	"middleware/types"
	"utility"
	"github.com/vmihailenco/msgpack"
)


func (chain *GroupChain) getGroupByHeight(height uint64) *types.Group {
	groupId, _ := chain.groupsHeight.Get(utility.UInt64ToByte(height))
	if nil != groupId {
		return chain.getGroupById(groupId)
	}

	return nil
}

func (chain *GroupChain) hasGroup(id []byte) bool {
    ok, _ := chain.groups.Has(id)
    return ok
}

func (chain *GroupChain) getGroupById(id []byte) *types.Group {
	data, _ := chain.groups.Get(id)
	if nil == data || 0 == len(data) {
		return nil
	}

	var group types.Group
	err := msgpack.Unmarshal(data, &group)
	if err != nil {
		return nil
	}
	return &group
}

func (chain *GroupChain) getGroupsAfterHeight(height uint64, limit int) ([]*types.Group) {
	result := make([]*types.Group, 0)
	iter := chain.groupsHeight.NewIterator()
	defer iter.Release()

	if !iter.Seek(utility.UInt64ToByte(height)) {
		return result
	}

	for limit > 0 {
		gid := iter.Value()
		g := chain.getGroupById(gid)
		if g != nil {
			result = append(result, g)
			limit--
		}
		if !iter.Next() {
			break
		}
	}
	return result
}

func (chain *GroupChain) loadLastGroup() *types.Group {
    gid, err := chain.groups.Get([]byte(GROUP_STATUS_KEY))
	if err != nil {
		return nil
	}
	return chain.getGroupById(gid)
}

func (chain *GroupChain) commitGroup(group *types.Group) error {
	data, err := msgpack.Marshal(group)
	if nil != err {
		return err
	}

	batch := chain.groups.CreateLDBBatch()
	defer batch.Reset()

	if err := chain.groups.AddKv(batch, group.Id, data); err != nil {
		return err
	}
	if err := chain.groupsHeight.AddKv(batch, utility.UInt64ToByte(group.GroupHeight), group.Id); err != nil {
		return err
	}
	if err := chain.groups.AddKv(batch, []byte(GROUP_STATUS_KEY), group.Id); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		return err
	}

	chain.lastGroup = group

	return nil
}
