package Server

import (
	"log"

	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"os"
	"reflect"
	//"strconv"
	"strings"
	"time"

	"code.myceliUs.com/CargoWebServer/Cargo/BPMS/BPMN20"
	"code.myceliUs.com/CargoWebServer/Cargo/BPMS/BPMS_Runtime"
	"code.myceliUs.com/CargoWebServer/Cargo/JS"
	"code.myceliUs.com/CargoWebServer/Cargo/Persistence/CargoEntities"
	"code.myceliUs.com/CargoWebServer/Cargo/Utility"
)

const (
	// make the processor more responsive...
	resolution = 5
)

/**
 * Tag used to identified a process instance during it lifetime.
 * The tag will be reatributed latter...
 */
type ColorTag struct {
	Name       string // Acid Green
	Number     string // Hexadecimal #B0BF1A
	InstanceId string // if is used by an instance.
}

type WorkflowProcessor struct {
	// The workflow manage stop when that variable
	// is set to true.
	abortedByEnvironment chan bool

	// The trigger channel.
	triggerChan chan *BPMS_Runtime.Trigger

	// The runtime...
	runtime *BPMS_Runtime_RuntimesEntity

	// Process instance have a color tag to be
	// easier to regonized by the end user. The color must be unique
	// for the lifetime of the instance, but the color can be reuse latter by
	// other instance.
	colorTagMap map[string]*ColorTag

	// Contain the list of available tag number...
	availableTagNumber []string
}

func newWorkflowProcessor() *WorkflowProcessor {

	// The workflow manger...
	workflowProcessor := new(WorkflowProcessor)
	workflowProcessor.abortedByEnvironment = make(chan bool)

	// The trigger chanel...
	workflowProcessor.triggerChan = make(chan *BPMS_Runtime.Trigger)

	// The runtime data...
	runtimeUUID := BPMS_RuntimeRuntimesExists("runtime")
	if len(runtimeUUID) == 0 {
		// Create a new runtime...
		workflowProcessor.runtime = GetServer().GetEntityManager().NewBPMS_RuntimeRuntimesEntity("runtime", nil)
		workflowProcessor.runtime.SaveEntity()

	} else {
		// Get the current runtime...
		workflowProcessor.runtime = GetServer().GetEntityManager().NewBPMS_RuntimeRuntimesEntity(runtimeUUID, nil)
		// Initialyse the runtime...
		workflowProcessor.runtime.InitEntity(runtimeUUID)
	}

	return workflowProcessor
}

func (this *WorkflowProcessor) Initialize() {
	// Here I will read the color tag...
	this.colorTagMap = make(map[string]*ColorTag, 0)
	colorNamePath := GetServer().GetConfigurationManager().GetDataPath() + "/colorName.csv"
	colorNameFile, _ := os.Open(colorNamePath)
	csvReader := csv.NewReader(bufio.NewReader(colorNameFile))

	for {
		record, err := csvReader.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}
		tag := new(ColorTag)
		for i := 0; i < len(record); i++ {
			if i == 0 {
				tag.Name = record[i]
			} else if i == 1 {
				tag.Number = record[i]
				this.availableTagNumber = append(this.availableTagNumber, tag.Number)
			}
		}
		this.colorTagMap[tag.Number] = tag
	}

	// That function will evaluate the processes and put it in the good
	// state to start...
	this.runTopLevelProcesses()
	this.getNextProcessInstanceNumber()
}

// Start processing the workflow...
func (this *WorkflowProcessor) run() {

	for {
		select {
		case trigger := <-this.triggerChan:
			this.processTrigger(trigger) // Process the triggers...
		case done := <-this.abortedByEnvironment:
			if done {
				log.Println("Stop processing workflow")
				return
			}
		default:
			// Run the to level process each 300 millisecond.
			this.runTopLevelProcesses()
		}
		// Does nothing for the next 300 millisecond...
		time.Sleep(300 * time.Millisecond)

	}
}

/**
 * Process new trigger... The trigger contain information, data, that the
 * workflow processor use to evaluate the next action evalute...
 */
