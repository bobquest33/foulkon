package api

import (
	"fmt"
	"time"

	"github.com/satori/go.uuid"
	"github.com/tecsisa/authorizr/database"
)

// Policy domain
type Policy struct {
	ID         string       `json:"ID, omitempty"`
	Name       string       `json:"Name, omitempty"`
	Path       string       `json:"Path, omitempty"`
	Org        string       `json:"Org, omitempty"`
	CreateAt   time.Time    `json:"CreateAt, omitempty"`
	Urn        string       `json:"Urn, omitempty"`
	Statements *[]Statement `json:"Statements, omitempty"`
}

func (p Policy) GetUrn() string {
	return p.Urn
}

// Identifier for policy that allow you to retrieve from Database
type PolicyIdentity struct {
	Org  string `json:"Org, omitempty"`
	Name string `json:"Name, omitempty"`
}

type Statement struct {
	Effect    string   `json:"Effect, omitempty"`
	Action    []string `json:"Action, omitempty"`
	Resources []string `json:"Resources, omitempty"`
}

func (api AuthAPI) GetPolicyByName(authenticatedUser AuthenticatedUser, org string, policyName string) (*Policy, error) {
	// Validate fields
	if !IsValidName(policyName) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy name %v", policyName),
		}
	}
	// Validate org
	if !IsValidOrg(org) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: org %v", org),
		}
	}

	// Call repo to retrieve the policy
	policy, err := api.PolicyRepo.GetPolicyByName(org, policyName)

	// Error handling
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		// Policy doesn't exist in DB
		if dbError.Code == database.POLICY_NOT_FOUND {
			return nil, &Error{
				Code:    POLICY_BY_ORG_AND_NAME_NOT_FOUND,
				Message: dbError.Message,
			}
		} else { // Unexpected error
			return nil, &Error{
				Code:    UNKNOWN_API_ERROR,
				Message: dbError.Message,
			}
		}
	}

	// Check restrictions
	policiesFiltered, err := api.GetPoliciesAuthorized(authenticatedUser, policy.Urn, POLICY_ACTION_GET_POLICY, []Policy{*policy})
	if err != nil {
		return nil, err
	}

	// Check if we have our user authorized
	if len(policiesFiltered) > 0 {
		policyFiltered := policiesFiltered[0]
		return &policyFiltered, nil
	} else {
		return nil, &Error{
			Code: UNAUTHORIZED_RESOURCES_ERROR,
			Message: fmt.Sprintf("User with external ID %v is not allowed to access to resource %v",
				authenticatedUser.Identifier, policy.Urn),
		}
	}
}

func (api AuthAPI) GetListPolicies(authenticatedUser AuthenticatedUser, org string, pathPrefix string) ([]PolicyIdentity, error) {
	// Validate path prefix
	if len(pathPrefix) > 0 && !IsValidPath(pathPrefix) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: PathPrefix %v", pathPrefix),
		}
	}
	// Validate org
	if !IsValidOrg(org) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: org %v", org),
		}
	}
	if len(pathPrefix) == 0 {
		pathPrefix = "/"
	}

	// Call repo to retrieve the policies
	policies, err := api.PolicyRepo.GetPoliciesFiltered(org, pathPrefix)

	// Error handling
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		return nil, &Error{
			Code:    UNKNOWN_API_ERROR,
			Message: dbError.Message,
		}
	}

	// Check restrictions to list
	urnPrefix := GetUrnPrefix(org, RESOURCE_POLICY, pathPrefix)
	policiesFiltered, err := api.GetPoliciesAuthorized(authenticatedUser, urnPrefix, POLICY_ACTION_LIST_POLICIES, policies)
	if err != nil {
		return nil, err
	}

	policyReferenceIds := []PolicyIdentity{}
	for _, p := range policiesFiltered {
		policyReferenceIds = append(policyReferenceIds, PolicyIdentity{
			Org:  p.Org,
			Name: p.Name,
		})
	}

	return policyReferenceIds, nil
}

