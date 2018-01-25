package Server

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"code.myceliUs.com/CargoWebServer/Cargo/Entities/CargoEntities"
	"code.myceliUs.com/Utility"
	"code.myceliUs.com/XML_Schemas"
)

////////////////////////////////////////////////////////////////////////////////
//						Entity
////////////////////////////////////////////////////////////////////////////////

/**
 * Struct used to keep information about references.
 */
type EntityRef struct {
	Name      string
	OwnerUuid string
	Value     string
}

////////////////////////////////////////////////////////////////////////////////
//						Entity Manager
////////////////////////////////////////////////////////////////////////////////
type EntityManager struct {

	/**
	 * ref -> entity
	 */
	referenced map[string][]EntityRef // TODO remove when reference are initialyse.

	/**
	 * entity -> ref
	 */
	reference map[string][]EntityRef

	/**
	 * Use to protected the ressource access...
	 */
	sync.RWMutex
}

var entityManager *EntityManager

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

	// References
	entityManager.referenced = make(map[string][]EntityRef, 0)
	entityManager.reference = make(map[string][]EntityRef, 0)

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

	// I will create the cargo entities if it dosent already exist.
	cargoEntitiesUuid := CargoEntitiesEntitiesExists("CARGO_ENTITIES")
	if len(cargoEntitiesUuid) == 0 {
		cargoEntities := this.NewCargoEntitiesEntitiesEntity("", "CARGO_ENTITIES", nil)
		cargoEntities.object.M_id = "CARGO_ENTITIES"
		cargoEntities.object.M_name = "Cargo entities"
		cargoEntities.object.M_version = "1.0"
		cargoEntities.object.NeedSave = true
		cargoEntities.SaveEntity()
	}

	// Force complete intialysation of action.
	this.getEntities("CargoEntities.Action", nil, "CargoEntities", false)

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

/**
 * Cargo entities contains files, accounts, etc...
 */
func (this *EntityManager) getCargoEntities() *CargoEntities_EntitiesEntity {
	cargoEntitiesUuid := CargoEntitiesEntitiesExists("CARGO_ENTITIES")
	cargoEntities, _ := this.getEntityByUuid(cargoEntitiesUuid, true)
	return cargoEntities.(*CargoEntities_EntitiesEntity)
}

/**
 * Wrapper to the cacheManager's 'contains()' function.
 * Determines if the entity exists in the cacheManager map.
 */
func (this *EntityManager) contain(uuid string) (Entity, bool) {
	// Set the cache...
	return server.GetCacheManager().contains(uuid)
}

/**
 * Wrapper to the cacheManager's 'setEntity()' function.
 * Inserts entity in the cacheManager's map if it doesn't already exist.
 * Otherwise replaces the entity in the map with this entity.
 */
func (this *EntityManager) insert(entity Entity) {
	if entity.GetObject() != nil {
		server.GetCacheManager().setEntity(entity)
	}
}

/**
 * Wrapper to the cacheManager's 'removeEntity()' function.
 * Removes an existing entity with a given uuid from the cacheManager's map.
 */
func (this *EntityManager) removeEntity(uuid string) {
	server.GetCacheManager().removeEntity(uuid)
}

/**
 * That function is use to delete an entity from the store.
 */
func (this *EntityManager) deleteEntity(toDelete Entity) {
	log.Println(toDelete)
	// first of all i will remove it from the cache.
	this.removeEntity(toDelete.GetUuid())

	storeId := toDelete.GetTypeName()[0:strings.Index(toDelete.GetTypeName(), ".")]
	if reflect.TypeOf(GetServer().GetDataManager().getDataStore(storeId)).String() == "*Server.SqlDataStore" {
		// I will try to found the prototype inside sql_info instead.
		storeId = "sql_info"
	}

	// remove it's data from the database.
	var deleteEntityQuery EntityQuery
	deleteEntityQuery.TypeName = toDelete.GetTypeName()
	deleteEntityQuery.Indexs = append(deleteEntityQuery.Indexs, "UUID="+toDelete.GetUuid())
	var params []interface{}
	query, _ := json.Marshal(deleteEntityQuery)

	// Here I will try to delete the data... sometime because of the cascade
	// rule of sql the data is already deleted so error here dosent stop
	// the execution of the reste of entity suppression.
	GetServer().GetDataManager().deleteData(storeId, string(query), params)

	// delete it's childs.
	for i := 0; i < len(toDelete.GetChildsUuid()); i++ {
		child, _ := GetServer().GetEntityManager().getEntityByUuid(toDelete.GetChildsUuid()[i], false)
		child.DeleteEntity()
	}

	// This variable will keep track of other entity to save after that entity will be
	// deleted.
	toSaves := make([]Entity, 0)

	// Now it's references/childs in it's owner/parent...
	for i := 0; i < len(toDelete.GetReferenced()); i++ {
		ref := toDelete.GetReferenced()[i]
		if this.isExist(ref.OwnerUuid) {
			refOwner, err := this.getEntityByUuid(ref.OwnerUuid, true)
			if err == nil {
				prototype := refOwner.GetPrototype()

				fieldIndex := prototype.getFieldIndex(ref.Name)
				if !strings.HasPrefix(ref.Name, "M_") && fieldIndex == -1 {
					fieldIndex = prototype.getFieldIndex("M_" + ref.Name)
				}

				fieldType := prototype.FieldsType[fieldIndex]

				isRef := strings.HasSuffix(fieldType, ":Ref")
				if isRef {
					refOwner.RemoveReference(ref.Name, toDelete)
				} else {
					refOwner.RemoveChild(ref.Name, toDelete.GetUuid())
				}
			}

			// append save only once.
			isExist := false
			for i := 0; i < len(toSaves); i++ {
				if toSaves[i].GetUuid() == refOwner.GetUuid() {
					isExist = true
				}
			}
			if !isExist {
				toSaves = append(toSaves, refOwner)
			}
		}
	}

	// Now I will remove it from it parent if is not already deleted...
	if this.isExist(toDelete.GetParentUuid()) {
		if toDelete.GetParentPtr() != nil {
			// First I will remove it from parent childs...
			parent := toDelete.GetParentPtr()
			parent.RemoveChild(toDelete.GetParentLnk(), toDelete.GetUuid())
			parent.SetNeedSave(true)
			parent.SaveEntity()
		}
	}

	// Save refeferenced entity...
	for i := 0; i < len(toSaves); i++ {
		// Save it only if it dosen't already deleted.
		if toSaves[i].Exist() {
			toSaves[i].SaveEntity()
		}
	}

	// Send event message...
	var eventDatas []*MessageData
	evtData := new(MessageData)
	evtData.TYPENAME = "Server.MessageData"
	evtData.Name = "entity"

	evtData.Value = toDelete.GetObject()
	eventDatas = append(eventDatas, evtData)
	evt, _ := NewEvent(DeleteEntityEvent, EntityEvent, eventDatas)
	GetServer().GetEventManager().BroadcastEvent(evt)

	// Remove the ownership if there is one.
	this.removeEntityOwner(toDelete)
	log.Println("--> entity ", toDelete.GetUuid(), " is remove ", !this.isExist(toDelete.GetUuid()))
}

/**
 * Append a reference to an object. (Owner Uuid)
 */
func (this *EntityManager) appendReferences(ref EntityRef) {
	this.Lock()
	defer this.Unlock()
	if this.reference[ref.OwnerUuid] == nil {
		this.reference[ref.OwnerUuid] = make([]EntityRef, 0)
	}
	this.reference[ref.OwnerUuid] = append(this.reference[ref.OwnerUuid], ref)
}

/**
 * Return the list of reference for a given object.
 */
func (this *EntityManager) getReferences(uuid string) []EntityRef {
	this.Lock()
	defer this.Unlock()
	references := this.reference[uuid]
	return references
}

/**
 * Remove a references of an entity.
 */
func (this *EntityManager) removeReferences(uuid string) {
	this.Lock()
	defer this.Unlock()
	delete(this.reference, uuid)
}

/**
 * Append a reference to an object. (Owner Uuid)
 */
func (this *EntityManager) appendReferenceds(targetId string, ref EntityRef) {
	this.Lock()
	defer this.Unlock()
	if this.referenced[targetId] == nil {
		this.referenced[targetId] = make([]EntityRef, 0)
	}
	this.referenced[targetId] = append(this.referenced[targetId], ref)
}

/**
 * Return the list of reference for a given object.
 */
