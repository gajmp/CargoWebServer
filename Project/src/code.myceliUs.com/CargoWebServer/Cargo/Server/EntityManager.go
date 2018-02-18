package Server

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"strconv"
	"strings"

	"code.myceliUs.com/CargoWebServer/Cargo/Entities/CargoEntities"
	//"code.myceliUs.com/CargoWebServer/Cargo/Entities/Config"

	"code.myceliUs.com/Utility"
)

////////////////////////////////////////////////////////////////////////////////
//						Entity Manager
////////////////////////////////////////////////////////////////////////////////

// Struct use to tranfer internal informations about entity
type EntityInfo struct {
	typeName string
	ids      []interface{}
	uuid     string

	// that map is use to cache entitie in memory.
	entities chan []Entity
}

type EntityManager struct {
	// Cache the entitie in memory...
	m_entities map[string]Entity

	// Map of entity (reverse) references in cache
	m_entitiesRefs map[string]map[string][]string

	// Access the tow map via those channel...
	m_setEntityChan    chan Entity
	m_getEntityChan    chan EntityInfo
	m_removeEntityChan chan Entity
}

var entityManager *EntityManager

// Function to be use a function pointer.
var getEntityFct = func(uuid string) (interface{}, error) {
	entity, err := entityManager.getEntityByUuid(uuid)
	if err != nil {
		return nil, errors.New(err.GetBody())
	}
	return entity, nil
}

func (this *Server) GetEntityManager() *EntityManager {
	if entityManager == nil {
		entityManager = newEntityManager()
	}
	return entityManager
}

func newEntityManager() *EntityManager {

	entityManager = new(EntityManager)

	// Create prototypes for config objects and entities objects...
	entityManager.createConfigPrototypes()
	entityManager.createCargoEntitiesPrototypes()
	entityManager.registerConfigObjects()
	entityManager.registerCargoEntitiesObjects()

	// Entity prototype is a dynamic type.
	// ** Dynamic type must have the TYPENAME property!
	Utility.RegisterType((*EntityPrototype)(nil))
	Utility.RegisterType((*Restriction)(nil))
	Utility.RegisterType((*DynamicEntity)(nil))
	Utility.RegisterType((*MessageData)(nil))
	Utility.RegisterType((*TaskInstanceInfo)(nil))
	Utility.RegisterType((*EntityQuery)(nil))

	// The Cache...
	entityManager.m_entities = make(map[string]Entity, 0)
	entityManager.m_entitiesRefs = make(map[string]map[string][]string)

	// Cache accessor.
	entityManager.m_getEntityChan = make(chan EntityInfo, 0)
	entityManager.m_removeEntityChan = make(chan Entity, 0)
	entityManager.m_setEntityChan = make(chan Entity, 0)

	// Cache processing loop...
	go func() {
		entityManager := GetServer().GetEntityManager()
		for {
			select {
			case entityInfo := <-entityManager.m_getEntityChan:
				if len(entityInfo.uuid) > 0 {
					entity := entityManager.m_entities[entityInfo.uuid]
					entities := make([]Entity, 0)
					if entity != nil {
						entities = append(entities, entity)
					}
					entityInfo.entities <- entities
				} else if len(entityInfo.ids) > 0 {
					// The uuid generated here is local and not the entity uuid because the
					// parentUuid is not know... clash can append if entity with same ids and typeName exist at same time.
					id := entityManager.GenerateEntityUUID(entityInfo.typeName, "", entityInfo.ids, "", "")
					entity := entityManager.m_entities[id]
					entities := make([]Entity, 0)
					if entity != nil {
						entities = append(entities, entity)
					}
					entityInfo.entities <- entities
				} else if len(entityInfo.typeName) > 0 {
					// entityInfo.entities <- nil
					entities := make([]Entity, 0)
					// Here I will append the list of entities in the results.
					for _, entity := range entityManager.m_entities {
						if entity.GetTypeName() == entityInfo.typeName {
							entities = append(entities, entity)
						}
					}
					entityInfo.entities <- entities
				} else {
					entityInfo.entities <- nil
				}
			case entity := <-entityManager.m_removeEntityChan:

				// If the entity was referenced by entity in the cache
				// I will remove it from this entity
				ref := entityManager.m_entitiesRefs[entity.GetUuid()]
				if ref != nil {
					for field, uuids := range ref {
						for i := 0; i < len(uuids); i++ {
							// Only entities in the case must be updates.
							if entityManager.m_entities[uuids[i]] != nil {
								// In that case the entity referenced is in the
								// cache map so I need to update it.
								entity_ := entityManager.m_entities[uuids[i]]
								prototype, _ := entityManager.getEntityPrototype(entity_.GetTypeName(), strings.Split(entity_.GetTypeName(), ".")[0])
								if reflect.TypeOf(entity_).String() == "*Server.DynamicEntity" {
									refs_ := entity_.(*DynamicEntity).getValue(field)
									if refs_ != nil {
										refs := make([]string, 0)
										for j := 0; j < len(refs_.([]string)); j++ {
											if refs_.([]string)[j] != entity.GetUuid() {
												refs = append(refs, refs_.([]string)[j])
											}
										}
										entity_.(*DynamicEntity).setValue(field, refs)
									}
								} else {
									removeName := strings.Replace(field, "M_", "", -1)
									removeName = "Remove" + strings.ToUpper(removeName[0:1]) + removeName[1:] + "Ref"
									params := make([]interface{}, 1)
									params[0] = entity
									_, err := Utility.CallMethod(entity_, removeName, params)
									if err == nil {
										// Here I will send update entity event...
										log.Println("--------> remove ref ", entity_.GetUuid(), field, entity.GetUuid())
									}
								}

								// Sent the update event.
								eventData := make([]*MessageData, 2)
								msgData0 := new(MessageData)
								msgData0.Name = "entity"
								if reflect.TypeOf(entity_).String() == "*Server.DynamicEntity" {
									msgData0.Value = entity_.(*DynamicEntity).getObject()
								} else {
									msgData0.Value = entity_
								}
								eventData[0] = msgData0

								msgData1 := new(MessageData)
								msgData1.Name = "prototype"
								msgData1.Value = prototype
								eventData[1] = msgData1

								evt, _ := NewEvent(UpdateEntityEvent, EntityEvent, eventData)
								GetServer().GetEventManager().BroadcastEvent(evt)
							}
						}
					}

					// Remove it from the map.
					if len(entity.Ids()) > 0 {
						id := entityManager.GenerateEntityUUID(entity.GetTypeName(), "", entity.Ids(), "", "")
						delete(entityManager.m_entities, id)
					}
					delete(entityManager.m_entities, entity.GetUuid())
				}

			case entity := <-entityManager.m_setEntityChan:
				// Append in the map:
				// Set it entity getter.
				entity.SetEntityGetter(getEntityFct)
				// By id
				if len(entity.Ids()) > 0 {
					id := entityManager.GenerateEntityUUID(entity.GetTypeName(), "", entity.Ids(), "", "")
					entityManager.m_entities[id] = entity
				}
				// By uuid
				entityManager.m_entities[entity.GetUuid()] = entity

				// Keep track of reverse references releationship. (referenced by)
				prototype, _ := entityManager.getEntityPrototype(entity.GetTypeName(), strings.Split(entity.GetTypeName(), ".")[0])
				for i := 0; i < len(prototype.Fields); i++ {
					if strings.HasSuffix(prototype.FieldsType[i], ":Ref") {
						refs := make([]string, 0)
						var refs_ interface{}
						var err interface{}
						if reflect.TypeOf(entity).String() == "*Server.DynamicEntity" {
							refs_ = entity.(*DynamicEntity).getValue(prototype.Fields[i])
						} else {
							getterName := strings.Replace(prototype.Fields[i], "M_", "", -1)
							getterName = "Get" + strings.ToUpper(getterName[0:1]) + getterName[1:] + "Str"
							refs_, err = Utility.CallMethod(entity, getterName, []interface{}{})
						}
						if err == nil {
							if reflect.TypeOf(refs_).Kind() == reflect.String {
								refs = append(refs, refs_.(string))
							} else if reflect.TypeOf(refs_).String() == "[]string" {
								refs = refs_.([]string)
							} else if reflect.TypeOf(refs_).String() == "[]interface {}" {
								for j := 0; j < len(refs_.([]interface{})); j++ {
									refs = append(refs, refs_.([]interface{})[j].(string))
								}
							}

							// Now I go the reference i will keep it in the map.
							for j := 0; j < len(refs); j++ {
								if entityManager.m_entitiesRefs[refs[j]] == nil {
									entityManager.m_entitiesRefs[refs[j]] = make(map[string][]string, 0)
								}
								if entityManager.m_entitiesRefs[refs[j]][prototype.Fields[i]] == nil {
									entityManager.m_entitiesRefs[refs[j]][prototype.Fields[i]] = make([]string, 0)
								}
								if !Utility.Contains(entityManager.m_entitiesRefs[refs[j]][prototype.Fields[i]], entity.GetUuid()) {
									entityManager.m_entitiesRefs[refs[j]][prototype.Fields[i]] = append(entityManager.m_entitiesRefs[refs[j]][prototype.Fields[i]], entity.GetUuid())
								}
							}
						}
					}
				}
			}
		}
	}()

	return entityManager
}

