// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package gce_test

import (
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/instance"
	"github.com/juju/juju/provider/common"
	"github.com/juju/juju/provider/gce"
	"github.com/juju/juju/provider/gce/google"
	"github.com/juju/juju/storage"
)

type environAZSuite struct {
	gce.BaseSuite
}

var _ = gc.Suite(&environAZSuite{})

func (s *environAZSuite) TestAvailabilityZones(c *gc.C) {
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("a-zone", google.StatusUp, "", ""),
		google.NewZone("b-zone", google.StatusUp, "", ""),
	}

	zones, err := s.Env.AvailabilityZones()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(zones, gc.HasLen, 2)
	c.Check(zones[0].Name(), gc.Equals, "a-zone")
	c.Check(zones[0].Available(), jc.IsTrue)
	c.Check(zones[1].Name(), gc.Equals, "b-zone")
	c.Check(zones[1].Available(), jc.IsTrue)
}

func (s *environAZSuite) TestAvailabilityZonesDeprecated(c *gc.C) {
	zone := google.NewZone("a-zone", google.StatusUp, "DEPRECATED", "b-zone")

	c.Check(zone.Deprecated(), jc.IsTrue)
}

func (s *environAZSuite) TestAvailabilityZonesAPI(c *gc.C) {
	s.FakeConn.Zones = []google.AvailabilityZone{}

	_, err := s.Env.AvailabilityZones()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(s.FakeConn.Calls, gc.HasLen, 1)
	c.Check(s.FakeConn.Calls[0].FuncName, gc.Equals, "AvailabilityZones")
	c.Check(s.FakeConn.Calls[0].Region, gc.Equals, "us-east1")
}

func (s *environAZSuite) TestInstanceAvailabilityZoneNames(c *gc.C) {
	s.FakeEnviron.Insts = []instance.Instance{s.Instance}

	ids := []instance.Id{instance.Id("spam")}
	zones, err := s.Env.InstanceAvailabilityZoneNames(ids)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(zones, jc.DeepEquals, []string{"home-zone"})
}

func (s *environAZSuite) TestInstanceAvailabilityZoneNamesAPIs(c *gc.C) {
	s.FakeEnviron.Insts = []instance.Instance{s.Instance}

	ids := []instance.Id{instance.Id("spam")}
	_, err := s.Env.InstanceAvailabilityZoneNames(ids)
	c.Assert(err, jc.ErrorIsNil)

	s.FakeEnviron.CheckCalls(c, []gce.FakeCall{{
		FuncName: "GetInstances", Args: gce.FakeCallArgs{"switch": s.Env},
	}})
}

func (s *environAZSuite) TestStartInstanceAvailabilityZones(c *gc.C) {
	s.FakeCommon.AZInstances = []common.AvailabilityZoneInstances{
		{ZoneName: "az2",
			Instances: []instance.Id{s.Instance.Id()}},
		{ZoneName: "az3",
			Instances: []instance.Id{},
		}}

	zones, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(zones, jc.DeepEquals, []string{"az2", "az3"})
}

func (s *environAZSuite) TestStartInstanceAvailabilityZoneParam(c *gc.C) {
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("az1", google.StatusDown, "", ""),
		google.NewZone("az2", google.StatusUp, "", ""),
		google.NewZone("az3", google.StatusUp, "", ""),
	}
	s.StartInstArgs.AvailabilityZone = "az3"

	zones, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)

	// Prior to the introduction of StartInstanceParams.AvailabilityZone
	// "az2" would have been chosen as the availability zone by the provider.
	// Ensure that the new value is taking precedence.
	c.Check(zones, jc.DeepEquals, []string{"az3"})
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesPlacement(c *gc.C) {
	s.StartInstArgs.Placement = "zone=a-zone"
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("a-zone", google.StatusUp, "", ""),
	}

	zones, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(zones, jc.DeepEquals, []string{"a-zone"})
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesPlacementAPI(c *gc.C) {
	s.StartInstArgs.Placement = "zone=a-zone"
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("a-zone", google.StatusUp, "", ""),
	}

	_, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)

	s.FakeEnviron.CheckCalls(c, []gce.FakeCall{})
	s.FakeCommon.CheckCalls(c, []gce.FakeCall{})
	c.Check(s.FakeConn.Calls, gc.HasLen, 1)
	c.Check(s.FakeConn.Calls[0].FuncName, gc.Equals, "AvailabilityZones")
	c.Check(s.FakeConn.Calls[0].Region, gc.Equals, "us-east1")
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesPlacementUnavailable(c *gc.C) {
	s.StartInstArgs.Placement = "zone=a-zone"
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("a-zone", google.StatusDown, "", ""),
	}

	_, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)

	c.Check(err, gc.ErrorMatches, `.*availability zone "a-zone" is DOWN`)
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesNoneFound(c *gc.C) {
	_, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesVolumeAttachments(c *gc.C) {
	s.StartInstArgs.VolumeAttachments = []storage.VolumeAttachmentParams{{
		VolumeId: "home-zone--c930380d-8337-4bf5-b07a-9dbb5ae771e4",
	}}

	zones, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zones, jc.DeepEquals, []string{"home-zone"})
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesVolumeAttachmentsDifferentZones(c *gc.C) {
	s.StartInstArgs.VolumeAttachments = []storage.VolumeAttachmentParams{{
		VolumeId: "home-zone--c930380d-8337-4bf5-b07a-9dbb5ae771e4",
	}, {
		VolumeId: "away-zone--c930380d-8337-4bf5-b07a-9dbb5ae771e4",
	}}

	_, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, gc.ErrorMatches, `cannot attach volumes from multiple availability zones: home-zone--c930380d-8337-4bf5-b07a-9dbb5ae771e4 is in home-zone, away-zone--c930380d-8337-4bf5-b07a-9dbb5ae771e4 is in away-zone`)
}