func (this *WorkflowProcessor) processTrigger(trigger *BPMS_Runtime.Trigger) *CargoEntities.Error {
	// Now i will process the trigger...

	// New process trigger was created...
	if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Start {

		// Get the bpmn process...
		processEntity, err := GetServer().GetEntityManager().getEntityByUuid(trigger.M_processUUID)
		if err != nil {
			return err
		}

		// Get it definitions.
		definitionsEntity := processEntity.GetParentPtr()
		definitionsInstance := this.getDefinitionInstance(definitionsEntity.GetObject().(*BPMN20.Definitions).GetId(), definitionsEntity.GetObject().(*BPMN20.Definitions).GetUUID())

		/** So Here I will create a new process instance **/
		processInstance := new(BPMS_Runtime.ProcessInstance)
		processInstance.UUID = "BPMS_Runtime.ProcessInstance%" + Utility.RandomUUID()
		processInstance.M_number = this.getNextProcessInstanceNumber()

		index := rand.Intn(len(this.availableTagNumber))
		colorTag := this.colorTagMap[this.availableTagNumber[index]]
		processInstance.M_colorNumber = colorTag.Number
		processInstance.M_colorName = colorTag.Name

		// Here I will associate the bmpn process instance id.
		processInstance.M_bpmnElementId = trigger.M_processUUID

		// Now the start event...
		var startEvent *BPMN20.StartEvent
		for i := 0; i < len(processEntity.GetChildsPtr()); i++ {
			if processEntity.GetChildsPtr()[i].GetTypeName() == "BPMN20.StartEvent" {
				startEvent = processEntity.GetChildsPtr()[i].GetObject().(*BPMN20.StartEvent)
				break
			}
		}

		// Evaluation of the source expression if there is so...
		// Get a reference to the process.
		process := processEntity.GetObject().(*BPMN20.Process)
		processInstance.SetId(process.M_id)

		// Set the process instances...
		definitionsInstance.SetProcessInstances(processInstance)

		// Now I will create the start event...
		startEventInstance := this.createInstance(startEvent, processInstance, nil, trigger.GetDataRef(), trigger.M_sessionId)

		// Set the new definition instance...
		this.runtime.object.SetDefinitions(definitionsInstance)

		// Save the new entity...
		this.runtime.SaveEntity()

		// we are done with the start event.
		startEventInstance.(BPMS_Runtime.FlowNodeInstance).SetLifecycleState(BPMS_Runtime.LifecycleState_Completing)

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_None {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Timer {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Conditional {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Message {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Signal {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Multiple {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_ParallelMultiple {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Escalation {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Error {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Compensation {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Terminate {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Cancel {

	} else if trigger.GetEventTriggerType() == BPMS_Runtime.EventTriggerType_Link {

	}

	// No error here...
	return nil
}

/**
 * Evaluate processes instance for each bpmn processes.
 */
func (this *WorkflowProcessor) runTopLevelProcesses() {
	// Get all processes...
	processes := GetServer().GetWorkflowManager().getProcess()
	for i := 0; i < len(processes); i++ {
		this.workflowTransitionInterpreter(processes[i])
	}
}

/**
 * Evaluate the transition.
 */
func (this *WorkflowProcessor) workflowTransitionInterpreter(process *BPMN20.Process) {
	activeProcessInstances := this.getActiveProcessInstances(process)
	for i := 0; i < len(activeProcessInstances); i++ {
		log.Println("Evalute transition for process ", activeProcessInstances[i].M_id)
		for j := 0; j < len(activeProcessInstances[i].GetFlowNodeInstances()); j++ {
			this.workflowTransition(activeProcessInstances[i].GetFlowNodeInstances()[j], activeProcessInstances[i])
		}
	}
}

/**
 * Evaluate the transition...
 */
func (this *WorkflowProcessor) workflowTransition(flowNode BPMS_Runtime.FlowNodeInstance, processInstance *BPMS_Runtime.ProcessInstance) {

	// I will retreive the entity related to the instance...
	bpmnElementEntity, err := GetServer().GetEntityManager().getEntityByUuid(flowNode.(BPMS_Runtime.Instance).GetBpmnElementId())

	if err == nil {

		if flowNode.GetLifecycleState() == BPMS_Runtime.LifecycleState_Active {
			// Here I will process the active instances...
			if reflect.TypeOf(flowNode).String() == "*BPMS_Runtime.ActivityInstance" {
				err := this.executeActivityInstance(flowNode.(*BPMS_Runtime.ActivityInstance), "")
				if err == nil {
					flowNode.SetLifecycleState(BPMS_Runtime.LifecycleState_Completing)
				}
			} else if reflect.TypeOf(flowNode).String() == "*BPMS_Runtime.EventInstance" {
				if reflect.TypeOf(bpmnElementEntity.GetObject()).String() == "*BPMN20.EndEvent" {
					// here I just set the life cycle to complete...
					flowNode.SetLifecycleState(BPMS_Runtime.LifecycleState_Completing)
				}

			} else if reflect.TypeOf(flowNode).String() == "*BPMS_Runtime.GatewayInstance" {

			} else if reflect.TypeOf(flowNode).String() == "*BPMS_Runtime.SubprocessInstance" {

			}

		} else if flowNode.GetLifecycleState() == BPMS_Runtime.LifecycleState_Completing {
			bpmnElement := bpmnElementEntity.GetObject().(BPMN20.FlowNode)

			// Here i will process the completed instance...
			for i := 0; i < len(bpmnElement.GetOutgoing()); i++ {

				// TODO in case of a gateway evaluate the sequence flow condition...
				seqFlow := bpmnElement.GetOutgoing()[i]
				connectingObj := new(BPMS_Runtime.ConnectingObject)
				connectingObj.UUID = "BPMS_Runtime.ConnectingObject%" + Utility.RandomUUID()
				connectingObj.M_bpmnElementId = seqFlow.GetId()
				connectingObj.SetSourceRef(flowNode)
				connectingObj.SetId(bpmnElement.GetOutgoing()[i].GetId())

				// Now I will create the next node

				// The item reference must be set...
				items := make([]*BPMS_Runtime.ItemAwareElementInstance, 0)

				// Create the new instance.
				nextStep := this.createInstance(seqFlow.GetTargetRef(), processInstance, connectingObj, items, "")

				connectingObj.SetTargetRef(nextStep)

				flowNode.SetOutputRef(connectingObj)

				// Set the connecting object and save it...
				processInstance.SetConnectingObjects(connectingObj)

				processInstanceEntity := GetServer().GetEntityManager().NewBPMS_RuntimeProcessInstanceEntityFromObject(processInstance)

				// Save the change...
				processInstanceEntity.SaveEntity()

			}

			// Set completed...
			this.completeInstance(flowNode)
		}
	}
}

/**
 * Do the work inside the activity instance if there some work todo...
 */
func (this *WorkflowProcessor) executeActivityInstance(instance BPMS_Runtime.Instance, sessionId string) *CargoEntities.Error {

	// I will retreive the activity entity related to the the instance.
	activityEntity, err := GetServer().GetEntityManager().getEntityByUuid(instance.GetBpmnElementId())
	if err != nil {
		return err
	}

	// Get the activity object.
	activity := activityEntity.GetObject()

	switch v := activity.(type) {
	case *BPMN20.AdHocSubProcess:

	case *BPMN20.BusinessRuleTask:

	case *BPMN20.CallActivity:

	case *BPMN20.SubProcess_impl:

	case *BPMN20.ManualTask:

	case *BPMN20.ScriptTask:
		_, err := this.runScript(activity.(*BPMN20.ScriptTask).GetScript().GetScript(), instance, sessionId)

		// Set state to completing...
		if err != nil {
			instance.(*BPMS_Runtime.ActivityInstance).M_lifecycleState = BPMS_Runtime.LifecycleState_Failed
			//instance.SetLogInfoRef()
		} else {
			instance.(*BPMS_Runtime.ActivityInstance).M_lifecycleState = BPMS_Runtime.LifecycleState_Completed
		}

		return nil
	case *BPMN20.UserTask:

	case *BPMN20.Task_impl:

	default:
		log.Println("-------> Unknow activity type ", v)
	}

	return NewError(Utility.FileLine(), ACTION_EXECUTE_ERROR, SERVER_ERROR_CODE, errors.New("No action was found!"))
}

/**
 * That function is use to evaluate a given script.
 */
func (this *WorkflowProcessor) runScript(src string, instance BPMS_Runtime.Instance, sessionId string) (result interface{}, err error) {

	/** Here I will execute the script task **/
	functionStr := src[9 : len(src)-3]

	// Set the instance on the context...
	vm := JS.GetJsRuntimeManager().GetVm(sessionId).Copy()
	vm.Set("instance", instance)

	// Here i wil find the name of the function...
	startIndex := strings.Index(functionStr, " ")
	endIndex := strings.Index(functionStr, "(")

	functionName := strings.Trim(functionStr[startIndex:endIndex], " ")

	result, err = vm.Run(functionStr)
	if err != nil {
		log.Println("Error in code of ", functionName)
		return
	}

	result, err = vm.Run(functionName + "();")
	if err != nil {
		log.Println("Error in code of ", functionName)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////////////
// Various constructor function.
////////////////////////////////////////////////////////////////////////////////
/**
 * Create the instance from
 */
func (this *WorkflowProcessor) createInstance(flowNode BPMN20.FlowNode, processInstance *BPMS_Runtime.ProcessInstance, input *BPMS_Runtime.ConnectingObject, items []*BPMS_Runtime.ItemAwareElementInstance, sessionId string) BPMS_Runtime.Instance {
	var instance BPMS_Runtime.Instance

	switch v := flowNode.(type) {
	case BPMN20.Activity:
		log.Println("-------> create activity ", v)
		instance = this.createActivityInstance(v, processInstance, items, sessionId)
		instance.(*BPMS_Runtime.ActivityInstance).SetInputRef(input)
		log.Println("-------> after create activity process data:", processInstance.GetData())
	case BPMN20.Gateway:
		log.Println("-------> create gateway ", v)
		instance.(*BPMS_Runtime.GatewayInstance).SetInputRef(input)

	case BPMN20.Event:
		log.Println("-------> create event ", v)
		instance = this.createEventInstance(v, processInstance, items, sessionId)
		if input != nil {
			instance.(*BPMS_Runtime.EventInstance).SetInputRef(input)
		}
		log.Println("-------> after start event creation process data:", processInstance.GetData())
	case *BPMN20.Transaction:
		// Nothing todo here... for now...

	default:
		log.Println("-------> not define ", v)
	}

	// Set the log information here...
	if instance != nil {
		this.setLogInfo(instance.(BPMS_Runtime.FlowNodeInstance), "New instance created", sessionId)
	}

	// Get the entity for the process instance.
	processInstance.NeedSave = true
	processEntity := GetServer().GetEntityManager().NewBPMS_RuntimeProcessInstanceEntityFromObject(processInstance)
	processEntity.SaveEntity()

	return instance
}

/**
 * Create Activity instance
 */
func (this *WorkflowProcessor) createActivityInstance(activity BPMN20.Activity, processInstance *BPMS_Runtime.ProcessInstance, items []*BPMS_Runtime.ItemAwareElementInstance, sessionId string) *BPMS_Runtime.ActivityInstance {

	instanceEntity := GetServer().GetEntityManager().NewBPMS_RuntimeActivityInstanceEntity("BPMS_Runtime.ActivityInstance%"+Utility.RandomUUID(), nil)
	instance := instanceEntity.GetObject().(*BPMS_Runtime.ActivityInstance)
	instance.M_bpmnElementId = activity.GetUUID()
	instance.M_id = activity.(BPMN20.BaseElement).GetId()

	instance.M_bpmnElementId = activity.GetUUID()

	// Set state to ready...
	instance.M_lifecycleState = BPMS_Runtime.LifecycleState_Ready

	// I will copy the data reference into the start event...
	for i := 0; i < len(items); i++ {
		instance.SetDataRef(items[i])
	}

	// Now I will set the data association...
	var dataInput []BPMN20.BaseElement
	if activity.GetIoSpecification() != nil {
		for i := 0; i < len(activity.GetIoSpecification().GetDataInput()); i++ {
			dataInput = append(dataInput, activity.GetIoSpecification().GetDataInput()[i])
		}
	}
	var dataInputAssociations []BPMN20.DataAssociation
	for i := 0; i < len(activity.GetDataInputAssociation()); i++ {
		dataInputAssociations = append(dataInputAssociations, activity.GetDataInputAssociation()[i])
	}

	this.setDataAssocication(processInstance, instance, dataInputAssociations, dataInput)

	switch v := activity.(type) {
	case *BPMN20.AdHocSubProcess:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_AdHocSubprocess
		instance.M_activityType = BPMS_Runtime.ActivityType_AdHocSubprocess
	case *BPMN20.BusinessRuleTask:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_BusinessRuleTask
		instance.M_activityType = BPMS_Runtime.ActivityType_BusinessRuleTask
	case *BPMN20.CallActivity:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_CallActivity
		instance.M_activityType = BPMS_Runtime.ActivityType_CallActivity
	case *BPMN20.SubProcess_impl:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_EmbeddedSubprocess
		instance.M_activityType = BPMS_Runtime.ActivityType_EmbeddedSubprocess
	case *BPMN20.ManualTask:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_ManualTask
		instance.M_activityType = BPMS_Runtime.ActivityType_ManualTask
	case *BPMN20.ScriptTask:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_ScriptTask
		instance.M_activityType = BPMS_Runtime.ActivityType_ScriptTask
	case *BPMN20.UserTask:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_UserTask
		instance.M_activityType = BPMS_Runtime.ActivityType_UserTask
	case *BPMN20.Task_impl:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_AbstractTask
		instance.M_activityType = BPMS_Runtime.ActivityType_AbstractTask
	default:
		log.Println("-------> Unknow activity type ", v)
	}

	var dataOutput []BPMN20.BaseElement
	if activity.GetIoSpecification() != nil {
		for i := 0; i < len(activity.GetIoSpecification().GetDataOutput()); i++ {
			dataOutput = append(dataOutput, activity.GetIoSpecification().GetDataOutput()[i])
		}
	}
	var dataOutputAssociations []BPMN20.DataAssociation
	for i := 0; i < len(activity.GetDataOutputAssociation()); i++ {
		dataOutputAssociations = append(dataOutputAssociations, activity.GetDataOutputAssociation()[i])
	}

	this.setDataAssocication(instance, processInstance, dataOutputAssociations, dataOutput)

	// Set the start event for the processInstance...
	processInstance.SetFlowNodeInstances(instance)

	// I can activate the instance...
	this.activateInstance(instance)

	return instance
}

/**
 * Create Event Instance
 */
func (this *WorkflowProcessor) createEventInstance(event BPMN20.Event, processInstance *BPMS_Runtime.ProcessInstance, items []*BPMS_Runtime.ItemAwareElementInstance, sessionId string) *BPMS_Runtime.EventInstance {
	// Now I will create the start event...
	instanceEntity := GetServer().GetEntityManager().NewBPMS_RuntimeEventInstanceEntity("BPMS_Runtime.EventInstance%"+Utility.RandomUUID(), nil)
	instance := instanceEntity.GetObject().(*BPMS_Runtime.EventInstance)
	instance.M_bpmnElementId = event.GetUUID()
	instance.M_id = event.(BPMN20.BaseElement).GetId()

	// I will copy the data reference into the start event...
	for i := 0; i < len(items); i++ {
		instance.SetDataRef(items[i])
	}

	// Select the good event type...
	switch v := event.(type) {
	case *BPMN20.StartEvent:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_StartEvent
		instance.M_eventType = BPMS_Runtime.EventType_StartEvent
	case *BPMN20.EndEvent:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_EndEvent
		instance.M_eventType = BPMS_Runtime.EventType_EndEvent
	case *BPMN20.BoundaryEvent:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_BoundaryEvent
		instance.M_eventType = BPMS_Runtime.EventType_BoundaryEvent
	case *BPMN20.IntermediateCatchEvent:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_IntermediateCatchEvent
		instance.M_eventType = BPMS_Runtime.EventType_IntermediateCatchEvent
	case *BPMN20.IntermediateThrowEvent:
		instance.M_flowNodeType = BPMS_Runtime.FlowNodeType_IntermediateThrowEvent
		instance.M_eventType = BPMS_Runtime.EventType_IntermediateThrowEvent
	default:
		log.Println("------> event ", v, " has not instance type...")
	}

	// Set the instance to ready state...
	instance.M_lifecycleState = BPMS_Runtime.LifecycleState_Ready

	// Set the start event for the processInstance...
	processInstance.SetFlowNodeInstances(instance)

	// Get the data objects from the flow elements
	bpmnElementEntity, err := GetServer().GetEntityManager().getEntityByUuid(processInstance.GetBpmnElementId())

	if err == nil {
		process := bpmnElementEntity.GetObject().(*BPMN20.Process)

		var dataAssociations []BPMN20.DataAssociation

		// Set output data association...
		switch v := event.(type) {
		case BPMN20.CatchEvent:
			for i := 0; i < len(v.GetDataOutputAssociation()); i++ {
				dataAssociations = append(dataAssociations, v.GetDataOutputAssociation()[i])
			}
			// Get the data...
			if len(v.GetDataOutputAssociation()) > 0 {
				data := this.getProcessDataInput(process)
				this.setDataAssocication(instance, processInstance, dataAssociations, data)
			}
		case BPMN20.ThrowEvent:
			for i := 0; i < len(v.GetDataInputAssociation()); i++ {
				dataAssociations = append(dataAssociations, v.GetDataInputAssociation()[i])
			}
			if len(v.GetDataInputAssociation()) > 0 {
				data := this.getProcessDataOutput(process)
				this.setDataAssocication(processInstance, instance, dataAssociations, data)
			}

		}
	}

	// I can activate the instance...
	this.activateInstance(instance)

	return instance
}

/**
 * Create a new Item aware element instance for a given bpmnElementId...
 * The bpmn element must be a ItemAwareElement.
 */
func (this *WorkflowProcessor) createItemAwareElementInstance(bpmnElementId string, data interface{}) (*BPMS_Runtime.ItemAwareElementInstance, *CargoEntities.Error) {

	bpmnElementEntity, err := GetServer().GetEntityManager().getEntityByUuid(bpmnElementId)
	if err != nil {
		return nil, err
	}

	// Here I will simply serialyse the data and save it...
	buffer := new(bytes.Buffer)
	enc := json.NewEncoder(buffer)
	err_ := enc.Encode(data)
	if err != nil {
		return nil, NewError(Utility.FileLine(), JSON_MARSHALING_ERROR, SERVER_ERROR_CODE, err_)
	}

	// Here I will set the instance id...
	instanceEntity := GetServer().GetEntityManager().NewBPMS_RuntimeItemAwareElementInstanceEntity("", nil)

	instance := instanceEntity.GetObject().(*BPMS_Runtime.ItemAwareElementInstance)
	instance.M_bpmnElementId = bpmnElementId
	instance.M_id = bpmnElementEntity.GetObject().(BPMN20.BaseElement).GetId()
	instance.M_data = buffer.Bytes()

	// Now I will save the entity
	instanceEntity.SaveEntity()

	return instance, nil
}

////////////////////////////////////////////////////////////////////////////////
// State transition function.
////////////////////////////////////////////////////////////////////////////////

// Utility function to save a given entity
func (this *WorkflowProcessor) saveInstance(instance BPMS_Runtime.Instance) {
	var entity Entity

	switch v := instance.(type) {
	case *BPMS_Runtime.ActivityInstance:
		entity = GetServer().GetEntityManager().NewBPMS_RuntimeActivityInstanceEntityFromObject(v)
	case *BPMS_Runtime.ProcessInstance:
		entity = GetServer().GetEntityManager().NewBPMS_RuntimeProcessInstanceEntityFromObject(v)
	case *BPMS_Runtime.EventInstance:
		entity = GetServer().GetEntityManager().NewBPMS_RuntimeEventInstanceEntityFromObject(v)
	case *BPMS_Runtime.GatewayInstance:
		entity = GetServer().GetEntityManager().NewBPMS_RuntimeGatewayInstanceEntityFromObject(v)
	}

	log.Println("=--------------> save instance: ", entity.GetObject())

	if entity != nil {
		entity.SaveEntity()
	}
}

/**
 * Activate a given instance.
 */
func (this *WorkflowProcessor) activateInstance(instance BPMS_Runtime.FlowNodeInstance) error {
	if instance.GetLifecycleState() != BPMS_Runtime.LifecycleState_Ready {
		return errors.New("The input flow node instance must have life cycle state at Ready")
	}

	// TODO do activation stuff here.

	instance.SetLifecycleState(BPMS_Runtime.LifecycleState_Active)
	this.saveInstance(instance.(BPMS_Runtime.Instance))
	return nil
}

/**
 * Terminate a given instance.
 */
func (this *WorkflowProcessor) completeInstance(instance BPMS_Runtime.FlowNodeInstance) error {
	if instance.GetLifecycleState() != BPMS_Runtime.LifecycleState_Completing {
		return errors.New("The input flow node instance must have life cycle state at Completing")
	}

	// TODO do completing stuff here.
	instance.SetLifecycleState(BPMS_Runtime.LifecycleState_Completed)
	this.saveInstance(instance.(BPMS_Runtime.Instance))
	return nil
}

/**
 * Terminate a given instance.
 */
func (this *WorkflowProcessor) terminateInstance(instance BPMS_Runtime.FlowNodeInstance) error {
	if instance.GetLifecycleState() != BPMS_Runtime.LifecycleState_Terminating {
		return errors.New("The input flow node instance must have life cycle state at Terminating")
	}

	// TODO do terminating stuff here.

	instance.SetLifecycleState(BPMS_Runtime.LifecycleState_Terminated)
	this.saveInstance(instance.(BPMS_Runtime.Instance))

	return nil
}

/**
 * Compensate a given instance.
 */
func (this *WorkflowProcessor) compensateInstance(instance BPMS_Runtime.FlowNodeInstance) error {
	if instance.GetLifecycleState() != BPMS_Runtime.LifecycleState_Compensating {
		return errors.New("The input flow node instance must have life cycle state at Compensating")
	}

	// TODO do compensating stuff here.

	instance.SetLifecycleState(BPMS_Runtime.LifecycleState_Compensated)
	this.saveInstance(instance.(BPMS_Runtime.Instance))

	return nil
}

/**
 * Abord a given instance.
 */
func (this *WorkflowProcessor) abordInstance(instance BPMS_Runtime.FlowNodeInstance) error {
	if instance.GetLifecycleState() != BPMS_Runtime.LifecycleState_Failing {
		return errors.New("The input flow node instance must have life cycle state at Failing")
	}

	// TODO do failling stuff here.
	instance.SetLifecycleState(BPMS_Runtime.LifecycleState_Failed)
	this.saveInstance(instance.(BPMS_Runtime.Instance))

	return nil
}

/**
 * Delete a process instance from the runtime when is no more needed...
 */
func (this *WorkflowProcessor) deleteInstance(processInstance *BPMS_Runtime.ProcessInstance) {
	processInstaceEntity := GetServer().GetEntityManager().NewBPMS_RuntimeProcessInstanceEntityFromObject(processInstance)
	if processInstaceEntity != nil {
		log.Println("---------> remove process instance: ", processInstance.GetUUID())
		// Remove the entity...
		processInstaceEntity.DeleteEntity()
	}
}

/**
 * Determine if a process is still active.
 */
func (this *WorkflowProcessor) stillActive(processInstance *BPMS_Runtime.ProcessInstance) bool {
	// Here If the process instance contain a end event instance with state
	// completed then it must be considere completed
	for i := 0; i < len(processInstance.GetFlowNodeInstances()); i++ {
		instance := processInstance.GetFlowNodeInstances()[i]
		if reflect.TypeOf(instance).String() == "*BPMS_Runtime.EventInstance" {
			evt := instance.(*BPMS_Runtime.EventInstance)
			log.Println("---> ", evt.GetBpmnElementId())
			if strings.HasPrefix(evt.GetBpmnElementId(), "BPMN20.EndEvent%") {
				if evt.GetLifecycleState() == BPMS_Runtime.LifecycleState_Completed {
					return false
				}
			}
		}
	}

	// TODO evalute here other case where the process instance must be considere
	// innactive.

	return true
}

////////////////////////////////////////////////////////////////////////////////
// Getter/setter
////////////////////////////////////////////////////////////////////////////////

/**
 * Get the definition instance with a given id.
 */
func (this *WorkflowProcessor) getDefinitionInstance(definitionsId string, bpmnDefinitionsId string) *BPMS_Runtime.DefinitionsInstance {
	var definitionsInstance *BPMS_Runtime.DefinitionsInstance
	// Now I will try to find the definitions instance for that definitions.
	definitionsInstanceEntity, err := GetServer().GetEntityManager().getEntityById("BPMS_Runtime.DefinitionsInstance", definitionsId)

	if err != nil {
		// Here i will create the definitions instance.
		definitionsInstanceEntity = GetServer().GetEntityManager().NewBPMS_RuntimeDefinitionsInstanceEntity(definitionsId, nil)
		definitionsInstance = definitionsInstanceEntity.GetObject().(*BPMS_Runtime.DefinitionsInstance)
		definitionsInstance.M_id = definitionsId
		definitionsInstance.M_bpmnElementId = bpmnDefinitionsId
	} else {
		definitionsInstance = definitionsInstanceEntity.GetObject().(*BPMS_Runtime.DefinitionsInstance)
	}

	return definitionsInstance
}

/**
 * That function return the list of active process instances for a given process
 * It also delete innactive instances en passant.
 */
func (this *WorkflowProcessor) getActiveProcessInstances(process *BPMN20.Process) []*BPMS_Runtime.ProcessInstance {
	instances, _ := GetServer().GetWorkflowManager().getInstances(process.GetUUID(), "BPMS_Runtime.ProcessInstance")
	var activeProcessInstances []*BPMS_Runtime.ProcessInstance

	// The list of processes instance...
	for i := 0; i < len(instances); i++ {
		processInstance := instances[i].(*BPMS_Runtime.ProcessInstance)
		if this.stillActive(processInstance) == false {
			this.deleteInstance(processInstance)
			this.availableTagNumber = append(this.availableTagNumber, processInstance.M_colorNumber)
		} else {
			activeProcessInstances = append(activeProcessInstances, processInstance)

			if tag, ok := this.colorTagMap[processInstance.M_colorNumber]; ok {
				// Set the color...
				tag.InstanceId = processInstance.UUID
				var availableTagNumber_ []string
				for j := 0; j < len(this.availableTagNumber); j++ {
					if this.availableTagNumber[j] != processInstance.M_colorNumber {
						availableTagNumber_ = append(availableTagNumber_, this.availableTagNumber[j])
					}
				}
				this.availableTagNumber = availableTagNumber_
			}
		}
	}

	return activeProcessInstances
}

/**
 * That function return the next process number...
 */
func (this *WorkflowProcessor) getNextProcessInstanceNumber() int {
	var number int

	// Now I will get all defintions names...
	var intancesQuery EntityQuery
	intancesQuery.TypeName = "BPMS_Runtime.ProcessInstance"

	intancesQuery.Fields = append(intancesQuery.Fields, "M_number")

	var filedsType []interface{} // not use...
	var params []interface{}
	query, _ := json.Marshal(intancesQuery)

	values, err := GetServer().GetDataManager().readData(BPMS_RuntimeDB, string(query), filedsType, params)

	if err == nil {
		for i := 0; i < len(values); i++ {
			n := values[i][0].(int)
			if n > number {
				number = n
			}
		}
	} else {
		log.Println("-------> process number error: ", err)
	}

	// The next number...
	number += 1

	return number
}

func (this *WorkflowProcessor) getProcessData(process *BPMN20.Process) []BPMN20.BaseElement {
	var data []BPMN20.BaseElement
	for i := 0; i < len(process.GetFlowElement()); i++ {
		flowElement := process.GetFlowElement()[i]
		if reflect.TypeOf(flowElement).String() == "*BPMN20.DataObject" {
			data = append(data, flowElement.(BPMN20.BaseElement))
		}
	}

	// Apppend process properties...
	for i := 0; i < len(process.GetProperty()); i++ {
		data = append(data, process.GetProperty()[i])
	}

	return data
}

func (this *WorkflowProcessor) getProcessDataInput(process *BPMN20.Process) []BPMN20.BaseElement {
	data := this.getProcessData(process)
	if process.GetIoSpecification() != nil {
		for i := 0; i < len(process.GetIoSpecification().GetDataInput()); i++ {
			data = append(data, process.GetIoSpecification().GetDataInput()[i])
		}
	}

	return data
}

func (this *WorkflowProcessor) getProcessDataOutput(process *BPMN20.Process) []BPMN20.BaseElement {
	data := this.getProcessData(process)
	if process.GetIoSpecification() != nil {
		for i := 0; i < len(process.GetIoSpecification().GetDataOutput()); i++ {
			data = append(data, process.GetIoSpecification().GetDataOutput()[i])
		}
	}
	return data
}

/**
 * Set output data association
 */
func (this *WorkflowProcessor) setDataAssocication(source BPMS_Runtime.Instance, target BPMS_Runtime.Instance, dataAssociations []BPMN20.DataAssociation, data []BPMN20.BaseElement) {
	// Index the item aware in a map to access it by theire id's
	srcItemawareElementMap := make(map[string]*BPMS_Runtime.ItemAwareElementInstance, 0)

	// I will copy the data reference into the instance.
	for i := 0; i < len(source.GetDataRef()); i++ {
		srcItemawareElementMap[source.GetDataRef()[i].M_id] = source.GetDataRef()[i]
	}

	for i := 0; i < len(source.GetData()); i++ {
		srcItemawareElementMap[source.GetData()[i].M_id] = source.GetData()[i]
	}

	trgItemawareElementMap := make(map[string]BPMN20.ItemAwareElement)

	// The data to be set...
	for i := 0; i < len(data); i++ {
		trgItemawareElementMap[data[i].GetId()] = data[i].(BPMN20.ItemAwareElement)
	}

	// Set the data from the source instance to the
	// target instance.
	for i := 0; i < len(dataAssociations); i++ {
		dataAssociation := dataAssociations[i]
		sourceRefs := dataAssociation.GetSourceRef()
		targetRef := dataAssociation.GetTargetRef()
		for j := 0; j < len(sourceRefs); j++ {
			sourceRef := sourceRefs[j]
			if itemawareElementSrc, ok := srcItemawareElementMap[sourceRef.(BPMN20.BaseElement).GetId()]; ok {
				if _, ok := trgItemawareElementMap[targetRef.(BPMN20.BaseElement).GetId()]; ok {
					var itemawareElementTrg *BPMS_Runtime.ItemAwareElementInstance
					// I will try to find the item in the item...
					for i := 0; i < len(target.GetData()); i++ {
						if target.GetData()[i].M_id == targetRef.(BPMN20.BaseElement).GetId() {
							itemawareElementTrg = target.GetData()[i]
						}
					}
					for i := 0; i < len(target.GetDataRef()); i++ {
						if target.GetDataRef()[i].M_id == targetRef.(BPMN20.BaseElement).GetId() {
							itemawareElementTrg = target.GetDataRef()[i]
						}
					}
					if itemawareElementTrg == nil {
						itemawareElementTrg = new(BPMS_Runtime.ItemAwareElementInstance)
						itemawareElementTrg.UUID = "BPMS_Runtime.ItemAwareElementInstance%" + Utility.RandomUUID()
						itemawareElementTrg.M_bpmnElementId = itemawareElementSrc.M_bpmnElementId
						itemawareElementTrg.M_id = targetRef.(BPMN20.BaseElement).GetId()
						target.SetData(itemawareElementTrg)
					}
					itemawareElementTrg.SetData(itemawareElementSrc.M_data)
				}
			}
		}
	}
}

func (this *WorkflowProcessor) setLogInfo(instance BPMS_Runtime.FlowNodeInstance, descripion string, sessionId string) {

	// Now the log information...
	logInfo := new(BPMS_Runtime.LogInfo)
	logInfo.UUID = "BPMS_Runtime.LogInfo%" + Utility.RandomUUID()
	logInfo.SetRuntimesPtr(this.runtime.object)
	logInfo.M_date = Utility.MakeTimestamp()
	logInfo.M_id = Utility.RandomUUID()
	logInfo.SetObject(instance)

	if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_AbstractTask {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_ServiceTask {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_ManualTask {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_BusinessRuleTask {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_ScriptTask {
		logInfo.M_action = "Run Script" // Start a new process...
	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_EmbeddedSubprocess {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_EventSubprocess {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_AdHocSubprocess {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_Transaction {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_CallActivity {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_ParallelGateway {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_ExclusiveGateway {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_EventBasedGateway {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_ComplexGateway {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_StartEvent {
		logInfo.M_action = "Start Process" // Start a new process...
	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_IntermediateCatchEvent {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_BoundaryEvent {

	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_EndEvent {
		logInfo.M_action = "End Process" // Start a new process...
	} else if instance.GetFlowNodeType() == BPMS_Runtime.FlowNodeType_IntermediateThrowEvent {

	}

	if len(sessionId) > 0 {
		// Set the actor...
		session := GetServer().GetSessionManager().GetActiveSessionById(sessionId)
		// Set the account pointer to the session...
		logInfo.SetActor(session.GetAccountPtr().GetUUID())
	} else {
		// The runtime is the actor in that case.
		logInfo.SetActor(this.runtime.GetUuid())
	}

	// Set as a reference
	instance.(BPMS_Runtime.Instance).SetLogInfoRef(logInfo)

	// The parent of the logInfo...
	this.runtime.object.SetLogInfos(logInfo)

}

////////////////////////////////////////////////////////////////////////////////
// Api.
////////////////////////////////////////////////////////////////////////////////

/**
 * That function create a new Item aware element instance for a given bpmn element
 * id, who is an instance of ItemAwareElement.
 */
func (this *WorkflowProcessor) NewItemAwareElementInstance(bpmnElementId string, data interface{}, messageId string, sessionId string) *BPMS_Runtime.ItemAwareElementInstance {
	instance, err := this.createItemAwareElementInstance(bpmnElementId, data)

	// Retunr an error in that case...
	if err != nil {
		GetServer().reportErrorMessage(messageId, sessionId, err)
		return nil
	}

	return instance
}