////////////////////////////////////////////////////////////////////////////////
// Service functions
////////////////////////////////////////////////////////////////////////////////

/**
 * Initialization.
 */
func (this *EntityManager) initialize() {
	log.Println("--> Initialize EntityManager")

	// Create the default configurations
	GetServer().GetConfigurationManager().setServiceConfiguration(this.getId(), -1)

	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// must be call at least once at start time.
	this.getCargoEntities()
}

func (this *EntityManager) getId() string {
	return "EntityManager"
}

func (this *EntityManager) start() {
	log.Println("--> Start EntityManager")
}

func (this *EntityManager) stop() {
	log.Println("--> Stop EntityManager")
}

// Return the default Cargo Entities, create it if is not already exist.
func (this *EntityManager) getCargoEntities() *CargoEntities.Entities {
	var cargoEntities *CargoEntities.Entities
	entities, _ := GetServer().GetEntityManager().getEntityById("CargoEntities.Entities", "CargoEntities", []interface{}{"CARGO_ENTITIES"})
	if entities != nil {
		cargoEntities = entities.(*CargoEntities.Entities)
	} else {
		// I will create the cargo entities if it dosent already exist.
		cargoEntities = new(CargoEntities.Entities)
		cargoEntities.SetId("CARGO_ENTITIES")
		cargoEntities.SetName("Cargo entities")
		cargoEntities.SetVersion("1.0")
		this.saveEntity(cargoEntities)
	}
	return cargoEntities
}

// Return the uuid for the CargoEntities
func (this *EntityManager) getCargoEntitiesUuid() string {
	return this.GenerateEntityUUID("CargoEntities.Entities", "", []interface{}{"CARGO_ENTITIES"}, "", "")
}

func (this *EntityManager) getEntityPrototype(typeName string, storeId string) (*EntityPrototype, error) {
	store := GetServer().GetDataManager().getDataStore(storeId)
	if store != nil {
		prototype, err := store.GetEntityPrototype(typeName)
		return prototype, err
	}
	return nil, errors.New("No Data store found with id: " + storeId)
}

func (this *EntityManager) getEntityOwner(entity Entity) Entity {
	log.Println("getEntityOwner")
	return nil
}

func (this *EntityManager) isEntityExist(uuid string) bool {
	log.Println("isEntityExist")
	return false
}

func (this *EntityManager) getEntities(typeName string, storeId string, query *EntityQuery) ([]Entity, *CargoEntities.Error) {
	var info EntityInfo
	info.typeName = typeName
	info.entities = make(chan []Entity)
	this.m_getEntityChan <- info

	// wait to answer...
	entities := <-info.entities
	if entities != nil {
		return entities, nil
	}

	// TODO get entities from the datastore.

	return nil, nil
}

func (this *EntityManager) getEntityByUuid(uuid string) (Entity, *CargoEntities.Error) {
	var info EntityInfo
	info.typeName = strings.Split(uuid, "%")[0]
	info.uuid = uuid
	info.entities = make(chan []Entity)
	this.m_getEntityChan <- info

	// wait to answer...
	entities := <-info.entities
	if len(entities) == 1 {
		return entities[0], nil
	}

	// Todo get the entity from the datastore.

	return nil, nil
}

