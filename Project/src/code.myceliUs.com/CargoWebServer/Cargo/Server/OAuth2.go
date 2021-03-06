/**
 * That file contain the code for OAuth2 service.
 * As exemple...
 * id: 			1234
 * secret: 		aabbccdd
 * redirect: 	http://localhost:9393/oauth2callback
 * token: 		http://localhost:9393/token
 * authorize: 	http://localhost:9393/authorize
 */

package Server

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"code.myceliUs.com/CargoWebServer/Cargo/Entities/CargoEntities"
	"code.myceliUs.com/CargoWebServer/Cargo/Entities/Config"
	"code.myceliUs.com/Utility"
	"github.com/RangelReale/osin"
	"gopkg.in/square/go-jose.v1"
)

// variable use betheewen the http handlers and the OAuth service.
var (
	// The manager.
	oauth2Manager *OAuth2Manager
	channels      map[string]chan []string
)

// The ID Token represents a JWT passed to the client as part of the token response.
//
// https://openid.net/specs/openid-connect-core-1_0.html#IDToken
type IDToken struct {
	// Specifies the issuing authority (iss).
	Issuer string `json:"iss"`
	// Asserts the identity of the user, called subject in OpenID (sub).
	UserID string `json:"sub"`
	// Is generated for a particular audience, i.e. client (aud).
	ClientID string `json:"aud"`

	// Has an issue (iat) and an expiration date (exp).
	Expiration int64 `json:"exp"`
	IssuedAt   int64 `json:"iat"`

	// May contain a nonce (nonce).
	Nonce string `json:"nonce,omitempty"` // Non-manditory fields MUST be "omitempty"

	// Custom claims supported by this server.
	//
	// See: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Name       string `json:"name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	Locale     string `json:"locale,omitempty"`
}

/**
 * The OAuth2 Server.
 */
type OAuth2Manager struct {

	// the data stores.
	m_store *OAuth2Store

	// the oauth sever.
	m_server *osin.Server

	// openId authentication.
	m_jwtSigner  jose.Signer
	m_publicKeys *jose.JsonWebKeySet
}

func (this *Server) GetOAuth2Manager() *OAuth2Manager {
	if oauth2Manager == nil {
		oauth2Manager = newOAuth2Manager()
	}
	return oauth2Manager
}

func newOAuth2Manager() *OAuth2Manager {

	// The oauth manager.
	oauth2Manager := new(OAuth2Manager)

	return oauth2Manager
}

func savePEMKey(fileName string, key *rsa.PrivateKey) error {
	outFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outFile, privateKey)
	if err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Service functions
////////////////////////////////////////////////////////////////////////////////

/**
 * That function is use to synchronize the information of a ldap server
 * with a given id.
 */
func (this *OAuth2Manager) initialize() {
	// register service avalaible action here.
	log.Println("--> initialyze OAuth2Manager")

	// Create the default configurations
	GetServer().GetConfigurationManager().setServiceConfiguration(this.getId(), -1)

	channels = make(map[string]chan []string, 0)

}

func (this *OAuth2Manager) getId() string {
	return "OAuth2Manager"
}

func (this *OAuth2Manager) start() {
	log.Println("--> Start OAuth2Manager")
	activeConfigurations := GetServer().GetConfigurationManager().getActiveConfigurations()
	cfg := activeConfigurations.GetOauth2Configuration()

	var sconfig *osin.ServerConfig
	if cfg == nil {
		// Get the default configuration.
		sconfig = osin.NewServerConfig()

		// Set default parameters here.
		sconfig.AllowedAuthorizeTypes = osin.AllowedAuthorizeType{osin.CODE, osin.TOKEN}
		sconfig.AllowedAccessTypes = osin.AllowedAccessType{osin.AUTHORIZATION_CODE,
			osin.REFRESH_TOKEN, osin.PASSWORD, osin.CLIENT_CREDENTIALS, osin.ASSERTION}
		sconfig.AllowGetAccessRequest = true
		sconfig.AllowClientSecretInParams = true

		// Save it into the entity.
		cfg = new(Config.OAuth2Configuration)
		cfg.SetId("OAuth2Config")
		cfg.SetAccessExpiration(int64(sconfig.AccessExpiration))
		cfg.SetAllowClientSecretInParams(sconfig.AllowClientSecretInParams)
		cfg.SetAllowGetAccessRequest(sconfig.AllowGetAccessRequest)

		cfg.SetUuidGenerator(generateUuidFct)
		cfg.SetEntityGetter(getEntityFct)
		cfg.SetEntitySetter(setEntityFct)

		for i := 0; i < len(sconfig.AllowedAuthorizeTypes); i++ {
			if sconfig.AllowedAuthorizeTypes[i] == osin.CODE {
				cfg.AppendAllowedAuthorizeTypes("code")
			} else if sconfig.AllowedAuthorizeTypes[i] == osin.TOKEN {
				cfg.AppendAllowedAuthorizeTypes("token")
			}
		}

		for i := 0; i < len(sconfig.AllowedAccessTypes); i++ {
			if sconfig.AllowedAccessTypes[i] == osin.AUTHORIZATION_CODE {
				cfg.AppendAllowedAccessTypes("authorization_code")
			} else if sconfig.AllowedAccessTypes[i] == osin.REFRESH_TOKEN {
				cfg.AppendAllowedAccessTypes("refresh_token")
			} else if sconfig.AllowedAccessTypes[i] == osin.PASSWORD {
				cfg.AppendAllowedAccessTypes("password")
			} else if sconfig.AllowedAccessTypes[i] == osin.CLIENT_CREDENTIALS {
				cfg.AppendAllowedAccessTypes("client_credentials")
			} else if sconfig.AllowedAccessTypes[i] == osin.ASSERTION {
				cfg.AppendAllowedAccessTypes("assertion")
			} else if sconfig.AllowedAccessTypes[i] == osin.IMPLICIT {
				cfg.AppendAllowedAccessTypes("__implicit")
			}
		}

		cfg.SetAuthorizationExpiration(int(sconfig.AuthorizationExpiration))
		cfg.SetErrorStatusCode(sconfig.ErrorStatusCode)
		cfg.SetRedirectUriSeparator(sconfig.RedirectUriSeparator)
		cfg.SetTokenType(sconfig.TokenType)

		// Now the key.
		reader := rand.Reader
		bitSize := 2048

		key, err := rsa.GenerateKey(reader, bitSize)

		// The file name will be a random number.
		fileName := Utility.RandomUUID()

		if err == nil {
			savePEMKey(GetServer().GetConfigurationManager().GetDataPath()+"/Config/"+fileName+".pem", key)
		}

		cfg.SetPrivateKey(fileName)

		// Create the new configuration entity.
		GetServer().GetEntityManager().createEntity(activeConfigurations, "M_oauth2Configuration", cfg)

	} else {
		sconfig = osin.NewServerConfig()
		// Set the access expiration time.
		sconfig.AccessExpiration = int32(cfg.GetAccessExpiration())
		sconfig.AllowClientSecretInParams = cfg.IsAllowClientSecretInParams()
		sconfig.AllowGetAccessRequest = cfg.IsAllowGetAccessRequest()

		for i := 0; i < len(cfg.GetAllowedAuthorizeTypes()); i++ {
			if cfg.GetAllowedAuthorizeTypes()[i] == "code" {
				sconfig.AllowedAuthorizeTypes = append(sconfig.AllowedAuthorizeTypes, osin.CODE)
			} else if cfg.GetAllowedAuthorizeTypes()[i] == "token" {
				sconfig.AllowedAuthorizeTypes = append(sconfig.AllowedAuthorizeTypes, osin.TOKEN)
			}
		}

		for i := 0; i < len(cfg.GetAllowedAccessTypes()); i++ {
			if cfg.GetAllowedAccessTypes()[i] == "authorization_code" {
				sconfig.AllowedAccessTypes = append(sconfig.AllowedAccessTypes, osin.AUTHORIZATION_CODE)
			} else if cfg.GetAllowedAccessTypes()[i] == "refresh_token" {
				sconfig.AllowedAccessTypes = append(sconfig.AllowedAccessTypes, osin.REFRESH_TOKEN)
			} else if cfg.GetAllowedAccessTypes()[i] == "password" {
				sconfig.AllowedAccessTypes = append(sconfig.AllowedAccessTypes, osin.PASSWORD)
			} else if cfg.GetAllowedAccessTypes()[i] == "client_credentials" {
				sconfig.AllowedAccessTypes = append(sconfig.AllowedAccessTypes, osin.CLIENT_CREDENTIALS)
			} else if cfg.GetAllowedAccessTypes()[i] == "assertion" {
				sconfig.AllowedAccessTypes = append(sconfig.AllowedAccessTypes, osin.ASSERTION)
			} else if cfg.GetAllowedAccessTypes()[i] == "__implicit" {
				sconfig.AllowedAccessTypes = append(sconfig.AllowedAccessTypes, osin.IMPLICIT)
			}
		}

		sconfig.AuthorizationExpiration = int32(cfg.GetAuthorizationExpiration())
		sconfig.ErrorStatusCode = cfg.GetErrorStatusCode()
		sconfig.RedirectUriSeparator = cfg.GetRedirectUriSeparator()
		sconfig.TokenType = cfg.GetTokenType()

		// Cleanup
		this.cleanup()
	}

	// Load signing key.
	b, _ := ioutil.ReadFile(GetServer().GetConfigurationManager().GetDataPath() + "/Config/" + cfg.GetPrivateKey() + ".pem")
	block, _ := pem.Decode(b)

	if block == nil {
		log.Println("no private key found")
		return
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Println("failed to parse key: ", key, err)
		return
	}

	// Configure jwtSigner and public keys.
	privateKey := &jose.JsonWebKey{
		Key:       key,
		Algorithm: "RS256",
		Use:       "sig",
		KeyID:     "1", // KeyID should use the key thumbprint.
	}

	this.m_jwtSigner, err = jose.NewSigner(jose.RS256, privateKey)
	if err != nil {
		log.Println("failed to create jwtSigner:", this.m_jwtSigner, err)
		return
	}

	this.m_publicKeys = &jose.JsonWebKeySet{
		Keys: []jose.JsonWebKey{
			jose.JsonWebKey{Key: &key.PublicKey,
				Algorithm: "RS256",
				Use:       "sig",
				KeyID:     "1",
			},
		},
	}

	// Start the oauth service.
	this.m_store = newOauth2Store()
	this.m_server = osin.NewServer(sconfig, this.m_store)

}

