// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package rpc

import (
	"context"
	"net/url"
	"os"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&RPCSuite{})

const contextKeyTestTokens = "testTokens"

type RPCSuite struct {
	log  logrus.FieldLogger
	ctx  context.Context
	conn *Conn
}

func (s *RPCSuite) SetUpTest(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	s.ctx = context.WithValue(ctx, contextKeyTestTokens, []string{arvadostest.ActiveToken})
	s.conn = NewConn("zzzzz", &url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_TEST_API_HOST")}, true, func(ctx context.Context) ([]string, error) {
		return ctx.Value(contextKeyTestTokens).([]string), nil
	})
}

func (s *RPCSuite) TestCollectionCreate(c *check.C) {
	coll, err := s.conn.CollectionCreate(s.ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
		"owner_uuid":         arvadostest.ActiveUserUUID,
		"portable_data_hash": "d41d8cd98f00b204e9800998ecf8427e+0",
	}})
	c.Check(err, check.IsNil)
	c.Check(coll.UUID, check.HasLen, 27)
}

func (s *RPCSuite) TestSpecimenCRUD(c *check.C) {
	sp, err := s.conn.SpecimenCreate(s.ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
		"owner_uuid": arvadostest.ActiveUserUUID,
		"properties": map[string]string{"foo": "bar"},
	}})
	c.Check(err, check.IsNil)
	c.Check(sp.UUID, check.HasLen, 27)
	c.Check(sp.Properties, check.HasLen, 1)
	c.Check(sp.Properties["foo"], check.Equals, "bar")

	spGet, err := s.conn.SpecimenGet(s.ctx, arvados.GetOptions{UUID: sp.UUID})
	c.Check(spGet.UUID, check.Equals, sp.UUID)
	c.Check(spGet.Properties["foo"], check.Equals, "bar")

	spList, err := s.conn.SpecimenList(s.ctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", sp.UUID}}})
	c.Check(spList.ItemsAvailable, check.Equals, 1)
	c.Assert(spList.Items, check.HasLen, 1)
	c.Check(spList.Items[0].UUID, check.Equals, sp.UUID)
	c.Check(spList.Items[0].Properties["foo"], check.Equals, "bar")

	anonCtx := context.WithValue(context.Background(), contextKeyTestTokens, []string{arvadostest.AnonymousToken})
	spList, err = s.conn.SpecimenList(anonCtx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", sp.UUID}}})
	c.Check(spList.ItemsAvailable, check.Equals, 0)
	c.Check(spList.Items, check.HasLen, 0)

	spDel, err := s.conn.SpecimenDelete(s.ctx, arvados.DeleteOptions{UUID: sp.UUID})
	c.Check(spDel.UUID, check.Equals, sp.UUID)
}