func (s *environAZSuite) TestStartInstanceAvailabilityZonesVolumeAttachmentsConflictsPlacement(c *gc.C) {
	s.StartInstArgs.Placement = "zone=away-zone"
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("away-zone", google.StatusUp, "", ""),
	}
	s.StartInstArgs.VolumeAttachments = []storage.VolumeAttachmentParams{{
		VolumeId: "home-zone--c930380d-8337-4bf5-b07a-9dbb5ae771e4",
	}}

	_, err := gce.StartInstanceAvailabilityZones(s.Env, s.StartInstArgs)
	c.Assert(err, gc.ErrorMatches, `cannot create instance with placement "zone=away-zone", as this will prevent attaching the requested disks in zone "home-zone"`)
}

func (s *environAZSuite) TestDeriveAvailabilityZone(c *gc.C) {
	s.StartInstArgs.Placement = "zone=test-available"
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("test-available", google.StatusUp, "", ""),
	}
	zone, err := s.Env.DeriveAvailabilityZone(s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zone, gc.Equals, "test-available")
}

func (s *environAZSuite) TestDeriveAvailabilityZoneVolumeNoPlacement(c *gc.C) {
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("az1", google.StatusDown, "", ""),
		google.NewZone("az2", google.StatusUp, "", ""),
	}
	s.StartInstArgs.VolumeAttachments = []storage.VolumeAttachmentParams{{
		VolumeId: "az2--c930380d-8337-4bf5-b07a-9dbb5ae771e4",
	}}
	zone, err := s.Env.DeriveAvailabilityZone(s.StartInstArgs)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zone, gc.Equals, "az2")
}

func (s *environAZSuite) TestDeriveAvailabilityZoneUnavailable(c *gc.C) {
	s.StartInstArgs.Placement = "zone=test-unavailable"
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("test-unavailable", google.StatusDown, "", ""),
	}
	zone, err := s.Env.DeriveAvailabilityZone(s.StartInstArgs)
	c.Check(err, gc.ErrorMatches, `.*availability zone "test-unavailable" is DOWN`)
	c.Assert(zone, gc.Equals, "")
}

func (s *environAZSuite) TestDeriveAvailabilityZoneUnknown(c *gc.C) {
	s.StartInstArgs.Placement = "zone=test-unknown"
	zone, err := s.Env.DeriveAvailabilityZone(s.StartInstArgs)
	c.Assert(err, gc.ErrorMatches, `invalid availability zone "test-unknown" not found`)
	c.Assert(zone, gc.Equals, "")
}

func (s *environAZSuite) TestDeriveAvailabilityZoneConflictsVolume(c *gc.C) {
	s.FakeConn.Zones = []google.AvailabilityZone{
		google.NewZone("az1", google.StatusUp, "", ""),
		google.NewZone("az2", google.StatusUp, "", ""),
	}
	s.StartInstArgs.Placement = "zone=az1"
	s.StartInstArgs.VolumeAttachments = []storage.VolumeAttachmentParams{{
		VolumeId: "az2--c930380d-8337-4bf5-b07a-9dbb5ae771e4",
	}}
	zone, err := s.Env.DeriveAvailabilityZone(s.StartInstArgs)
	c.Assert(err, gc.ErrorMatches, `cannot create instance with placement "zone=az1", as this will prevent attaching the requested disks in zone "az2"`)
	c.Assert(zone, gc.Equals, "")
}
