package main

import (
	"math/rand"
	"time"

	"github.com/xiaonanln/goworld"
	"github.com/xiaonanln/goworld/common"
	"github.com/xiaonanln/goworld/entity"
	"github.com/xiaonanln/goworld/gwlog"
	"github.com/xiaonanln/goworld/kvdb"
)

type Account struct {
	entity.Entity
	username string
	logining bool
}

func (a *Account) OnInit() {

}

func (a *Account) OnCreated() {
	//gwlog.Info("%s created: client=%v", a, a.GetClient())
}

func (a *Account) getAvatarID(username string, callback func(entityID common.EntityID, err error)) {
	kvdb.Get(username, func(val string, err error) {
		if a.IsDestroyed() {
			return
		}
		callback(common.EntityID(val), err)
	})
}

func (a Account) setAvatarID(username string, avatarID common.EntityID) {
	kvdb.Put(username, string(avatarID), nil)
}

func (a *Account) Login_Client(username string, password string) {
	if a.logining {
		// logining
		gwlog.Error("%s is already logining", a)
		return
	}

	gwlog.Info("%s logining with username %s password %s ...", a, username, password)
	if password != "123456" {
		a.CallClient("OnLogin", false)
		return
	}

	a.logining = true
	a.CallClient("OnLogin", true)
	a.getAvatarID(username, func(avatarID common.EntityID, err error) {
		if err != nil {
			gwlog.Panic(err)
		}

		gwlog.Debug("Username %s get avatar id = %s", username, avatarID)
		if avatarID.IsNil() {
			// avatar not found, create new avatar
			avatarID = goworld.CreateEntityLocally("Avatar")
			a.setAvatarID(username, avatarID)

			avatar := goworld.GetEntity(avatarID)
			a.onAvatarEntityFound(avatar)
		} else {
			goworld.LoadEntityAnywhere("Avatar", avatarID)
			a.Call(avatarID, "GetSpaceID", a.ID) // request for avatar space ID
		}
	})
}

func (a *Account) OnGetAvatarSpaceID_Server(avatarID common.EntityID, spaceID common.EntityID) {
	a.Attrs.Set("loginAvatarID", avatarID)
	a.EnterSpace(spaceID, entity.Position{}) // TODO: not use 0, 0, 0
}

func (a *Account) onAvatarEntityFound(avatar *entity.Entity) {
	a.GiveClientTo(avatar)
}

func (a *Account) OnClientDisconnected() {
	a.Destroy()
}

func (a *Account) OnMigrateIn() {
	loginAvatarID := common.EntityID(a.Attrs.GetStr("loginAvatarID"))
	avatar := goworld.GetEntity(loginAvatarID)
	gwlog.Debug("%s migrating in, attrs=%v, loginAvatarID=%s, avatar=%v, client=%s", a, a.Attrs.ToMap(), loginAvatarID, avatar, a.GetClient())

	if avatar != nil {
		a.onAvatarEntityFound(avatar)
	} else {
		// failed ? try again
		a.AddCallback(time.Millisecond*time.Duration(rand.Intn(3000)), func() {
			goworld.LoadEntityAnywhere("Avatar", loginAvatarID)
			a.Call(loginAvatarID, "GetSpaceID", a.ID) // request for avatar space ID
		})
	}
}

func (a *Account) OnMigrateOut() {
	gwlog.Debug("%s migrating out ...", a)
}