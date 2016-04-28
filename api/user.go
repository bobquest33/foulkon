package api

import (
	"fmt"
	"time"

	"github.com/tecsisa/authorizr/database"
)

// User domain
type User struct {
	ID         string    `json:"ID, omitempty"`
	ExternalID string    `json:"ExternalID, omitempty"`
	Path       string    `json:"Path, omitempty"`
	CreateAt   time.Time `json:"CreateAt, omitempty"`
	Urn        string    `json:"Urn, omitempty"`
}

// User api
type UsersAPI struct {
	UserRepo UserRepo
}

// Retrieve user by id
func (u *UsersAPI) GetUserByExternalId(id string) (*User, error) {
	// Call repo to retrieve the user
	user, err := u.UserRepo.GetUserByExternalID(id)

	// Error handling
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		// User doesn't exist in DB
		if dbError.Code == database.USER_NOT_FOUND {
			return nil, &Error{
				Code:    USER_BY_EXTERNAL_ID_NOT_FOUND,
				Message: dbError.Message,
			}
		} else { // Unexpected error
			return nil, &Error{
				Code:    UNKNOWN_API_ERROR,
				Message: dbError.Message,
			}
		}
	}

	// Return user
	return user, nil

}

// Retrieve users that has path
func (u *UsersAPI) GetListUsers(pathPrefix string) ([]User, error) {

	// Retrieve users with specified path prefix
	users, err := u.UserRepo.GetUsersFiltered(pathPrefix)

	// Error handling
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		return nil, &Error{
			Code:    UNKNOWN_API_ERROR,
			Message: dbError.Message,
		}
	}

	// Return users
	return users, nil
}

// Add an User to database if not exist
func (u *UsersAPI) AddUser(user User) (*User, error) {
	// Check if user already exist
	userDB, err := u.UserRepo.GetUserByExternalID(user.ExternalID)

	// If user exist it can't create it
	if userDB != nil {
		return nil, &Error{
			Code:    USER_ALREADY_EXIST,
			Message: fmt.Sprintf("Unable to create user, user with ExternalID %v already exist", user.ExternalID),
		}
	}

	// Check if there is an unexpected error in DB
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		if dbError.Code != database.USER_NOT_FOUND {
			return nil, &Error{
				Code:    UNKNOWN_API_ERROR,
				Message: dbError.Message,
			}
		}
	}

	// Create user
	userCreated, err := u.UserRepo.AddUser(user)

	// Check if there is an unexpected error in DB
	if err != nil {
		//Transform to DB error
		dbError := err.(*database.Error)
		return nil, &Error{
			Code:    UNKNOWN_API_ERROR,
			Message: dbError.Message,
		}
	}

	// Return user created
	return userCreated, nil
}

// Remove user with this id
func (u *UsersAPI) RemoveUserById(id string) error {
	// Remove user with given external id
	err := u.UserRepo.RemoveUser(id)

	if err != nil {
		//Transform to DB error
		dbError := err.(database.Error)
		// If user doesn't exist
		if dbError.Code == database.USER_NOT_FOUND {
			return &Error{
				Code:    USER_BY_EXTERNAL_ID_NOT_FOUND,
				Message: dbError.Message,
			}
		} else { // Unexpected error
			return &Error{
				Code:    UNKNOWN_API_ERROR,
				Message: dbError.Message,
			}
		}
	}

	return nil
}

// Get groups for an user
func (u *UsersAPI) GetGroupsByUserId(id string) ([]Group, error) {
	return u.UserRepo.GetGroupsByUserID(id)
}