func (this *EntityManager) getReferenceds(uuid string) []EntityRef {
	this.Lock()
	defer this.Unlock()
	references := this.referenced[uuid]
	return references
}

/**
 * Remove a references of an entity.
 */
func (this *EntityManager) removeReferenceds(uuid string) {
	this.Lock()
	defer this.Unlock()
	delete(this.reference, uuid)
}

/**
 * Set the list of reference of a given owner entity.
 */
func (this *EntityManager) setReferences(owner Entity) {
	prototype := owner.GetPrototype()
	for i := 0; i < len(prototype.FieldsType); i++ {
		fieldType := prototype.FieldsType[i]
		if strings.HasSuffix(fieldType, ":Ref") {
			fieldName := prototype.Fields[i]
			uuids := make([]string, 0)
			if reflect.TypeOf(owner.GetObject()).String() == "map[string]interface {}" {
				// Dynamic entity here.
				objectMap := owner.GetObject().(map[string]interface{})
				if objectMap[fieldName] != nil {
					if reflect.TypeOf(objectMap[fieldName]).String() == "string" {
						uuids = append(uuids, objectMap[fieldName].(string))
					} else if reflect.TypeOf(objectMap[fieldName]).String() == "[]interface {}" {
						for j := 0; j < len(objectMap[fieldName].([]interface{})); j++ {
							fieldValue := objectMap[fieldName].([]interface{})[j]
							if reflect.TypeOf(objectMap[fieldName]).String() == "string" {
								uuids = append(uuids, fieldValue.(string))
							}
						}
					} else if reflect.TypeOf(objectMap[fieldName]).String() == "[]string" {
						for j := 0; j < len(objectMap[fieldName].([]string)); j++ {
							uuids = append(uuids, objectMap[fieldName].([]string)[j])
						}
					}
				}
			} else {
				// In case of static entity...
				ps := reflect.ValueOf(owner.GetObject())
				s := ps.Elem()

				if s.Kind() == reflect.Struct {
					if strings.HasPrefix(fieldName, "M_") {
						fieldName = "m_" + fieldName[2:]
					}

					f := s.FieldByName(fieldName)
					if f.IsValid() {
						if f.Kind() == reflect.String {
							uuid := f.String()
							if len(uuid) > 0 {
								uuids = append(uuids, uuid)
							}
						} else if f.Kind() == reflect.Slice {
							for j := 0; j < f.Len(); j++ {
								if f.Index(j).Kind() == reflect.String {
									uuid := f.Index(j).String()
									if len(uuid) > 0 {
										uuids = append(uuids, uuid)
									}
								} else {
									if f.Index(j).Kind() == reflect.Ptr {
										// Here I have a structure.
										uuid := f.Index(j).Elem().FieldByName("UUID").String()
										if len(uuid) > 0 {
											uuids = append(uuids, uuid)
										}

									}
								}
							}
						} else if f.Kind() == reflect.Ptr {
							// Here I have a structure.
							if !f.IsNil() {
								uuid := f.Elem().FieldByName("UUID").String()
								if len(uuid) > 0 {
									uuids = append(uuids, uuid)
								}
							}
						}
					}
				}
			}

			// Now I will try to append the reference inside the
			// entity.
			for j := 0; j < len(uuids); j++ {
				reference, err := this.getEntityByUuid(uuids[j], true)
				if err == nil {
					owner.AppendReference(reference)
				}
			}
		}
	}
}

/**
 * Set the content of a target object whit the source object. use by static entity
 * only...
 */
