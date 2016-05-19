package http

import (
	"encoding/json"
	"net/http"

	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/tecsisa/authorizr/api"
	"github.com/tecsisa/authorizr/authorizr"
)

// Requests

type CreateGroupRequest struct {
	Name string `json:"Name, omitempty"`
	Path string `json:"Path, omitempty"`
}

type UpdateGroupRequest struct {
	Name string `json:"Name, omitempty"`
	Path string `json:"Path, omitempty"`
}

// Responses

type CreateGroupResponse struct {
	Group *api.Group
}

type UpdateGroupResponse struct {
	Group *api.Group
}

type GetGroupNameResponse struct {
	Group *api.Group
}

type GetGroupsResponse struct {
	Groups []api.Group
}

type GetGroupMembersResponse struct {
	Members *api.GroupMembers
}

type GetGroupPolicies struct {
	Group    api.Group
	Policies []api.Policy
}

type GroupHandler struct {
	core *authorizr.Core
}

func (g *GroupHandler) handleCreateGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Decode request
	request := CreateGroupRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		g.core.Logger.Errorln(err)
		RespondBadRequest(w)
		return
	}

	org := ps.ByName(ORG_NAME)
	// Call group API to create an group
	result, err := g.core.GroupApi.AddGroup(org, request.Name, request.Path)

	// Error handling
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		switch apiError.Code {
		case api.GROUP_ALREADY_EXIST:
			RespondConflict(w)
		case api.INVALID_PARAMETER_ERROR:
			RespondBadRequest(w)
		default: // Unexpected API error
			RespondInternalServerError(w)
		}
		return
	}

	response := &CreateGroupResponse{
		Group: result,
	}

	// Write group to response
	RespondOk(w, response)
}

func (g *GroupHandler) handleDeleteGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group org and name from path
	org := ps.ByName(ORG_NAME)
	name := ps.ByName(GROUP_NAME)

	// Call user API to delete group
	err := g.core.GroupApi.RemoveGroup(org, name)

	// Check if there were errors
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		// If group doesn't exist
		if apiError.Code == api.GROUP_BY_ORG_AND_NAME_NOT_FOUND {
			RespondNotFound(w)
		} else { // Unexpected error
			RespondInternalServerError(w)
		}
	} else { // Respond without content
		RespondNoContent(w)
	}
}

func (g *GroupHandler) handleGetGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group org and name from path
	org := ps.ByName(ORG_NAME)
	name := ps.ByName(GROUP_NAME)

	// Call group API to retrieve group
	result, err := g.core.GroupApi.GetGroupByName(org, name)

	// Error handling
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		if apiError.Code == api.GROUP_BY_ORG_AND_NAME_NOT_FOUND {
			RespondNotFound(w)
			return
		} else { // Unexpected API error
			RespondInternalServerError(w)
			return
		}
	}

	response := GetGroupNameResponse{
		Group: result,
	}

	// Write group to response
	RespondOk(w, response)
}

func (g *GroupHandler) handleListGroups(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group org from path
	org := ps.ByName(ORG_NAME)

	// Retrieve query param if exist
	pathPrefix := r.URL.Query().Get("PathPrefix")

	// Call group API to retrieve groups
	result, err := g.core.GroupApi.GetListGroups(org, pathPrefix)
	if err != nil {
		g.core.Logger.Errorln(err)
		RespondInternalServerError(w)
		return
	}

	// Create response
	response := &GetGroupsResponse{
		Groups: result,
	}

	// Return data
	RespondOk(w, response)

}