func (this *OAuth2Manager) stop() {
	log.Println("--> Stop OAuth2Manager")
}

/**
 * That function remove expire access and authorization and renew access
 * if refresh exist.
 */
func (this *OAuth2Manager) cleanup() {
	activeConfiguration := GetServer().GetConfigurationManager().getActiveConfigurations()
	config := activeConfiguration.GetOauth2Configuration()

	// First of all I will renew the access...
	for i := 0; i < len(config.GetAccess()); i++ {
		access := config.GetAccess()[i]
		expirationTime := time.Unix(access.GetCreatedAt(), 0).Add(time.Duration(access.GetExpiresIn()) * time.Second)
		if expirationTime.Before(time.Now()) {
			if access.GetRefreshToken() != nil {
				accessEntity, _ := GetServer().GetEntityManager().getEntityByUuid(access.UUID)
				access = accessEntity.(*Config.OAuth2Access)
				// Reset the creation time instead of delete it and recreated it...
				access.SetCreatedAt(time.Now().Unix())
				GetServer().GetEntityManager().saveEntity(access)

				// Now it expire time.
				ids := []interface{}{access.GetId()}
				expireEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Expires", "Config", ids)
				expireEntity.(*Config.OAuth2Expires).SetExpiresAt(time.Unix(access.GetCreatedAt(), 0).Add(time.Duration(access.GetExpiresIn()) * time.Second).Unix())
				GetServer().GetEntityManager().saveEntity(expireEntity)
			}
		}
	}

	// Here I will remove all expired authorization and access.
	for i := 0; i < len(config.GetExpire()); i++ {
		expireTime := time.Unix(config.GetExpire()[i].GetExpiresAt(), 0)
		if expireTime.Before(time.Now()) {
			// I that case the value must be remove expired values...
			this.m_store.RemoveAccess(config.GetExpire()[i].GetId())
			this.m_store.RemoveAuthorize(config.GetExpire()[i].GetId())
		} else {
			setCodeExpiration(config.GetExpire()[i].GetId(), expireTime.Sub(time.Now()))
		}
	}

	// Now I will remove all refresh without access...
	for i := 0; i < len(config.GetRefresh()); i++ {
		refresh := config.GetRefresh()[i]
		if refresh.GetAccess() == nil {
			refreshEntity, _ := GetServer().GetEntityManager().getEntityByUuid(refresh.GetUuid())
			GetServer().GetEntityManager().deleteEntity(refreshEntity)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// The api
////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////// OpenID ///////////////////////////////////

/**
 * handleDiscovery returns the OpenID Connect discovery object, allowing clients
 * to discover OAuth2 resources.
 */
func DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	// For other example see: https://accounts.google.com/.well-known/openid-configuration
	config := GetServer().GetConfigurationManager().getServiceConfigurationById(GetServer().GetOAuth2Manager().getId())

	// The hostname and port must be correctly configure here, localhost will not work in many case.
	issuer := "https://" + config.GetHostName() + ":" + strconv.Itoa(config.GetPort())
	data := map[string]interface{}{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"jwks_uri":                              issuer + "/publickeys",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic"},
		"claims_supported": []string{
			"aud", "email", "email_verified", "exp",
			"family_name", "given_name", "iat", "iss",
			"locale", "name", "sub",
		},
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("failed to marshal data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Write(raw)
}

// handlePublicKeys publishes the public part of this server's signing keys.
// This allows clients to verify the signature of ID Tokens.
func PublicKeysHandler(w http.ResponseWriter, r *http.Request) {
	raw, err := json.MarshalIndent(GetServer().GetOAuth2Manager().m_publicKeys, "", "  ")
	if err != nil {
		log.Printf("failed to marshal data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Write(raw)
}

///////////////////////////////////// OAuth2 ///////////////////////////////////

// @api 1.0
// That function is use to get a given ressource for a given client.
// The client id is the id define in the configuration
// The scope are the scope of the ressources, ex. public_profile, email...
// The query is an http query from various api like facebook graph api.
// will start an authorization process if nothing is found.
// @param {string} clientId The oauth2 client identifier
// @param {string} scope string The resource scope
// @param {string} query string
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @param {string} query The http query string
// @param {string} idTokenUuid The id token uuid.
// @param {string} accessUuid The access uuid.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//OAuth2Manager.prototype.getResource = function (clientId, scope, query, successCallback, errorCallback, caller) {
//    // server is the client side singleton.
//    // Account uuid are set and reset at the time of login and logout respectively.
//    var idTokenUuid = ""
//    if(localStorage.getItem("idTokenUuid") != undefined){
//        idTokenUuid = localStorage.getItem("idTokenUuid")
//    }
//    var params = []
//    params.push(createRpcData(clientId, "STRING", "clientId"))
//    params.push(createRpcData(scope, "STRING", "scope"))
//    params.push(createRpcData(query, "STRING", "query"))
//    params.push(createRpcData(idTokenUuid, "STRING", "idTokenUuid"))
//    params.push(createRpcData("", "STRING", "accessUuid"))
//    // Call it on the server.
//    server.executeJsFunction(
//        "OAuth2ManagerGetResource", // The function to execute remotely on server
//        params, // The parameters to pass to that function
//        function (index, total, caller) { // The progress callback
//            // Nothing special to do here.
//        },
//        function (results, caller) {
//            caller.successCallback(results[0], caller.caller)
//        },
//        function (errMsg, caller) {
//            // display the message in the console.
//            // call the immediate error callback.
//            caller.errorCallback(errMsg, caller.caller)
//            // dispatch the message.
//            server.errorManager.onError(errMsg)
//        }, // Error callback
//        { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback } // The caller
//    )
//}
func (this *OAuth2Manager) GetResource(clientId string, scope string, query string, idTokenUuid string, accessUuid string, messageId string, sessionId string) interface{} {
	var access *Config.OAuth2Access
	// I will get the client...
	ids := []interface{}{clientId}
	clientEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	client := clientEntity.(*Config.OAuth2Client)
	if len(accessUuid) == 0 && len(idTokenUuid) > 0 {
		// Try to find the access...

		// Get the config.
		activeConfigurations := GetServer().GetConfigurationManager().getActiveConfigurations()
		config := activeConfigurations.GetOauth2Configuration()

		// Get the accesses
		accesses := config.GetAccess()

		// Now the client was found I will try to get an access code for the given
		// scope and client.
		for i := 0; i < len(accesses) && access == nil; i++ {
			a := accesses[i]
			if a.GetClient().GetId() == client.GetId() {
				values := strings.Split(a.GetScope(), " ")
				values_ := strings.Split(scope, " ")
				hasScope := true
				for j := 0; j < len(values_); j++ {
					if !Utility.Contains(values, values_[j]) {
						hasScope = false
						break
					}
				}
				// If the access has the correct scope.
				if hasScope && a.M_userData == idTokenUuid {
					// We found an access to the ressource!-)
					access = a
				}
			}
		}
	} else if len(accessUuid) > 0 {
		// Here the accessUuid is given so I will use it to get ressources.
		entity, err := GetServer().GetEntityManager().getEntityByUuid(accessUuid)
		if err == nil {
			access = entity.(*Config.OAuth2Access)
		} else {
			GetServer().reportErrorMessage(messageId, sessionId, err)
			return nil
		}
	}

	// If the ressource is found all we have to do is to get the actual resource.
	if access != nil {
		// Here I will made the API call.
		log.Println("-------------> dowload the ressource")
		result, err := DownloadRessource(query, access.GetId(), "Bearer")
		if err == nil {
			log.Println("-------------> result found ", result)
			return result
		} else {
			errObj := NewError(Utility.FileLine(), RESSOURCE_NOT_FOUND_ERROR, SERVER_ERROR_CODE, err)
			GetServer().reportErrorMessage(messageId, sessionId, errObj)
			return nil
		}

	} else {
		log.Println("-----------> ask for authorization")
		// No access was found so here I will initiated the authorization process...
		// To do so I will create the href where the user will be ask to
		// authorize the client application to access ressources.
		var authorizationLnk = client.GetAuthorizationUri()
		authorizationLnk += "?response_type=code&client_id=" + client.GetId()

		// I will create the request and send it to the client...
		msgId := Utility.RandomUUID()
		authorizationLnk += "&state=" + msgId + ":" + sessionId + ":" + clientId + ":" + scope + "&scope=" + scope + "&access_type=offline&approval_prompt=force"
		authorizationLnk += "&redirect_uri=" + client.GetRedirectUri()

		// Here if there is no user logged for the given session I will send an authentication request.
		var method string
		method = "OAuth2Authorization"
		params := make([]*MessageData, 1)
		data := new(MessageData)
		data.TYPENAME = "Server.MessageData"
		data.Name = "authorizationLnk"
		data.Value = authorizationLnk
		params[0] = data
		to := make([]*WebSocketConnection, 1)
		to[0] = GetServer().getConnectionById(sessionId)

		// synchronize the routine with a channel...
		done := make(chan bool)

		var authorizationCode string
		/** The authorize request **/
		oauth2AuthorizeRqst, _ := NewRequestMessage(msgId, method, params, to,
			func(done chan bool, idTokenUuid *string, accessUuid *string, authorizationCode *string) func(*message, interface{}) {
				return func(rspMsg *message, caller interface{}) {
					// I will retreive the access uuid from the result.
					results := rspMsg.msg.Rsp.GetResults()
					if len(results) == 2 {
						if Utility.IsValidEntityReferenceName(string(results[0].GetDataBytes())) {
							// In that case is the access uuid
							*accessUuid = string(results[0].GetDataBytes())
							*idTokenUuid = string(results[1].GetDataBytes())
						} else {
							// Here is the authorization code.
							*authorizationCode = string(results[0].GetDataBytes())
						}
					}
					done <- true
				}
			}(done, &idTokenUuid, &accessUuid, &authorizationCode), nil,
			func(done chan bool) func(*message, interface{}) {
				return func(rspMsg *message, caller interface{}) {
					done <- false
				}
			}(done), nil)

		// Send the request.
		GetServer().getProcessor().m_sendRequest <- oauth2AuthorizeRqst

		// So here I must block the execution of that function and wait
		// for the authorization. To do so I will made use of channel and
		// a callback and closure tree powerfull tools...
		// Wait for success or error...
		closeAuthorizeDialog := func() {
			var method string
			method = "closeAuthorizeDialog"
			params := make([]*MessageData, 0)
			to := make([]*WebSocketConnection, 1)
			to[0] = GetServer().getConnectionById(sessionId)
			oauth2AuthorizeEnd, err := NewRequestMessage(Utility.RandomUUID(), method, params, to, nil, nil, nil, nil)
			if err == nil {
				// Send the request.
				GetServer().getProcessor().m_sendRequest <- oauth2AuthorizeEnd
			}
		}

		// That function finalyse the access and close the issuer window
		// and save the idTokenUuid on the client side.
		finalyseAuthorize := func(idTokenUuid string) {
			var method string
			method = "finalyseAuthorize"
			params := make([]*MessageData, 1)

			// Put the id token uuid.
			params[0] = new(MessageData)
			params[0].Name = "idTokenUuid"
			params[0].Value = idTokenUuid

			to := make([]*WebSocketConnection, 1)
			to[0] = GetServer().getConnectionById(sessionId)
			oauth2AuthorizeEnd, err := NewRequestMessage(Utility.RandomUUID(), method, params, to, nil, nil, nil, nil)
			if err == nil {
				// Send the request.
				GetServer().getProcessor().m_sendRequest <- oauth2AuthorizeEnd
			}
		}

		// Wait for authorization
		if <-done {
			closeAuthorizeDialog()
			if len(accessUuid) == 0 {
				log.Println("-----------> authorization accept")
				log.Println("-----------> Ask for access")

				// That channel will contain the accessUuid
				channels[msgId] = make(chan []string)

				log.Println("------> wait for channel ", msgId)

				// Wait for the response from AppAuthCodeHandler.
				values := <-channels[msgId]

				// wait for access...
				if len(values) > 0 {
					// Recal the method with the grant access uuid...
					log.Println("590 -----------> access accept ", accessUuid)
					finalyseAuthorize(values[0])
					return this.GetResource(clientId, scope, query, values[0], values[1], messageId, sessionId)
				} else {
					log.Println("594 -----------> access refuse")
					errObj := NewError(Utility.FileLine(), ACCESS_DENIED_ERROR, SERVER_ERROR_CODE, errors.New("Access denied to get resource with scope "+scope+" for client with id "+clientId))
					GetServer().reportErrorMessage(messageId, sessionId, errObj)
				}

				delete(channels, msgId)

			} else {
				// Recal the method with the grant access uuid...
				log.Println("604 -----------> access accept")
				finalyseAuthorize(idTokenUuid)
				return this.GetResource(clientId, scope, query, idTokenUuid, accessUuid, messageId, sessionId)
			}

		} else {
			// Here I will report the error to the user.
			log.Println("-----------> authorization refuse")
			closeAuthorizeDialog()
			errObj := NewError(Utility.FileLine(), AUTHORIZATION_DENIED_ERROR, SERVER_ERROR_CODE, errors.New("Authorization denied to get resource with scope "+scope+" for client with id "+clientId))
			GetServer().reportErrorMessage(messageId, sessionId, errObj)
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Helper function
////////////////////////////////////////////////////////////////////////////////

/**
 * Clear expiring authorization and (access/refresh)
 */
func clearCodeExpired(code string) {

	// Remove the expire
	expireEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Expires", "Config", []interface{}{code})
	if expireEntity != nil {
		GetServer().GetEntityManager().deleteEntity(expireEntity)
	}

	// Remove the authorization
	authorizationEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Authorize", "Config", []interface{}{code})
	if authorizationEntity != nil {
		GetServer().GetEntityManager().deleteEntity(authorizationEntity)
	}

	// Remove the access
	accessEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", []interface{}{code})
	if accessEntity != nil {
		GetServer().GetEntityManager().deleteEntity(accessEntity)
	}

}

/**
 * Use a timer to execute clearExpiredCode when it came at end...
 */
func setCodeExpiration(code string, duration time.Duration) {
	// Create a closure and wrap the code.
	f := func(code string) func() {
		return func() {

			// In case of access token... Refresh it if it can.
			ids := []interface{}{code}
			entity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", ids)
			if errObj == nil {
				access := entity.(*Config.OAuth2Access)
				if access.GetRefreshToken() != nil {
					createAccessToken("refresh_token", access.GetClient(), access.GetAuthorize(), access.GetRefreshToken().GetId(), access.GetScope())

				}
			}

			// Remove the old access...
			clearCodeExpired(code)
		}
	}(code)

	// The function will be call after the duration.
	time.AfterFunc(duration, f)
}

/**
 * AddExpireAtData add info in expires table
 */
func addExpireAtData(code string, expireAt time.Time) error {
	ids := []interface{}{code}
	configEntity := GetServer().GetConfigurationManager().getOAuthConfigurationEntity()
	expireEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Expires", "Config", ids)

	var expire *Config.OAuth2Expires
	if expireEntity == nil {
		expire = new(Config.OAuth2Expires)
		expire.SetId(code)

		// append to config.
		expireEntity, _ = GetServer().GetEntityManager().createEntity(configEntity, "M_expire", expire)

	} else {
		expire = expireEntity.(*Config.OAuth2Expires)
	}

	// Set the date.
	expire.SetExpiresAt(expireAt.Unix())
	GetServer().GetEntityManager().saveEntity(expireEntity)

	// Start the timer.
	duration := expireAt.Sub(time.Now())

	// Set it expiration function.
	setCodeExpiration(code, duration)

	return nil
}

/**
 * Create Access token from refresh_token or authorization_code.
 */
func createAccessToken(grantType string, client *Config.OAuth2Client, authorizationCode string, refreshToken string, scope string) (*Config.OAuth2Access, error) {

	// The map that will contain the results
	jr := make(map[string]interface{})

	// build access code url
	parameters := url.Values{}
	parameters.Add("grant_type", grantType)
	parameters.Add("client_id", client.GetId())
	parameters.Add("client_secret", client.GetSecret())
	parameters.Add("redirect_uri", client.GetRedirectUri())

	if grantType == "refresh_token" {
		if len(refreshToken) > 0 {
			parameters.Add("refresh_token", refreshToken)
		}
	} else if grantType == "authorization_code" {
		parameters.Add("code", authorizationCode)
	}

	jr, err := RetrieveToken(client.GetId(), client.GetSecret(), client.GetTokenUri(), parameters)
	var access *Config.OAuth2Access
	if err != nil {
		return nil, err
	} else {
		if jr["access_token"] != nil {
			// Here I will save the new access token.
			configEntity := GetServer().GetConfigurationManager().getOAuthConfigurationEntity()

			// Here I will create a new access token if is not already exist.
			accessEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", []interface{}{jr["access_token"]})
			if accessEntity == nil {
				// Here I will create a new access from json data.
				access = new(Config.OAuth2Access)
				// Set the id
				access.SetId(jr["access_token"].(string))

				// set the access uuid
				accessEntity, _ = GetServer().GetEntityManager().createEntity(configEntity, "M_access", access)

				// Set the creation time.
				access.SetCreatedAt(time.Now().Unix())
				// Set the expiration delay.
				access.SetExpiresIn(int64(jr["expires_in"].(float64)))

				// Set it scope.
				access.SetScope(scope)

				/**
				// Set the custom parameters in the extra field.
				extra, err := json.Marshal(jr["custom_parameter"])
				if err == nil {
					accessToken.SetExtra(extra) // Set as json struct...
				}
				*/

				// Set the expiration...
				expirationTime := time.Unix(access.GetCreatedAt(), 0).Add(time.Duration(access.GetExpiresIn()) * time.Second)

				// Add the expire time.
				addExpireAtData(access.GetId(), expirationTime)

				// Set the client.
				access.SetClient(client)

				// Set the authorization code.
				access.SetAuthorize(authorizationCode)

				// If authorization object are found locally...
				ids := []interface{}{authorizationCode}
				authorizationEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Authorize", "Config", ids)

				if errObj == nil {
					authorization := authorizationEntity.(*Config.OAuth2Authorize)
					if len(authorization.GetRedirectUri()) > 0 {
						access.SetRedirectUri(authorization.GetRedirectUri())
					}
				}

				// Now the refresh token if there some.
				if jr["refresh_token"] != nil {
					refreshToken = jr["refresh_token"].(string)
				}

				if len(refreshToken) > 0 {
					// Here I will create the refresh token.
					refresh := new(Config.OAuth2Refresh)
					refresh.SetId(refreshToken)
					refresh.SetAccess(access)

					// Set into it parent.
					GetServer().GetEntityManager().createEntity(configEntity, "M_refresh", refresh)

					// Set the access
					access.SetRefreshToken(refresh)

					// Save the entity with it refresh token object.
					GetServer().GetEntityManager().saveEntity(accessEntity)
				}

				// Now the id token.
				if jr["id_token"] != nil {
					idToken, err := decodeIdToken(jr["id_token"].(string))
					if err == nil {
						userData := saveIdToken(idToken)
						access.SetUserData(userData)
						// Save the entity with it refresh token object.
						GetServer().GetEntityManager().saveEntity(accessEntity)
					}
				}

				// Save the new access token.
				GetServer().GetEntityManager().saveEntity(configEntity)
			} else {
				// set access to the existing object.
				access = accessEntity.(*Config.OAuth2Access)
			}
		} else if jr["error"] != nil {
			return nil, errors.New(jr["error"].(string))
		}
	}

	return access, nil
}

/**
 * That function is use to decode id token.
 */
func decodeIdToken(encoded string) (*IDToken, error) {
	parts := strings.Split(encoded, ".")

	var val []byte
	var err error

	// Read the body part.
	val, err = b64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	// I will initialyse the body from it data.
	idToken := new(IDToken)
	err = json.Unmarshal(val, idToken)
	if err != nil {
		return nil, err
	}

	// Test if the token is expired.
	if time.Unix(idToken.Expiration, 0).Before(time.Now()) {
		return idToken, errors.New("The token is expired!")
	}

	// The issuer configuration address.
	// TODO implement the https server and remove comment after.
	// http://www.kaihag.com/https-and-go/
	//issuerConfigAddress := "https://" + idToken.Issuer + "/.well-known/openid-configuration"

	return idToken, nil //validateIdToken(encoded, issuerConfigAddress)
}

/**
 * That function is use to validate a token id.
 */
func validateIdToken(tokenStr string, issuerConfigAddress string) error {

	// The original values.
	w := strings.Split(tokenStr, ".")

	h_, s_ := w[0], w[2]

	if m := len(h_) % 4; m != 0 {
		h_ += strings.Repeat("=", 4-m)
	}
	if m := len(s_) % 4; m != 0 {
		s_ += strings.Repeat("=", 4-m)
	}

	//extract kid from token header
	var header interface{}
	headerOauth, err := b64.URLEncoding.DecodeString(h_)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(string(headerOauth)), &header)
	if err != nil {
		return err
	}

	kid := header.(map[string]interface{})["kid"]

	// Now I will retreive the open-id configuration.

	// Retreive the issuer configuration.
	client := &http.Client{}
	req, _ := http.NewRequest("GET", issuerConfigAddress, nil)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	issuerConfig := make(map[string]interface{}, 0)
	json.Unmarshal(bodyBytes, &issuerConfig)

	// Now I will retreive the public keys.
	req, _ = http.NewRequest("GET", issuerConfig["jwks_uri"].(string), nil)

	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pulbicKeys := make(map[string]interface{}, 0)
	json.Unmarshal(bodyBytes, &pulbicKeys)

	// So now i got the isser configuration and it's public keys I can validate
	// the id token.
	keys := pulbicKeys["keys"].([]interface{})
	var publicKey map[string]interface{}
	for i := 0; i < len(keys); i++ {
		if keys[i].(map[string]interface{})["kid"].(string) == kid {
			publicKey = keys[i].(map[string]interface{})
			break
		}
	}

	// If the public key dosent exist.
	if publicKey == nil {
		return errors.New("No public key with id " + kid.(string) + " was found!")
	}

	//build the google pub key
	nStr := publicKey["n"].(string)
	if m := len(nStr) % 4; m != 0 {
		nStr += strings.Repeat("=", 4-m)
	}

	decN, err := b64.URLEncoding.DecodeString(nStr)
	if err != nil {
		return err
	}

	n := big.NewInt(0)
	n.SetBytes(decN)
	eStr := publicKey["e"].(string)
	if m := len(eStr) % 4; m != 0 {
		eStr += strings.Repeat("=", 4-m)
	}

	decE, err := b64.URLEncoding.DecodeString(eStr)
	if err != nil {
		return err
	}

	var eBytes []byte
	if len(decE) < 8 {
		eBytes = make([]byte, 8-len(decE), 8)
		eBytes = append(eBytes, decE...)
	} else {
		eBytes = decE
	}

	eReader := bytes.NewReader(eBytes)
	var e uint64
	err = binary.Read(eReader, binary.BigEndian, &e)
	if err != nil {
		return err
	}

	pKey := rsa.PublicKey{N: n, E: int(e)}
	toHash := w[0] + "." + w[1]
	digestOauth, err := b64.URLEncoding.DecodeString(s_)

	hasherOauth := sha256.New()
	hasherOauth.Write([]byte(toHash))

	// verification of the signature
	err = rsa.VerifyPKCS1v15(&pKey, crypto.SHA256, hasherOauth.Sum(nil), digestOauth)
	if err != nil {
		fmt.Printf("Error verifying key %s", err.Error())
		return err
	}

	return nil
}

/**
 * Retreive access token from a given url.
 */
func RetrieveToken(clientID, clientSecret, tokenURL string, v url.Values) (map[string]interface{}, error) {

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	pclient := &http.Client{}
	r, err := pclient.Do(req)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("oauth2: cannot fetch token: %v", err)
	}

	if code := r.StatusCode; code < 200 || code > 299 {
		return nil, fmt.Errorf("oauth2: cannot fetch token: %v\nResponse: %s", r.Status, body)
	}

	// values return.
	token := make(map[string]interface{}, 0)

	content, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch content {
	case "application/x-www-form-urlencoded", "text/plain":
		vals, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}

		token["access_token"] = vals.Get("access_token")
		token["token_type"] = vals.Get("token_type")
		token["refresh_token"] = vals.Get("refresh_token")
		token["id_token"] = vals.Get("id_token")

		e := vals.Get("expires_in")

		if e == "" {
			// TODO(jbd): Facebook's OAuth2 implementation is broken and
			// returns expires_in field in expires. Remove the fallback to expires,
			// when Facebook fixes their implementation.
			e = vals.Get("expires")
		}
		expires, _ := strconv.Atoi(e)
		if expires != 0 {
			token["expires_in"] = time.Now().Add(time.Duration(expires) * time.Second)
		}

	default:
		if err = json.Unmarshal(body, &token); err != nil {
			return nil, err
		}
	}

	// Don't overwrite `RefreshToken` with an empty value
	// if this was a token refreshing request.
	if token["refresh_token"] == "" {
		token["refresh_token"] = v.Get("refresh_token")
	}

	return token, nil
}

/**
 * Download ressource specify with a given query.
 * TODO interface http api call here...
 */
func DownloadRessource(query string, accessToken string, tokenType string) (map[string]interface{}, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", query, nil)
	req.Header.Add("Authorization", tokenType+" "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		output := make(map[string]interface{}, 0)
		json.Unmarshal(bodyBytes, &output)
		return output, nil
	}

	return nil, err
}

////////////////////////////////////////////////////////////////////////////////
// Auth Http handler.
////////////////////////////////////////////////////////////////////////////////

/**
 * If the use is not logged...
 */
func HandleAuthenticationPage(ar *osin.AuthorizeRequest, w http.ResponseWriter, r *http.Request) bool {
	r.ParseForm()
	user := r.Form.Get("login")
	pwd := r.Form.Get("password")
	state := r.URL.Query()["state"][0]

	var sessionId string
	var messageId string
	if len(strings.Split(state, ":")) == 4 {
		messageId = strings.Split(state, ":")[0]
		sessionId = strings.Split(state, ":")[1]
	}

	// If the user is already logged in i will return true.
	if len(sessionId) > 0 {
		if GetServer().GetSessionManager().getActiveSessionById(sessionId) != nil {
			if GetServer().GetSessionManager().getActiveSessionById(sessionId).GetAccountPtr() != nil {
				return true
			}
		}
	}

	// Here I will authenticate the user...
	if r.Method == "POST" && len(sessionId) > 0 && len(messageId) > 0 {
		// Try to open a new session user here...
		session := GetServer().GetSessionManager().Login(user, pwd, "", messageId, sessionId)
		if session != nil {
			return true
		} else {
			return false
		}
	}

	w.Write([]byte("<html><body>"))

	// if the user is no logged...
	w.Write([]byte(fmt.Sprintf("<form class='oauth2-form' action=\"/authorize?%s\" method=\"POST\">", r.URL.RawQuery)))
	w.Write([]byte("Login: <input type=\"text\" name=\"login\" /><br/>"))
	w.Write([]byte("Password: <input type=\"password\" name=\"password\" /><br/>"))
	w.Write([]byte("<input type=\"submit\"/>"))
	w.Write([]byte("</form>"))
	w.Write([]byte("</body></html>"))

	return false
}

/**
 * Ask for Authorization.
 */
func HandleAuthorizationPage(ar *osin.AuthorizeRequest, w http.ResponseWriter, r *http.Request) bool {
	// Here I will
	r.ParseForm()
	answer := r.FormValue("submitbutton")

	if answer == "Yes" {
		// Here the user accept the access to the ressource
		return true
	} else if answer == "No" {
		// Here The user refuse the acess to the ressource.
		// Here I will write the response that specifie that the user refuse
		// the authorization to the ressource.
		return false
	}

	// Here the user is logged and he need to ask if he give permission to
	// the request.
	w.Write([]byte("<html><body>"))
	w.Write([]byte(fmt.Sprintf("<form class='oauth2-form' action=\"/authorize?%s\" method=\"POST\">", r.URL.RawQuery)))
	w.Write([]byte("<span>Did you accept?</span></br>"))
	w.Write([]byte("<input type=\"submit\" name=\"submitbutton\" value=\"Yes\"/>"))
	w.Write([]byte("<input type=\"submit\" name=\"submitbutton\" value=\"No\"/>"))
	w.Write([]byte("</form>"))
	w.Write([]byte("</body></html>"))

	return false
}

/**
 * OAuth Authorization handler.
 */
func AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	server := GetServer().GetOAuth2Manager().m_server
	if server == nil {
		fmt.Printf("ERROR: %s\n", errors.New("no OAuth2 service configure!"))
		fmt.Fprintf(w, "no OAuth2 service configure!")
		return
	}
	resp := server.NewResponse()
	defer resp.Close()

	if ar := server.HandleAuthorizeRequest(resp, r); ar != nil {
		// The state contain the messageId:sessionId:clientId
		state := r.URL.Query()["state"][0]
		var sessionId string
		var messageId string
		var clientId string
		if len(strings.Split(state, ":")) == 4 {
			messageId = strings.Split(state, ":")[0]
			sessionId = strings.Split(state, ":")[1]
			clientId = strings.Split(state, ":")[2]
		}
		ids := []interface{}{clientId}

		clientEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids)
		if errObj != nil {
			// Print the message.
			log.Println(errObj.GetBody())
			return
		}
		client := clientEntity.(*Config.OAuth2Client)

		if !HandleAuthenticationPage(ar, w, r) {
			return
		}

		if !HandleAuthorizationPage(ar, w, r) {
			if len(r.FormValue("submitbutton")) == 0 {
				// No answer was given yet...
				return
			} else {
				// In that case the user refuse the authorization.
				ar.Authorized = false
				ar.RedirectUri = client.GetRedirectUri()
				server.FinishAuthorizeRequest(resp, r, ar)

				// Here I will create an error message...
				to := make([]*WebSocketConnection, 1)
				to[0] = GetServer().getConnectionById(sessionId)
				var errData []byte
				authorizationDenied := NewErrorMessage(messageId, 1, "Permission Denied by user", errData, to)
				GetServer().getProcessor().m_incomingChannel <- authorizationDenied
				return
			}
		}

		// OpenId part.
		scopes := make(map[string]bool)
		for _, s := range strings.Fields(ar.Scope) {
			scopes[s] = true
		}

		// If the "openid" connect scope is specified, attach an ID Token to the
		// authorization response.

		// The ID Token will be serialized and signed during the code for token exchange.
		if scopes["openid"] {
			config := GetServer().GetConfigurationManager().getServiceConfigurationById(GetServer().GetOAuth2Manager().getId())
			issuer := config.GetHostName() + ":" + strconv.Itoa(config.GetPort())

			// These values would be tied to the end user authorizing the client.
			now := time.Now()
			idToken := IDToken{
				Issuer:     issuer,
				UserID:     "",
				ClientID:   ar.Client.GetId(),
				Expiration: now.Add(time.Hour).Unix(),
				IssuedAt:   now.Unix(),
				Nonce:      r.URL.Query().Get("nonce"),
			}

			// From the session I will retreive the user session->account->user
			session := GetServer().GetSessionManager().getActiveSessionById(sessionId)

			// If the scope contain a profile
			if scopes["profile"] {
				idToken.UserID = session.GetAccountPtr().GetId()
				idToken.GivenName = session.GetAccountPtr().GetName()
				if session.GetAccountPtr().GetUserRef() != nil {
					idToken.Name = session.GetAccountPtr().GetUserRef().GetFirstName()
					idToken.FamilyName = session.GetAccountPtr().GetUserRef().GetLastName()
				}
				idToken.Locale = "us"
			}

			// Now if the scope contain a email...
			if scopes["email"] {
				t := true
				idToken.Email = session.GetAccountPtr().GetEmail()
				idToken.EmailVerified = &t
			}

			// NOTE: The storage must be able to encode and decode this object.
			ar.UserData = &idToken
		}

		// The user give the authorization.
		ar.State = state // Set the state to.
		ar.Authorized = true
		ar.RedirectUri = client.GetRedirectUri()
		server.FinishAuthorizeRequest(resp, r, ar)

		// Here I will create a response for the authorization.
		results := make([]*MessageData, 1)
		data := new(MessageData)
		data.TYPENAME = "Server.MessageData"
		data.Name = "code"
		data.Value = resp.Output["code"]
		results[0] = data
		to := make([]*WebSocketConnection, 1)
		to[0] = GetServer().getConnectionById(sessionId)
		authorizationAccept, _ := NewResponseMessage(messageId, results, to)

		GetServer().getProcessor().m_incomingChannel <- authorizationAccept

		redirectUrl, err := resp.GetRedirectUrl()
		if err == nil {
			// Redirect to the url here.
			client := &http.Client{}
			req, _ := http.NewRequest("GET", redirectUrl, nil)
			client.Do(req)
		}
	}
	if resp.IsError && resp.InternalError != nil {
		fmt.Printf("ERROR: %s\n", resp.InternalError)
		// Here I will create an error message to complete the workflow.
	}
	osin.OutputJSON(resp, w, r)
}

/**
 * Access token endpoint
 */
func TokenHandler(w http.ResponseWriter, r *http.Request) {
	server := GetServer().GetOAuth2Manager().m_server
	if server == nil {
		fmt.Printf("ERROR: %s\n", errors.New("no OAuth2 service configure!"))
		fmt.Fprintf(w, "no OAuth2 service configure!")
		return
	}
	resp := server.NewResponse()
	defer resp.Close()

	if ar := server.HandleAccessRequest(resp, r); ar != nil {
		switch ar.Type {
		case osin.AUTHORIZATION_CODE:
			ar.Authorized = true
		case osin.REFRESH_TOKEN:
			ar.Authorized = true
		case osin.PASSWORD:
			if ar.Username == "test" && ar.Password == "test" {
				ar.Authorized = true
			}
		case osin.CLIENT_CREDENTIALS:
			ar.Authorized = true
		case osin.ASSERTION:
			if ar.AssertionType == "urn:osin.example.complete" && ar.Assertion == "osin.data" {
				ar.Authorized = true
			}
		}

		server.FinishAccessRequest(resp, r, ar)

		// If an ID Token was encoded as the UserData, serialize and sign it.
		if idToken, ok := ar.UserData.(*IDToken); ok && idToken != nil {
			encodeIDToken(resp, idToken, GetServer().GetOAuth2Manager().m_jwtSigner)
		}
	}
	if resp.IsError && resp.InternalError != nil {
		fmt.Printf("ERROR: %s\n", resp.InternalError)
	}
	if !resp.IsError {
		resp.Output["custom_parameter"] = 19923
	}

	osin.OutputJSON(resp, w, r)
}

// encodeIDToken serializes and signs an ID Token then adds a field to the token response.
func encodeIDToken(resp *osin.Response, idToken *IDToken, singer jose.Signer) {
	resp.InternalError = func() error {
		payload, err := json.Marshal(idToken)
		if err != nil {
			return fmt.Errorf("failed to marshal token: %v", err)
		}
		jws, err := GetServer().GetOAuth2Manager().m_jwtSigner.Sign(payload)
		if err != nil {
			return fmt.Errorf("failed to sign token: %v", err)
		}
		raw, err := jws.CompactSerialize()
		if err != nil {
			return fmt.Errorf("failed to serialize token: %v", err)
		}
		resp.Output["id_token"] = raw
		return nil
	}()

	// Record errors as internal server errors.
	if resp.InternalError != nil {
		resp.IsError = true
		resp.ErrorId = osin.E_SERVER_ERROR
	}
}

/**
 * Information endpoint
 */
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	server := GetServer().GetOAuth2Manager().m_server
	if server == nil {
		fmt.Printf("ERROR: %s\n", errors.New("no OAuth2 service configure!"))
		fmt.Fprintf(w, "no OAuth2 service configure!")
		return
	}
	resp := server.NewResponse()
	defer resp.Close()

	if ir := server.HandleInfoRequest(resp, r); ir != nil {
		server.FinishInfoRequest(resp, r, ir)
	}
	osin.OutputJSON(resp, w, r)
}

/**
 * This is the client redirect handler.
 */
func AppAuthCodeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	errorCode := r.Form.Get("error")
	state := r.Form.Get("state")

	log.Println("-----> request ", r)
	// Send authentication end message...
	var clientId string
	var sessionId string
	var messageId string
	var scope string

	if len(strings.Split(state, ":")) == 4 {
		messageId = strings.Split(state, ":")[0]
		sessionId = strings.Split(state, ":")[1]
		clientId = strings.Split(state, ":")[2]
		scope = strings.Split(state, ":")[3]
	}

	log.Println("------> 1452 Autorize callback call. clientId ", clientId, "sessionId", sessionId, "messageId", messageId, "scope", scope)

	// I will get a reference to the client who generate the request.
	ids := []interface{}{clientId}
	clientEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids)
	if errObj != nil {
		log.Println(errObj.GetBody())
		return
	}
	client := clientEntity.(*Config.OAuth2Client)

	if len(errorCode) != 0 {
		// The authorization fail!
		log.Println("--------> create access error.")
		errorDescription := r.Form.Get("error_description")
		to := make([]*WebSocketConnection, 1)
		to[0] = GetServer().getConnectionById(sessionId)
		var errData []byte
		accessDenied := NewErrorMessage(messageId, 1, errorDescription, errData, to)
		GetServer().getProcessor().m_incomingChannel <- accessDenied
	} else {
		log.Println("--------> try to create access grant response whit id: ", messageId)
		access, err := createAccessToken("authorization_code", client, r.Form.Get("code"), "", scope)

		if err == nil {
			log.Println("--------> create access grant response.", messageId)
			var idTokenUuid string
			if access.GetUserData() != nil {
				idTokenUuid = access.GetUserData().GetUuid()
			}

			// Send back the response to access request.
			results := make([]*MessageData, 2)
			// The access uuid
			data0 := new(MessageData)
			data0.TYPENAME = "Server.MessageData"
			data0.Name = "accessUuid"
			data0.Value = access.GetUuid()
			results[0] = data0

			// The id token uuid.
			data1 := new(MessageData)
			data1.TYPENAME = "Server.MessageData"
			data1.Name = "idTokenUuid"
			data1.Value = idTokenUuid
			results[1] = data1

			to := make([]*WebSocketConnection, 2)
			to[0] = GetServer().getConnectionById(sessionId)
			accessGrantResp, _ := NewResponseMessage(messageId, results, to)
			GetServer().getProcessor().m_incomingChannel <- accessGrantResp

			// Set the values inside a string array and send it over channel.
			values := make([]string, 2)
			values[0] = idTokenUuid
			values[1] = access.GetUuid()

			channels[messageId] <- values

		} else {
			// send error
			log.Println("--------> access error: ", err)
			to := make([]*WebSocketConnection, 1)
			to[0] = GetServer().getConnectionById(sessionId)
			var errData []byte
			accessDenied := NewErrorMessage(messageId, 1, err.Error(), errData, to)
			GetServer().getProcessor().m_incomingChannel <- accessDenied
			channels[messageId] <- make([]string, 0) // deblock the channel...
		}
	}

}

// TODO remove the hidden field from the results.
// manage action permission and authentication.
/**
* That function handle http query as form of what so called API.
* exemple of use.

  Get all entity prototype from CargoEntities
  ** note the access_token can change over time.
  http://mon176:9696/api/Server/EntityManager/GetEntityPrototypes?storeId=CargoEntities&access_token=C4X_UsRXRCqwqsWfuEdgFA

  Get an entity object with a given uuid.
  * Note because % is in the uuid string it must be escape with %25 so here
  	 the uuid is CargoEntities.Action%7facc2a5-dcb7-4ae7-925a-fb0776a9da00
  http://localhost:9393/api/Server/EntityManager/GetObjectByUuid?p0=CargoEntities.Action%257facc2a5-dcb7-4ae7-925a-fb0776a9da00
*/
func HttpQueryHandler(w http.ResponseWriter, r *http.Request) {

	// So the request will contain...
	// The last tow parameters must be empty because we don't use the websocket
	// here.
	ids := strings.Split(r.URL.Path[5:], "/")

	// The action to be execute.
	var errObj *CargoEntities.Error
	var action *CargoEntities.Action

	// I will get the action entity from the values.
	if len(ids) == 3 {
		var entity Entity
		ids := []interface{}{ids[0] + "." + ids[1] + "." + ids[2]}
		entity, errObj = GetServer().GetEntityManager().getEntityById("CargoEntities.Action", "CargoEntities", ids)
		if errObj != nil {
			w.Header().Set("Content-Type", "application/text")
			w.Write([]byte(errObj.GetBody()))
			return
		}
		action = entity.(*CargoEntities.Action)
	} else {
		msg := "Incorrect number of parameter got " + strconv.Itoa(len(ids)) + " expected 3. "
		w.Write([]byte(msg))
		return
	}

	// The array of parameters.
	params := make([]interface{}, 0)

	// Here I will get the service object...
	service, err := Utility.CallMethod(GetServer(), "Get"+ids[1], params)
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte(err.(error).Error()))
		return
	}

	// The parameter values.
	values := r.URL.Query()

	// Now I will try to call the action.
	// First of all I will create the parameters.
	// The last tow parameters are always sessionId and messageId and
	// are use by the websocket and not the http.

	for i := 0; i < len(action.GetParameters()); i++ {
		// Here I will make type mapping...
		param := action.GetParameters()[i]

		if param.IsArray() {
			// Here the values inside the query must be parse...

		} else {

			// Not an array here.
			if param.GetType() == "string" {
				// The first parameter is a string.
				//r.URL.Query()
				v := values.Get(param.GetName())
				log.Println(v, reflect.TypeOf(v).Kind())
				if reflect.TypeOf(v).Kind() != reflect.String {
					w.Header().Set("Content-Type", "application/text")
					msg := "Incorrect parameter value for param " + param.GetName()
					w.Write([]byte(msg))
					return // report error here.
				}

				// Append the parameter to the parameter list.
				params = append(params, v)
			}
		}
	}

	var accessTokenId string

	// Try to get access token from the list of parameters.
	accessTokenId = values.Get("access_token")

	if len(accessTokenId) == 0 {
		values := strings.Split(r.Header.Get("Authorization"), " ")
		log.Println("values: ", values)
		if len(values) == 2 {
			if strings.ToLower(values[0]) == "bearer" {
				accessTokenId = values[1]
			}
		}
	}

	if len(accessTokenId) == 0 {
		if len(r.Form["access_token"]) == 1 {
			accessTokenId = r.Form["access_token"][0]
		}
	}

	// The access token variable.
	var accessToken *Config.OAuth2Access

	if len(accessTokenId) > 0 {
		ids := []interface{}{accessTokenId}
		entity, err := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", ids)
		if err != nil {
			w.Header().Set("Content-Type", "application/text")
			w.Write([]byte(err.GetBody()))
			return
		}

		// Get the access token here.
		accessToken = entity.(*Config.OAuth2Access)

	} else {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Access denied!"))
		return
	}

	log.Println("open id token: ", accessToken.GetUserData())

	// Here I will call the function on the service.
	results, err := Utility.CallMethod(service, ids[2], params)
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte(err.(error).Error()))
		return
	}

	// Here I will get the res
	resultStr, err := json.Marshal(results)
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte(err.(error).Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resultStr, _ = Utility.PrettyPrint(resultStr)
	w.Write(resultStr)
}

