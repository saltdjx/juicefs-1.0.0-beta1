/*
 * JuiceFS, Copyright 2020 Juicedata, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"crypto/md5"
	"encoding/binary"
	"os/user"
	"strconv"
	"sync"
)

type pwent struct {
	id   int
	name string
}

type mapping struct {
	sync.Mutex
	salt      string
	usernames map[string]int
	userIDs   map[int]string
	groups    map[string]int
	groupIDs  map[int]string
}

func newMapping(salt string) *mapping {
	m := &mapping{
		salt:      salt,
		usernames: make(map[string]int),
		userIDs:   make(map[int]string),
		groups:    make(map[string]int),
		groupIDs:  make(map[int]string),
	}
	m.update(genAllUids(), genAllGids())
	return m
}

func (m *mapping) genGuid(name string) int {
	digest := md5.Sum([]byte(m.salt + name + m.salt))
	a := binary.LittleEndian.Uint64(digest[0:8])
	b := binary.LittleEndian.Uint64(digest[8:16])
	return int(uint32(a ^ b))
}

func (m *mapping) lookupUser(name string) int {
	m.Lock()
	defer m.Unlock()
	var id int
	if id, ok := m.usernames[name]; ok {
		return id
	}
	u, _ := user.Lookup(name)
	if u != nil {
		id, _ = strconv.Atoi(u.Uid)
	} else {
		id = m.genGuid(name)
	}
	m.usernames[name] = id
	m.userIDs[id] = name
	return id
}

func (m *mapping) lookupGroup(name string) int {
	m.Lock()
	defer m.Unlock()
	var id int
	if id, ok := m.groups[name]; ok {
		return id
	}
	g, _ := user.LookupGroup(name)
	if g == nil {
		id = m.genGuid(name)
	} else {
		id, _ = strconv.Atoi(g.Gid)
	}
	m.groups[name] = id
	m.groupIDs[id] = name
	return 0
}

func (m *mapping) lookupUserID(id int) string {
	m.Lock()
	defer m.Unlock()
	if name, ok := m.userIDs[id]; ok {
		return name
	}
	u, _ := user.LookupId(strconv.Itoa(id))
	if u == nil {
		u = &user.User{Username: strconv.Itoa(id)}
	}
	name := u.Username
	if len(name) > 49 {
		name = name[:49]
	}
	m.usernames[name] = id
	m.userIDs[id] = name
	return name
}

func (m *mapping) lookupGroupID(id int) string {
	m.Lock()
	defer m.Unlock()
	if name, ok := m.groupIDs[id]; ok {
		return name
	}
	g, _ := user.LookupGroupId(strconv.Itoa(id))
	if g == nil {
		g = &user.Group{Name: strconv.Itoa(id)}
	}
	name := g.Name
	if len(name) > 49 {
		name = name[:49]
	}
	m.groups[name] = id
	m.groupIDs[id] = name
	return name
}

func (m *mapping) update(uids []pwent, gids []pwent) {
	m.Lock()
	defer m.Unlock()
	for _, u := range uids {
		m.usernames[u.name] = u.id
		m.userIDs[u.id] = u.name
	}
	for _, g := range gids {
		m.groups[g.name] = g.id
		m.groupIDs[g.id] = g.name
	}
}