func (this *EntityManager) getEntityById(typeName string, storeId string, ids []interface{}) (Entity, *CargoEntities.Error) {

	// First I will get a look in the cash to see if the entity is already initialyse...
	var info EntityInfo
	info.typeName = typeName
	info.ids = ids
	info.entities = make(chan []Entity)
	this.m_getEntityChan <- info

	// wait to answer...
	entities := <-info.entities
	if len(entities) == 1 {
		return entities[0], nil
	}

	// TODO get the entity from the data store.

	// So here I will retreive the entity uuid from the entity id.
	var query EntityQuery
	query.TYPENAME = typeName
	queryStr, _ := json.Marshal(query)

	// Make the query over the store...
	store := GetServer().GetDataManager().getDataStore(storeId)
	store.Read(string(queryStr), []interface{}{}, []interface{}{})

	errObj := NewError(Utility.FileLine(), ENTITY_ID_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Fail to retreive entity by id "))

	return nil, errObj
}

/**
 * Recursively generate Quad from structure values.
 */
func ToTriples(values map[string]interface{}) ([]interface{}, error) {
	triples := make([]interface{}, 0)
	var uuid string
	if values["UUID"] != nil {
		uuid = values["UUID"].(string)
		typeName := values["TYPENAME"].(string)
		storeId := typeName[0:strings.Index(typeName, ".")]
		prototype, _ := GetServer().GetEntityManager().getEntityPrototype(typeName, storeId)

		for k, v := range values {
			if v != nil {
				if reflect.TypeOf(v).String() == "map[string]interface {}" {
					triples_, err := ToTriples(v.(map[string]interface{}))
					if err == nil {
						triples = append(triples, triples_...)
					} else {
						return triples, err
					}
				} else {
					if strings.HasPrefix(k, "M_") {
						fieldType := prototype.FieldsType[prototype.getFieldIndex(k)]
						if strings.HasPrefix(fieldType, "[]") {
							// In case of array...
							if strings.HasSuffix(fieldType, ":Ref") {
								if reflect.TypeOf(v).String() == "[]interface {}" {
									for i := 0; i < len(v.([]interface{})); i++ {
										triples = append(triples, "<#"+uuid+">")
										triples = append(triples, "<#"+k+">")
										triples = append(triples, "<#"+v.([]interface{})[i].(string)+">")
									}
								} else if reflect.TypeOf(v).String() == "[]string" {
									for i := 0; i < len(v.([]string)); i++ {
										triples = append(triples, "<#"+uuid+">")
										triples = append(triples, "<#"+k+">")
										triples = append(triples, "<#"+v.([]string)[i]+">")
									}
								}
							} else {

								if reflect.TypeOf(v).String() == "[]interface {}" {
									if len(v.([]interface{})) > 0 {
										if reflect.TypeOf(v.([]interface{})[0]).String() == "map[string]interface {}" {
											// Sub-object..
											for i := 0; i < len(v.([]interface{})); i++ {
												triples_, err := ToTriples(v.([]interface{})[i].(map[string]interface{}))
												if err == nil {
													triples = append(triples, triples_...)
												} else {
													return triples, err
												}
											}
										}
									}
								} else {
									str, err := json.Marshal(v)
									if err == nil {
										triples = append(triples, "<#"+uuid+">")
										triples = append(triples, "<#"+k+">")
										triples = append(triples, `"`+string(str)+`"`)
									}
								}

							}
						} else {
							if strings.HasSuffix(fieldType, ":Ref") {
								if len(v.(string)) > 0 {
									triples = append(triples, "<#"+uuid+">")
									triples = append(triples, "<#"+k+">")
									triples = append(triples, "<#"+v.(string)+">")
								}
							} else {
								if reflect.TypeOf(v).Kind() == reflect.String {
									if len(v.(string)) > 0 {
										triples = append(triples, "<#"+uuid+">")
										triples = append(triples, "<#"+k+">")
										triples = append(triples, `"`+v.(string)+`"`)
									}
								} else {
									triples = append(triples, "<#"+uuid+">")
									triples = append(triples, "<#"+k+">")
									triples = append(triples, v)
								}
							}
						}
					}
				}
			}
		}
	}
	return triples, nil
}

/**
 * Create an new entity.
 */
func (this *EntityManager) createEntity(parentUuid string, attributeName string, typeName string, objectId string, entity Entity) (Entity, *CargoEntities.Error) {
	// Set the entity values here.
	entity.GetTypeName() // Set the type name if not already set...
	entity.SetParentLnk(attributeName)
	entity.SetParentUuid(parentUuid)

	// Here I will set the uuid if is not already set
	uuid := this.GenerateEntityUUID(typeName, parentUuid, entity.Ids(), "", "")
	entity.SetUuid(uuid)

	storeId := typeName[0:strings.Index(typeName, ".")]
	prototype, _ := GetServer().GetEntityManager().getEntityPrototype(typeName, storeId)

	// Generate the quads, it will also set the entity uuid at the same time
	// if not already set.
	var values map[string]interface{}
	var err error
	// Get values as map[string]interface{} and also set the entity in it parent.
	if reflect.TypeOf(values).String() == "*Server.DynamicEntity" {
		parent, cargoError := this.getEntityByUuid(parentUuid)
		if cargoError != nil {
			return nil, cargoError
		}

		fieldType := prototype.FieldsType[prototype.getFieldIndex(attributeName)]
		if strings.HasPrefix(fieldType, "[]") {
			parent.(*DynamicEntity).appendValue(attributeName, entity.(*DynamicEntity).getObject())
		} else {
			parent.(*DynamicEntity).setValue(attributeName, entity.(*DynamicEntity).getObject())
		}
		values = entity.(*DynamicEntity).getObject()
	} else {

		parent, cargoError := this.getEntityByUuid(parentUuid)
		if cargoError != nil {
			return nil, cargoError
		}

		setMethodName := strings.Replace(attributeName, "M_", "", -1)
		setMethodName = "Set" + strings.ToUpper(setMethodName[0:1]) + setMethodName[1:]
		params := make([]interface{}, 1)
		params[0] = entity
		_, err_ := Utility.CallMethod(parent, setMethodName, params)
		if err_ != nil {
			log.Println("fail to call method ", setMethodName, " on ", parent)
			cargoError := NewError(Utility.FileLine(), ATTRIBUTE_NAME_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, err_.(error))
			return nil, cargoError
		}

		values, err = Utility.ToMap(entity)
		if err != nil {
			cargoError := NewError(Utility.FileLine(), ENTITY_TO_QUADS_ERROR, SERVER_ERROR_CODE, err)
			return nil, cargoError
		}
	}

	// Here I will set the entity on the cache...
	this.m_setEntityChan <- entity

	if GetServer().GetServiceManager().m_isReady == false {
		return entity, nil
	}

	// Now I will call the meth
	var triples []interface{}
	triples, err = ToTriples(values)

	if err != nil {
		cargoError := NewError(Utility.FileLine(), ENTITY_TO_QUADS_ERROR, SERVER_ERROR_CODE, err)
		return nil, cargoError
	}

	// Now entity are quadify I will save it in the graph store.
	store := GetServer().GetDataManager().getDataStore(storeId)

	// So here I will simply append the quads in the database...
	_, err = store.Create("", triples)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), ENTITY_CREATION_ERROR, SERVER_ERROR_CODE, err)
		return nil, cargoError
	}

	// The event data...
	eventData := make([]*MessageData, 2)
	msgData0 := new(MessageData)
	msgData0.Name = "entity"
	if reflect.TypeOf(entity).String() == "*Server.DynamicEntity" {
		msgData0.Value = entity.(*DynamicEntity).getObject()
	} else {
		msgData0.Value = entity
	}
	eventData[0] = msgData0

	msgData1 := new(MessageData)
	msgData1.Name = "prototype"
	msgData1.Value = prototype
	eventData[1] = msgData1

	evt, _ := NewEvent(NewEntityEvent, EntityEvent, eventData)
	GetServer().GetEventManager().BroadcastEvent(evt)

	return entity, nil
}

func (this *EntityManager) saveEntity(entity Entity) *CargoEntities.Error {
	typeName := entity.GetTypeName() // Set the type name if not already set...
	storeId := typeName[0:strings.Index(typeName, ".")]
	prototype, _ := GetServer().GetEntityManager().getEntityPrototype(typeName, storeId)

	// Here I will set the uuid if is not already set
	if len(entity.GetUuid()) == 0 {
		uuid := this.GenerateEntityUUID(typeName, entity.GetParentUuid(), entity.Ids(), "", "")
		entity.SetUuid(uuid)
	}

	var values map[string]interface{}
	var err error

	if reflect.TypeOf(values).String() == "*Server.DynamicEntity" {
		values = entity.(*DynamicEntity).getObject()
	} else {
		values, err = Utility.ToMap(entity)
		if err != nil {
			cargoError := NewError(Utility.FileLine(), ENTITY_TO_QUADS_ERROR, SERVER_ERROR_CODE, err)
			return cargoError
		}
	}

	// Here I will set the entity on the cache...
	this.m_setEntityChan <- entity

	var triples []interface{}
	triples, err = ToTriples(values)

	if err != nil {
		cargoError := NewError(Utility.FileLine(), ENTITY_TO_QUADS_ERROR, SERVER_ERROR_CODE, err)
		return cargoError
	}

	// Now entity are quadify I will save it in the graph store.
	store := GetServer().GetDataManager().getDataStore(storeId)

	// So here I will simply append the quads in the database...
	err = store.Update("", triples, []interface{}{})
	if err != nil {
		cargoError := NewError(Utility.FileLine(), ENTITY_CREATION_ERROR, SERVER_ERROR_CODE, err)
		return cargoError
	}

	// The event data...
	eventData := make([]*MessageData, 2)
	msgData0 := new(MessageData)
	msgData0.Name = "entity"
	if reflect.TypeOf(entity).String() == "*Server.DynamicEntity" {
		msgData0.Value = entity.(*DynamicEntity).getObject()
	} else {
		msgData0.Value = entity
	}
	eventData[0] = msgData0

	msgData1 := new(MessageData)
	msgData1.Name = "prototype"
	msgData1.Value = prototype
	eventData[1] = msgData1

	evt, _ := NewEvent(UpdateEntityEvent, EntityEvent, eventData)
	GetServer().GetEventManager().BroadcastEvent(evt)

	return nil
}