////////////////////////////////////////////////////////////////////////////////
// OAuth2 Store Implementation.
////////////////////////////////////////////////////////////////////////////////
type OAuth2Store struct {
}

// Create and intialyse the OAuth2 Store.
func newOauth2Store() *OAuth2Store {
	store := new(OAuth2Store)
	return store
}

/**
 * Close the store.
 */
func (this *OAuth2Store) Close() {

}

/**
 * Return pointer...
 */
func (s *OAuth2Store) Clone() osin.Storage {
	return s
}

/**
 * Retrun a given client.
 */
func (this *OAuth2Store) GetClient(id string) (osin.Client, error) {
	// From the list of registred client I will retreive the client
	// with the given id.
	ids := []interface{}{id}
	clientEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids)

	if errObj != nil {
		return nil, errors.New("No client found with id " + id)
	}

	// Set the client.
	client := clientEntity.(*Config.OAuth2Client)

	// Create the corresponding client.
	c := new(osin.DefaultClient)
	c.Id = client.M_id
	c.Secret = client.M_secret
	c.RedirectUri = client.M_redirectUri
	c.UserData = client.M_extra
	return c, nil
}

/**
 * Set the client value.
 */
func (this *OAuth2Store) SetClient(id string, client osin.Client) error {
	// The configuration.
	configEntity := GetServer().GetConfigurationManager().getOAuthConfigurationEntity()

	// Create a client configuration from the osin.Client.
	c := new(Config.OAuth2Client)
	c.M_id = id
	c.M_extra = client.GetUserData().([]uint8)
	c.M_secret = client.GetSecret()
	c.M_redirectUri = client.GetRedirectUri()

	// append a new client.
	GetServer().GetEntityManager().createEntity(configEntity, "M_client", c)

	return nil
}

