/**
 * This file contain various error type...
 */

package Server

import (
	//"log"

	"code.myceliUs.com/CargoWebServer/Cargo/Entities/CargoEntities"
)

func NewError(errorPath string, errorId string, errorCode int, err error) *CargoEntities.Error {

	// Create a error object.
	errorObject := new(CargoEntities.Error)
	errorObject.SetErrorPath(errorPath)
	errorObject.SetId(errorId)

	if err != nil {
		errorObject.SetBody(err.Error())
	} else {
		errorObject.SetBody(errorId)
	}

	errorObject.SetCode(errorCode)

	// Uncomment pour logger dans le logger. Trop lourd pour le moment.
	/*
		server.entityManager.cargoEntities.GetObject().(*CargoEntities.Entities).SetEntities(errorObject)
		server.entityManager.cargoEntities.SaveEntity()

		// Create the log information for that error.
		GetServer().GetDefaultErrorLogger().AppendLogEntry(errorObject)
	*/

	//log.Println("ERROR: ", errorObject)
	return errorObject
}

const (
	// Error Code
	SERVER_ERROR_CODE = 0
	CLIENT_ERROR_CODE = 1

	// If the function is no yet implemented.
	NOT_IMPLEMENTED_ERROR = "NOT_IMPLEMENTED_ERROR"

	// Generic errors
	LDAP_ERROR                    = "LDAP_ERROR"
	PARAMETER_TYPE_ERROR          = "PARAMETER_TYPE_ERROR"
	ACTION_EXECUTE_ERROR          = "ACTION_EXECUTE_ERROR"
	SECURITY_MANAGER_ERROR        = "SECURITY_MANAGER_ERROR"
	SESSION_MANAGER_ERROR         = "SESSION_MANAGER_ERROR"
	ACCOUNT_MANAGER_ERROR         = "ACCOUNT_MANAGER_ERROR"
	FILE_MANAGER_ERROR            = "FILE_MANAGER_ERROR"
	DATASTORE_ERROR               = "DATASTORE_ERROR"
	EMAIL_ERROR                   = "EMAIL_ERROR"
	ACCOUNT_ID_DOESNT_EXIST_ERROR = "ACCOUNT_ID_DOESNT_EXIST_ERROR"
	ACTION_DOESNT_EXIST_ERROR     = "ACTION_DOESNT_EXIST_ERROR"
	PROTOTYPE_ERROR               = "PROTOTYPE_ERROR"
	MIMETYPE_DOESNT_EXIST_ERROR   = "MIMETYPE_DOESNT_EXIST_ERROR"
	INVALID_PACKAGE_NAME_ERROR    = "INVALID_PACKAGE_NAME_ERROR"
	INVALID_VARIABLE_NAME_ERROR   = "INVALID_VARIABLE_NAME_ERROR"
	INVALID_REFERENCE_NAME_ERROR  = "INVALID_REFERENCE_NAME_ERROR"
	ENTITY_PROTOTYPE_ERROR        = "ENTITY_PROTOTYPE_ERROR"
	EVENT_ERROR                   = "EVENT_ERROR"

	// Datastore errors
	DATASTORE_DOESNT_EXIST_ERROR  = "DATASTORE_DOESNT_EXIST_ERROR"
	DATASTORE_ALREADY_EXIST_ERROR = "DATASTORE_ALREADY_EXIST_ERROR"
	DATASTORE_INDEXATION_ERROR    = "DATASTORE_INDEXATION_ERROR"
	DATASTORE_KEY_NOT_FOUND_ERROR = "DATASTORE_KEY_NOT_FOUND_ERROR"

	// Security errors
	ROLE_ID_ALEADY_EXISTS_ERROR    = "ROLE_ID_ALEADY_EXISTS_ERROR"
	ROLE_ID_DOESNT_EXIST_ERROR     = "ROLE_ID_DOESNT_EXIST_ERROR"
	RESTRICTION_ACTION_ROLE_ERROR  = "RESTRICTION_ACTION_ROLE_ERROR"
	ROLE_DOESNT_HAVE_ACCOUNT_ERROR = "ROLE_DOESNT_HAVE_ACCOUNT_ERROR"
	ROLE_DOESNT_HAVE_ACTION_ERROR  = "ROLE_DOESNT_HAVE_ACTION_ERROR"
	PERMISSION_DENIED_ERROR        = "PERMISSION_DENIED_ERROR"

	// Session errors
	PASSWORD_MISMATCH_ERROR      = "PASSWORD_MISMATCH_ERROR"
	NO_SESSION_FOUND_ERROR       = "NO_SESSION_FOUND_ERROR"
	ACCOUNT_DOESNT_EXIST_ERROR   = "ACCOUNT_DOESNT_EXIST_ERROR"
	SESSION_ID_NOT_ACTIVE        = "SESSION_ID_NOT_ACTIVE"
	SESSION_UUID_NOT_FOUND_ERROR = "SESSION_UUID_NOT_FOUND_ERROR"

	// Account errors
	ACCOUNT_ALREADY_EXISTS_ERROR = "ACCOUNT_ALREADY_EXISTS_ERROR"
	USER_ID_DOESNT_EXIST_ERROR   = "USER_ID_DOESNT_EXIST_ERROR"

	// File errors
	INVALID_DIRECTORY_PATH_ERROR = "INVALID_DIRECTORY_PATH_ERROR"
	FILE_ALREADY_EXISTS_ERROR    = "FILE_ALREADY_EXISTS_ERROR"
	FILE_OPEN_ERROR              = "FILE_OPEN_ERROR"
	GET_FILE_STAT_ERROR          = "GET_FILE_STAT_ERROR"
	FILE_NOT_FOUND_ERROR         = "FILE_NOT_FOUND_ERROR"
	FILE_READ_ERROR              = "FILE_READ_ERROR"
	FILE_DELETE_ERROR            = "FILE_DELETE_ERROR"
	FILE_WRITE_ERROR             = "FILE_WRITE_ERROR"

	// LDAP errors
	COMPUTER_IP_DOESNT_EXIST_ERROR = "COMPUTER_IP_DOESNT_EXIST_ERROR"

	// Email errors
	EMAIL_ATTACHEMENT_FAIL_ERROR = "EMAIL_ATTACHEMENT_FAIL_ERROR"

	// Entity errors
	PROTOTYPE_DOESNT_EXIST_ERROR      = "PROTOTYPE_DOESNT_EXIST_ERROR"
	PROTOTYPE_RESTRICTIONS_ERROR      = "PROTOTYPE_RESTRICTIONS_ERROR"
	ENTITY_UUID_DOESNT_EXIST_ERROR    = "ENTITY_UUID_DOESNT_EXIST_ERROR"
	ENTITY_ID_DOESNT_EXIST_ERROR      = "ENTITY_ID_DOESNT_EXIST_ERROR"
	PROTOTYPE_CREATION_ERROR          = "PROTOTYPE_CREATION_ERROR"
	PROTOTYPE_UPDATE_ERROR            = "PROTOTYPE_UPDATE_ERROR"
	PROTOTYPE_DELETE_ERROR            = "PROTOTYPE_DELETE_ERROR"
	ATTRIBUTE_NAME_DOESNT_EXIST_ERROR = "ATTRIBUTE_NAME_DOESNT_EXIST_ERROR"
	TYPENAME_DOESNT_EXIST_ERROR       = "TYPENAME_DOESNT_EXIST_ERROR"
	ENTITY_ALREADY_EXIST_ERROR        = "ENTITY_ALREADY_EXIST_ERROR"
	ENTITY_TO_QUADS_ERROR             = "ENTITY_TO_QUADS_ERROR"
	ENTITY_CREATION_ERROR             = "ENTITY_CREATION_ERROR"
	ENTITY_UPDATE_ERROR               = "ENTITY_UPDATE_ERROR"

	// OAuth2 errors.
	REGISTER_CLIENT_ERROR      = "REGISTER_CLIENT_ERROR"
	RESSOURCE_NOT_FOUND_ERROR  = "RESSOURCE_NOT_FOUND_ERROR"
	AUTHORIZATION_DENIED_ERROR = "AUTHORIZATION_DENIED_ERROR"
	ACCESS_DENIED_ERROR        = "ACCESS_DENIED_ERROR"

	XML_READ_ERROR        = "XML_READ_ERROR"
	JSON_MARSHALING_ERROR = "JSON_MARSHALING_ERROR"
)
