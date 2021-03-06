package postgresql

import (
	"fmt"
	"time"

	"github.com/Tecsisa/foulkon/api"
	"github.com/Tecsisa/foulkon/database"
)

// GROUP REPOSITORY IMPLEMENTATION

func (g PostgresRepo) AddGroup(group api.Group) (*api.Group, error) {

	// Create group model
	groupDB := &Group{
		ID:       group.ID,
		Name:     group.Name,
		Path:     group.Path,
		CreateAt: group.CreateAt.UnixNano(),
		Urn:      group.Urn,
		Org:      group.Org,
	}

	// Store group
	err := g.Dbmap.Create(groupDB).Error

	// Error handling
	if err != nil {
		return nil, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return dbGroupToAPIGroup(groupDB), nil
}

func (g PostgresRepo) GetGroupByName(org string, name string) (*api.Group, error) {
	group := &Group{}
	query := g.Dbmap.Where("org like ? AND name like ?", org, name).First(group)

	// Check if group exists
	if query.RecordNotFound() {
		return nil, &database.Error{
			Code:    database.GROUP_NOT_FOUND,
			Message: fmt.Sprintf("Group with organization %v and name %v not found", org, name),
		}
	}

	// Error Handling
	if err := query.Error; err != nil {
		return nil, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return dbGroupToAPIGroup(group), nil
}

func (g PostgresRepo) GetGroupById(id string) (*api.Group, error) {
	group := &Group{}
	query := g.Dbmap.Where("id like ?", id).First(group)

	// Check if group exists
	if query.RecordNotFound() {
		return nil, &database.Error{
			Code:    database.GROUP_NOT_FOUND,
			Message: fmt.Sprintf("Group with id %v not found", id),
		}
	}

	// Error Handling
	if err := query.Error; err != nil {
		return nil, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return dbGroupToAPIGroup(group), nil
}

func (g PostgresRepo) GetGroupsFiltered(org string, filter *api.Filter) ([]api.Group, int, error) {
	var total int
	groups := []Group{}
	query := g.Dbmap
	if len(org) > 0 {
		query = query.Where("org like ? ", org)
	}
	if len(filter.PathPrefix) > 0 {
		query = query.Where("path like ? ", filter.PathPrefix+"%")
	}
	// Error handling
	if err := query.Find(&groups).Count(&total).Offset(filter.Offset).Limit(filter.Limit).Find(&groups).Error; err != nil {
		return nil, total, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	// Transform users for API
	var apiGroups []api.Group
	if groups != nil {
		apiGroups = make([]api.Group, len(groups), cap(groups))
		for i, g := range groups {
			apiGroups[i] = *dbGroupToAPIGroup(&g)
		}
		return apiGroups, total, nil
	}

	// No data to return
	return apiGroups, total, nil
}

func (g PostgresRepo) UpdateGroup(group api.Group, newName string, newPath string, urn string) (*api.Group, error) {

	// Create new group
	updatedGroup := Group{
		Name: newName,
		Path: newPath,
		Urn:  urn,
	}

	groupDB := Group{
		ID:       group.ID,
		Name:     group.Name,
		Path:     group.Path,
		CreateAt: group.CreateAt.UTC().UnixNano(),
		Urn:      group.Urn,
		Org:      group.Org,
	}

	// Update group
	query := g.Dbmap.Model(&groupDB).Update(updatedGroup)

	// Check if group exist
	if query.RecordNotFound() {
		return nil, &database.Error{
			Code:    database.GROUP_NOT_FOUND,
			Message: fmt.Sprintf("Group with name %v not found", group.Name),
		}
	}

	// Error Handling
	if err := query.Error; err != nil {
		return nil, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return dbGroupToAPIGroup(&groupDB), nil
}

func (g PostgresRepo) RemoveGroup(id string) error {
	transaction := g.Dbmap.Begin()
	// Delete group
	transaction.Where("id like ?", id).Delete(&Group{})

	// Error handling
	if err := transaction.Error; err != nil {
		transaction.Rollback()
		return &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	// Delete all group relations
	transaction.Where("group_id like ?", id).Delete(&GroupUserRelation{})

	// Error handling
	if err := transaction.Error; err != nil {
		transaction.Rollback()
		return &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	transaction.Commit()
	return nil
}

func (g PostgresRepo) AddMember(userID string, groupID string) error {

	// Create relation
	relation := &GroupUserRelation{
		UserID:  userID,
		GroupID: groupID,
	}

	// Store relation
	err := g.Dbmap.Create(relation).Error

	// Error handling
	if err != nil {
		return &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return nil
}

func (g PostgresRepo) RemoveMember(userID string, groupID string) error {
	err := g.Dbmap.Where("user_id like ? AND group_id like ?", userID, groupID).Delete(&GroupUserRelation{}).Error

	// Error handling
	if err != nil {
		return &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}
	return nil
}

func (g PostgresRepo) IsMemberOfGroup(userID string, groupID string) (bool, error) {
	relation := GroupUserRelation{}
	query := g.Dbmap.Where("user_id like ? AND group_id like ?", userID, groupID).First(&relation)

	// Check if relation exists
	if query.RecordNotFound() {
		return false, nil
	}

	// Error Handling
	if err := query.Error; err != nil {
		return false, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return true, nil
}

func (g PostgresRepo) GetGroupMembers(groupID string, filter *api.Filter) ([]api.User, int, error) {
	var total int
	members := []GroupUserRelation{}
	query := g.Dbmap.Where("group_id like ?", groupID)

	// Error handling
	if err := query.Find(&members).Count(&total).Offset(filter.Offset).Limit(filter.Limit).Find(&members).Error; err != nil {
		return nil, total, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	var apiUsers []api.User
	// Transform relations to API domain
	if members != nil {
		apiUsers = make([]api.User, len(members), cap(members))
		for i, m := range members {
			user, err := g.GetUserByID(m.UserID)
			// Error handling
			if err != nil {
				return nil, total, &database.Error{
					Code:    database.INTERNAL_ERROR,
					Message: err.Error(),
				}
			}

			apiUsers[i] = *user
		}
	}

	return apiUsers, total, nil
}

func (g PostgresRepo) AttachPolicy(groupID string, policyID string) error {
	// Create relation
	relation := &GroupPolicyRelation{
		GroupID:  groupID,
		PolicyID: policyID,
	}

	// Store relation
	err := g.Dbmap.Create(relation).Error

	// Error handling
	if err != nil {
		return &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return nil
}

func (g PostgresRepo) DetachPolicy(groupID string, policyID string) error {
	// Remove relation
	err := g.Dbmap.Where("group_id like ? AND policy_id like ?", groupID, policyID).Delete(&GroupPolicyRelation{}).Error

	// Error handling
	if err != nil {
		return &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return nil
}

func (g PostgresRepo) IsAttachedToGroup(groupID string, policyID string) (bool, error) {
	relation := GroupPolicyRelation{}
	query := g.Dbmap.Where("group_id like ? AND policy_id like ?", groupID, policyID).First(&relation)

	// Check if relation exists
	if query.RecordNotFound() {
		return false, nil
	}

	// Error Handling
	if err := query.Error; err != nil {
		return false, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}

	return true, nil
}

func (g PostgresRepo) GetAttachedPolicies(groupID string, filter *api.Filter) ([]api.Policy, int, error) {
	var total int
	relations := []GroupPolicyRelation{}
	query := g.Dbmap.Where("group_id like ?", groupID)

	// Error Handling
	if err := query.Find(&relations).Count(&total).Offset(filter.Offset).Limit(filter.Limit).Find(&relations).Error; err != nil {
		return nil, total, &database.Error{
			Code:    database.INTERNAL_ERROR,
			Message: err.Error(),
		}
	}
	var apiPolicies []api.Policy
	// Transform relations to API domain
	if relations != nil {
		apiPolicies = make([]api.Policy, len(relations), cap(relations))
		for i, r := range relations {
			policy, err := g.GetPolicyById(r.PolicyID)
			// Error handling
			if err != nil {
				return nil, total, &database.Error{
					Code:    database.INTERNAL_ERROR,
					Message: err.Error(),
				}
			}

			apiPolicies[i] = *policy
		}
	}

	return apiPolicies, total, nil
}

// PRIVATE HELPER METHODS

// Transform a Group retrieved from db into a group for API
func dbGroupToAPIGroup(groupdb *Group) *api.Group {
	return &api.Group{
		ID:       groupdb.ID,
		Name:     groupdb.Name,
		Path:     groupdb.Path,
		CreateAt: time.Unix(0, groupdb.CreateAt).UTC(),
		Urn:      groupdb.Urn,
		Org:      groupdb.Org,
	}
}