/**
 * Save a given autorization.
 */
func (this *OAuth2Store) SaveAuthorize(data *osin.AuthorizeData) error {

	// Get the config entity
	configEntity := GetServer().GetConfigurationManager().getOAuthConfigurationEntity()

	a := new(Config.OAuth2Authorize)

	// Set the client.
	ids := []interface{}{data.Client.GetId()}
	c, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids)

	if errObj != nil {
		return errors.New("No client found with id " + data.Client.GetId())
	}

	// Set the value from the data.
	a.SetId(data.Code)

	// Initialyse getUuid, setEntity etc...
	GetServer().GetEntityManager().setEntity(a)
	log.Println("---> expire In ", data.ExpiresIn)
	a.SetExpiresIn(int64(data.ExpiresIn))
	a.SetScope(data.Scope)
	a.SetRedirectUri(data.RedirectUri)
	a.SetState(data.State)
	a.SetCreatedAt(data.CreatedAt.Unix())

	// Set the client.
	a.SetClient(c.(*Config.OAuth2Client))

	// Save the id token if found.
	if data.UserData != nil {
		idToken := saveIdToken(data.UserData.(*IDToken))
		// Set into the user user
		a.SetUserData(idToken)
	}

	// append a new Authorize.
	GetServer().GetEntityManager().createEntity(configEntity, "M_authorize", a)

	// Add expire data.
	if err := addExpireAtData(data.Code, data.ExpireAt()); err != nil {
		return err
	}

	return nil
}