func (api AuthAPI) AddPolicy(authenticatedUser AuthenticatedUser, name string, path string, org string, statements []Statement) (*Policy, error) {
	// Validate fields
	if !IsValidName(name) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy name %v", name),
		}
	}
	// Validate org
	if !IsValidOrg(org) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: org %v", org),
		}
	}
	if !IsValidPath(path) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Path %v", path),
		}

	}

	err := IsValidStatement(&statements)
	if err != nil {
		apiError := err.(*Error)
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: apiError.Message,
		}

	}

	policy := createPolicy(name, path, org, &statements)

	// Check restrictions
	policiesFiltered, err := api.GetPoliciesAuthorized(authenticatedUser, policy.Urn, POLICY_ACTION_CREATE_POLICY, []Policy{policy})
	if err != nil {
		return nil, err
	}

	// Check if we have our user authorized
	if len(policiesFiltered) < 1 {
		return nil, &Error{
			Code: UNAUTHORIZED_RESOURCES_ERROR,
			Message: fmt.Sprintf("User with external ID %v is not allowed to access to resource %v",
				authenticatedUser.Identifier, policy.Urn),
		}
	}

	// Check if policy already exist
	_, err = api.PolicyRepo.GetPolicyByName(org, name)

	// Check if policy could be retrieved
	if err != nil {
		// Transform to DB error
		dbError := err.(*database.Error)
		switch dbError.Code {
		// Policy doesn't exist in DB
		case database.POLICY_NOT_FOUND:
			// Create policy
			policyCreated, err := api.PolicyRepo.AddPolicy(policy)

			// Check if there is an unexpected error in DB
			if err != nil {
				//Transform to DB error
				dbError := err.(*database.Error)
				return nil, &Error{
					Code:    UNKNOWN_API_ERROR,
					Message: dbError.Message,
				}
			}

			// Return policy created
			return policyCreated, nil
		default: // Unexpected error
			return nil, &Error{
				Code:    UNKNOWN_API_ERROR,
				Message: dbError.Message,
			}
		}
	} else { // If policy exist it can't create it
		return nil, &Error{
			Code:    POLICY_ALREADY_EXIST,
			Message: fmt.Sprintf("Unable to create policy, policy with org %v and name %v already exist", org, name),
		}
	}
}

func (api AuthAPI) UpdatePolicy(authenticatedUser AuthenticatedUser, org string, policyName string, newName string, newPath string,
	newStatements []Statement) (*Policy, error) {
	// Validate fields
	if !IsValidName(policyName) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy name %v", policyName),
		}
	}
	// Validate org
	if !IsValidOrg(org) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: org %v", org),
		}
	}
	if !IsValidName(newName) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy new name %v", newName),
		}
	}
	if !IsValidPath(newPath) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy new path %v", newPath),
		}

	}
	err := IsValidStatement(&newStatements)
	if err != nil {
		apiError := err.(*Error)
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: apiError.Message,
		}

	}

	// Call repo to retrieve the policy
	policyDB, err := api.GetPolicyByName(authenticatedUser, org, policyName)
	if err != nil {
		return nil, err
	}

	// Check restrictions
	policiesFiltered, err := api.GetPoliciesAuthorized(authenticatedUser, policyDB.Urn, POLICY_ACTION_UPDATE_POLICY, []Policy{*policyDB})
	if err != nil {
		return nil, err
	}

	// Check if we have our user authorized
	if len(policiesFiltered) < 1 {
		return nil, &Error{
			Code: UNAUTHORIZED_RESOURCES_ERROR,
			Message: fmt.Sprintf("User with external ID %v is not allowed to access to resource %v",
				authenticatedUser.Identifier, policyDB.Urn),
		}
	}

	// Check if policy with newName exist
	targetPolicy, err := api.GetPolicyByName(authenticatedUser, org, newName)

	if err == nil && targetPolicy.ID != policyDB.ID {
		// Policy already exists
		return nil, &Error{
			Code:    POLICY_ALREADY_EXIST,
			Message: fmt.Sprintf("Policy name: %v already exists", newName),
		}
	}

	if err != nil {
		if apiError := err.(*Error); apiError.Code == UNAUTHORIZED_RESOURCES_ERROR || apiError.Code == UNKNOWN_API_ERROR {
			return nil, err
		}
	}

	// Get Policy Updated
	policyToUpdate := createPolicy(newName, newPath, org, &newStatements)

	// Check restrictions
	policiesFiltered, err = api.GetPoliciesAuthorized(authenticatedUser, policyToUpdate.Urn, POLICY_ACTION_UPDATE_POLICY, []Policy{policyToUpdate})
	if err != nil {
		return nil, err
	}

	// Check if we have our user authorized
	if len(policiesFiltered) < 1 {
		return nil, &Error{
			Code: UNAUTHORIZED_RESOURCES_ERROR,
			Message: fmt.Sprintf("User with external ID %v is not allowed to access to resource %v",
				authenticatedUser.Identifier, policyToUpdate.Urn),
		}
	}

	// Update policy
	policy, err := api.PolicyRepo.UpdatePolicy(*policyDB, newName, newPath, policyToUpdate.Urn, newStatements)

	// Check if there is an unexpected error in DB
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		return nil, &Error{
			Code:    UNKNOWN_API_ERROR,
			Message: dbError.Message,
		}
	}

	return policy, nil
}