func (g *GroupHandler) handleUpdateGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Decode request
	request := UpdateGroupRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		g.core.Logger.Errorln(err)
		RespondBadRequest(w)
		return
	}

	// Check parameters
	if len(strings.TrimSpace(request.Name)) == 0 ||
		len(strings.TrimSpace(request.Path)) == 0 {
		g.core.Logger.Errorf("There are mising parameters: Name %v, Path %v", request.Name, request.Path)
		RespondBadRequest(w)
		return
	}

	// Retrieve group, org from path
	org := ps.ByName(ORG_NAME)
	groupName := ps.ByName(GROUP_NAME)

	// Call group API to update group
	result, err := g.core.GroupApi.UpdateGroup(org, groupName, request.Name, request.Path)

	// Check errors
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		switch apiError.Code {
		case api.GROUP_BY_ORG_AND_NAME_NOT_FOUND:
			RespondNotFound(w)
		case api.GROUP_ALREADY_EXIST:
			RespondConflict(w)
		case api.INVALID_PARAMETER_ERROR:
			RespondBadRequest(w)
		default:
			RespondInternalServerError(w)
		}
		return
	}

	// Create response
	response := &UpdateGroupResponse{
		Group: result,
	}

	// Write group to response
	RespondOk(w, response)
}

func (g *GroupHandler) handleListMembers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group, org
	org := ps.ByName(ORG_NAME)
	group := ps.ByName(GROUP_NAME)

	// Call group API to list members
	result, err := g.core.GroupApi.ListMembers(org, group)

	// Check errors
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		switch apiError.Code {
		case api.GROUP_BY_ORG_AND_NAME_NOT_FOUND:
			RespondNotFound(w)
		default:
			RespondInternalServerError(w)
		}
		return
	}

	// Create response
	response := &GetGroupMembersResponse{
		Members: result,
	}

	// Write GroupMembers to response
	RespondOk(w, response)

}

func (g *GroupHandler) handleAddMember(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group, org and user from path
	org := ps.ByName(ORG_NAME)
	user := ps.ByName(USER_ID)
	group := ps.ByName(GROUP_NAME)

	// Call group API to create an group
	err := g.core.GroupApi.AddMember(user, group, org)
	// Error handling
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		switch apiError.Code {
		case api.GROUP_BY_ORG_AND_NAME_NOT_FOUND, api.USER_BY_EXTERNAL_ID_NOT_FOUND:
			RespondNotFound(w)
		case api.USER_IS_ALREADY_A_MEMBER_OF_GROUP:
			RespondConflict(w)
		default:
			RespondInternalServerError(w)
		}
		return
	} else { // Respond without content
		RespondNoContent(w)
	}
}

func (g *GroupHandler) handleRemoveMember(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group, org and user from path
	org := ps.ByName(ORG_NAME)
	user := ps.ByName(USER_ID)
	group := ps.ByName(GROUP_NAME)

	// Call group API to create an group
	err := g.core.GroupApi.RemoveMember(user, group, org)
	// Error handling
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		switch apiError.Code {
		case api.GROUP_BY_ORG_AND_NAME_NOT_FOUND, api.USER_BY_EXTERNAL_ID_NOT_FOUND:
			RespondNotFound(w)
		case api.USER_IS_NOT_A_MEMBER_OF_GROUP:
			RespondConflict(w)
		default:
			RespondInternalServerError(w)
		}
		return
	} else { // Respond without content
		RespondNoContent(w)
	}

}

func (g *GroupHandler) handleAttachGroupPolicy(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Retrieve group, org and policy from path
	org := ps.ByName(ORG_NAME)
	groupName := ps.ByName(GROUP_NAME)
	policyName := ps.ByName(POLICY_NAME)

	// Call group API to attach policy to group
	err := g.core.GroupApi.AttachPolicyToGroup(org, groupName, policyName)

	// Error handling
	if err != nil {
		g.core.Logger.Errorln(err)
		// Transform to API errors
		apiError := err.(*api.Error)
		switch apiError.Code {
		case api.GROUP_BY_ORG_AND_NAME_NOT_FOUND, api.POLICY_BY_ORG_AND_NAME_NOT_FOUND:
			RespondNotFound(w)
		case api.POLICY_IS_ALREADY_ATTACHED_TO_GROUP:
			RespondConflict(w)
		default: // Unexpected API error
			RespondInternalServerError(w)
		}
		return

	} else { // Respond without content
		RespondNoContent(w)
	}

}

func (g *GroupHandler) handleDetachGroupPolicy(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

}

func (g *GroupHandler) handleListAttachedGroupPolicies(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

}

func (g *GroupHandler) handleListAllGroups(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

}