// LoadAuthorize looks up AuthorizeData by a code.
// Client information MUST be loaded together.
// Optionally can return error if expired.
func (this *OAuth2Store) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	ids := []interface{}{code}
	authorizeEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Authorize", "Config", ids)
	if errObj != nil {
		// No data was found.
		return nil, errors.New("No authorize data found with code " + code)
	}

	// Get the object.
	authorize := authorizeEntity.(*Config.OAuth2Authorize)

	var data *osin.AuthorizeData
	data = new(osin.AuthorizeData)
	data.Code = code
	data.ExpiresIn = int32(authorize.GetExpiresIn())
	data.Scope = authorize.GetScope()
	data.RedirectUri = authorize.GetRedirectUri()
	data.State = authorize.GetState()
	data.CreatedAt = time.Unix(authorize.GetCreatedAt(), 0)

	// set the user data here.
	if authorize.GetUserData() != nil {
		data.UserData = loadIdToken(authorize.GetUserData())
	}

	c, err := this.GetClient(authorize.GetClient().GetId())
	if err != nil {
		return nil, err
	}
	data.Client = c

	// Now I will test the expiration time.
	if data.ExpireAt().Before(time.Now()) {
		return nil, errors.New("Token expired at " + data.ExpireAt().String())
	}

	return data, nil

}

