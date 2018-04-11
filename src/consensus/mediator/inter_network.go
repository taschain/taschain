package mediator

import (
	"consensus/groupsig"
)

///////////////////////////////////////////////////////////////////////////////
type PostGroupMemberI func(uid groupsig.ID, msg interface{}) int

type PostGroupI func(gid groupsig.ID, msg interface{}) int

type PostGlobalI func(msg interface{}) int