func (this *EntityManager) setObjectValues(target Entity, source interface{}) {

	// here we have a static object...
	prototype := target.GetPrototype()
	if prototype == nil {
		log.Println("No prototype found for ---> ", target)
	}

	// The need save evaluation...
	mapValues, _ := Utility.ToMap(source)
	needSave := target.GetChecksum() != Utility.GetChecksum(mapValues)
	target.SetNeedSave(needSave)

	// in case of dynamic object...
	if reflect.TypeOf(source).String() == "map[string]interface {}" {
		target.(*DynamicEntity).SetObjectValues(source.(map[string]interface{}))
	}

	// I will get the target object.
	targetReflexObject := reflect.ValueOf(target.GetObject())
	sourceFelfexObject := reflect.ValueOf(source)

	// First of all I will reset the values that are not in the target but are
	// in the source.
	for i := 0; i < len(prototype.FieldsType); i++ {
		fieldName := prototype.Fields[i]
		fieldType := prototype.FieldsType[i]
		isArray := strings.HasPrefix(fieldType, "[]")
		isRef := strings.HasSuffix(fieldType, ":Ref")
		isBaseType := strings.HasPrefix(fieldType, "[]xs.") || strings.HasPrefix(fieldType, "xs.")
		isEnum := strings.HasPrefix(fieldType, "enum")

		var targetField reflect.Value
		if targetReflexObject.Kind() == reflect.Ptr {
			targetField = targetReflexObject.Elem().FieldByName(fieldName)
		} else {
			targetField = targetReflexObject.FieldByName(fieldName)
		}

		var sourceField reflect.Value
		if targetReflexObject.Kind() == reflect.Ptr {
			sourceField = sourceFelfexObject.Elem().FieldByName(fieldName)
		} else {
			sourceField = sourceFelfexObject.FieldByName(fieldName)
		}

		if targetField.IsValid() {
			if isBaseType || isEnum {
				if !sourceField.IsValid() {
					if isArray {
						targetField.Set(reflect.ValueOf(make([]interface{}, 0)))
					} else {
						var val interface{}
						targetField.Set(reflect.ValueOf(val))
					}
				}
			} else if isRef {
				var removeMethode = "Remove" + strings.ToUpper(fieldName[2:3]) + fieldName[3:]
				var setMethode = "Set" + strings.ToUpper(fieldName[2:3]) + fieldName[3:]
				if isArray {
					for i := 0; i < targetField.Len(); i++ {
						ref := targetField.Index(i)
						needToBeRemove := true
						for j := 0; j < sourceField.Len(); j++ {
							if sourceField.Index(j) == ref {
								needToBeRemove = false
								break
							}
						}
						if needToBeRemove && len(ref.String()) > 0 {
							toRemove, _ := this.getEntityByUuid(ref.String(), true)
							target.RemoveReference(fieldName, toRemove)
							toRemove.RemoveReferenced(fieldName, target)
							// I will call remove function...
							params := make([]interface{}, 1)
							params[0] = toRemove.GetObject()
							Utility.CallMethod(target.GetObject(), removeMethode, params)
						}
					}
					// Append the references...
					for i := 0; i < sourceField.Len(); i++ {
						if len(sourceField.Index(i).String()) > 0 {
							ref, err := this.getEntityByUuid(sourceField.Index(i).String(), true)
							if err == nil {
								target.AppendReference(ref)
								ref.AppendReferenced(fieldName, target)
								// Now I will call the append method...
								params := make([]interface{}, 1)
								params[0] = ref.GetObject()
								Utility.CallMethod(target.GetObject(), setMethode, params)
							} else {
								params := make([]interface{}, 1)
								params[0] = sourceField.Index(i).String()
								Utility.CallMethod(target.GetObject(), setMethode, params)
							}
						}
					}
				} else {
					// Remove the reference...
					if sourceField.String() != targetField.String() {
						if len(targetField.String()) > 0 {
							toRemove, err := this.getEntityByUuid(targetField.String(), true)
							if err == nil {
								target.RemoveReference(fieldName, toRemove)
								toRemove.RemoveReferenced(fieldName, target)
								// I will call remove function...
								params := make([]interface{}, 1)
								params[0] = toRemove.GetObject()
								Utility.CallMethod(target.GetObject(), removeMethode, params)
							}
						}
					}
					// Append the reference...
					if len(sourceField.String()) > 0 {
						ref, err := this.getEntityByUuid(sourceField.String(), true)
						if err == nil {
							target.AppendReference(ref)
							ref.AppendReferenced(fieldName, target)
							// Now I will call the append method...
							params := make([]interface{}, 1)
							params[0] = ref.GetObject()
							Utility.CallMethod(target.GetObject(), setMethode, params)
						} else {
							params := make([]interface{}, 1)
							params[0] = sourceField.String()
							Utility.CallMethod(target.GetObject(), setMethode, params)
						}
					}
				}
			}
		}
	}

	// set the new field values.
	for i := 0; i < len(prototype.FieldsType); i++ {
		fieldName := prototype.Fields[i]
		fieldType := prototype.FieldsType[i]
		isArray := strings.HasPrefix(fieldType, "[]")
		isRef := strings.HasSuffix(fieldType, ":Ref")
		isBaseType := strings.HasPrefix(fieldType, "[]xs.") || strings.HasPrefix(fieldType, "xs.")
		isEnum := strings.HasPrefix(fieldType, "enum")

		var targetField reflect.Value
		if targetReflexObject.Kind() == reflect.Ptr {
			targetField = targetReflexObject.Elem().FieldByName(fieldName)
		} else {
			targetField = targetReflexObject.FieldByName(fieldName)
		}

		var sourceField reflect.Value
		if targetReflexObject.Kind() == reflect.Ptr {
			sourceField = sourceFelfexObject.Elem().FieldByName(fieldName)
		} else {
			sourceField = sourceFelfexObject.FieldByName(fieldName)
		}

		if sourceField.IsValid() {
			if targetField.IsValid() {

				if isBaseType || isEnum {
					// set the value(s)...
					targetField.Set(sourceField)
				} else if !isRef {
					var removeMethode = "Remove" + strings.ToUpper(fieldName[2:3]) + fieldName[3:]
					if isArray {
						// First I will remove the object in the target that are no more in the
						// source.
						for i := 0; i < targetField.Len(); i++ {
							toRemove := targetField.Index(i)
							needToBeRemove := true
							for j := 0; j < sourceField.Len(); j++ {
								subObject := sourceField.Index(j)
								var params []interface{}
								var subObjectUuid string
								val0, err0 := Utility.CallMethod(subObject.Interface(), "GetUUID", params)
								if err0 == nil {
									subObjectUuid = val0.(string)
								}

								var toRemoveUuid string
								val1, err1 := Utility.CallMethod(toRemove.Interface(), "GetUUID", params)
								if err1 == nil {
									toRemoveUuid = val1.(string)
								}

								if toRemoveUuid == subObjectUuid {
									needToBeRemove = false
									break
								}
							}
							if needToBeRemove {
								params := make([]interface{}, 1)
								params[0] = toRemove.Interface()
								Utility.CallMethod(target.GetObject(), removeMethode, params)
							}
						}

						// Now I will set the field...
						for j := 0; j < sourceField.Len(); j++ {
							subObject := sourceField.Index(j)
							if subObject.Interface() != nil {
								var params []interface{}
								val, err := Utility.CallMethod(subObject.Interface(), "GetUUID", params)
								if err == nil {
									subObjectUuid := val.(string)
									subEntity, err := this.getEntityByUuid(subObjectUuid, true)
									if err == nil {
										this.setObjectValues(subEntity, subObject.Interface())
									}
									setMethodName := strings.Replace(fieldName, "M_", "", -1)
									setMethodName = "Set" + strings.ToUpper(setMethodName[0:1]) + setMethodName[1:]
									params := make([]interface{}, 1)
									params[0] = subObject.Interface()
									_, err_ := Utility.CallMethod(target.GetObject(), setMethodName, params)
									if err_ != nil {
										log.Println("fail to call method ", setMethodName, " on ", target.GetObject())
									}
								} else {
									log.Println("----------> fail to call method GetUUID on ", subObject)
								}
							} else {
								// TODO remove the value here.
								// Remove the value here...
							}
						}
					} else {
						// Clear the actual value...
						if !sourceField.IsNil() {
							// remove the existing value

							// Set the new value.
							var subObjectUuid string
							var params []interface{}
							if sourceField.Interface() != nil {
								val, err := Utility.CallMethod(sourceField.Interface(), "GetUUID", params)
								if err == nil {
									subObjectUuid = val.(string)
								} else {
									log.Println("----------> fail to call method GetUUID on ", sourceField.Interface())
								}

								subEntity, err := this.getEntityByUuid(subObjectUuid, true)
								if err == nil {
									this.setObjectValues(subEntity, sourceField.Interface())
									setMethodName := strings.Replace(fieldName, "M_", "", -1)
									setMethodName = "Set" + strings.ToUpper(setMethodName[0:1]) + setMethodName[1:]
									params := make([]interface{}, 1)
									params[0] = sourceField.Interface()
									_, err := Utility.CallMethod(target.GetObject(), setMethodName, params)
									if err != nil {
										log.Println("fail to call method ", setMethodName, " on ", target.GetObject())
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

/**
 * Return an entity with for a given type and id
 */
func (this *EntityManager) getEntityById(storeId string, typeName string, ids []interface{}, lazy bool) (Entity, *CargoEntities.Error) {
	// Verify that typeName is valid
	// interface{} is an exception...
	if !Utility.IsValidPackageName(typeName) && !strings.HasSuffix(typeName, "interface{}") {
		cargoError := NewError(Utility.FileLine(), INVALID_PACKAGE_NAME_ERROR, SERVER_ERROR_CODE, errors.New("Type name '"+typeName+"' is not valid."))
		return nil, cargoError
	}

	// If the store is not found I will return an error.
	if GetServer().GetDataManager().getDataStore(storeId) == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Datastore '"+storeId+"' dosen't exist."))
		return nil, cargoError
	}

	// The entity information are in sql_info and not the given store id...
	if reflect.TypeOf(GetServer().GetDataManager().getDataStore(storeId)).String() == "*Server.SqlDataStore" {
		// I will try to found the prototype inside sql_info instead.
		storeId = "sql_info"
	}

	prototype, err := this.getEntityPrototype(typeName, storeId)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Type name '"+typeName+"' does not exist."))
		return nil, cargoError
	}

	var results [][]interface{}
	if err == nil {
		var query EntityQuery
		query.TypeName = typeName
		query.Fields = append(query.Fields, "UUID")
		var fieldsType []interface{} // not used
		var params []interface{}

		if len(ids) != len(prototype.Ids)-1 {
			cargoError := NewError(Utility.FileLine(), DATASTORE_ERROR, SERVER_ERROR_CODE, errors.New("Unexpecter number of ids got "+strconv.Itoa(len(ids))+" expected "+strconv.Itoa(len(prototype.Ids))))
			return nil, cargoError
		}

		query.Indexs = make([]string, 0)
		for i := 1; i < len(prototype.Ids) && len(results) == 0; i++ {
			idField := prototype.Ids[i]
			query.Indexs = append(query.Indexs, idField+"="+Utility.ToString(ids[i-1]))
		}
		queryStr, _ := json.Marshal(query)
		results, err = GetServer().GetDataManager().readData(storeId, string(queryStr), fieldsType, params)
		if err != nil {
			// Create the error message
			cargoError := NewError(Utility.FileLine(), DATASTORE_ERROR, SERVER_ERROR_CODE, err)
			return nil, cargoError
		}

	}

	// In that case not information are found.
	if len(results) == 0 {
		// Before I will give up I will try to find value with sub-type.
		subTypes := prototype.SubstitutionGroup
		for i := 0; i < len(subTypes); i++ {
			entities, errObj := this.getEntityById(storeId, subTypes[i], ids, lazy)
			if errObj == nil {
				return entities, nil
			}
		}
		// Here I will send a an error...
		var id string
		for i := 0; i < len(ids); i++ {
			id += Utility.ToString(ids[i])
			if i < len(ids)-1 {
				id += " "
			}
		}
		cargoError := NewError(Utility.FileLine(), DATASTORE_ERROR, SERVER_ERROR_CODE, errors.New("No values found for type '"+typeName+"' and id '"+id+"'"))
		return nil, cargoError
	}

	entity, errObj := this.getEntityByUuid(results[0][0].(string), lazy)

	if errObj != nil || entity == nil {
		entity, errObj = this.getDynamicEntityByUuid(results[0][0].(string), lazy)
		if errObj != nil {
			return nil, errObj
		}
	}

	return entity, nil
}

/**
 * Return the list of entity type derived from a given type.
 */
func (this *EntityManager) getDerivedEntityType(typeName string) ([]*EntityPrototype, *CargoEntities.Error) {
	var derived []*EntityPrototype

	if !Utility.IsValidPackageName(typeName) {
		cargoError := NewError(Utility.FileLine(), INVALID_PACKAGE_NAME_ERROR, SERVER_ERROR_CODE, errors.New("Type name '"+typeName+"' is not valid."))
		return derived, cargoError
	}

	packageName := typeName[0:strings.Index(typeName, ".")]
	// Here I will retreive the supertype
	superTypePrototype, err := this.getEntityPrototype(typeName, packageName)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Type name '"+typeName+"' dosen't exist."))
		return derived, cargoError
	}

	for i := 0; i < len(superTypePrototype.SubstitutionGroup); i++ {
		packageName := superTypePrototype.SubstitutionGroup[i][0:strings.Index(superTypePrototype.SubstitutionGroup[i], ".")]
		substitutionGroup, err := this.getEntityPrototype(superTypePrototype.SubstitutionGroup[i], packageName)
		if err == nil {
			derived = append(derived, substitutionGroup)
		}
	}

	return derived, nil
}

/**
 * Return the list of entities for a given type name.
 */
func (this *EntityManager) getEntities(typeName string, query *EntityQuery, storeId string, lazy bool) ([]Entity, *CargoEntities.Error) {

	var entities []Entity

	if GetServer().GetDataManager().getDataStore(storeId) == nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("Datastore '"+storeId+"' dosen't exist."))
		return nil, cargoError
	}
	var dataStore DataStore
	dataStore = GetServer().GetDataManager().getDataStore(storeId)
	if reflect.TypeOf(dataStore).String() == "*Server.SqlDataStore" {
		dataStore = GetServer().GetDataManager().getDataStore("sql_info")
	}

	if query == nil {

		values, err := dataStore.(*KeyValueDataStore).getIndexation(typeName)
		if err != nil {
			errObj := NewError(Utility.FileLine(), DATASTORE_INDEXATION_ERROR, SERVER_ERROR_CODE, errors.New("No indexation for type '"+typeName+"'."))
			return entities, errObj
		}

		for i := 0; i < len(values); i++ {
			key := values[i].(string)
			values_, err := dataStore.(*KeyValueDataStore).getValues(key)
			if err != nil {
				return entities, NewError(Utility.FileLine(), DATASTORE_KEY_NOT_FOUND_ERROR, SERVER_ERROR_CODE, errors.New("No value found for key '"+key+"'."))
			}

			if len(values_) > 0 {
				uuid := values_[0].(string)
				entity, errObj := this.getEntityByUuid(uuid, lazy)
				if errObj != nil {
					entity, errObj = this.getDynamicEntityByUuid(uuid, lazy)
					if errObj != nil {
						return entities, errObj
					}
				}
				if entity != nil {
					entities = append(entities, entity)
				}
			}
		}

	} else {
		// Here I will create a new query and execute it...
		// I will retreive the uuid...
		query.TypeName = typeName
		query.Fields = []string{"UUID"} //append(query.Fields, "UUID")
		var fieldsType []interface{}    // not used
		var params []interface{}

		// Now I will execute the query...
		queryStr, _ := json.Marshal(query)

		results, err := GetServer().GetDataManager().readData(storeId, string(queryStr), fieldsType, params)
		if err != nil {
			// Create the error message
			cargoError := NewError(Utility.FileLine(), DATASTORE_ERROR, SERVER_ERROR_CODE, err)
			return nil, cargoError
		}

		if len(results) > 0 {
			for i := 0; i < len(results); i++ {
				uuid := results[i][0].(string)
				entity, _ := this.getEntityByUuid(uuid, lazy)
				entities = append(entities, entity)
			}
		}
	}

	return entities, nil
}

/**
 * Return the list of all entities for different types.
 */
func (this *EntityManager) getEntitiesByType(typeNames []string, storeId string, lazy bool) ([]Entity, *CargoEntities.Error) {
	entitiesMap := make(map[string]Entity)
	var entities []Entity

	for i := 0; i < len(typeNames); i++ {
		entities_, errObj := this.getEntities(typeNames[i], nil, storeId, lazy)
		if errObj != nil {
			return entities, errObj
		}
		for j := 0; j < len(entities_); j++ {
			entitiesMap[entities_[j].GetUuid()] = entities_[j]
		}
	}

	for _, entity := range entitiesMap {
		entities = append(entities, entity)
	}

	return entities, nil
}

func (this *EntityManager) getEntityByUuid(uuid string, lazy bool) (Entity, *CargoEntities.Error) {
	if !Utility.IsValidEntityReferenceName(uuid) {
		return nil, NewError(Utility.FileLine(), INVALID_REFERENCE_NAME_ERROR, SERVER_ERROR_CODE, errors.New("The uuid '"+uuid+"' is not valid."))
	}

	if val, ok := this.contain(uuid); ok {
		if lazy {
			return val, nil
		} else if !val.IsLazy() {
			return val, nil
		}

		// Remove the actual entity from the cache

		// Init it, it will introduce it after it.
		if strings.HasPrefix(uuid, "CargoEntities.Entities%") {
			return val, nil
		}

		val.SetInit(false) // force reinitialisation of the value.
		val.InitEntity(uuid, lazy)

		return val, nil
	}

	typeName := strings.Split(uuid, "%")[0]

	// Remove the suffix in that particular case.
	if strings.HasSuffix(typeName, "_impl") {
		typeName = strings.Replace(typeName, "_impl", "", -1)
	}

	funcName := "New" + strings.Replace(typeName, ".", "", -1) + "Entity"

	params := make([]interface{}, 3)
	params[0] = "" // No parent uuid needed.
	params[1] = uuid
	params[2] = nil

	this.Lock()
	result, err := Utility.CallMethod(this, funcName, params)
	this.Unlock()

	if err != nil {
		// Try with dynamic entity instead.
		entity, errObj := this.getDynamicEntityByUuid(uuid, lazy)

		if errObj != nil {
			return nil, errObj
		}
		return entity, nil
	}

	entity := result.(Entity)
	entity.InitEntity(uuid, lazy)
	// Here I will also set the reference for the entity...
	this.setReferences(entity)

	return entity, nil
}

func (this *EntityManager) getDynamicEntityByUuid(uuid string, lazy bool) (Entity, *CargoEntities.Error) {

	if val, ok := this.contain(uuid); ok {
		return val, nil
	}

	values := make(map[string]interface{}, 0)
	values["TYPENAME"] = strings.Split(uuid, "%")[0]
	values["UUID"] = uuid

	// here the parent uuid is not know.
	this.Lock()
	entity, errObj := this.newDynamicEntity("", values)
	this.Unlock()

	if errObj != nil {
		return nil, errObj
	}

	if !entity.Exist() {
		// Create the error message
		cargoError := NewError(Utility.FileLine(), ENTITY_UUID_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("The entity uuid '"+uuid+"' does not correspond to an existing entity."))
		// Return the error entity in the err return param.
		return nil, cargoError
	}

	// initialyse the entity.
	entity.InitEntity(uuid, lazy)

	return entity, nil

}

// Use at initialysation time (init)
func (this *EntityManager) appendReference(name string, ownerId string, value string) {

	if value != "null" && !strings.HasSuffix(value, "$$null") {
		var ref EntityRef
		ref.Name = name
		ref.OwnerUuid = ownerId
		ref.Value = value

		// append the reference.
		this.appendReferences(ref)

		owner, err := this.getEntityByUuid(ownerId, true)
		if err == nil {
			targetId := strings.Split(value, "$$")[1]
			if Utility.IsValidEntityReferenceName(targetId) {
				target, err := this.getEntityByUuid(targetId, true)
				if err == nil {
					// Set the reference...
					owner.AppendReference(target)
				}
			}
		}

	}
}

// Use at creation time by dynamic entity... (Save)
func (this *EntityManager) appendReferenced(name string, ownerId string, value string) {

	if value != "null" && !strings.HasSuffix(value, "$$null") {
		var ref EntityRef
		ref.Name = name
		ref.OwnerUuid = ownerId
		ref.Value = value

		// here the target id must be use to resolve references...
		targetId := value[strings.Index(value, "$$")+2:]

		// Here I will try to find the targeted object...
		if Utility.IsValidEntityReferenceName(targetId) {
			// The key is an uuid
			targetRef, err := GetServer().GetEntityManager().getEntityByUuid(targetId, true)
			if err == nil {
				ownerRef, _ := GetServer().GetEntityManager().getEntityByUuid(ownerId, true)
				ownerRef.AppendReference(targetRef)
				// Append the reference here...
				targetRef.AppendReferenced(name, ownerRef)
				//log.Println("-------> append referenced: ", targetId, ":", ownerId)
				// Save the target entity...
				targetRef.SaveEntity()
				return // nothing to do...
			}
		}

		// Here the target object does not exist so when it will be created and
		// save the reference to this owner object will be added to the target.
		this.appendReferenceds(targetId, ref)
	}
}

// used by dynamic entity only...
func (this *EntityManager) saveReferenced(entity Entity) {
	// Here I will try to find if there is pending reference for the object...
	typeName := entity.GetTypeName()
	packageName := typeName[0:strings.Index(typeName, ".")]
	prototype, err := this.getEntityPrototype(typeName, packageName)
	if err == nil {
		for i := 0; i < len(prototype.Ids); i++ {
			var referenced []EntityRef
			if i == 0 {
				// Get reference by uuid...
				referenced = this.getReferenceds(entity.GetUuid())
			} else {
				// Reference registered by other id...
				id := prototype.Ids[i]

				if reflect.TypeOf(entity.GetObject()).String() == "map[string]interface {}" {
					// A dynamic entity here...
					if entity.GetObject().(map[string]interface{})[id] != nil {
						var refId string
						if reflect.TypeOf(entity.GetObject().(map[string]interface{})[id]).Kind() == reflect.String {
							refId += entity.GetObject().(map[string]interface{})[id].(string)
						} else if reflect.TypeOf(entity.GetObject().(map[string]interface{})[id]).Kind() == reflect.Int {
							refId += strconv.Itoa(entity.GetObject().(map[string]interface{})[id].(int))
						} else if reflect.TypeOf(entity.GetObject().(map[string]interface{})[id]).Kind() == reflect.Int8 {
							refId += strconv.Itoa(int(entity.GetObject().(map[string]interface{})[id].(int8)))
						} else if reflect.TypeOf(entity.GetObject().(map[string]interface{})[id]).Kind() == reflect.Int16 {
							refId += strconv.Itoa(int(entity.GetObject().(map[string]interface{})[id].(int16)))
						} else if reflect.TypeOf(entity.GetObject().(map[string]interface{})[id]).Kind() == reflect.Int32 {
							refId += strconv.Itoa(int(entity.GetObject().(map[string]interface{})[id].(int32)))
						} else if reflect.TypeOf(entity.GetObject().(map[string]interface{})[id]).Kind() == reflect.Int64 {
							refId += strconv.Itoa(int(entity.GetObject().(map[string]interface{})[id].(int64)))
						}
						referenced = this.getReferenceds(refId)
					}
				} else {
					// A static entity here...
					id = strings.Replace(id, "M_", "", -1)
					methodName := "Get" + strings.ToUpper(id[0:1]) + id[1:]
					params := make([]interface{}, 0)
					result, err := Utility.CallMethod(entity.GetObject(), methodName, params)
					if err == nil {
						referenced = this.getReferenceds(result.(string))
					}
				}
			}

			for j := 0; j < len(referenced); j++ {
				// So here I will get the owner entity
				owner, _ := this.getEntityByUuid(referenced[j].OwnerUuid, true)
				if reflect.TypeOf(owner.GetObject()).String() == "map[string]interface {}" {
					// Set the uuid as value
					if owner.GetObject().(map[string]interface{})[referenced[j].Name] != entity.GetUuid() {
						owner.GetObject().(map[string]interface{})[referenced[j].Name] = entity.GetUuid()
						owner.SetNeedSave(true)
						owner.SaveEntity()
					}
				}

			}
		}
	}

}

/**
 * A function to initialyse an entity with a given id.
 */
func (this *EntityManager) InitEntity(entity Entity, lazy bool) {

	// Get the list of references.
	references := this.getReferences(entity.GetUuid())

	// Now I will call methode to initialyse the reference...
	for i := 0; i < len(references); i++ {
		ref := references[i]
		// Retreive the reference owner...
		refOwner, errObj := this.getEntityByUuid(ref.OwnerUuid, lazy)
		if errObj == nil {
			values := strings.Split(ref.Value, "$$")
			if len(values) == 2 && refOwner != nil {
				refUUID := values[1]
				typeName := values[0]
				var refTarget Entity
				if Utility.IsValidEntityReferenceName(refUUID) {
					// We have a uuid here
					refTarget, errObj = this.getEntityByUuid(refUUID, lazy)
				} else if len(refUUID) > 0 {
					// Here we have an id not a uuid...
					storeId := typeName[:strings.Index(typeName, ".")]
					ids := []interface{}{refUUID}
					refTarget, errObj = this.getEntityById(storeId, typeName, ids, lazy)
				}
				// The set methode name...
				if errObj == nil && refTarget != nil {
					methodName := "Set" + strings.ToUpper(ref.Name[0:1]) + ref.Name[1:]
					params := make([]interface{}, 1)
					params[0] = refTarget.GetObject()
					_, invalidMethod := Utility.CallMethod(refOwner.(Entity).GetObject(), methodName, params)
					if invalidMethod != nil {
						// Also append referenced into the ref owner.
						fieldIndex := refOwner.GetPrototype().getFieldIndex(ref.Name)
						fieldType := refOwner.GetPrototype().FieldsType[fieldIndex]
						if strings.HasSuffix(fieldType, ":Ref") {
							// TODO verify if the reference need to be set here.
							/*if strings.HasPrefix(fieldType, "[]") {
								if reflect.TypeOf(refOwner.(*DynamicEntity).object[ref.Name]).String() == "[]string" {
									if refOwner.(*DynamicEntity).object[ref.Name] == nil {
										refOwner.(*DynamicEntity).object[ref.Name] = make([]string, 0)
									}
									if !Utility.Contains(refOwner.(*DynamicEntity).object[ref.Name].([]string), refTarget.GetUuid()) {
										refOwner.(*DynamicEntity).object[ref.Name] = append(refOwner.(*DynamicEntity).object[ref.Name].([]string), refTarget.GetUuid())
									}

								} else if reflect.TypeOf(refOwner.(*DynamicEntity).object[ref.Name]).String() == "[]interface {}" {
									log.Println("--> wrong reference type")
								}
							} else {
								refOwner.(*DynamicEntity).object[ref.Name] = refTarget.GetUuid()
							}*/
							// Append the referenced
							refTarget.AppendReferenced(ref.Name, refOwner)
							refOwner.AppendReference(refTarget)
						}
					}

				} else if len(refUUID) > 0 {
					log.Println("--------> reference target not found:", refUUID)
				}
			}

		} else {
			log.Println("--------> reference owner not found:", ref.OwnerUuid)
			return
		}
	}

	// Now the entity is in the cache...
	this.insert(entity)

	// remove it from the list...
	this.removeReferences(entity.GetUuid())
}

/**
 * Return the list of entity prototype for a given package...
 */
func (this *EntityManager) getEntityPrototypes(storeId string, schemaId string) ([]*EntityPrototype, error) {
	if GetServer().GetDataManager().getDataStore(storeId) == nil {
		return nil, errors.New("The dataStore with id '" + storeId + "' doesn't exist.")
	}

	dataStore := GetServer().GetDataManager().getDataStore(storeId)
	if reflect.TypeOf(dataStore).String() == "*Server.SqlDataStore" {
		// I will try to found the prototype inside sql_info instead.
		dataStore = GetServer().GetDataManager().getDataStore("sql_info")
	}

	protos, err := dataStore.GetEntityPrototypes()
	prototypes := make([]*EntityPrototype, 0)

	if err == nil {
		for i := 0; i < len(protos); i++ {
			if strings.HasPrefix(protos[i].TypeName, schemaId) {
				prototypes = append(prototypes, protos[i])
			}
		}
	} else {
		log.Println("--->", err)
	}

	return prototypes, err
}

/**
 * Return the entity prototype for an object of a given name.
 */
func (this *EntityManager) getEntityPrototype(typeName string, storeId string) (*EntityPrototype, error) {

	if GetServer().GetDataManager().getDataStore(storeId) == nil {
		return nil, errors.New("The dataStore with id '" + storeId + "' doesn't exist.")
	}

	dataStore := GetServer().GetDataManager().getDataStore(storeId)
	if reflect.TypeOf(dataStore).String() == "*Server.SqlDataStore" {
		// I will try to found the prototype inside sql_info instead.
		dataStore = GetServer().GetDataManager().getDataStore("sql_info")
	}

	proto, err := dataStore.GetEntityPrototype(typeName)

	if err != nil {
		err = errors.New("Prototype for entity '" + typeName + "' was not found." + err.Error())
		return nil, err
	}

	return proto, err
}

/**
 * Remove a given prototype from a store and clean up all object of it type.
 */
func (this *EntityManager) deleteEntityPrototype(storeId string, prototype *EntityPrototype) error {

	// Now I will remove the prototype itself from the db.
	store := GetServer().GetDataManager().getDataStore(storeId)
	store.DeleteEntityPrototype(prototype.TypeName)

	// send event here.
	var eventDatas []*MessageData

	evtData := new(MessageData)
	evtData.TYPENAME = "Server.MessageData"
	evtData.Name = "prototype"
	evtData.Value = prototype

	// push in the data array.
	eventDatas = append(eventDatas, evtData)

	evt, _ := NewEvent(DeletePrototypeEvent, PrototypeEvent, eventDatas)
	GetServer().GetEventManager().BroadcastEvent(evt)

	return nil
}

/**
 * Create a new entity with default value and append it inside it parent...
 *
 * TODO Est que "The attributeName is the name of the entity in it's parent whitout the M_" est vrai ou on doit lui donner avec le M_?
 */
func (this *EntityManager) createEntity(parentUuid string, attributeName string, typeName string, objectId string, values interface{}) (Entity, *CargoEntities.Error) {

	if !Utility.IsValidPackageName(typeName) {
		cargoError := NewError(Utility.FileLine(), INVALID_PACKAGE_NAME_ERROR, SERVER_ERROR_CODE, errors.New("The typeName '"+typeName+"' is not valid."))
		return nil, cargoError
	}

	var parentPtr Entity
	if len(parentUuid) > 0 {
		if !Utility.IsValidEntityReferenceName(parentUuid) {
			cargoError := NewError(Utility.FileLine(), INVALID_REFERENCE_NAME_ERROR, SERVER_ERROR_CODE, errors.New("The parentUuid '"+parentUuid+"' is not valid."))
			return nil, cargoError
		}
		var errObj *CargoEntities.Error
		parentPtr, errObj = this.getEntityByUuid(parentUuid, true)
		if errObj != nil {
			return nil, errObj
		}
	}

	// Try to cast the value as needed...
	if reflect.TypeOf(values).String() == "map[string]interface {}" {
		restrictionError := applyEntityRestrictions(values.(map[string]interface{}))
		if restrictionError != nil {
			errObj := NewError(Utility.FileLine(), PROTOTYPE_RESTRICTIONS_ERROR, SERVER_ERROR_CODE, restrictionError)
			return nil, errObj
		}

		values_, err := Utility.InitializeStructure(values.(map[string]interface{}))
		if err == nil {
			values = values_.Interface()
		}
	}

	params := make([]interface{}, 3)
	params[0] = parentUuid
	params[1] = objectId
	params[2] = values

	methodName := "New" + strings.Replace(typeName, ".", "", -1) + "Entity"
	entity, invalidMethod := Utility.CallMethod(this, methodName, params)

	if invalidMethod != nil {
		// Try to create a dynamic entity...
		if reflect.TypeOf(values).String() == "map[string]interface {}" {
			values.(map[string]interface{})["ParentUuid"] = parentUuid
			var errObj *CargoEntities.Error
			entity, errObj = this.newDynamicEntity(parentUuid, values.(map[string]interface{}))
			if errObj != nil {
				return nil, errObj
			}
		} else {
			cargoError := NewError(Utility.FileLine(), TYPENAME_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("The typeName '"+typeName+"' is does not exist."))
			return nil, cargoError
		}
	}

	// Set the relation name with it parent.
	entity.(Entity).SetParentLnk(attributeName)
	if this.isExist(entity.(Entity).GetUuid()) {
		// Here because I use the New methode the value is in the cache so
		// I need to remove it...
		this.removeEntity(entity.(Entity).GetUuid())
	}

	// Save the entity...
	entity.(Entity).SetNeedSave(true)

	// Now I will save it parent if there one.
	if parentPtr != nil {
		// Set it parent.
		entity.(Entity).SaveEntity()
		parentPtrTypeName := parentPtr.GetTypeName()
		parentPtrStoreId := parentPtrTypeName[:strings.Index(parentPtrTypeName, ".")]
		parentPrtPrototype, _ := this.getEntityPrototype(parentPtrTypeName, parentPtrStoreId)
		if parentPrtPrototype.getFieldIndex(attributeName) < 0 {
			cargoError := NewError(Utility.FileLine(), ATTRIBUTE_NAME_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("The attribute name '"+attributeName+"' does not exist in the parent entity with uuid '"+parentUuid+"'."))
			return nil, cargoError
		} else {
			// Append the child into it parent and save it.
			parentPtr.AppendChild(attributeName, entity.(Entity))
			// Set need save at true.
			parentPtr.SetNeedSave(true)
			parentPtr.SaveEntity()
		}
	} else {
		entity.(Entity).SaveEntity()
	}

	// Return the object.
	return entity.(Entity), nil
}

/**
 * Those function will be use to set entity ownership.
 */
func (this *EntityManager) setEntityOwner(owner *CargoEntities.Account, entity Entity) {
	// First of all I will retreive the kv store.
	uuid := entity.GetUuid()
	storeId := uuid[0:strings.Index(uuid, ".")]
	store := GetServer().GetDataManager().getDataStore(storeId)
	if reflect.TypeOf(store).String() == "*Server.SqlDataStore" {
		store = GetServer().GetDataManager().getDataStore("sql_info")
	}

	if this.getEntityOwner(entity) != nil {
		this.removeEntityOwner(entity)
	}

	// Here I will simply set the entity ownership to the given account.
	store.(*KeyValueDataStore).setValue([]byte(uuid+"_owner"), []byte(owner.GetUUID()))
}

/**
 * Retreive the entity owner for a given entity
 */
func (this *EntityManager) getEntityOwner(entity Entity) *CargoEntities.Account {

	// First of all I will retreive the kv store.
	uuid := entity.GetUuid()
	storeId := uuid[0:strings.Index(uuid, ".")]
	store := GetServer().GetDataManager().getDataStore(storeId)
	if reflect.TypeOf(store).String() == "*Server.SqlDataStore" {
		store = GetServer().GetDataManager().getDataStore("sql_info")
	}

	// Retreive the owner
	val, err := store.(*KeyValueDataStore).getValue(uuid + "_owner")
	if err != nil {
		return nil
	}

	// In that case the owner was retreive.
	ownerEntity, errObj := this.getEntityByUuid(string(val), true)

	if errObj != nil {
		return nil
	}

	return ownerEntity.GetObject().(*CargoEntities.Account)
}

/**
 * Remove the entity owner.
 */
func (this *EntityManager) removeEntityOwner(entity Entity) {

	uuid := entity.GetUuid()
	storeId := uuid[0:strings.Index(uuid, ".")]
	store := GetServer().GetDataManager().getDataStore(storeId)
	if reflect.TypeOf(store).String() == "*Server.SqlDataStore" {
		store = GetServer().GetDataManager().getDataStore("sql_info")
	}

	// Remove the ownership...
	store.(*KeyValueDataStore).deleteValue(entity.GetUuid() + "_owner")
}

/**
 * Return true if an entity with a given uuid exist in the store.
 */
func (this *EntityManager) isExist(uuid string) bool {
	if len(uuid) == 0 {
		return false
	}

	storeId := uuid[0:strings.Index(uuid, ".")]
	store := GetServer().GetDataManager().getDataStore(storeId)
	// Here the code is not nil
	if store != nil {
		if reflect.TypeOf(store).String() == "*Server.SqlDataStore" {
			store = GetServer().GetDataManager().getDataStore("sql_info")
		}

		_, err := store.(*KeyValueDataStore).getValue(uuid)
		if err == nil {
			return true
		}
	}

	return false
}

/**
 * Sort an array of entities.
 */
func (this *EntityManager) sortEntities(entities []Entity, orderBy []interface{}, startIndex int, endIndex int, asc bool) []Entity {

	// Get the order to sort...
	if len(orderBy) > 0 {
		order := orderBy[0].(string)
		// and remove it from the list...
		orderBy = orderBy[1:]
		// First of all I will sort the array
		sort.Slice(entities[startIndex:endIndex], func(order string, asc bool) func(int, int) bool {
			return func(i, j int) bool {
				// I will get values to compare...
				entity0 := entities[i]
				entity1 := entities[j]
				if entity0 == nil || entity1 == nil {
					return false
				}

				var val0, val1 interface{}
				if reflect.TypeOf(entity0).String() == "*Server.DynamicEntity" {
					// Dynamic entity.
					val0 = entity0.(*DynamicEntity).getValue(order)
					val1 = entity1.(*DynamicEntity).getValue(order)
				} else {
					methodName := "Get" + strings.ToUpper(order[0:1]) + order[1:]
					params := make([]interface{}, 0)
					val0, _ = Utility.CallMethod(entity0, methodName, params)
					val1, _ = Utility.CallMethod(entity1, methodName, params)
				}
				if asc {
					return Utility.Less(val0, val1)
				} else {
					return Utility.Less(val1, val0)
				}
			}
		}(order, asc))
	}
	// TODO recursively sort with other fields...
	/*for i := 0; i < len(entities)-1; i++ {
		entity0 := entities[i]
		entity1 := entities[i+1]
		var val0, val1 interface{}
		if reflect.TypeOf(entity0).String() == "*Server.DynamicEntity" {
			// Dynamic entity.
			val0 = entity0.(*DynamicEntity).getValue(order)
			val1 = entity1.(*DynamicEntity).getValue(order)
		} else {
			methodName := "Get" + strings.ToUpper(order[0:1]) + order[1:]
			params := make([]interface{}, 0)
			val0, _ = Utility.CallMethod(entity0, methodName, params)
			val1, _ = Utility.CallMethod(entity1, methodName, params)
		}

		//log.Println("--------> ", entity0.GetTypeName(), " ", order, " val0 ", val0, " val1 ", val1)
	}*/

	return entities
}

/**
 * That function is use by CreateEntity and SaveEntity it validate entity fields
 * over it's defined restrictions.
 */
func applyEntityRestrictions(values map[string]interface{}) error {
	typeName := values["TYPENAME"].(string)
	prototype, err := GetServer().GetEntityManager().getEntityPrototype(typeName, typeName[0:strings.Index(typeName, ".")])

	// Test restriction for the base types.
	if prototype.getFieldIndex("M_listOf") != -1 {
		values_, err := applyRestrictions("M_listOf", prototype, values["M_listOf"], true)
		if err == nil {
			values["M_listOf"] = values_
		}

	} else if prototype.getFieldIndex("M_valueOf") != -1 {
		value_, err := applyRestrictions("M_valueOf", prototype, values["M_valueOf"], false)
		if err == nil {
			values["M_valueOf"] = value_
		}
	}

	if err == nil {
		for i := 0; i < len(prototype.FieldsType); i++ {
			// Before I set the value I will test if there's a restriction on that field.
			fieldType := prototype.FieldsType[i]
			isArray := strings.HasPrefix(fieldType, "[]")
			fieldType = strings.Replace(fieldType, "[]", "", -1)
			fieldType = strings.Replace(fieldType, ":Ref", "", -1)
			index := strings.Index(fieldType, ".")
			if index != -1 {
				// Here I will
				fieldPrototype, err := GetServer().GetEntityManager().getEntityPrototype(fieldType, fieldType[0:index])
				if err == nil {
					// Here I will get the entity value.
					value := values[prototype.Fields[i]]

					// Apply the restriction over the value.
					value, err = applyRestrictions(typeName+"_"+prototype.Fields[i], fieldPrototype, value, isArray)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return err
}

/**
 * That function is use to apply restriction to the input value.
 */
func applyRestrictions(field string, prototype *EntityPrototype, value interface{}, isArray bool) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if !isArray {
		// Here I will apply restriction if there is some.
		for i := 0; i < len(prototype.Restrictions); i++ {
			restriction := prototype.Restrictions[i]
			typeName := prototype.TypeName
			if !strings.HasPrefix(typeName, "xs.") {
				for j := 0; j < len(prototype.SuperTypeNames); i++ {
					if strings.HasPrefix(prototype.SuperTypeNames[j], "xs.") {
						typeName = prototype.SuperTypeNames[j]
						break
					}
				}
			}
			// In case of a string...
			if XML_Schemas.IsXsString(typeName) {
				var str string
				if reflect.TypeOf(value).Kind() == reflect.String {
					str = value.(string)
				} else if reflect.TypeOf(value).String() == "map[string]interface {}" {
					if value.(map[string]interface{})["M_valueOf"] != nil {
						str = value.(map[string]interface{})["M_valueOf"].(string)
					}
				}
				if restriction.Type == RestrictionType_MaxLength {
					length, err := strconv.Atoi(restriction.Value)
					if err != nil {
						return nil, err
					}
					if len(str) > length {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Maximum expected length is ` + restriction.Value + ` not ` + strconv.Itoa(len(str)) + `"}`)
					}
				} else if restriction.Type == RestrictionType_MinLength {
					length, err := strconv.Atoi(restriction.Value)
					if err != nil {
						return nil, err
					}
					if len(str) < length {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Minimum expected length is ` + restriction.Value + ` not ` + strconv.Itoa(len(str)) + `"}`)
					}
				} else if restriction.Type == RestrictionType_WhiteSpace {
					// Here tree value can be there...
					if restriction.Value == "preserve" {
						// No normalization is done, the value is the ·normalized value·
					} else if restriction.Value == "replace" {
						// All occurrences of #x9 (tab), #xA (line feed) and #xD (carriage return) are replaced > with #x20 (space).
						str = strings.Replace(str, "\t", " ", -1)
						str = strings.Replace(str, "\r\n", " ", -1)
						str = strings.Replace(str, "\n", " ", -1)
					} else if restriction.Value == "collapse" {
						// Subsequent to the replacements specified above under replace, contiguous sequences of #x20s are collapsed to a single #x20, and initial and/or final #x20s are deleted.
						str = strings.TrimSpace(str)
						re := regexp.MustCompile("  +")
						str = string(re.ReplaceAll(bytes.TrimSpace([]byte(str)), []byte(" ")))
					}

				} else if restriction.Type == RestrictionType_Enumeration {
					/** Nothing to do here because enumeration is the index and not the value **/
				} else if restriction.Type == RestrictionType_Pattern {
					/** Here I will apply the regex **/
					match, _ := regexp.MatchString(restriction.Value, str)
					if !match {
						var escapedRestrictionValue = restriction.Value
						escapedRestrictionValue = strings.Replace(escapedRestrictionValue, "\\", "\\\\", -1) // Escape \ symbole
						escapedRestrictionValue = strings.Replace(escapedRestrictionValue, "\"", "\\\"", -1) // escape " symbole
						return nil, errors.New(`{"field":"` + field + `", "msg":"The given value dosen't match the regular expression ` + escapedRestrictionValue + `"}`)
					}
				}

				if reflect.TypeOf(value).Kind() == reflect.String {
					value = str // set it new value.
				} else if reflect.TypeOf(value).String() == "map[string]interface {}" {
					if value.(map[string]interface{})["M_valueOf"] != nil {
						value.(map[string]interface{})["M_valueOf"] = str
					}
				}
			} else if XML_Schemas.IsXsNumeric(typeName) || XML_Schemas.IsXsMoney(typeName) {
				numericVal := Utility.ToNumeric(value)
				restrictionVal := Utility.ToNumeric(restriction.Value)
				if restriction.Type == RestrictionType_MaxExclusive {
					if numericVal >= restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Maximum exclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_MaxInclusive {
					if numericVal > restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Maximum inclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_MinExclusive {
					if numericVal <= restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Minimum exclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_MinInclusive {
					if numericVal < restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Minimum inclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_FractionDigits {
					numVal := Utility.Round(Utility.ToNumeric(value), Utility.ToInt(restriction.Value))
					if reflect.TypeOf(value).Kind() == reflect.Float64 {
						value = numVal // set it new value.
					} else if reflect.TypeOf(value).Kind() == reflect.Float32 {
						value = float32(numVal) // set it new value.
					} else if reflect.TypeOf(value).String() == "map[string]interface {}" {
						if value.(map[string]interface{})["M_valueOf"] != nil {
							value.(map[string]interface{})["M_valueOf"] = numVal
						}
					}
				} else if restriction.Type == RestrictionType_TotalDigits {
					str := Utility.ToString(value)
					numVal := Utility.ToNumeric(str[0:Utility.ToInt(restriction.Value)])
					if reflect.TypeOf(value).Kind() == reflect.Float64 {
						value = numVal // set it new value.
					} else if reflect.TypeOf(value).Kind() == reflect.Float32 {
						value = float32(numVal) // set it new value.
					} else if reflect.TypeOf(value).String() == "map[string]interface {}" {
						if value.(map[string]interface{})["M_valueOf"] != nil {
							value.(map[string]interface{})["M_valueOf"] = numVal
						}
					}
				}
			} else if XML_Schemas.IsXsInt(typeName) {
				// Here We got a numeric value...
				intVal := Utility.ToInt(value)
				restrictionVal := Utility.ToInt(restriction.Value)

				if restriction.Type == RestrictionType_MaxExclusive {
					if intVal >= restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Maximum exclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_MaxInclusive {
					if intVal > restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Maximum inclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_MinExclusive {
					if intVal <= restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Minimum exclusive value is ` + restriction.Value + `"}`)
					}
				} else if restriction.Type == RestrictionType_MinInclusive {
					if intVal < restrictionVal {
						return nil, errors.New(`{"field":"` + field + `", "msg":"Minimum inclusive value is ` + restriction.Value + `"}`)
					}
				}
			}
		}
	} else {
		// Here I will test array restriction.
		for i := 0; i < len(prototype.Restrictions); i++ {
			restriction := prototype.Restrictions[i]
			if restriction.Type == RestrictionType_Length || restriction.Type == RestrictionType_MinLength || restriction.Type == RestrictionType_MaxLength {
				l := len(value.([]interface{}))
				length, err := strconv.Atoi(restriction.Value)
				if err == nil {
					if restriction.Type == RestrictionType_Length {
						value_ := make([]interface{}, length)
						for j := 0; j < length && j < len(value.([]interface{})); j++ {
							value_[j] = value.([]interface{})[j]
						}
						return value_, nil // Trunk the array ass needed.
					} else if restriction.Type == RestrictionType_MinLength {
						var value_ []interface{}
						if l < length {
							value_ = make([]interface{}, length)
						} else {
							value_ = make([]interface{}, l)
						}

						for j := 0; j < len(value.([]interface{})); j++ {
							value_[j] = value.([]interface{})[j]
						}
						return value_, nil // Trunk the array ass needed.
					} else if restriction.Type == RestrictionType_MaxLength {
						var value_ []interface{}
						if l > length {
							value_ = make([]interface{}, length)
						} else {
							value_ = make([]interface{}, l)
						}

						for j := 0; j < length && j < len(value.([]interface{})); j++ {
							value_[j] = value.([]interface{})[j]
						}
						return value_, nil // Trunk the array ass needed.
					}

				} else {
					return nil, err
				}
			}

		}
	}

	return value, nil
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
	prototype.(*EntityPrototype).Create(storeId)

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
	prototype.(*EntityPrototype).Save(storeId)

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

	err = this.deleteEntityPrototype(storeId, prototype)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), DATASTORE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, err)
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
	store.(*KeyValueDataStore).DeleteEntityPrototype(oldName)
	store.(*KeyValueDataStore).saveEntityPrototype(prototype.(*EntityPrototype))

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

	return prototype.(*EntityPrototype)

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

	var schemaId string
	if strings.Index(storeId, ".") > 0 {
		schemaId = storeId[strings.Index(storeId, ".")+1:]
		storeId = storeId[0:strings.Index(storeId, ".")]
	}
	protos, err := this.getEntityPrototypes(storeId, schemaId)
	if err != nil {
		cargoError := NewError(Utility.FileLine(), PROTOTYPE_DOESNT_EXIST_ERROR, SERVER_ERROR_CODE, errors.New("There is no prototypes in store '"+storeId+"'."))
		GetServer().reportErrorMessage(messageId, sessionId, cargoError)
		return nil
	}

	return protos
}

// @api 1.0
// That function will retreive the list of derived entity prototype from a given type.
// @param {string} typeName The type name of the parent entity.
// @param {string} messageId The request id that need to access this method.
// @param {string} sessionId The user session.
// @result{[]*EntityPrototype} Return the retreived list of entity prototype
// @scope {public}
// @param {callback} successCallback The function is call in case of success and the result parameter contain objects we looking for.
// @param {callback} errorCallback In case of error.
// @src
//EntityManager.prototype.getDerivedEntityPrototypes = function (typeName, successCallback, errorCallback, caller) {
//    // server is the client side singleton.
//    var params = []
//    params.push(createRpcData(typeName, "STRING", "typeName"))
//    // Call it on the server.
//    server.executeJsFunction(
//        "EntityManagerGetDerivedEntityPrototypes", // The function to execute remotely on server
//        params, // The parameters to pass to that function
//        function (index, total, caller) { // The progress callback
//            // Nothing special to do here.
//        },
//        function (results, caller) {
//            var prototypes = []
//            if (results[0] != null) {
//                for (var i = 0; i < results[0].length; i++) {
//                    var result = results[0][i]
//                    if (entityPrototypes[results[0][i].TypeName] != undefined) {
//                        prototypes.push(entityPrototypes[results[0][i].TypeName])
//                    } else {
//                        var proto = new EntityPrototype()
//                        proto.init(results[0][i])
//                        entityPrototypes[results[0][i].TypeName] = proto
//                    }
//                }
//            }
//            // return the list of prototype object.
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
func (this *EntityManager) GetDerivedEntityPrototypes(typeName string, messageId string, sessionId string) []*EntityPrototype {
	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	protos, errObj := this.getDerivedEntityType(typeName)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
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

	result, errObj := this.createEntity(parentUuid, attributeName, typeName, objectId, values)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	// Here I will set the ownership of the entity.
	session := GetServer().GetSessionManager().getActiveSessionById(sessionId)
	if session != nil {
		this.setEntityOwner(session.GetAccountPtr(), result)
	}

	return result.GetObject()
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

	// Need to be call before any new entity function to test the new value
	// with the actual one.
	funcName := "New" + strings.Replace(typeName, ".", "", -1) + "EntityFromObject"

	// Try to cast the value as needed...
	if reflect.TypeOf(values).String() == "map[string]interface {}" {
		values_, err := Utility.InitializeStructure(values.(map[string]interface{}))
		if err == nil {
			values = values_.Interface()
		}
	}

	params := make([]interface{}, 1)
	params[0] = values

	entity, err := Utility.CallMethod(this, funcName, params)

	if err != nil {
		// I will try with dynamic entity insted...
		var errObj *CargoEntities.Error
		entity, errObj = this.newDynamicEntity("", values.(map[string]interface{}))
		if errObj != nil {
			GetServer().reportErrorMessage(messageId, sessionId, errObj)
			return nil
		}
	}

	// Now I will save the entity.
	entity.(Entity).SaveEntity()

	return entity.(Entity).GetObject()
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

	// The entity to remove.
	var entity Entity

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
//                    if (result[0] == null) {
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
func (this *EntityManager) GetEntities(typeName string, storeId string, query *EntityQuery, offset int, limit int, orderBy []interface{}, asc bool, messageId string, sessionId string) []interface{} {

	errObj := GetServer().GetSecurityManager().canExecuteAction(sessionId, Utility.FunctionName())
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	entities, errObj := this.getEntities(typeName, query, storeId, false)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}

	//log.Println("-----> Number of entities found ", len(entities))

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
	this.sortEntities(entities, orderBy, 0, len(entities), asc)

	var objects []interface{}

	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return objects
	}

	if limit <= 0 {
		// all results are require.
		limit = len(entities)
	}

	for i := offset; i < limit && i < len(entities); i++ {
		// If the entity was deleted before the time i sent it back to the

		if entities[i] != nil {
			// Init the entity
			entities[i].InitEntity(entities[i].GetUuid(), false)
			objects = append(objects, entities[i].GetObject())
		}
	}

	return objects
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

	entity, errObj := this.getEntityByUuid(uuid, false)
	if errObj != nil {
		entity, errObj = this.getDynamicEntityByUuid(uuid, false)
		if errObj != nil {
			GetServer().reportErrorMessage(messageId, sessionId, errObj)
			return nil
		}
	}

	return entity.GetObject()
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
	entity, errObj := this.getEntityById(storeId, typeName, ids, false)
	if errObj != nil {
		GetServer().reportErrorMessage(messageId, sessionId, errObj)
		return nil
	}
	return entity.GetObject()
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
	return Utility.GenerateUUID(keyInfo)
}

////////////////////////////////////////////////////////////////////////////////
// XSD/XML Schemas...
////////////////////////////////////////////////////////////////////////////////

func (this *EntityManager) generateXsdSchema(schemaId string, filePath string) (*EntityPrototype, error) {

	/** First of all I will generate the javascript file **/
	// Execute the command...

	// First of all I will

	return nil, nil
}