/**
 * Remove authorize from the db.
 */
func (this *OAuth2Store) RemoveAuthorize(code string) error {

	entity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Authorize", "Config", []interface{}{code})
	if entity != nil {
		GetServer().GetEntityManager().deleteEntity(entity)
		// Remove the related expire code if there one.
		entity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Expires", "Config", []interface{}{code})
		if entity != nil {
			GetServer().GetEntityManager().deleteEntity(entity)
		}
		return nil
	}

	return errors.New("No authorization with code " + code + " was found!")
}

/**
 * Load a given id token.
 */
func loadIdToken(idToken *Config.OAuth2IdToken) *IDToken {
	it := new(IDToken)
	it.ClientID = idToken.GetClient().GetId()
	it.Email = idToken.GetEmail()
	emailVerified := idToken.IsEmailVerified()
	it.EmailVerified = &emailVerified
	it.Expiration = idToken.GetExpiration()
	it.FamilyName = idToken.GetFamilyName()
	it.GivenName = idToken.GetGivenName()
	it.IssuedAt = idToken.GetIssuedAt()
	it.Issuer = idToken.GetIssuer()
	it.Locale = idToken.GetLocal()
	it.Name = idToken.GetName()
	it.Nonce = idToken.GetNonce()
	it.UserID = idToken.GetId()
	return it
}