func (this *EntityManager) deleteEntity(entity Entity) *CargoEntities.Error {

	if len(entity.GetParentUuid()) > 0 {
		// I will get the parent uuid link.
		parent, err := GetServer().GetEntityManager().getEntityByUuid(entity.GetParentUuid())
		if err != nil {
			return err
		}

		// Here I will remove it from it parent...
		// Get values as map[string]interface{} and also set the entity in it parent.
		if reflect.TypeOf(entity).String() == "*Server.DynamicEntity" {
			parent.(*DynamicEntity).removeValue(entity.GetParentLnk(), entity.GetUuid())
		} else {
			removeMethode := strings.Replace(entity.GetParentLnk(), "M_", "", -1)
			removeMethode = "Remove" + strings.ToUpper(removeMethode[0:1]) + removeMethode[1:]
			params := make([]interface{}, 1)
			params[0] = entity
			_, err_ := Utility.CallMethod(parent, removeMethode, params)
			if err_ != nil {
				log.Println("fail to call method ", removeMethode, " on ", parent.GetTypeName(), parent.GetUuid())
				cargoError := NewError(Utility.FileLine(), ATTRIBUTE_NAME_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, err_.(error))
				return cargoError
			}

			//log.Println("-------> entity ", entity.GetUuid(), " is now remove from ", parent.GetUuid())
		}
	}

	// remove it from the cache.
	this.m_removeEntityChan <- entity

	// Send event message...
	var eventDatas []*MessageData
	evtData := new(MessageData)
	evtData.TYPENAME = "Server.MessageData"
	evtData.Name = "entity"
	if reflect.TypeOf(entity).String() == "*Server.DynamicEntity" {
		evtData.Value = entity.(*DynamicEntity).getObject()
	} else {
		evtData.Value = entity
	}

	eventDatas = append(eventDatas, evtData)
	evt, _ := NewEvent(DeleteEntityEvent, EntityEvent, eventDatas)
	GetServer().GetEventManager().BroadcastEvent(evt)

	// TODO remove it from the database.

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// API
////////////////////////////////////////////////////////////////////////////////

////////////////////////////// Prototypes //////////////////////////////////////

// @api 1.0
// Create a new entity prototype.
// @param {string} storeId The store id, where to create the new prototype.
// @param {interface{}} prototype The prototype object to create.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{*EntityPrototype} Return the created entity prototype
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.createEntityPrototype = function(storeId, prototype, successCallback, errorCallback, caller){
//	var params = []
//	params.push(createRpcData(storeId, "STRING", "storeId"))
//	params.push(createRpcData(prototype, "JSON_STR", "prototype"))
//	server.executeJsFunction(
//	"EntityManagerCreateEntityPrototype",
//	params,
//	undefined, //progress callback
//	function (results, caller) { // Success callback
// 	   if(caller.successCallback!=undefined){
// 			var prototype = new EntityPrototype()
//			prototype.init(results[0])
//      	caller.successCallback(prototype, caller.caller)
//          caller.successCallback = undefined
//		}
//	},
//	function (errMsg, caller) { // Error callback
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//	},{"successCallback":successCallback, "errorCallback":errorCallback, "caller": caller})
//}
func (this *EntityManager) CreateEntityPrototype(storeId string, prototype interface{}, messageId string, sessionId string) *EntityPrototype {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// Cast it as needed...
	if reflect.TypeOf(prototype).String() == "map[string]interface {}" {
		prototype.(map[string]interface{})["TYPENAME"] = "Server.EntityPrototype"
		values, err := Utility.InitializeStructure(prototype.(map[string]interface{}))
		if err == nil {
			prototype = values.Interface()
		}
	}

	if reflect.TypeOf(prototype).String() != "*Server.EntityPrototype" {
		cargoError := NewError(Utility.FileLine(), PARAMETER_TYPE_ERROR, SERVER_ERROR_CODE, errors.New("Expected '*Server.EntityPrototype' but got '"+reflect.TypeOf(prototype).String()+"' instead."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	// Get the store...
	store := GetServer().GetDataManager().getDataStore(storeId)
	if store == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Datastore '"+storeId+"' dosen't exist."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil

	}

	// Save the prototype...
	err := store.CreateEntityPrototype(prototype.(*EntityPrototype))
	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_CREATION_ERROR, SERVER_ERROR_CODE, err)
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	return prototype.(*EntityPrototype)
}

// @api 1.0
// Save existing entity prototype.
// @param {string} storeId The store id, where to create the new prototype.
// @param {interface{}} prototype The prototype object to create.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{*EntityPrototype} Return the saved entity prototype
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.saveEntityPrototype = function(storeId, prototype, successCallback, errorCallback, caller){
//	var params = []
//	params.push(createRpcData(storeId, "STRING", "storeId"))
//	params.push(createRpcData(prototype, "JSON_STR", "prototype"))
//	server.executeJsFunction(
//	"EntityManagerSaveEntityPrototype",
//	params,
//	undefined, //progress callback
//	function (results, caller) { // Success callback
// 	   if(caller.successCallback!=undefined){
// 			 var prototype = new EntityPrototype()
//			 prototype.init(results[0])
//      	 caller.successCallback(prototype, caller.caller)
//           caller.successCallback = undefined
//		}
//	},
//	function (errMsg, caller) { // Error callback
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//	},{"successCallback":successCallback, "errorCallback":errorCallback, "caller": caller})
//}
func (this *EntityManager) SaveEntityPrototype(storeId string, prototype interface{}, messageId string, sessionId string) *EntityPrototype {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// Cast it as needed...
	if reflect.TypeOf(prototype).String() == "map[string]interface {}" {
		prototype.(map[string]interface{})["TYPENAME"] = "Server.EntityPrototype"
		values, err := Utility.InitializeStructure(prototype.(map[string]interface{}))
		if err == nil {
			prototype = values.Interface()
		} else {
			log.Println("fail to initialyse EntityPrototype from map[string]interface {} ", err)
			cargoError := NewError(Utility.FileLine(), PARAMETER_TYPE_ERROR, SERVER_ERROR_CODE, err)
			GetServer().reportErrorMessage(messageId, sessionId, cargoError)
			return nil
		}
	}

	if reflect.TypeOf(prototype).String() != "*Server.EntityPrototype" {
		cargoError := NewError(Utility.FileLine(), PARAMETER_TYPE_ERROR, SERVER_ERROR_CODE, errors.New("Expected '*Server.EntityPrototype' but got '"+reflect.TypeOf(prototype).String()+"' instead."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	// Get the store...
	store := GetServer().GetDataManager().getDataStore(storeId)
	if store == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Datastore '"+storeId+"' dosen't exist."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil

	}

	// Save the prototype...
	err := store.SaveEntityPrototype(prototype.(*EntityPrototype))
	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_UPDATE_ERROR, SERVER_ERROR_CODE, err)
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	return prototype.(*EntityPrototype)
}

// @api 1.0
// Delete existing entity prototype.
// @param {string} typeName The prototype id.
// @param {string} storeId The store id, where to create the new prototype.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
func (this *EntityManager) DeleteEntityPrototype(typeName string, storeId string, messageId string, sessionId string) {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return
	}

	prototype, err := this.getEntityPrototype(typeName, storeId)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, err)
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return
	}

	// Get the store...
	store := GetServer().GetDataManager().getDataStore(storeId)
	if store == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Datastore '"+storeId+"' dosen't exist."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return
	}

	err = store.DeleteEntityPrototype(prototype.TypeName)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_DELETE_ERROR, SERVER_ERROR_CODE, err)
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
	}

}