func (api AuthAPI) DeletePolicy(authenticatedUser AuthenticatedUser, org string, name string) error {
	// Validate fields
	if !IsValidName(name) {
		return &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy name %v", name),
		}
	}
	// Validate org
	if !IsValidOrg(org) {
		return &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: org %v", org),
		}
	}

	// Call repo to retrieve the policy
	policy, err := api.GetPolicyByName(authenticatedUser, org, name)
	if err != nil {
		return err
	}

	// Check restrictions
	policiesFiltered, err := api.GetPoliciesAuthorized(authenticatedUser, policy.Urn, POLICY_ACTION_DELETE_POLICY, []Policy{*policy})
	if err != nil {
		return err
	}

	// Check if we have our user authorized
	if len(policiesFiltered) < 1 {
		return &Error{
			Code: UNAUTHORIZED_RESOURCES_ERROR,
			Message: fmt.Sprintf("User with external ID %v is not allowed to access to resource %v",
				authenticatedUser.Identifier, policy.Urn),
		}
	}

	err = api.PolicyRepo.RemovePolicy(policy.ID)
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		return &Error{
			Code:    UNKNOWN_API_ERROR,
			Message: dbError.Message,
		}
	}

	// Return no error
	return nil
}

func (api AuthAPI) GetPolicyAttachedGroups(authenticatedUser AuthenticatedUser, org string, policyName string) ([]GroupIdentity, error) {
	// Validate fields
	if !IsValidName(policyName) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: Policy name %v", policyName),
		}
	}
	// Validate org
	if !IsValidOrg(org) {
		return nil, &Error{
			Code:    INVALID_PARAMETER_ERROR,
			Message: fmt.Sprintf("Invalid parameter: org %v", org),
		}
	}

	// Call repo to retrieve the policy
	policy, err := api.GetPolicyByName(authenticatedUser, org, policyName)
	if err != nil {
		return nil, err
	}

	// Check restrictions
	policiesFiltered, err := api.GetPoliciesAuthorized(authenticatedUser, policy.Urn, POLICY_ACTION_LIST_ATTACHED_GROUPS, []Policy{*policy})
	if err != nil {
		return nil, err
	}

	// Check if we have our user authorized
	if len(policiesFiltered) < 1 {
		return nil, &Error{
			Code: UNAUTHORIZED_RESOURCES_ERROR,
			Message: fmt.Sprintf("User with external ID %v is not allowed to access to resource %v",
				authenticatedUser.Identifier, policy.Urn),
		}
	}

	// Call repo to retrieve the attached groups
	groups, err := api.PolicyRepo.GetAllPolicyGroupRelation(policy.ID)

	// Error handling
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		return nil, &Error{
			Code:    UNKNOWN_API_ERROR,
			Message: dbError.Message,
		}
	}

	groupReferenceIDs := []GroupIdentity{}
	for _, g := range groups {
		groupReferenceIDs = append(groupReferenceIDs, GroupIdentity{
			Org:  g.Org,
			Name: g.Name,
		})
	}

	return groupReferenceIDs, nil
}

func createPolicy(name string, path string, org string, statements *[]Statement) Policy {
	urn := CreateUrn(org, RESOURCE_POLICY, path, name)
	policy := Policy{
		ID:         uuid.NewV4().String(),
		Name:       name,
		Path:       path,
		Org:        org,
		Urn:        urn,
		Statements: statements,
	}

	return policy
}