/**
 * Save Token id.
 */
func saveIdToken(data *IDToken) *Config.OAuth2IdToken {
	// Get needed entities.
	configEntity := GetServer().GetConfigurationManager().getOAuthConfigurationEntity()
	ids := []interface{}{data.ClientID}
	clientEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids)
	if errObj != nil {
		log.Println(errObj.GetBody())
		return nil
	}
	client := clientEntity.(*Config.OAuth2Client)

	// Create id token (OpenId)
	var idToken *Config.OAuth2IdToken
	entity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2IdToken", "Config", []interface{}{data.UserID})

	if entity != nil {
		// Update value...
		idToken = entity.(*Config.OAuth2IdToken)
		idToken.SetEmail(data.Email)
		idToken.SetEmailVerified(*data.EmailVerified)
		idToken.SetExpiration(data.Expiration)
		idToken.SetFamilyName(data.FamilyName)
		idToken.SetGivenName(data.GivenName)
		idToken.SetIssuedAt(data.IssuedAt)
		idToken.SetIssuer(data.Issuer)
		idToken.SetLocal(data.Locale)
		idToken.SetName(data.Name)
		idToken.SetNonce(data.Nonce)
		GetServer().GetEntityManager().saveEntity(idToken)

	} else {
		// Create the id token.
		idToken = new(Config.OAuth2IdToken)
		idToken.SetId(data.UserID)
		GetServer().GetEntityManager().setEntity(idToken)
		idToken.SetClient(client)
		idToken.SetEmail(data.Email)
		idToken.SetEmailVerified(*data.EmailVerified)
		idToken.SetExpiration(data.Expiration)
		idToken.SetFamilyName(data.FamilyName)
		idToken.SetGivenName(data.GivenName)
		idToken.SetIssuedAt(data.IssuedAt)
		idToken.SetIssuer(data.Issuer)
		idToken.SetLocal(data.Locale)
		idToken.SetName(data.Name)
		idToken.SetNonce(data.Nonce)

		// Save into the config.
		GetServer().GetEntityManager().createEntity(configEntity, "M_ids", idToken)
	}
	return idToken
}

