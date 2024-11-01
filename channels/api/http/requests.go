// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/internal/api"
	"github.com/absmach/supermq/pkg/apiutil"
	mgclients "github.com/absmach/supermq/pkg/clients"
)

type createChannelReq struct {
	Channel channels.Channel
}

func (req createChannelReq) validate() error {
	if len(req.Channel.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.Channel.ID != "" {
		return api.ValidateUUID(req.Channel.ID)
	}

	return nil
}

type createChannelsReq struct {
	Channels []channels.Channel
}

func (req createChannelsReq) validate() error {
	if len(req.Channels) == 0 {
		return apiutil.ErrEmptyList
	}
	for _, channel := range req.Channels {
		if channel.ID != "" {
			if err := api.ValidateUUID(channel.ID); err != nil {
				return err
			}
		}
		if len(channel.Name) > api.MaxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type viewChannelReq struct {
	id string
}

func (req viewChannelReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listChannelsReq struct {
	status     mgclients.Status
	offset     uint64
	limit      uint64
	name       string
	tag        string
	permission string
	visibility string
	userID     string
	listPerms  bool
	metadata   mgclients.Metadata
	id         string
}

func (req listChannelsReq) validate() error {
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	if req.visibility != "" &&
		req.visibility != api.AllVisibility &&
		req.visibility != api.MyVisibility &&
		req.visibility != api.SharedVisibility {
		return apiutil.ErrInvalidVisibilityType
	}
	if len(req.name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateChannelReq struct {
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateChannelTagsReq struct {
	id   string
	Tags []string `json:"tags,omitempty"`
}

func (req updateChannelTagsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type setChannelParentGroupReq struct {
	id            string
	ParentGroupID string `json:"parent_group_id"`
}

func (req setChannelParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.ParentGroupID == "" {
		return apiutil.ErrMissingParentGroupID
	}

	return nil
}

type removeChannelParentGroupReq struct {
	id string
}

func (req removeChannelParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeChannelStatusReq struct {
	id string
}

func (req changeChannelStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type connectChannelThingsRequest struct {
	channelID string
	ThingIds  []string `json:"thing_ids,omitempty"`
}

func (req *connectChannelThingsRequest) validate() error {
	if req.channelID == "" {
		return apiutil.ErrMissingID
	}

	if err := api.ValidateUUID(req.channelID); err != nil {
		return err
	}

	if len(req.ThingIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ThingIds {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}
	return nil
}

type disconnectChannelThingsRequest struct {
	channelID string
	ThingIds  []string `json:"thing_ids,omitempty"`
}

func (req *disconnectChannelThingsRequest) validate() error {
	if req.channelID == "" {
		return apiutil.ErrMissingID
	}

	if err := api.ValidateUUID(req.channelID); err != nil {
		return err
	}

	if len(req.ThingIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ThingIds {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}
	return nil
}

type connectRequest struct {
	ChannelIds []string `json:"channel_ids,omitempty"`
	ThingIds   []string `json:"thing_ids,omitempty"`
}

func (req *connectRequest) validate() error {
	if len(req.ChannelIds) == 0 {
		return apiutil.ErrMissingID
	}
	for _, cid := range req.ChannelIds {
		if err := api.ValidateUUID(cid); err != nil {
			return err
		}
	}

	if len(req.ThingIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ThingIds {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}
	return nil
}

type disconnectRequest struct {
	ChannelIds []string `json:"channel_ids,omitempty"`
	ThingIds   []string `json:"thing_ids,omitempty"`
}

func (req *disconnectRequest) validate() error {
	if len(req.ChannelIds) == 0 {
		return apiutil.ErrMissingID
	}
	for _, cid := range req.ChannelIds {
		if err := api.ValidateUUID(cid); err != nil {
			return err
		}
	}

	if len(req.ThingIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ThingIds {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}
	return nil
}

type deleteChannelReq struct {
	id string
}

func (req deleteChannelReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