// @api 1.0
// Rename existing entity prototype.
// @param {string} typeName The new prototype name.
// @param {string} prototype The prototype to rename.
// @param {string} storeId The store id, where to create the new prototype.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{*EntityPrototype} Return the renamed entity prototype
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.renameEntityPrototype = function(typeName, prototype, storeId, successCallback, errorCallback, caller){
//	var params = []
//	params.push(createRpcData(typeName, "STRING", "typeName"))
//	params.push(createRpcData(prototype, "JSON_STR", "prototype"))
//	params.push(createRpcData(storeId, "STRING", "storeId"))
//	server.executeJsFunction(
//	"EntityManagerRenameEntityPrototype",
//	params,
//	undefined, //progress callback
//	function (results, caller) { // Success callback
// 	   if(caller.successCallback!=undefined){
// 			 var prototype = new EntityPrototype()
//			 prototype.init(results[0])
//      	 caller.successCallback(prototype, caller.caller)
//           caller.successCallback = undefined
//		}
//	},
//	function (errMsg, caller) { // Error callback
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//	},{"successCallback":successCallback, "errorCallback":errorCallback, "caller": caller})
//}
func (this *EntityManager) RenameEntityPrototype(typeName string, prototype interface{}, storeId string, messageId string, sessionId string) *EntityPrototype {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// Cast it as needed...
	/*if reflect.TypeOf(prototype).String() == "map[string]interface {}" {
		prototype.(map[string]interface{})["TYPENAME"] = "Server.EntityPrototype"
		values, err := Utility.InitializeStructure(prototype.(map[string]interface{}))
		if err == nil {
			prototype = values.Interface()
		} else {
			log.Println("fail to initialyse EntityPrototype from map[string]interface {} ", err)
			cargoError := NewError(Utility.FileLine(), PARAMETER_TYPE_ERROR, SERVER_ERROR_CODE, err)
			GetServer().reportErrorMessage(messageId, sessionId, cargoError)
			return nil
		}
	}

	if reflect.TypeOf(prototype).String() != "*Server.EntityPrototype" {
		cargoError := NewError(Utility.FileLine(), PARAMETER_TYPE_ERROR, SERVER_ERROR_CODE, errors.New("Expected '*Server.EntityPrototype' but got '"+reflect.TypeOf(prototype).String()+"' instead."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	// Get the store...
	store := GetServer().GetDataManager().getDataStore(storeId)
	if store == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Datastore '"+storeId+"' dosen't exist."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	oldName := prototype.(*EntityPrototype).TypeName
	// Those types can not be rename.
	if strings.HasPrefix(oldName, "xs.") || strings.HasPrefix(oldName, "sqltypes.") || strings.HasPrefix(oldName, "XMI_types.") || strings.HasPrefix(oldName, "Config.") || strings.HasPrefix(oldName, "CargoEntities.") || strings.HasPrefix(oldName, "sql_infos.") {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_UPDATE_ERROR, SERVER_ERROR_CODE, errors.New("Prototype "+oldName+" cannot be rename!"))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	// So here I will get the list of all entities for that type.
	entities, _ := this.getEntities(oldName, nil, storeId, false)

	// Now I will change the prototype name
	prototype.(*EntityPrototype).TypeName = typeName

	// Save info in substitution groups...
	for i := 0; i < len(prototype.(*EntityPrototype).SubstitutionGroup); i++ {
		subTypeName := prototype.(*EntityPrototype).SubstitutionGroup[i]
		subType, err := this.getEntityPrototype(subTypeName, subTypeName[0:strings.Index(subTypeName, ".")])
		if err == nil {
			for j := 0; j < len(subType.SuperTypeNames); j++ {
				if subType.SuperTypeNames[j] == oldName {
					subType.SuperTypeNames[j] = typeName
				}
			}
			// Save it...
			subType.Save(subTypeName[0:strings.Index(subTypeName, ".")])
		}

	}

	// Save info in supertypes
	for i := 0; i < len(prototype.(*EntityPrototype).SuperTypeNames); i++ {
		superTypeName := prototype.(*EntityPrototype).SuperTypeNames[i]
		superType, err := this.getEntityPrototype(superTypeName, superTypeName[0:strings.Index(superTypeName, ".")])
		if err == nil {
			for j := 0; j < len(superType.SubstitutionGroup); j++ {
				if superType.SubstitutionGroup[j] == oldName {
					superType.SubstitutionGroup[j] = typeName
				}
			}
			superType.Save(superTypeName[0:strings.Index(superTypeName, ".")])
		}
	}

	// Now I must make tour of all prototypes in the data store and replace
	// field that made use of that prototype with it new typename.
	prototypes, err := this.getEntityPrototypes(storeId, typeName[0:strings.Index(typeName, ".")])
	if err == nil {
		for i := 0; i < len(prototypes); i++ {
			p := prototypes[i]
			needSave := false
			for j := 0; j < len(p.FieldsType); j++ {
				if strings.Index(p.FieldsType[j], oldName) > 0 {
					needSave = true
					strings.Replace(p.FieldsType[j], oldName, typeName, -1)
				}
			}
			if needSave == true {
				// save the prototype.
				p.Save(storeId)
			}
		}
	} else {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_UPDATE_ERROR, SERVER_ERROR_CODE, err)
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	prototype.(*EntityPrototype).TypeName = typeName
	store.(*GraphStore).DeleteEntityPrototype(oldName)
	store.(*GraphStore).saveEntityPrototype(prototype.(*EntityPrototype))

	// Now I must update entities...
	for i := 0; i < len(entities); i++ {
		if reflect.TypeOf(entities[i]).String() == "*Server.DynamicEntity" {
			//
			entity := entities[i].(*DynamicEntity)
			ids := make([]interface{}, 0)
			p, _ := this.getEntityPrototype(entity.GetTypeName(), entity.GetTypeName()[0:strings.Index(entity.GetTypeName(), ".")])
			for j := 0; j < len(p.Ids); j++ {
				ids = append(ids, entity.getValue(p.Ids[j]))
			}

			// Here I will delete the existing entity from the db...
			entity.setValue("UUID", nil)          // Set it uuid to nil
			entity.setValue("TYPENAME", typeName) // Set it new typeName
			// Recreate it with it new type
			newEntity, errObj := this.newDynamicEntity(entity.GetParentUuid(), entity.GetObject().(map[string]interface{}))
			if errObj != nil {
				newEntity.SaveEntity() // Save the new entity
			}
		}

	}

	return prototype.(*EntityPrototype)*/
	return nil

}

// @api 1.0
// That function will retreive all prototypes of a store.
// @param {string} storeId The store id, where to create the new prototype.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{[]*EntityPrototype} Return the retreived list of entity prototype
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.getEntityPrototypes = function (storeId, successCallback, errorCallback, caller) {
//    // server is the client side singleton.
//    var params = []
//    params.push(createRpcData(storeId, "STRING", "storeId"))
//    // Call it on the server.
//    server.executeJsFunction(
//        "EntityManagerGetEntityPrototypes", // The function to execute remotely on server
//        params, // The parameters to pass to that function
//        function (index, total, caller) { // The progress callback
//            // Nothing special to do here.
//        },
//        function (results, caller) {
//            var results = results[0]
//            var prototypes = []
//            if (results != null) {
//                for (var i = 0; i < results.length; i++) {
//                    var proto = new EntityPrototype()
//                    entityPrototypes[results[i].TypeName] = proto
//                    proto.init(results[i])
//                    prototypes.push(proto)
//                }
//            }
// 			 if(caller.successCallback!=undefined){
//            	caller.successCallback(prototypes, caller.caller)
//            	caller.successCallback = undefined
//			 }
//        },
//        function (errMsg, caller) {
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//        }, // Error callback
//        { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback } // The caller
//    )
//}
func (this *EntityManager) GetEntityPrototypes(storeId string, messageId string, sessionId string) []*EntityPrototype {

	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	if strings.Index(storeId, ".") > 0 {
		storeId = storeId[0:strings.Index(storeId, ".")]
	}

	store := GetServer().GetDataManager().getDataStore(storeId)
	if store == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("There is no store with id '"+storeId+"'."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	protos, err := store.GetEntityPrototypes()

	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("There is no prototypes in store '"+storeId+"'."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	return protos
}

// @api 1.0
// That function will retreive the entity prototype with a given type name.
// @param {string} typeName The type name of the prototype to retreive.
// @param {string} storeId The store id, where to create the new prototype.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{*EntityPrototype} Return the retreived entity prototype
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.getEntityPrototype = function (typeName, storeId, successCallback, errorCallback, caller) {
//    // Retrun entity prototype that aleady exist.
//    if (entityPrototypes[typeName] != undefined) {
//        successCallback(entityPrototypes[typeName], caller)
//        successCallback = undefined
//        return
//    }
//    // server is the client side singleton.
//    var params = []
//    params.push(createRpcData(typeName, "STRING", "typeName"))
//    params.push(createRpcData(storeId, "STRING", "storeId"))
//    // Call it on the server.
//    server.executeJsFunction(
//        "EntityManagerGetEntityPrototype", // The function to execute remotely on server
//        params, // The parameters to pass to that function
//        function (index, total, caller) { // The progress callback
//            // Nothing special to do here.
//        },
//        function (results, caller) {
//            var proto = new EntityPrototype()
//            entityPrototypes[results[0].TypeName] = proto
//            proto.init(results[0])
//			 if(caller.successCallback!=undefined){
//            	caller.successCallback(proto, caller.caller)
//            	caller.successCallback = undefined
//        	}
//        },
//        function (errMsg, caller) {
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//        }, // Error callback
//        { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback } // The caller
//    )
//}
func (this *EntityManager) GetEntityPrototype(typeName string, storeId string, messageId string, sessionId string) *EntityPrototype {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	proto, err := this.getEntityPrototype(typeName, storeId)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), ENTITY_PROTOTYPE_ERROR, SERVER_ERROR_CODE, err)
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	return proto
}

//////////////////////////////// Entities //////////////////////////////////////

// @api 1.0
// Event handler function.
// @param {interface{}} values The entity to set.
// @scope {public}
// @src
//EntityManager.prototype.onEvent = function (evt) {
//    // Set the internal object.
//    if (evt.code == UpdateEntityEvent || evt.code == NewEntityEvent) {
//        if (entityPrototypes[evt.dataMap["entity"].TYPENAME] == undefined) {
//            console.log("Type " + evt.dataMap["entity"].TYPENAME + " not define!")
//            return
//        }
//        if (entities[evt.dataMap["entity"].UUID] == undefined) {
//            var entity = eval("new " + evt.dataMap["entity"].TYPENAME + "()")
//            entity.initCallback = function (self, evt, entity) {
//                return function (entity) {
//                    server.entityManager.setEntity(entity)
//                    EventHub.prototype.onEvent.call(self, evt)
//                }
//            } (this, evt, entity)
//            entity.init(evt.dataMap["entity"])
//        } else {
//            // update the object values.
//            // but before I call the event I will be sure the entity have
//            var entity = entities[evt.dataMap["entity"].UUID]
//            entity.initCallback = function (self, evt, entity) {
//                return function (entity) {
//                    // Test if the object has change here befor calling it.
//                    server.entityManager.setEntity(entity)
//                    if (evt.done == undefined) {
//                        EventHub.prototype.onEvent.call(self, evt)
//                    }
//                    evt.done = true // Cut the cyclic recursion.
//                }
//            } (this, evt, entity)
//            setObjectValues(entity, evt.dataMap["entity"])
//        }
//    } else if (evt.code == DeleteEntityEvent) {
//        var entity = entities[evt.dataMap["entity"].UUID]
//        if (entity != undefined) {
//            this.resetEntity(entity)
//            EventHub.prototype.onEvent.call(this, evt)
//        }
//    }
//}
func (this *EntityManager) OnEvent(evt interface{}) {
	/** empty function here... **/
}

// @api 1.0
// Set the value of an entity on the entityManager.
// @param {interface{}} values The entity to set.
// @scope {public}
// @src
//EntityManager.prototype.setEntity = function (entity) {
//    this.getEntityPrototype(entity.TYPENAME, entity.TYPENAME.split(".")[0],
//        function (prototype, caller) {
//            var id_ = entity.TYPENAME + ":"
//            for (var i = 0; i < prototype.Ids.length; i++) {
//                var id = prototype.Ids[i]
//                if (id == "UUID" || id == "uuid") {
//					  if(entity.UUID != undefined){
//					  	if(entities[entity.UUID] != undefined && entity.UUID.length > 0){
//							entity.ParentLnk = entities[entity.UUID].ParentLnk
//					  	}
//					  	entities[entity.UUID] = entity
//					  }
//                } else if(entity[id] != undefined) {
//                    if (entity[id].length > 0) {
//                        id_ += entity[id]
//                        if (i < prototype.Ids.length - 1) {
//                            id_ += "_"
//                        }
//						  if(i == prototype.Ids.length - 1){
//					  		if(entities[id_] != undefined){
//								entity.ParentLnk = entities[id_].ParentLnk
//					  		}
//							entities[id_] = entity
//						  }
//                    }
//                }
//            }
//        },
//        function (errMsg, caller) {
//            /** Nothing to do here. */
//        },
//        {})
//}
func (this *EntityManager) SetEntity(values interface{}) {
	/** empty function here... **/
}

// @api 1.0
// Reset the value of an entity on the entityManager.
// @param {interface{}} values The entity to set.
// @scope {public}
// @src
//EntityManager.prototype.resetEntity = function (entity) {
//    var prototype = entityPrototypes[entity.TYPENAME]
//    delete entities[entity.UUID]
//    var id = entity.TYPENAME + ":"
//    for (var i = 0; i < prototype.Ids.length; i++) {
//        id += entity[prototype.Ids[i]]
//        if (i < prototype.Ids.length - 1) {
//            id += "_"
//        }
//    }
//    if (entities[id] != undefined) {
//        delete entities[id]
//    }
//}
func (this *EntityManager) ResetEntity(values interface{}) {
	/** empty function here... **/
}

// @api 1.0
// That function is use to create a new entity of a given type..
// @param {string} parentUuid The uuid of the parent entity if there is one, null otherwise.
// @param {string} attributeName The attribute name is the name of the new entity in his parent. (parent.attributeName = this)
// @param {string} typeName The type name of the new entity.
// @param {string} objectId The id of the new entity. There is no restriction on the value entered.
// @param {interface{}} values the entity to be save, it can be nil.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{interface{}} Return the created entity
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.createEntity = function (parentUuid, attributeName, typeName, id, entity, successCallback, errorCallback, caller) {
//    // server is the client side singleton.
//    var params = []
//    params.push(createRpcData(parentUuid, "STRING", "parentUuid"))
//    params.push(createRpcData(attributeName, "STRING", "attributeName"))
//    params.push(createRpcData(typeName, "STRING", "typeName"))
//    params.push(createRpcData(id, "STRING", "id"))
//    params.push(createRpcData(entity, "JSON_STR", "entity"))
//    // Call it on the server.
//    server.executeJsFunction(
//        "EntityManagerCreateEntity", // The function to execute remotely on server
//        params, // The parameters to pass to that function
//        function (index, total, caller) { // The progress callback
//            // Nothing special to do here.
//        },
//        function (result, caller) {
//            var entity = eval("new " + result[0].TYPENAME + "()")
//            entity.initCallback = function () {
//                return function (entity) {
//                    if (caller.successCallback != undefined) {
//                        caller.successCallback(entity, caller.caller)
//                        caller.successCallback = undefined
//                    }
//                }
//            } (caller)
//            entity.init(result[0])
//        },
//        function (errMsg, caller) {
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//        }, // Error callback
//        { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback } // The caller
//    )
//}
func (this *EntityManager) CreateEntity(parentUuid string, attributeName string, typeName string, objectId string, values interface{}, messageId string, sessionId string) interface{} {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	result, errObj := this.createEntity(parentUuid, attributeName, typeName, objectId, values.(Entity))
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// Here I will set the ownership of the entity.
	session := GetServer().GetSessionManager().getActiveSessionById(sessionId)
	if session != nil {
		// this.setEntityOwner(session.GetAccountPtr(), result)
	}

	return result
}

// @api 1.0
// Save The entity. If the entity does not exist it creates it.
// @param {interface{}} values The entity to save.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{interface{}} Return an object (Entity)
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.saveEntity = function (entity, successCallback, errorCallback, caller) {
//    // server is the client side singleton.
//    entity.NeedSave = true
//    var params = []
//    params.push(createRpcData(entity, "JSON_STR", "entity"))
//    params.push(createRpcData(entity.TYPENAME, "STRING", "typeName"))
//    // Call it on the server.
//    server.executeJsFunction(
//        "EntityManagerSaveEntity", // The function to execute remotely on server
//        params, // The parameters to pass to that function
//        function (index, total, caller) { // The progress callback
//            // Nothing special to do here.
//        },
//        function (result, caller) {
//            var entity = eval("new " + result[0].TYPENAME + "()")
//            entity.initCallback = function () {
//                return function (entity) {
//                    // Set the new entity values...
//                    server.entityManager.setEntity(entity)
//                    if (caller.successCallback != undefined) {
//                        caller.successCallback(entity, caller.caller)
//                        caller.successCallback = undefined
//                    }
//                }
//            } (caller)
//            entity.init(result[0])
//        },
//        function (errMsg, caller) {
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//        }, // Error callback
//        { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback } // The caller
//    )
//}
func (this *EntityManager) SaveEntity(values interface{}, typeName string, messageId string, sessionId string) interface{} {

	var errObj *CargoEntities.Error
	errObj = GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// Now I will save the entity.
	errObj = this.saveEntity(values.(Entity))
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	return values.(Entity)
}

// @api 1.0
// That function is use to remove an entity with a given uuid.
// @param {string} uuid The uuid of entity to delete. Must have the form TypeName%UUID
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
func (this *EntityManager) RemoveEntity(uuid string, messageId string, sessionId string) {
	var errObj *CargoEntities.Error
	errObj = GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return
	}

	errObj = NewError(Utility.FileLine(), NOT_IMPLEMENTED_ERROR, SERVER_ERROR_CODE, errors.New("EntityManager.RemoveEntity is not implemented!"))
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
	}

	// The entity to remove.
	/*var entity Entity

	// validate the action. TODO active it latter...
	errObj = GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return // exit here.
	}

	entity, errObj = this.getEntityByUuid(uuid, false)
	if errObj != nil {
		entity, errObj = this.getDynamicEntityByUuid(uuid, false)
	}

	if entity != nil {
		// validate over the entity TODO active it latter...
		//errObj = GetServer().GetSecurityManager().hasPermission(sessionId, CargoEntities.PermissionType_Delete, entity)
		if errObj != nil {
			GetServer().reportErrorMessage(messageId, sessionId, errObj)
			return // exit here.
		}

		// Suppress the enitity...
		entity.DeleteEntity()

		// exit here.
		return
	}

	// Repport the error
	GetServer().reportErrorMessage(messageId, sessionId, errObj)
	*/

}

// @api 1.0
// That function is use to retreive objects with a given type.
// @param {string} typeName The name of the type we looking for in the form packageName.typeName
// @param {string} storeId The name of the store where the information is saved.
// @param {EntityQuery} query It contain the code of a function to be executed by the server to filter specific values.
// @param {int} offset	Results offset
// @param {int} limit	The number of results to return. Can be use to create page of results.
// @param {[]string} orderBy the list of field that specifie the result order.
// @param {bool} asc the list of field that specifie the result order.
// @result{[]interface{}} Return an array of object's (Entities)
// @scope {public}
// @param {callback} progressCallback The function is call when chunk of response is received.
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.getEntities = function (typeName, storeId, query, offset, limit, orderBy, asc, progressCallback, successCallback, errorCallback, caller) {
//    // First of all i will get the entity prototype.
//    server.entityManager.getEntityPrototype(typeName, storeId,
//        // The success callback.
//        function (result, caller) {
//            // Set the parameters.
//            var typeName = caller.typeName
//            var storeId = caller.storeId
//            var query = caller.query
//            var successCallback = caller.successCallback
//            var progressCallback = caller.progressCallback
//            var errorCallback = caller.errorCallback
//            var caller = caller.caller
//            // Create the list of parameters.
//            var params = []
//            params.push(createRpcData(typeName, "STRING", "typeName"))
//            params.push(createRpcData(storeId, "STRING", "storeId"))
//            params.push(createRpcData(query, "JSON_STR", "query"))
//            params.push(createRpcData(offset, "INTEGER", "offset"))
//            params.push(createRpcData(limit, "INTEGER", "limit"))
//			  params.push(createRpcData(orderBy, "JSON_STR", "orderBy", "[]string"))
//			  params.push(createRpcData(asc, "BOOLEAN", "asc"))
//            // Call it on the server.
//            server.executeJsFunction(
//                "EntityManagerGetEntities", // The function to execute remotely on server
//                params, // The parameters to pass to that function
//                function (index, total, caller) { // The progress callback
//                    // Keep track of the file transfert.
//                    caller.progressCallback(index, total, caller.caller)
//                },
//                function (result, caller) {
//                    var entities = []
//                    if (result[0] != undefined) {
//                        for (var i = 0; i < result[0].length; i++) {
//                            var entity = eval("new " + caller.prototype.TypeName + "(caller.prototype)")
//                            if (i == result[0].length - 1) {
//                                entity.initCallback = function (caller) {
//                                    return function (entity) {
//                                        server.entityManager.setEntity(entity)
//                                        if( caller.successCallback != undefined){
//                                        		caller.successCallback(entities, caller.caller)
//                                        		caller.successCallback = undefined
//                                    		}
//                                    }
//                                } (caller)
//                            } else {
//                                entity.initCallback = function (entity) {
//                                    server.entityManager.setEntity(entity)
//                                }
//                            }
//                            // push the entitie before init it...
//                            entities.push(entity)
//                            // call init...
//                            entity.init(result[0][i])
//                        }
//                    }
//                    if (result[0] == null || result[0].length==0) {
//                        if( caller.successCallback != undefined){
//                        	caller.successCallback(entities, caller.caller)
//                            caller.successCallback = undefined
//                    	}
//                    }
//                },
//                function (errMsg, caller) {
//                    // call the immediate error callback.
//                    if( caller.errorCallback != undefined){
//                    		caller.errorCallback(errMsg, caller.caller)
//							caller.errorCallback = undefined
//					  }
//                    // dispatch the message.
//                    server.errorManager.onError(errMsg)
//                }, // Error callback
//                { "caller": caller, "successCallback": successCallback, "progressCallback": progressCallback, "errorCallback": errorCallback, "prototype": result } // The caller
//            )
//        },
//        // The error callback.
//        function (errMsg, caller) {
//          	// call the immediate error callback.
//         		if( caller.errorCallback != undefined){
//            		caller.errorCallback(errMsg, caller.caller)
//					caller.errorCallback = undefined
//				}
//            // dispatch the message.
//            server.errorManager.onError(errMsg)
//        }, { "typeName": typeName, "storeId": storeId, "query": query, "caller": caller, "successCallback": successCallback, "progressCallback": progressCallback, "errorCallback": errorCallback })
//}
func (this *EntityManager) GetEntities(typeName string, storeId string, query *EntityQuery, offset int, limit int, orderBy []interface{}, asc bool, messageId string, sessionId string) []Entity {

	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	entities, errObj := this.getEntities(typeName, storeId, query)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// If no order ar specified i will use the id's as order.
	if len(orderBy) == 0 {
		// Here I will sort by it it's without it uuid...
		prototype, err := this.getEntityPrototype(typeName, typeName[0:strings.Index(typeName, ".")])
		if err != nil {
			return nil // The prototype was no foud here.
		}
		for i := 1; i < len(prototype.Ids); i++ {
			if !strings.HasPrefix("[]", prototype.FieldsType[prototype.getFieldIndex(prototype.Ids[i])]) {
				orderBy = append(orderBy, prototype.Ids[i])
			}
		}
	}

	// Sort the entities
	/*this.sortEntities(entities, orderBy, 0, len(entities), asc)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return objects
	}*/

	if limit <= 0 {
		// all results are require.
		limit = len(entities)
	}

	// Return the subset of entities.
	return entities[offset:limit]

}

// @api 1.0
// That function is use to retreive objects with a given type.
// @param {string} uuid The uuid of the entity we looking for. The uuid must has form typeName%UUID.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{interface{}} Return an object (Entity)
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.getEntityByUuid = function (uuid, successCallback, errorCallback, caller) {
//    if(uuid.length == 0){
//		console.log("No uuid to found!")
//		return
//	  }
//    var entity = entities[uuid]
//    if (entity != undefined) {
//        if (entity.TYPENAME == entity.__class__ && entity.IsInit == true) {
//            successCallback(entity, caller)
//            return // break it here.
//        }
//    }
//    var typeName = uuid.substring(0, uuid.indexOf("%"))
//    var storeId = typeName.substring(0, typeName.indexOf("."))
//    // Create the entity prototype here.
//    var entity = eval("new " + typeName + "(caller.prototype)")
//    entity.UUID = uuid
//    entity.TYPENAME = typeName
//    server.entityManager.setEntity(entity)
//    // First of all i will get the entity prototype.
//    server.entityManager.getEntityPrototype(typeName, storeId,
//        // The success callback.
//        function (result, caller) {
//            // Set the parameters.
//            var uuid = caller.uuid
//            var successCallback = caller.successCallback
//            var progressCallback = caller.progressCallback
//            var errorCallback = caller.errorCallback
//            var caller = caller.caller
//            var params = []
//            params.push(createRpcData(uuid, "STRING", "uuid"))
//            // Call it on the server.
//            server.executeJsFunction(
//                "EntityManagerGetEntityByUuid", // The function to execute remotely on server
//                params, // The parameters to pass to that function
//                function (index, total, caller) { // The progress callback
//                    // Nothing special to do here.
//                },
//                function (result, caller) {
//                    var entity = entities[result[0].UUID]
//                    entity.initCallback = function (caller) {
//                        return function (entity) {
//                          server.entityManager.setEntity(entity)
//							if(caller.successCallback != undefined){
//                            	caller.successCallback(entity, caller.caller)
//								caller.successCallback = undefined
//							}
//                        }
//                    } (caller)
//                    if (entity.IsInit == false) {
//                        entity.init(result[0])
//                    } else {
//						if(caller.successCallback != undefined){
//                            caller.successCallback(entity, caller.caller)
//							caller.successCallback = undefined
//						}
//                    }
//                },
//                function (errMsg, caller) {
//                  server.errorManager.onError(errMsg)
//         			if( caller.errorCallback != undefined){
//            			caller.errorCallback(errMsg, caller.caller)
//						caller.errorCallback = undefined
//					}
//                }, // Error callback
//                { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback, "prototype": result } // The caller
//            )
//        },
//        // The error callback.
//        function (errMsg, caller) {
//          server.errorManager.onError(errMsg)
//         	if( caller.errorCallback != undefined){
//          	caller.errorCallback(errMsg, caller.caller)
//				caller.errorCallback = undefined
//			}
//        }, { "uuid": uuid, "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback })
//}
func (this *EntityManager) GetEntityByUuid(uuid string, messageId string, sessionId string) interface{} {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	entity, errObj := this.getEntityByUuid(uuid)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	return entity
}

// @api 1.0
// Retrieve an entity with a given typename and id.
// @param {string} typeName The object type name.
// @param {string} storeId The object type name.
// @param {string} ids The id's (not uuid) of the object to look for.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{interface{}} Return an object (Entity)
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.getEntityById = function (typeName, storeId, ids, successCallback, errorCallback, caller, parent) {
//    if (!isArray(ids)) {
//        console.log("ids must be an array! ", ids)
//    }
//    // key in the server.
//    var id = typeName + ":"
//    for (var i = 0; i < ids.length; i++) {
//        id += ids[i]
//        if (i < ids.length - 1) {
//            id += "_"
//        }
//    }
//    if (entities[id] != undefined) {
//        successCallback(entities[id], caller)
//        return // break it here.
//    }
//    // First of all i will get the entity prototype.
//    server.entityManager.getEntityPrototype(typeName, storeId,
//        // The success callback.
//        function (result, caller) {
//            // Set the parameters.
//            var storeId = caller.storeId
//            var typeName = caller.typeName
//            var ids = caller.ids
//            var successCallback = caller.successCallback
//            var progressCallback = caller.progressCallback
//            var errorCallback = caller.errorCallback
//            var caller = caller.caller
//            var params = []
//            params.push(createRpcData(typeName, "STRING", "typeName"))
//            params.push(createRpcData(storeId, "STRING", "storeId"))
//            params.push(createRpcData(ids, "JSON_STR", "ids")) // serialyse as an JSON object array...
//            // Call it on the server.
//            server.executeJsFunction(
//                "EntityManagerGetEntityById", // The function to execute remotely on server
//                params, // The parameters to pass to that function
//                function (index, total, caller) { // The progress callback
//                    // Nothing special to do here.
//                },
//                function (result, caller) {
//                    if (result[0] == null) {
//                        return
//                    }
//                    // In case of existing entity.
//                    if (entities[result[0].UUID] != undefined && result[0].TYPENAME == result[0].__class__) {
//						if(caller.successCallback != undefined){
//                        	caller.successCallback(entities[result[0].UUID], caller.caller)
//							caller.successCallback = undefined
//						}
//                        return // break it here.
//                    }
//                    var entity = eval("new " + caller.prototype.TypeName + "(caller.prototype)")
//                    entity.initCallback = function () {
//                        return function (entity) {
//							if(caller.successCallback != undefined){
//                            	caller.successCallback(entity, caller.caller)
//								caller.successCallback = undefined
//							}
//                        }
//                    } (caller)
//                    entity.init(result[0])
//                },
//                function (errMsg, caller) {
//          		server.errorManager.onError(errMsg)
//         			if( caller.errorCallback != undefined){
//          			caller.errorCallback(errMsg, caller.caller)
//						caller.errorCallback = undefined
//					}
//                }, // Error callback
//                { "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback, "prototype": result, "parent": parent, "ids": ids } // The caller
//            )
//        },
//        // The error callback.
//        function (errMsg, caller) {
//            server.errorManager.onError(errMsg)
//            caller.errorCallback(errMsg, caller)
//        }, { "storeId": storeId, "typeName": typeName, "ids": ids, "caller": caller, "successCallback": successCallback, "errorCallback": errorCallback })
//}
func (this *EntityManager) GetEntityById(typeName string, storeId string, ids []interface{}, messageId string, sessionId string) interface{} {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	entity, errObj := this.getEntityById(typeName, storeId, ids)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}
	return entity
}

// @api 1.0
// Take an array of id's in the same order as the entity prototype Id's and
// generate a dertermistic UUID from it.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{string} Return Derteministic Universal Unique Identifier string
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
func (this *EntityManager) GenerateEntityUUID(typeName string, parentUuid string, ids []interface{}, messageId string, sessionId string) string {
	if len(ids) == 0 {
		// if there is no ids in the entity I will generate a random uuid.
		return typeName + "%" + Utility.RandomUUID()
	}

	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return ""
	}
	var keyInfo string
	if len(parentUuid) > 0 {
		keyInfo += parentUuid + ":"
	}
	keyInfo = typeName + ":"
	for i := 0; i < len(ids); i++ {
		if reflect.TypeOf(ids[i]).Kind() == reflect.String {
			keyInfo += ids[i].(string)
		} else if reflect.TypeOf(ids[i]).Kind() == reflect.Int {
			keyInfo += strconv.Itoa(ids[i].(int))
		} else if reflect.TypeOf(ids[i]).Kind() == reflect.Int8 {
			keyInfo += strconv.Itoa(int(ids[i].(int8)))
		} else if reflect.TypeOf(ids[i]).Kind() == reflect.Int16 {
			keyInfo += strconv.Itoa(int(ids[i].(int16)))
		} else if reflect.TypeOf(ids[i]).Kind() == reflect.Int32 {
			keyInfo += strconv.Itoa(int(ids[i].(int32)))
		} else if reflect.TypeOf(ids[i]).Kind() == reflect.Int64 {
			keyInfo += strconv.Itoa(int(ids[i].(int64)))
		}
		// Append underscore for readability in case of problem...
		if i < len(ids)-1 {
			keyInfo += "_"
		}
	}
	// Return the uuid from the input information.
	return typeName + "%" + Utility.GenerateUUID(keyInfo)
}