/**
 * Save the access Data.
 */
func (this *OAuth2Store) SaveAccess(data *osin.AccessData) error {
	configEntity := GetServer().GetConfigurationManager().getOAuthConfigurationEntity()
	ids := []interface{}{data.AccessToken}
	accessEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", ids)
	var access *Config.OAuth2Access
	if errObj != nil {
		access = new(Config.OAuth2Access)
		access.SetId(data.AccessToken)
		accessEntity, _ = GetServer().GetEntityManager().createEntity(configEntity, "M_access", access)
	}

	// Cast entity to *Config.OAuth2Access
	access = accessEntity.(*Config.OAuth2Access)

	prev := ""
	authorizeData := &osin.AuthorizeData{}

	if data.AccessData != nil {
		prev = data.AccessData.AccessToken
	}

	if data.AuthorizeData != nil {
		authorizeData = data.AuthorizeData
	}

	if data.Client == nil {
		return errors.New("data.Client must not be nil")
	}

	// Set the client.
	ids_ := []interface{}{data.Client.GetId()}
	clientEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Client", "Config", ids_)
	if errObj != nil {
		log.Println(errObj.GetBody())
		return errors.New(errObj.GetBody())
	}
	client := clientEntity.(*Config.OAuth2Client)
	access.SetClient(client)

	// Set the authorization.
	if authorizeData == nil {
		return errors.New("authorize data must not be nil")
	}

	// Keep only the code here no the object because it will be deleted after
	// the access creation.
	access.SetAuthorize(authorizeData.Code)

	// Set other values.
	access.SetPrevious(prev)

	access.SetExpiresIn(int64(data.ExpiresIn))
	access.SetScope(data.Scope)
	access.SetRedirectUri(data.RedirectUri)

	// Set the unix time.
	access.SetCreatedAt(data.CreatedAt.Unix())

	if data.UserData != nil {
		idToken := saveIdToken(data.UserData.(*IDToken))
		// Set into the user user
		access.SetUserData(idToken)
	}

	// Add expire data.
	if err := addExpireAtData(data.AccessToken, data.ExpireAt()); err != nil {
		return err
	}

	// Now the refresh token.
	if len(data.RefreshToken) > 0 {
		// In that case I will save the refresh token.
		ids := []interface{}{data.RefreshToken}
		refreshEntity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Refresh", "Config", ids)
		var refresh *Config.OAuth2Refresh
		if refreshEntity == nil {
			refresh = new(Config.OAuth2Refresh)
			refresh.SetId(data.RefreshToken)
			// save the access.
			refreshEntity, _ = GetServer().GetEntityManager().createEntity(configEntity, "M_refresh", refresh)
		}

		// Cast refresh entity to *Config.OAuth2Refresh
		refresh = refreshEntity.(*Config.OAuth2Refresh)

		// Here the refresh token dosent exist so i will create it.
		refresh.SetAccess(access) // Ref.

		// Set the access
		access.SetRefreshToken(refresh) // Ref

		GetServer().GetEntityManager().saveEntity(access)
		GetServer().GetEntityManager().saveEntity(refresh)
	}

	return nil
}

/**
 * Load the access for a given code.
 */
func (this *OAuth2Store) LoadAccess(code string) (*osin.AccessData, error) {
	ids := []interface{}{code}
	accessEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", ids)
	if errObj != nil {
		return nil, errors.New("No access found with code " + code)
	}

	var access *osin.AccessData
	a := accessEntity.(*Config.OAuth2Access)

	access = new(osin.AccessData)
	access.AccessToken = code
	access.ExpiresIn = int32(a.GetExpiresIn())
	access.Scope = a.GetScope()
	access.RedirectUri = a.GetRedirectUri()
	access.CreatedAt = time.Unix(int64(a.GetCreatedAt()), 0)

	if a.GetUserData() != nil {
		access.UserData = loadIdToken(a.GetUserData())
	}

	// The refresh token
	if a.GetRefreshToken() != nil {
		access.RefreshToken = a.GetRefreshToken().GetId()
	}

	// The access token
	access.AccessToken = a.GetId()

	// Now the client
	c, err := this.GetClient(a.GetClient().GetId())
	if err != nil {
		return nil, err
	}
	access.Client = c

	// The authorize
	auth, err := this.LoadAuthorize(a.GetAuthorize())
	if err != nil {
		// Try to get authorize...
		refreshToken := a.GetRefreshToken()
		if refreshToken == nil {
			return nil, err
		}

		// Get the configuration object.
		config := GetServer().GetConfigurationManager().getActiveConfigurations()

		// So here The refresh token is valid i will create a new authorization
		authorizeData := new(osin.AuthorizeData)
		authorizeData.Client = c

		// I will reuse the say authorization code...
		authorizeData.Code = a.GetAuthorize()
		authorizeData.CreatedAt = time.Now()
		authorizeData.ExpiresIn = int32(config.GetOauth2Configuration().GetAuthorizationExpiration())
		authorizeData.RedirectUri = c.GetRedirectUri()
		authorizeData.Scope = a.GetScope()

		// TODO does is needed?
		authorizeData.State = ""

		// Set the new authorization here...
		access.AuthorizeData = authorizeData
	}
	access.AuthorizeData = auth

	return access, nil
}

/**
 * Remove an access code.
 */
func (this *OAuth2Store) RemoveAccess(code string) error {
	entity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Access", "Config", []interface{}{code})
	if entity != nil {
		GetServer().GetEntityManager().deleteEntity(entity)
		// Remove the related expire code if there one.
		entity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Expires", "Config", []interface{}{code})
		if entity != nil {
			GetServer().GetEntityManager().deleteEntity(entity)
		}
		return nil
	}
	return errors.New("No access with code " + code + " was found.")
}

/**
 * Load the access data from it refresh code.
 */
func (this *OAuth2Store) LoadRefresh(code string) (*osin.AccessData, error) {
	ids := []interface{}{code}
	refreshEntity, errObj := GetServer().GetEntityManager().getEntityById("Config.OAuth2Refresh", "Config", ids)
	if errObj != nil {
		return nil, errors.New("Now refresh token found with code " + code)
	}

	refresh := refreshEntity.(*Config.OAuth2Refresh)

	// Get the access...
	access, err := this.LoadAccess(refresh.GetAccess().GetId())
	if err != nil {
		// In that case I will create a new access and associated it
		// with the refresh...
		return nil, err
	}

	// Here the access token can be
	return access, err
}

/**
 * Remove refresh.
 */
func (this *OAuth2Store) RemoveRefresh(code string) error {
	entity, _ := GetServer().GetEntityManager().getEntityById("Config.OAuth2Refresh", "Config", []interface{}{code})
	if entity != nil {
		GetServer().GetEntityManager().deleteEntity(entity)
		return nil
	}
	return errors.New("No refresh was found with code " + code)
}
